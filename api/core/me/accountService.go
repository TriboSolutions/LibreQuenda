package me

import (
	"context"
	"errors"
	"librequenda/shared"
	"librequenda/zitadel"
	"net/http"
)

// The link in Zitadel's verification email opens /perfil, which verifies the code itself.
func verifyEmailLink() string {
	return shared.PUBLIC_URL + "/perfil?verificar_correo={{.Code}}"
}

// accountError explains a Zitadel refusal to the resident; anything else is a failure.
func accountError(err error, failed string) *shared.ErrorResponse {
	switch {
	case errors.Is(err, zitadel.ErrWrongPassword):
		return &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.current_password_wrong"}
	case errors.Is(err, zitadel.ErrInvalidCode):
		return &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.invalid_verification_code"}
	case errors.Is(err, zitadel.ErrRejected):
		return &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: failed}
	}
	shared.Logger.Error("zitadel account: %v", err)
	return &shared.ErrorResponse{Code: http.StatusBadGateway, Message: "errors.account_unavailable"}
}

// GetAccount reads the account from Zitadel. userId is the token's subject: residents only reach their own.
func (s *meService) GetAccount(ctx context.Context, userId string) (*AccountResponse, *shared.ErrorResponse) {
	var account, err = zitadel.GetAccount(ctx, userId)
	if err != nil {
		return nil, accountError(err, "errors.account_unavailable")
	}
	var rules *zitadel.PasswordRules
	if rules, err = zitadel.GetPasswordRules(ctx); err != nil {
		return nil, accountError(err, "errors.account_unavailable")
	}
	return &AccountResponse{
		GivenName:     account.GivenName,
		FamilyName:    account.FamilyName,
		Email:         account.Email,
		EmailVerified: account.EmailVerified,
		PasswordRules: rules,
	}, nil
}

func (s *meService) SetName(ctx context.Context, userId string, input NameInput) *shared.ErrorResponse {
	if key := input.validate(); key != "" {
		return &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	if err := zitadel.SetName(ctx, userId, input.GivenName, input.FamilyName); err != nil {
		return accountError(err, "errors.invalid_name")
	}
	return nil
}

func (s *meService) SetPassword(ctx context.Context, userId string, input PasswordInput) *shared.ErrorResponse {
	if input.CurrentPassword == "" {
		return &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.current_password_wrong"}
	}
	var rules, err = zitadel.GetPasswordRules(ctx)
	if err != nil {
		return accountError(err, "errors.account_unavailable")
	}
	if !rules.Meets(input.NewPassword) {
		return &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.password_too_weak"}
	}
	if err = zitadel.SetPassword(ctx, userId, input.CurrentPassword, input.NewPassword); err != nil {
		return accountError(err, "errors.password_too_weak")
	}
	return nil
}

// SetEmail changes the address and emails it a code. Until it is verified, roles
// waiting for this person are not granted (SyncCurrentUser).
func (s *meService) SetEmail(ctx context.Context, userId string, input EmailInput) *shared.ErrorResponse {
	if key := input.validate(); key != "" {
		return &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	if err := zitadel.SetEmail(ctx, userId, input.Email, verifyEmailLink()); err != nil {
		return accountError(err, "errors.invalid_email")
	}
	return nil
}

func (s *meService) ResendEmailCode(ctx context.Context, userId string) *shared.ErrorResponse {
	if err := zitadel.ResendEmailCode(ctx, userId, verifyEmailLink()); err != nil {
		return accountError(err, "errors.account_unavailable")
	}
	return nil
}

func (s *meService) VerifyEmail(ctx context.Context, userId string, input VerifyEmailInput) *shared.ErrorResponse {
	if key := input.validate(); key != "" {
		return &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	if err := zitadel.VerifyEmail(ctx, userId, input.Code); err != nil {
		return accountError(err, "errors.invalid_verification_code")
	}
	return nil
}
