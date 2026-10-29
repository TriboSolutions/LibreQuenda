package counciladmin

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/branding"
	"librequenda/db"
	"librequenda/mailer"
	"librequenda/model"
	"librequenda/secrets"
	"librequenda/shared"
	"net/http"
	"strconv"
	"strings"

	"github.com/angelbarreiros/Penguin/router/helpers"
	"github.com/google/uuid"
)

// savedRelay: the council's own relay, opened (nil when it has none).
func savedRelay(ctx context.Context, queries *model.Queries, tenantId uuid.UUID) (*mailer.Relay, error) {
	var row, err = queries.GetTenantSmtp(ctx, tenantId)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return mailer.OpenRelay(row.Config, tenantId)
}

// GetSmtp: the council's relay, without the password.
func (s *counciladminService) GetSmtp(ctx context.Context, tenantId uuid.UUID) (*SmtpDto, *shared.ErrorResponse) {
	var _, envRelay = mailer.EnvRelay()
	var row, err = model.New(s.db).GetTenantSmtp(ctx, tenantId)
	if errors.Is(err, sql.ErrNoRows) {
		return &SmtpDto{Port: 587, EnvRelay: envRelay}, nil
	}
	if err != nil {
		return nil, failed("errors.smtp_load_failed")
	}
	var relay *mailer.Relay
	if relay, err = mailer.OpenRelay(row.Config, tenantId); err != nil {
		// sealed with another key: the council has to type its settings again
		shared.Logger.Error("council %s SMTP settings: %v", tenantId, err)
		return &SmtpDto{Configured: true, Unreadable: true, Port: 587, UpdatedAt: &row.UpdatedAt, EnvRelay: envRelay}, nil
	}
	var port, _ = strconv.Atoi(relay.Port)
	return &SmtpDto{
		Configured:  true,
		Host:        relay.Host,
		Port:        int32(port),
		Username:    relay.Username,
		HasPassword: relay.Password != "",
		From:        relay.From,
		UpdatedAt:   &row.UpdatedAt,
		EnvRelay:    envRelay,
	}, nil
}

// SaveSmtp sets the council's own relay: its emails stop using the environment's.
func (s *counciladminService) SaveSmtp(ctx context.Context, tenantId uuid.UUID, input SmtpInput) (*SmtpDto, *shared.ErrorResponse) {
	if key := input.validate(); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	var err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		var relay = mailer.Relay{Host: input.Host, Port: strconv.Itoa(int(input.Port)), Username: input.Username, Password: input.Password, From: input.From}
		if relay.Username == "" {
			relay.Password = ""
		} else if relay.Password == "" {
			// keep the saved one; settings sealed with another key have none to keep
			if saved, openErr := savedRelay(ctx, queries, tenantId); openErr == nil && saved != nil {
				relay.Password = saved.Password
			}
		}
		var config, sealErr = mailer.SealRelay(relay, tenantId)
		if sealErr != nil {
			return sealErr
		}
		return queries.UpsertTenantSmtp(ctx, model.UpsertTenantSmtpParams{TenantID: tenantId, Config: config})
	})
	if errors.Is(err, secrets.ErrNoKey) {
		shared.Logger.Error("council SMTP settings not saved: %v", err)
		return nil, failed("errors.secrets_key_missing")
	}
	if err != nil {
		return nil, failed("errors.smtp_save_failed")
	}
	return s.GetSmtp(ctx, tenantId)
}

// DeleteSmtp: back to the environment's relay.
func (s *counciladminService) DeleteSmtp(ctx context.Context, tenantId uuid.UUID) *shared.ErrorResponse {
	var err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		var _, deleteErr = queries.DeleteTenantSmtp(ctx, tenantId)
		return deleteErr
	})
	if err != nil {
		return failed("errors.smtp_save_failed")
	}
	return nil
}

// TestSmtp sends a test email to the administrator, through the relay the
// council's emails would use right now.
func (s *counciladminService) TestSmtp(ctx context.Context, tenantId uuid.UUID, userId uuid.UUID, language string) (*SmtpTestDto, *shared.ErrorResponse) {
	var queries = model.New(s.db)
	var council, err = savedRelay(ctx, queries, tenantId)
	if err != nil {
		shared.Logger.Error("council %s SMTP settings: %v", tenantId, err)
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.smtp_password_unreadable"}
	}
	var via = "environment"
	if council != nil {
		via = "council"
	}
	var relay mailer.Relay
	if relay, err = mailer.Pick(council); err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.smtp_not_configured"}
	}
	var profile model.GetUserProfileRow
	if profile, err = queries.GetUserProfile(ctx, userId); err != nil {
		return nil, failed("errors.smtp_load_failed")
	}
	var councilName string
	if councilName, err = queries.GetTenantName(ctx, tenantId); err != nil {
		return nil, failed("errors.smtp_load_failed")
	}
	var t = func(key string, args ...any) string {
		return helpers.GetI18nInstance().TranslateWithLanguageWithVars("email.smtp_test."+key, language, args...)
	}
	var appName = branding.Name(ctx, s.db)
	var body = t("body", councilName, relay.Host, appName)
	err = mailer.Send(relay, mailer.Message{
		To:      profile.Email,
		Subject: t("subject", councilName, appName),
		Text:    body,
		HTML:    "<p>" + htmlEscape(body) + "</p>",
	})
	if err != nil {
		shared.Logger.Error("SMTP test for council %s: %v", tenantId, err)
		return nil, &shared.ErrorResponse{Code: http.StatusBadGateway, Message: "errors.smtp_test_failed", Args: []any{err.Error()}}
	}
	return &SmtpTestDto{To: profile.Email, Via: via}, nil
}

var htmlEscape = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace
