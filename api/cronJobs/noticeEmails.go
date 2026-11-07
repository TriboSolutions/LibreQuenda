package cronJobs

import (
	"bytes"
	"context"
	"database/sql"
	"librequenda/branding"
	"librequenda/mailer"
	"librequenda/model"
	"librequenda/shared"
	"log"
	"strings"
	"time"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

// SendNoticeEmails sends the councils' announcements to who was told in the
// bell, unless they turned notices off or the announcement expired meanwhile.
func SendNoticeEmails(ctx context.Context, db *sql.DB) (int, error) {
	var queries = model.New(db)
	var _, envRelay = mailer.EnvRelay()
	var rows, err = queries.ListPendingNoticeEmails(ctx, model.ListPendingNoticeEmailsParams{EnvRelay: envRelay, MaxRows: notificationEmailBatch})
	if err != nil {
		return 0, err
	}
	var sent int
	var now = time.Now()
	var brand = branding.ForEmail(ctx, db)
	for _, row := range rows {
		if !row.EmailNotifications || !row.ExpiresAt.After(now) {
			if err = queries.MarkNotificationEmailed(ctx, row.ID); err != nil {
				return sent, err
			}
			continue
		}
		var council, councilErr = mailer.OpenRelay(row.SmtpConfig, row.TenantID)
		if err = send(council, councilErr, noticeEmail(row, brand)); err != nil {
			log.Printf("notice email %s: %v", row.ID, err)
			continue
		}
		sent++
		if err = queries.MarkNotificationEmailed(ctx, row.ID); err != nil {
			return sent, err
		}
	}
	return sent, nil
}

func noticeEmail(row model.ListPendingNoticeEmailsRow, brand branding.Email) mailer.Message {
	var i18n = helpers.GetI18nInstance()
	var language = row.PreferredLanguage.String // "" = the default (gl)
	var t = func(key string, args ...any) string {
		return i18n.TranslateWithLanguageWithVars("email."+key, language, args...)
	}
	var view = emailView{
		Greeting:  t("greeting_anonymous"),
		Intro:     t("council_notice.intro", row.TenantName),
		Heading:   row.Title,
		Body:      row.Body,
		Manage:    t("council_notice.manage"),
		ManageURL: shared.PUBLIC_URL + "/" + row.TenantSlug,
		Footer:    t("council_notice.footer", brand.Name, row.TenantName),
	}
	if row.FacilityName != "" {
		view.Intro = t("council_notice.intro_facility", row.TenantName, row.FacilityName)
		view.ManageURL += "/" + row.FacilitySlug
	}
	if name := strings.Fields(row.FullName.String); len(name) > 0 {
		view.Greeting = t("greeting", name[0])
	}
	var html bytes.Buffer
	brandHeader(&view, brand)
	_ = emailTemplate.Execute(&html, view) // the template is fixed and checked by the tests
	var text = []string{view.Greeting, "", view.Intro, "", view.Heading, "", view.Body, "", view.Manage + ": " + view.ManageURL, "", "--", view.Footer}
	var message = mailer.Message{
		To:      row.Email,
		Subject: t("council_notice.subject", row.TenantName, row.Title),
		Text:    strings.Join(text, "\n"),
		HTML:    html.String(),
	}
	attachLogo(&message, brand)
	return message
}

// SendPollEmails asks the council's people by email too, while the poll is open,
// and tells them when its results are out (the polls that closed since the last
// run are found first: NotifyPollResults).
func SendPollEmails(ctx context.Context, db *sql.DB) (int, error) {
	var queries = model.New(db)
	if _, err := queries.NotifyPollResults(ctx); err != nil {
		return 0, err
	}
	var _, envRelay = mailer.EnvRelay()
	var rows, err = queries.ListPendingPollEmails(ctx, model.ListPendingPollEmailsParams{EnvRelay: envRelay, MaxRows: notificationEmailBatch})
	if err != nil {
		return 0, err
	}
	var sent int
	var brand = branding.ForEmail(ctx, db)
	for _, row := range rows {
		var stale = row.Type == model.NotificationTypePollOpened && !row.IsOpen // too late to vote
		if !row.EmailNotifications || stale {
			if err = queries.MarkNotificationEmailed(ctx, row.ID); err != nil {
				return sent, err
			}
			continue
		}
		var council, councilErr = mailer.OpenRelay(row.SmtpConfig, row.TenantID)
		if err = send(council, councilErr, pollEmail(row, brand)); err != nil {
			log.Printf("poll email %s: %v", row.ID, err)
			continue
		}
		sent++
		if err = queries.MarkNotificationEmailed(ctx, row.ID); err != nil {
			return sent, err
		}
	}
	return sent, nil
}

func pollEmail(row model.ListPendingPollEmailsRow, brand branding.Email) mailer.Message {
	var i18n = helpers.GetI18nInstance()
	var language = row.PreferredLanguage.String // "" = the default (gl)
	var t = func(key string, args ...any) string {
		return i18n.TranslateWithLanguageWithVars("email."+key, language, args...)
	}
	var zone, zoneErr = time.LoadLocation(row.Timezone)
	if zoneErr != nil {
		zone = time.UTC
	}
	var view = emailView{
		Greeting:  t("greeting_anonymous"),
		Intro:     t("poll_opened.intro", row.TenantName),
		Heading:   row.Question,
		Body:      t("poll_opened.closes", row.ClosesAt.In(zone).Format("02/01/2006 15:04")),
		Manage:    t("poll_opened.manage"),
		ManageURL: shared.PUBLIC_URL + "/enquisas?enquisa=" + row.PollID.String(),
		Footer:    t("poll_opened.footer", brand.Name, row.TenantName),
	}
	if row.FacilityName != "" {
		view.Intro = t("poll_opened.intro_facility", row.TenantName, row.FacilityName)
	}
	var subject = t("poll_opened.subject", row.TenantName, row.Question)
	if row.Type == model.NotificationTypePollClosed {
		view.Intro, view.Body, view.Manage = t("poll_closed.intro", row.TenantName), t("poll_closed.body"), t("poll_closed.manage")
		subject = t("poll_closed.subject", row.Question)
	}
	if name := strings.Fields(row.FullName.String); len(name) > 0 {
		view.Greeting = t("greeting", name[0])
	}
	var html bytes.Buffer
	brandHeader(&view, brand)
	_ = emailTemplate.Execute(&html, view) // the template is fixed and checked by the tests
	var text = []string{view.Greeting, "", view.Intro, "", view.Heading, "", view.Body, "", view.Manage + ": " + view.ManageURL, "", "--", view.Footer}
	var message = mailer.Message{
		To:      row.Email,
		Subject: subject,
		Text:    strings.Join(text, "\n"),
		HTML:    html.String(),
	}
	attachLogo(&message, brand)
	return message
}
