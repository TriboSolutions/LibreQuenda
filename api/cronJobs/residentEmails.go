package cronJobs

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"librequenda/branding"
	"librequenda/mailer"
	"librequenda/model"
	"librequenda/shared"
	"log"
	"strings"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

// SendResidentEmails: the council's administrators learn by email that
// someone asks to be verified as a resident (never with the DNI or address:
// those are seen in the Veciños tab), and the person learns how it decided,
// or that it took the verification back (and how many bookings went with it).
func SendResidentEmails(ctx context.Context, db *sql.DB) (int, error) {
	var queries = model.New(db)
	var _, envRelay = mailer.EnvRelay()
	var rows, err = queries.ListPendingResidentEmails(ctx, model.ListPendingResidentEmailsParams{EnvRelay: envRelay, MaxRows: notificationEmailBatch})
	if err != nil {
		return 0, err
	}
	var sent int
	var brand = branding.ForEmail(ctx, db)
	for _, row := range rows {
		if !row.EmailNotifications {
			if err = queries.MarkNotificationEmailed(ctx, row.ID); err != nil {
				return sent, err
			}
			continue
		}
		var council, councilErr = mailer.OpenRelay(row.SmtpConfig, row.TenantID)
		if err = send(council, councilErr, residentEmail(row, brand)); err != nil {
			log.Printf("resident email %s: %v", row.ID, err)
			continue
		}
		sent++
		if err = queries.MarkNotificationEmailed(ctx, row.ID); err != nil {
			return sent, err
		}
	}
	return sent, nil
}

func residentEmail(row model.ListPendingResidentEmailsRow, brand branding.Email) mailer.Message {
	var appName = brand.Name
	var i18n = helpers.GetI18nInstance()
	var language = row.PreferredLanguage.String // "" = the default (gl)
	var t = func(key string, args ...any) string {
		return i18n.TranslateWithLanguageWithVars("email."+key, language, args...)
	}
	var kind = string(row.Type)
	var notice struct {
		Name      string `json:"name"`
		Cancelled int    `json:"cancelled"` // resident_revoked: the bookings that went with it
	}
	_ = json.Unmarshal(row.Payload, &notice)
	if notice.Name == "" {
		notice.Name = t("someone")
	}
	var view = emailView{
		Greeting:  t("greeting_anonymous"),
		Intro:     t(kind+".intro", row.TenantName, notice.Name),
		Manage:    t(kind + ".manage"),
		ManageURL: shared.PUBLIC_URL + "/" + row.TenantSlug,
		Footer:    t("resident_footer", appName),
	}
	if row.Type == model.NotificationTypeResidentRequested {
		view.ManageURL = shared.PUBLIC_URL + "/" + row.TenantSlug + "/admin?vista=vecinos"
		view.Footer = t("issue_reported.footer", appName)
	}
	if row.Type == model.NotificationTypeResidentRevoked {
		view.Footer = t("resident_revoked.footer", appName)
		if notice.Cancelled > 0 {
			view.Reason = t("resident_revoked.cancelled", notice.Cancelled)
		}
	}
	if row.Type == model.NotificationTypeResidentRejected && row.TenantPhone.String != "" {
		view.Reason = t("resident_rejected.phone", row.TenantPhone.String)
	}
	if name := strings.Fields(row.FullName.String); len(name) > 0 {
		view.Greeting = t("greeting", name[0])
	}
	var html bytes.Buffer
	brandHeader(&view, brand)
	_ = emailTemplate.Execute(&html, view) // the template is fixed and checked by the tests
	var text = []string{view.Greeting, "", view.Intro}
	if view.Reason != "" {
		text = append(text, "", view.Reason)
	}
	text = append(text, "", view.Manage+": "+view.ManageURL, "", "--", view.Footer)
	var message = mailer.Message{
		To:      row.Email,
		Subject: t(kind+".subject", row.TenantName, notice.Name),
		Text:    strings.Join(text, "\n"),
		HTML:    html.String(),
	}
	attachLogo(&message, brand)
	return message
}
