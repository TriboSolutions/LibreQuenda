package analytics

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"librequenda/model"
	"librequenda/secrets"
)

// keyOwner: what the model's API key is sealed for (it opens with nothing else).
var keyOwner = []byte("ai_settings")

// Settings: the language model, as a superadministrator sees and sets it (never the key itself).
type Settings struct {
	Enabled   bool       `json:"enabled"`
	Provider  string     `json:"provider"` // openai (any OpenAI-compatible API) or anthropic
	BaseUrl   string     `json:"base_url"`
	Model     string     `json:"model"`
	HasKey    bool       `json:"has_key"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// SettingsInput: what the form sends; an empty api_key keeps the one saved.
type SettingsInput struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"`
	BaseUrl  string `json:"base_url"`
	Model    string `json:"model"`
	ApiKey   string `json:"api_key"`
}

// Validate tidies the input and answers the i18n key of the first problem, or "".
func (in *SettingsInput) Validate() string {
	in.Provider = strings.TrimSpace(in.Provider)
	in.BaseUrl = strings.TrimRight(strings.TrimSpace(in.BaseUrl), "/")
	in.Model = strings.TrimSpace(in.Model)
	in.ApiKey = strings.TrimSpace(in.ApiKey)
	switch {
	case in.Provider != "openai" && in.Provider != "anthropic":
		return "errors.ai_invalid_provider"
	case !validBaseURL(in.BaseUrl):
		return "errors.ai_invalid_url"
	case in.Model == "" || utf8.RuneCountInString(in.Model) > 100:
		return "errors.ai_invalid_model"
	case utf8.RuneCountInString(in.ApiKey) > 500:
		return "errors.ai_invalid_key"
	}
	return ""
}

// validBaseURL: http(s), with a host, no credentials, query or fragment; at most 255 characters.
func validBaseURL(raw string) bool {
	if raw == "" || len(raw) > 255 {
		return false
	}
	var u, err = url.Parse(raw)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" && u.User == nil && u.RawQuery == "" && u.Fragment == ""
}

// GetSettings: the model as saved; the defaults (off) before anyone set it.
func GetSettings(ctx context.Context, db *sql.DB) (*Settings, error) {
	var row, err = model.New(db).GetAISettings(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return &Settings{Provider: "openai"}, nil
	}
	if err != nil {
		return nil, err
	}
	var updated = row.UpdatedAt
	return &Settings{Enabled: row.Enabled, Provider: row.Provider, BaseUrl: row.BaseUrl, Model: row.Model, HasKey: len(row.ApiKey) > 0, UpdatedAt: &updated}, nil
}

// SaveSettings stores the model, the key sealed (an empty one keeps the saved key).
func SaveSettings(ctx context.Context, db *sql.DB, in SettingsInput) error {
	var params = model.SaveAISettingsParams{Enabled: in.Enabled, Provider: in.Provider, BaseUrl: in.BaseUrl, Model: in.Model}
	if in.ApiKey != "" {
		var sealed, err = secrets.Seal(in.ApiKey, keyOwner)
		if err != nil {
			return err
		}
		params.ApiKey = sealed
	}
	return model.New(db).SaveAISettings(ctx, params)
}

// llm: the model the assistant talks to, key opened; ErrNotConfigured when it is off or unset.
type llm struct {
	provider, baseURL, model, key string
}

var ErrNotConfigured = errors.New("the analytics assistant is off: a superadministrator sets its language model in the platform settings")

func loadLLM(ctx context.Context, db *sql.DB) (*llm, error) {
	var row, err = model.New(db).GetAISettings(ctx)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && (!row.Enabled || row.BaseUrl == "" || row.Model == "")) {
		return nil, ErrNotConfigured
	}
	if err != nil {
		return nil, err
	}
	var key string
	if len(row.ApiKey) > 0 {
		if key, err = secrets.Open(row.ApiKey, keyOwner); err != nil {
			return nil, errors.New("the model's API key cannot be read (saved with another SECRETS_KEY): type it again in the platform settings")
		}
	}
	return &llm{provider: row.Provider, baseURL: row.BaseUrl, model: row.Model, key: key}, nil
}
