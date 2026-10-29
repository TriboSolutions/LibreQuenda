package cronJobs

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"librequenda/branding"
	"librequenda/checkin"
	"librequenda/mailer"
	"librequenda/model"
	"librequenda/shared"
	"log"
	"strings"
	"time"

	"github.com/angelbarreiros/Penguin/router/helpers"
	"github.com/angelbarreiros/Penguin/scheduler"
)

const notificationEmailInterval = time.Minute
const notificationEmailBatch = 50

type notificationEmailJob struct {
	db *sql.DB
}

// registerNotificationEmailCronJob emails every notification: confirmations and
// reminders explain the facility's ways in (with the booking QR for a reader),
// cancellations say sorry.
func registerNotificationEmailCronJob(db *sql.DB, cronScheduler *scheduler.Scheduler) {
	if _, ok := mailer.EnvRelay(); !ok {
		log.Println("SMTP_HOST or SMTP_FROM not set: only councils with their own SMTP relay get emails")
	}
	var _, err = cronScheduler.ScheduleIntervalJob(
		notificationEmailInterval,
		scheduler.JobFunction(&notificationEmailJob{db: db}),
		scheduler.JobOptions{
			WaitForCompletion: true,
		},
	)
	if err != nil {
		log.Printf("failed to schedule notification email cron job: %v", err)
	}
}

func (j *notificationEmailJob) Execute() []any {
	var ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var sent, err = SendNotificationEmails(ctx, j.db)
	if err != nil {
		log.Printf("failed to send notification emails: %v", err)
		return []any{err}
	}
	var invitations int
	invitations, err = SendInvitationEmails(ctx, j.db)
	if err != nil {
		log.Printf("failed to send invitation emails: %v", err)
		return []any{err}
	}
	sent += invitations
	if sent > 0 {
		log.Printf("sent %d notification emails", sent)
	}
	return []any{sent}
}

// send goes through the council's relay, or else the environment's.
func send(council *mailer.Relay, councilErr error, message mailer.Message) error {
	if councilErr != nil {
		return councilErr
	}
	var relay, err = mailer.Pick(council)
	if err != nil {
		return err
	}
	return mailer.Send(relay, message)
}

// SendNotificationEmails works through the email queue. A notification leaves the
// queue once sent, or skipped when the email no longer makes sense (see shouldEmail).
// A failed send stays queued for the next run; so do the emails of a council with
// no relay while the environment has none either.
func SendNotificationEmails(ctx context.Context, db *sql.DB) (int, error) {
	var queries = model.New(db)
	var _, envRelay = mailer.EnvRelay()
	var rows, err = queries.ListPendingEmails(ctx, model.ListPendingEmailsParams{EnvRelay: envRelay, MaxRows: notificationEmailBatch})
	if err != nil {
		return 0, err
	}
	var sent int
	var now = time.Now()
	var appName = branding.Name(ctx, db)
	for _, row := range rows {
		if shouldEmail(row, now) {
			var message, buildErr = notificationEmail(row, appName, now)
			if buildErr == nil {
				var council, councilErr = mailer.OpenRelay(row.SmtpConfig, row.TenantID)
				buildErr = send(council, councilErr, message)
			}
			if buildErr != nil {
				log.Printf("notification email %s: %v", row.ID, buildErr)
				continue
			}
			sent++
		}
		if err = queries.MarkNotificationEmailed(ctx, row.ID); err != nil {
			return sent, err
		}
	}
	return sent, nil
}

// shouldEmail: reminders follow the profile switch and only make sense before the
// start; a confirmation of a booking already cancelled or over is dropped.
// Cancellations always go out.
func shouldEmail(row model.ListPendingEmailsRow, now time.Time) bool {
	var live = row.Status == model.BookingStatusConfirmed && row.EndsAt.After(now)
	switch row.Type {
	case model.NotificationTypeBookingReminder:
		return row.EmailReminders && live && row.StartsAt.After(now)
	case model.NotificationTypeBookingConfirmed, model.NotificationTypeBookingChanged:
		return live
	default:
		return true
	}
}

// usable: how to get in is explained while the booking can still be used.
func usable(row model.ListPendingEmailsRow, now time.Time) bool {
	return row.Status == model.BookingStatusConfirmed && row.EndsAt.After(now) &&
		row.Type != model.NotificationTypeBookingCancelled && row.Type != model.NotificationTypeBookingCancelledByCouncil
}

// waysIn: the facility's ways in, in the order the email explains them.
func waysIn(row model.ListPendingEmailsRow) []model.CheckinMethod {
	var ways []model.CheckinMethod
	for _, way := range []struct {
		on     bool
		method model.CheckinMethod
	}{
		{row.HasReader, model.CheckinMethodReader},
		{row.EntryCode, model.CheckinMethodEntryCode},
		{row.PrintedQr, model.CheckinMethodPrintedQr},
		{row.StaffConfirm, model.CheckinMethodStaff},
	} {
		if way.on {
			ways = append(ways, way.method)
		}
	}
	return ways
}

type emailView struct {
	Greeting, Intro, Reason, WaysTitle, Window, Address, Directions, Manage, ManageURL, Footer string
	DirectionsURL                                                                              string
	Ways                                                                                       []emailWay
	Contacts                                                                                   []emailContact
	HasQR                                                                                      bool // the booking QR, for a reader
}

// emailWay: one way to check in and how to use it; the reader's comes with the
// QR, the phone's with a link to where it is done.
type emailWay struct {
	Text, Link, LinkURL string
	HasQR               bool
}

// emailContact: who to call or write to, the facility or the council.
type emailContact struct {
	Label, Phone, Email string
	PhoneURL, EmailURL  template.URL // tel: and mailto:, which html/template would otherwise blank
}

// contact: nil when there is neither a phone nor an email.
func contact(label string, phone, email sql.NullString) *emailContact {
	var c = emailContact{Label: label, Phone: strings.TrimSpace(phone.String), Email: strings.TrimSpace(email.String)}
	if c.Phone == "" && c.Email == "" {
		return nil
	}
	if c.Phone != "" {
		c.PhoneURL = template.URL("tel:" + strings.Join(strings.Fields(c.Phone), ""))
	}
	if c.Email != "" {
		c.EmailURL = template.URL("mailto:" + c.Email)
	}
	return &c
}

// directionsURL: the route there in the visitor's own maps, as on the web (web/src/lib/maps.ts).
func directionsURL(latitude, longitude float64) string {
	return fmt.Sprintf("https://www.google.com/maps/dir/?api=1&destination=%v,%v", latitude, longitude)
}

// councilCancellation is what CancelBookingsInRange stores: why, and for a closure until when.
// bookings.StaffCancel stores by_staff instead: staff cancelled that one booking.
type councilCancellation struct {
	Reason      string     `json:"reason"`
	ClosedFrom  *time.Time `json:"closed_from"`
	ClosedUntil *time.Time `json:"closed_until"`
	ByStaff     bool       `json:"by_staff"`
}

var emailTemplate = template.Must(template.New("notification").Parse(`<!doctype html>
<html><body style="margin:0;padding:24px;background:#f6f9fb;font-family:system-ui,sans-serif;color:#111827;font-size:15px;line-height:22px">
<div style="max-width:480px;margin:0 auto;background:#ffffff;border:1px solid #e3e8ed;border-radius:8px;padding:24px">
<p style="margin:0 0 12px">{{.Greeting}}</p>
<p style="margin:0 0 16px">{{.Intro}}</p>
{{if .Reason}}<p style="margin:0 0 16px">{{.Reason}}</p>{{end}}
{{if .Ways}}<p style="margin:0 0 8px;font-weight:600">{{.WaysTitle}}</p>
<ul style="margin:0 0 12px;padding-left:20px">{{range .Ways}}<li style="margin:0 0 8px">{{.Text}}{{if .LinkURL}} <a href="{{.LinkURL}}" style="color:#0a6fb0">{{.Link}}</a>{{end}}{{if .HasQR}}<br><img src="cid:booking-qr" width="200" height="200" alt="QR" style="display:block;margin:8px 0 0">{{end}}</li>{{end}}</ul>
<p style="margin:0 0 16px;color:#5b6573">{{.Window}}</p>{{end}}
{{if or .Address .DirectionsURL}}<p style="margin:0 0 16px">{{if .Address}}<span style="color:#5b6573">{{.Address}}</span>{{end}}{{if and .Address .DirectionsURL}}<br>{{end}}{{if .DirectionsURL}}<a href="{{.DirectionsURL}}" style="color:#0a6fb0">{{.Directions}}</a>{{end}}</p>{{end}}
{{range .Contacts}}<p style="margin:0 0 12px"><span style="color:#5b6573">{{.Label}}</span>{{if .Phone}}<br><a href="{{.PhoneURL}}" style="color:#0a6fb0">{{.Phone}}</a>{{end}}{{if .Email}}<br><a href="{{.EmailURL}}" style="color:#0a6fb0">{{.Email}}</a>{{end}}</p>{{end}}
<p style="margin:0"><a href="{{.ManageURL}}" style="color:#0a6fb0">{{.Manage}}</a></p>
</div>
<p style="max-width:480px;margin:16px auto 0;color:#5b6573;font-size:12px;line-height:18px">{{.Footer}}</p>
</body></html>`))

// appName signs the footer: the installation's own name (branding).
func notificationEmail(row model.ListPendingEmailsRow, appName string, now time.Time) (mailer.Message, error) {
	var i18n = helpers.GetI18nInstance()
	var language = row.PreferredLanguage.String // "" = the default (gl)
	var t = func(key string, args ...any) string {
		return i18n.TranslateWithLanguageWithVars("email."+key, language, args...)
	}
	var kind = string(row.Type)

	var loc, err = time.LoadLocation(row.Timezone)
	if err != nil {
		loc = time.UTC
	}
	var date = row.StartsAt.In(loc).Format("02/01/2006")
	var start = row.StartsAt.In(loc).Format("15:04")
	var end = row.EndsAt.In(loc).Format("15:04")

	var view = emailView{
		Greeting:  t("greeting_anonymous"),
		Intro:     t(kind+".intro", row.FacilityName, row.TenantName, date, start, end),
		Manage:    t("manage"),
		ManageURL: shared.PUBLIC_URL + "/reservas",
		Footer:    t("footer", appName),
	}
	if row.Type == model.NotificationTypeBookingReminder {
		view.Footer = t("footer_reminders", shared.PUBLIC_URL+"/perfil")
	}
	if row.Type == model.NotificationTypeBookingCancelledByCouncil {
		var why councilCancellation
		_ = json.Unmarshal(row.Payload, &why) // older notices carry no reason: the plain text is still right
		if why.ByStaff {
			view.Intro = t(kind+".intro_staff", row.FacilityName, row.TenantName, date, start, end)
		}
		if why.ClosedFrom != nil && why.ClosedUntil != nil {
			const stamp = "02/01/2006 15:04"
			view.Intro = t(kind+".intro_closure", row.FacilityName, row.TenantName, date, start, end,
				why.ClosedFrom.In(loc).Format(stamp), why.ClosedUntil.In(loc).Format(stamp))
		}
		if why.Reason != "" {
			view.Reason = t("reason", why.Reason)
		}
	}
	if name := strings.Fields(row.FullName.String); len(name) > 0 {
		view.Greeting = t("greeting", name[0])
	}
	if usable(row, now) {
		var ways = waysIn(row)
		for _, way := range ways {
			var reader = way == model.CheckinMethodReader
			var item = emailWay{Text: t("checkin." + string(way)), HasQR: reader}
			if way == model.CheckinMethodEntryCode || way == model.CheckinMethodPrintedQr {
				item.Link, item.LinkURL = t("checkin.open"), shared.PUBLIC_URL+"/reservas"
			}
			view.Ways = append(view.Ways, item)
			view.HasQR = view.HasQR || reader
		}
		if len(ways) > 0 {
			view.WaysTitle = t("checkin.title_one")
			if len(ways) > 1 {
				view.WaysTitle = t("checkin.title_many")
			}
			view.Window = t("checkin.window")
		}
		if row.FacilityAddress.String != "" {
			view.Address = t("address", row.FacilityAddress.String)
		}
		if row.FacilityLatitude.Valid && row.FacilityLongitude.Valid {
			view.Directions, view.DirectionsURL = t("directions"), directionsURL(row.FacilityLatitude.Float64, row.FacilityLongitude.Float64)
		}
	}
	// who to ask: the facility's own contact, then the council's
	for _, c := range []*emailContact{
		contact(t("contact_facility", row.FacilityName), row.FacilityPhone, row.FacilityEmail),
		contact(t("contact_council", row.TenantName), row.TenantPhone, row.TenantEmail),
	} {
		if c != nil {
			view.Contacts = append(view.Contacts, *c)
		}
	}

	var html bytes.Buffer
	if err = emailTemplate.Execute(&html, view); err != nil {
		return mailer.Message{}, err
	}
	var text = []string{view.Greeting, "", view.Intro}
	if view.Reason != "" {
		text = append(text, "", view.Reason)
	}
	if len(view.Ways) > 0 {
		text = append(text, "", view.WaysTitle)
		for _, way := range view.Ways {
			var line = "- " + way.Text
			if way.LinkURL != "" {
				line += " " + way.Link + ": " + way.LinkURL
			}
			text = append(text, line)
		}
		text = append(text, view.Window)
	}
	if view.Address != "" {
		text = append(text, "", view.Address)
	}
	if view.DirectionsURL != "" {
		text = append(text, view.Directions+": "+view.DirectionsURL)
	}
	for _, c := range view.Contacts {
		var lines = []string{"", c.Label}
		if c.Phone != "" {
			lines = append(lines, c.Phone)
		}
		if c.Email != "" {
			lines = append(lines, c.Email)
		}
		text = append(text, lines...)
	}
	text = append(text, "", view.Manage+": "+view.ManageURL, "", "--", view.Footer)

	var message = mailer.Message{
		To:      row.Email,
		Subject: t(kind+".subject", row.FacilityName, date, start),
		Text:    strings.Join(text, "\n"),
		HTML:    html.String(),
	}
	if view.HasQR {
		var qr []byte
		if qr, err = checkin.PNG(row.CheckinToken, 480); err != nil {
			return mailer.Message{}, err
		}
		message.Inline = []mailer.Inline{{CID: "booking-qr", ContentType: "image/png", Filename: "reserva-qr.png", Data: qr}}
	}
	return message, nil
}

// SendInvitationEmails tells each new council administrator, whether or not they
// have an account yet. The role itself arrives on their first sign-in with that
// email verified, with a notice in the bell (AcceptPendingInvitations).
func SendInvitationEmails(ctx context.Context, db *sql.DB) (int, error) {
	var queries = model.New(db)
	var _, envRelay = mailer.EnvRelay()
	var rows, err = queries.ListPendingInvitationEmails(ctx, model.ListPendingInvitationEmailsParams{EnvRelay: envRelay, MaxRows: notificationEmailBatch})
	if err != nil {
		return 0, err
	}
	var sent int
	var appName = branding.Name(ctx, db)
	for _, row := range rows {
		var council, councilErr = mailer.OpenRelay(row.SmtpConfig, row.TenantID)
		if err = send(council, councilErr, invitationEmail(row, appName)); err != nil {
			log.Printf("invitation email %s: %v", row.ID, err)
			continue
		}
		sent++
		if err = queries.MarkInvitationEmailed(ctx, row.ID); err != nil {
			return sent, err
		}
	}
	return sent, nil
}

func invitationEmail(row model.ListPendingInvitationEmailsRow, appName string) mailer.Message {
	var i18n = helpers.GetI18nInstance()
	var language = row.PreferredLanguage.String // "" = the default (gl): they may have no account yet
	var t = func(key string, args ...any) string {
		return i18n.TranslateWithLanguageWithVars("email."+key, language, args...)
	}
	var view = emailView{
		Greeting:  t("greeting_anonymous"),
		Intro:     t("council_admin.intro", row.TenantName, row.Email, appName),
		Manage:    t("council_admin.manage"),
		ManageURL: shared.PUBLIC_URL + "/" + row.TenantSlug + "/admin",
		Footer:    t("council_admin.footer"),
	}
	if name := strings.Fields(row.FullName.String); len(name) > 0 {
		view.Greeting = t("greeting", name[0])
	}
	var html bytes.Buffer
	_ = emailTemplate.Execute(&html, view) // the template is fixed and checked by the tests
	return mailer.Message{
		To:      row.Email,
		Subject: t("council_admin.subject", row.TenantName),
		Text:    strings.Join([]string{view.Greeting, "", view.Intro, "", view.Manage + ": " + view.ManageURL, "", "--", view.Footer}, "\n"),
		HTML:    html.String(),
	}
}
