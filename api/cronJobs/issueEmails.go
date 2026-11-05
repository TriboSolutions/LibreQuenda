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

// SendIssueEmails tells each reporter, by email too, that the council solved
// or turned down the problem they reported, with the council's note. Like the
// booking notices, a notice leaves the queue once sent.
func SendIssueEmails(ctx context.Context, db *sql.DB) (int, error) {
	var queries = model.New(db)
	var _, envRelay = mailer.EnvRelay()
	var rows, err = queries.ListPendingIssueEmails(ctx, model.ListPendingIssueEmailsParams{EnvRelay: envRelay, MaxRows: notificationEmailBatch})
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
		if err = send(council, councilErr, issueEmail(row, brand)); err != nil {
			log.Printf("issue email %s: %v", row.ID, err)
			continue
		}
		sent++
		if err = queries.MarkNotificationEmailed(ctx, row.ID); err != nil {
			return sent, err
		}
	}
	return sent, nil
}

func issueEmail(row model.ListPendingIssueEmailsRow, brand branding.Email) mailer.Message {
	var appName = brand.Name
	var i18n = helpers.GetI18nInstance()
	var language = row.PreferredLanguage.String // "" = the default (gl)
	var t = func(key string, args ...any) string {
		return i18n.TranslateWithLanguageWithVars("email."+key, language, args...)
	}
	var kind = string(row.Type)
	var notice struct {
		Reason string `json:"reason"`
	}
	_ = json.Unmarshal(row.Payload, &notice) // without a note the email still says what happened
	var view = emailView{
		Greeting:  t("greeting_anonymous"),
		Intro:     t(kind+".intro", row.FacilityName, row.TenantName),
		Manage:    t("manage"),
		ManageURL: shared.PUBLIC_URL + "/reservas",
		Footer:    t("issue_footer", appName),
	}
	if row.Type == model.NotificationTypeIssueReported {
		view.Manage = t("issue_reported.manage")
		view.ManageURL = shared.PUBLIC_URL + "/" + row.TenantSlug + "/admin?vista=incidencias"
		view.Footer = t("issue_reported.footer", appName)
	}
	if notice.Reason != "" {
		view.Reason = t("issue_note", notice.Reason)
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
		Subject: t(kind+".subject", row.FacilityName),
		Text:    strings.Join(text, "\n"),
		HTML:    html.String(),
	}
	attachLogo(&message, brand)
	return message
}
