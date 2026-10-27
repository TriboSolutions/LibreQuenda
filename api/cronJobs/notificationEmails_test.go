package cronJobs

import (
	"librequenda/model"
	"strings"
	"testing"
	"time"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

func TestShouldEmailAndQR(t *testing.T) {
	var now = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	var soon = model.ListPendingEmailsRow{Status: model.BookingStatusConfirmed, StartsAt: now.Add(10 * time.Minute), EndsAt: now.Add(70 * time.Minute), EmailReminders: true}
	var with = func(change func(*model.ListPendingEmailsRow)) model.ListPendingEmailsRow {
		var row = soon
		change(&row)
		return row
	}
	var cases = []struct {
		name     string
		row      model.ListPendingEmailsRow
		send, qr bool
	}{
		{"confirmed", with(func(r *model.ListPendingEmailsRow) { r.Type = model.NotificationTypeBookingConfirmed }), true, true},
		{"confirmed, then cancelled", with(func(r *model.ListPendingEmailsRow) {
			r.Type = model.NotificationTypeBookingConfirmed
			r.Status = model.BookingStatusCancelled
		}), false, false},
		{"reminder", with(func(r *model.ListPendingEmailsRow) { r.Type = model.NotificationTypeBookingReminder }), true, true},
		{"reminder, reminders off", with(func(r *model.ListPendingEmailsRow) {
			r.Type = model.NotificationTypeBookingReminder
			r.EmailReminders = false
		}), false, true},
		{"reminder, already started", with(func(r *model.ListPendingEmailsRow) {
			r.Type = model.NotificationTypeBookingReminder
			r.StartsAt = now.Add(-time.Minute)
		}), false, true},
		{"cancelled by the council: always, no QR", with(func(r *model.ListPendingEmailsRow) {
			r.Type = model.NotificationTypeBookingCancelledByCouncil
			r.Status = model.BookingStatusCancelled
			r.EmailReminders = false
		}), true, false},
		{"cancelled by the user: no QR", with(func(r *model.ListPendingEmailsRow) {
			r.Type = model.NotificationTypeBookingCancelled
			r.Status = model.BookingStatusCancelled
		}), true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldEmail(c.row, now); got != c.send {
				t.Errorf("shouldEmail() = %v, want %v", got, c.send)
			}
			if got := withQR(c.row, now); got != c.qr {
				t.Errorf("withQR() = %v, want %v", got, c.qr)
			}
		})
	}
}

func TestNotificationEmailTexts(t *testing.T) {
	helpers.GetI18nInstance(helpers.SetDefaultLocale("gl"), helpers.SetDirectory("../i18n"))
	var now = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	var row = model.ListPendingEmailsRow{
		Status: model.BookingStatusCancelled, StartsAt: now.Add(48 * time.Hour), EndsAt: now.Add(49 * time.Hour),
		FacilityName: "Pista de tenis", TenantName: "Concello de Lalín", Timezone: "Europe/Madrid", Email: "a@example.com",
	}
	row.PreferredLanguage.String, row.PreferredLanguage.Valid = "en", true
	for _, kind := range []model.NotificationType{
		model.NotificationTypeBookingConfirmed, model.NotificationTypeBookingReminder, model.NotificationTypeBookingChanged,
		model.NotificationTypeBookingCancelled, model.NotificationTypeBookingCancelledByCouncil,
	} {
		row.Type = kind
		var message, err = notificationEmail(row, now)
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		if strings.Contains(message.Subject, "email.") || strings.Contains(message.Text, "email.") || !strings.Contains(message.Subject, "Pista de tenis") {
			t.Errorf("%s: missing translation: %q / %q", kind, message.Subject, message.Text)
		}
	}
	row.Type = model.NotificationTypeBookingCancelledByCouncil
	var message, _ = notificationEmail(row, now)
	if !strings.Contains(message.Text, "sorry") || len(message.Inline) != 0 {
		t.Errorf("council cancellation: %q, %d images", message.Text, len(message.Inline))
	}

	row.Payload = []byte(`{"reason":"Roof works","closed_from":"2026-11-15T00:00:00+01:00","closed_until":"2026-11-30T23:59:00+01:00"}`)
	message, _ = notificationEmail(row, now)
	if !strings.Contains(message.Text, "from 15/11/2026 00:00 to 30/11/2026 23:59") || !strings.Contains(message.Text, "Reason: Roof works") {
		t.Errorf("closure email lacks dates or reason: %q", message.Text)
	}
}

func TestInvitationEmail(t *testing.T) {
	helpers.GetI18nInstance(helpers.SetDefaultLocale("gl"), helpers.SetDirectory("../i18n"))
	var row = model.ListPendingInvitationEmailsRow{Email: "nova@example.com", TenantName: "Concello de Lalín", TenantSlug: "lalin"}
	var message = invitationEmail(row) // no account yet: Galician, no name
	if !strings.Contains(message.Subject, "Concello de Lalín") || !strings.Contains(message.Text, "nova@example.com") ||
		!strings.Contains(message.Text, "/lalin/admin") || !strings.HasPrefix(message.Text, "Ola,") || strings.Contains(message.Text, "email.") {
		t.Errorf("invitation email: %q / %q", message.Subject, message.Text)
	}
}
