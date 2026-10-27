// Package zitadel lets residents manage their own account (name, email, password)
// from LibreQuenda, so they never see Zitadel's console. It signs in as the
// "librequenda-api" service account, a user manager only (zitadel/setup.sh).
package zitadel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"librequenda/shared"
)

var (
	ErrUnavailable     = errors.New("zitadel: no service account token")
	ErrWrongPassword   = errors.New("zitadel: the current password is wrong")
	ErrRejected        = errors.New("zitadel: the request was rejected")       // a password or email Zitadel does not accept
	ErrInvalidCode     = errors.New("zitadel: the verification code is wrong") // or expired
	errUnexpectedReply = errors.New("zitadel: unexpected reply")
)

var client = &http.Client{Timeout: 10 * time.Second}

var tokenOnce sync.Once
var token string

// The token file is written by zitadel-setup, which runs before the API starts.
func serviceToken() string {
	tokenOnce.Do(func() {
		var content, err = os.ReadFile(shared.ZITADEL_USERS_PAT_FILE)
		if err != nil {
			shared.Logger.Error("zitadel service account token: %v", err)
			return
		}
		token = strings.TrimSpace(string(content))
	})
	return token
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func call(ctx context.Context, method string, path string, body any, out any) error {
	var bearer = serviceToken()
	if bearer == "" {
		return ErrUnavailable
	}
	var payload io.Reader
	if body != nil {
		var data, err = json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(data)
	}
	var req, err = http.NewRequestWithContext(ctx, method, shared.ZITADEL_INTERNAL_URL+path, payload)
	if err != nil {
		return err
	}
	req.Host = shared.ZITADEL_DOMAIN // Zitadel resolves the instance from the Host header
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Content-Type", "application/json")

	var res *http.Response
	if res, err = client.Do(req); err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		var reply apiError
		_ = json.NewDecoder(res.Body).Decode(&reply)
		return classify(res.StatusCode, reply)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}

// classify turns Zitadel's error into one the API can explain to the resident.
func classify(status int, reply apiError) error {
	switch {
	case strings.Contains(reply.Message, "COMMAND-3M0fs"): // "Password is invalid": the current one
		return ErrWrongPassword
	case strings.Contains(strings.ToLower(reply.Message), "code"): // wrong or expired verification code
		return ErrInvalidCode
	case status == http.StatusBadRequest || status == http.StatusConflict || status == http.StatusPreconditionFailed:
		return fmt.Errorf("%w: %s", ErrRejected, reply.Message)
	}
	return fmt.Errorf("%w: %d %s", errUnexpectedReply, status, reply.Message)
}

// --- account --------------------------------------------------------------

type Account struct {
	GivenName     string
	FamilyName    string
	Email         string
	EmailVerified bool
}

func GetAccount(ctx context.Context, userId string) (*Account, error) {
	var reply struct {
		User struct {
			Human struct {
				Profile struct {
					GivenName  string `json:"givenName"`
					FamilyName string `json:"familyName"`
				} `json:"profile"`
				Email struct {
					Email      string `json:"email"`
					IsVerified bool   `json:"isVerified"`
				} `json:"email"`
			} `json:"human"`
		} `json:"user"`
	}
	if err := call(ctx, http.MethodGet, "/v2/users/"+userId, nil, &reply); err != nil {
		return nil, err
	}
	var human = reply.User.Human
	return &Account{
		GivenName:     human.Profile.GivenName,
		FamilyName:    human.Profile.FamilyName,
		Email:         human.Email.Email,
		EmailVerified: human.Email.IsVerified,
	}, nil
}

func SetName(ctx context.Context, userId string, given string, family string) error {
	return call(ctx, http.MethodPut, "/v2/users/human/"+userId, map[string]any{
		"profile": map[string]string{"givenName": given, "familyName": family, "displayName": given + " " + family},
	}, nil)
}

// SetPassword changes it only if current is right (Zitadel checks it).
func SetPassword(ctx context.Context, userId string, current string, next string) error {
	return call(ctx, http.MethodPost, "/v2/users/"+userId+"/password", map[string]any{
		"newPassword":     map[string]any{"password": next, "changeRequired": false},
		"currentPassword": current,
	}, nil)
}

// SetEmail changes the address and emails a code to it; the link in that email
// opens linkTemplate, where Zitadel puts the code in place of {{.Code}}.
func SetEmail(ctx context.Context, userId string, email string, linkTemplate string) error {
	return call(ctx, http.MethodPost, "/v2/users/"+userId+"/email", map[string]any{
		"email":    email,
		"sendCode": map[string]string{"urlTemplate": linkTemplate},
	}, nil)
}

func ResendEmailCode(ctx context.Context, userId string, linkTemplate string) error {
	return call(ctx, http.MethodPost, "/v2/users/"+userId+"/email/resend", map[string]any{
		"sendCode": map[string]string{"urlTemplate": linkTemplate},
	}, nil)
}

func VerifyEmail(ctx context.Context, userId string, code string) error {
	return call(ctx, http.MethodPost, "/v2/users/"+userId+"/email/verify", map[string]string{"verificationCode": code}, nil)
}

// --- password rules -------------------------------------------------------

type PasswordRules struct {
	MinLength int  `json:"min_length"`
	Uppercase bool `json:"uppercase"`
	Lowercase bool `json:"lowercase"`
	Number    bool `json:"number"`
	Symbol    bool `json:"symbol"`
}

func GetPasswordRules(ctx context.Context) (*PasswordRules, error) {
	var reply struct {
		Settings struct {
			MinLength         string `json:"minLength"` // a string in Zitadel's JSON
			RequiresUppercase bool   `json:"requiresUppercase"`
			RequiresLowercase bool   `json:"requiresLowercase"`
			RequiresNumber    bool   `json:"requiresNumber"`
			RequiresSymbol    bool   `json:"requiresSymbol"`
		} `json:"settings"`
	}
	if err := call(ctx, http.MethodGet, "/v2/settings/password/complexity", nil, &reply); err != nil {
		return nil, err
	}
	var rules = &PasswordRules{
		Uppercase: reply.Settings.RequiresUppercase,
		Lowercase: reply.Settings.RequiresLowercase,
		Number:    reply.Settings.RequiresNumber,
		Symbol:    reply.Settings.RequiresSymbol,
	}
	fmt.Sscan(reply.Settings.MinLength, &rules.MinLength)
	return rules, nil
}

// Meets checks a new password before Zitadel does, to say which rule fails.
func (r *PasswordRules) Meets(password string) bool {
	var upper, lower, number, symbol bool
	for _, c := range password {
		switch {
		case c >= 'A' && c <= 'Z':
			upper = true
		case c >= 'a' && c <= 'z':
			lower = true
		case c >= '0' && c <= '9':
			number = true
		default:
			symbol = true
		}
	}
	return len([]rune(password)) >= r.MinLength && (!r.Uppercase || upper) && (!r.Lowercase || lower) && (!r.Number || number) && (!r.Symbol || symbol)
}
