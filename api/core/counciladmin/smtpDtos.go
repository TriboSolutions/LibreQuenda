package counciladmin

import (
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

// The council's own SMTP relay. The password is write-only: left empty, the
// saved one stays (while there is a username).
type SmtpInput struct {
	Host     string `json:"host"`
	Port     int32  `json:"port"` // 0 = 587
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"` // e.g. Concello de Lalín <reservas@lalin.gal>
}

// validate answers the i18n key of the problem, or "".
func (in *SmtpInput) validate() string {
	in.Host = strings.TrimSpace(in.Host)
	in.Username = strings.TrimSpace(in.Username)
	in.From = strings.TrimSpace(in.From)
	if in.Port == 0 {
		in.Port = 587
	}
	if in.Host == "" || len(in.Host) > 255 || strings.ContainsAny(in.Host, " /:@") {
		return "errors.invalid_smtp_host"
	}
	if in.Port < 1 || in.Port > 65535 {
		return "errors.invalid_smtp_port"
	}
	if utf8.RuneCountInString(in.Username) > 255 {
		return "errors.invalid_smtp_username"
	}
	if utf8.RuneCountInString(in.Password) > 255 {
		return "errors.invalid_smtp_password"
	}
	if _, err := mail.ParseAddress(in.From); err != nil || utf8.RuneCountInString(in.From) > 255 {
		return "errors.invalid_smtp_from"
	}
	return ""
}

type SmtpDto struct {
	Configured  bool       `json:"configured"` // false: the council's emails use the environment's relay
	Unreadable  bool       `json:"unreadable"` // saved with another SECRETS_KEY: to be typed again
	Host        string     `json:"host"`
	Port        int32      `json:"port"`
	Username    string     `json:"username"`
	HasPassword bool       `json:"has_password"`
	From        string     `json:"from"`
	UpdatedAt   *time.Time `json:"updated_at"`
	EnvRelay    bool       `json:"env_relay"` // the environment has a relay to fall back on
}

type SmtpTestDto struct {
	To  string `json:"to"`
	Via string `json:"via"` // council | environment
}
