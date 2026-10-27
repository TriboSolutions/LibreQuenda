package me

import (
	"librequenda/validate"
	"librequenda/zitadel"
	"strings"
	"unicode/utf8"
)

// AccountResponse: what /perfil shows and edits instead of Zitadel's console.
type AccountResponse struct {
	GivenName     string                 `json:"given_name"`
	FamilyName    string                 `json:"family_name"`
	Email         string                 `json:"email"`
	EmailVerified bool                   `json:"email_verified"`
	PasswordRules *zitadel.PasswordRules `json:"password_rules"`
}

type NameInput struct {
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
}

func (in *NameInput) validate() string {
	in.GivenName = strings.Join(strings.Fields(in.GivenName), " ")
	in.FamilyName = strings.Join(strings.Fields(in.FamilyName), " ")
	if in.GivenName == "" || in.FamilyName == "" || utf8.RuneCountInString(in.GivenName) > 100 || utf8.RuneCountInString(in.FamilyName) > 100 {
		return "errors.invalid_name"
	}
	return ""
}

type PasswordInput struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type EmailInput struct {
	Email string `json:"email"`
}

func (in *EmailInput) validate() string {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if !validate.Email(in.Email) {
		return "errors.invalid_email"
	}
	return ""
}

type VerifyEmailInput struct {
	Code string `json:"code"`
}

func (in *VerifyEmailInput) validate() string {
	in.Code = strings.TrimSpace(in.Code)
	if in.Code == "" || len(in.Code) > 32 {
		return "errors.invalid_verification_code"
	}
	return ""
}
