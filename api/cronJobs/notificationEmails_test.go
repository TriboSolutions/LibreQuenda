package cronJobs

import (
	"bytes"
	"database/sql"
	"librequenda/branding"
	"librequenda/model"
	"strings"
	"testing"
	"time"

	"github.com/angelbarreiros/Penguin/router/helpers"
	"github.com/google/uuid"
)

func TestShouldEmailAndUsable(t *testing.T) {
	var now = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	var soon = model.ListPendingEmailsRow{Status: model.BookingStatusConfirmed, StartsAt: now.Add(10 * time.Minute), EndsAt: now.Add(70 * time.Minute), EmailReminders: true, EmailNotifications: true}
	var with = func(change func(*model.ListPendingEmailsRow)) model.ListPendingEmailsRow {
		var row = soon
		change(&row)
		return row
	}
	var cases = []struct {
		name         string
		row          model.ListPendingEmailsRow
		send, usable bool
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
		{"cancelled by the council: always, no ways in", with(func(r *model.ListPendingEmailsRow) {
			r.Type = model.NotificationTypeBookingCancelledByCouncil
			r.Status = model.BookingStatusCancelled
			r.EmailReminders = false
		}), true, false},
		{"someone joined: emailed, without ways in", with(func(r *model.ListPendingEmailsRow) { r.Type = model.NotificationTypeBookingJoined }), true, false},
		{"emails turned off: none, not even a council cancellation", with(func(r *model.ListPendingEmailsRow) {
			r.Type = model.NotificationTypeBookingCancelledByCouncil
			r.EmailNotifications = false
		}), false, false},
		{"cancelled by the user: no ways in", with(func(r *model.ListPendingEmailsRow) {
			r.Type = model.NotificationTypeBookingCancelled
			r.Status = model.BookingStatusCancelled
		}), true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldEmail(c.row, now); got != c.send {
				t.Errorf("shouldEmail() = %v, want %v", got, c.send)
			}
			if got := usable(c.row, now); got != c.usable {
				t.Errorf("usable() = %v, want %v", got, c.usable)
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
		var message, err = notificationEmail(row, branding.Email{Name: "Reservas Lalín"}, now)
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		if strings.Contains(message.Subject, "email.") || strings.Contains(message.Text, "email.") || !strings.Contains(message.Subject, "Pista de tenis") {
			t.Errorf("%s: missing translation: %q / %q", kind, message.Subject, message.Text)
		}
		// the footer names the installation, not LibreQuenda (branding)
		if kind != model.NotificationTypeBookingReminder && (!strings.Contains(message.Text, "from Reservas Lalín") || strings.Contains(message.HTML, "LibreQuenda")) {
			t.Errorf("%s: footer does not name the app: %q", kind, message.Text)
		}
	}
	row.Type = model.NotificationTypeBookingCancelledByCouncil
	var message, _ = notificationEmail(row, branding.Email{Name: "Reservas Lalín"}, now)
	if !strings.Contains(message.Text, "sorry") || len(message.Inline) != 0 {
		t.Errorf("council cancellation: %q, %d images", message.Text, len(message.Inline))
	}

	row.Payload = []byte(`{"reason":"Roof works","closed_from":"2026-11-15T00:00:00+01:00","closed_until":"2026-11-30T23:59:00+01:00"}`)
	message, _ = notificationEmail(row, branding.Email{Name: "Reservas Lalín"}, now)
	if !strings.Contains(message.Text, "from 15/11/2026 00:00 to 30/11/2026 23:59") || !strings.Contains(message.Text, "Reason: Roof works") {
		t.Errorf("closure email lacks dates or reason: %q", message.Text)
	}
}

func TestInvitationEmail(t *testing.T) {
	helpers.GetI18nInstance(helpers.SetDefaultLocale("gl"), helpers.SetDirectory("../i18n"))
	var row = model.ListPendingInvitationEmailsRow{Email: "nova@example.com", TenantName: "Concello de Lalín", TenantSlug: "lalin"}
	var message = invitationEmail(row, branding.Email{Name: "Reservas Lalín"}) // no account yet: Galician, no name
	if !strings.Contains(message.Subject, "Concello de Lalín") || !strings.Contains(message.Text, "nova@example.com") ||
		!strings.Contains(message.Text, "/lalin/admin") || !strings.HasPrefix(message.Text, "Ola,") || strings.Contains(message.Text, "email.") ||
		!strings.Contains(message.Text, "en Reservas Lalín") {
		t.Errorf("invitation email: %q / %q", message.Subject, message.Text)
	}
}

func TestEmailExplainsOnlyTheFacilityWaysIn(t *testing.T) {
	helpers.GetI18nInstance(helpers.SetDefaultLocale("gl"), helpers.SetDirectory("../i18n"))
	var now = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	var row = model.ListPendingEmailsRow{
		Type: model.NotificationTypeBookingConfirmed, Status: model.BookingStatusConfirmed,
		StartsAt: now.Add(48 * time.Hour), EndsAt: now.Add(49 * time.Hour),
		FacilityName: "Pavillón", TenantName: "Concello de Lalín", Timezone: "Europe/Madrid", Email: "a@example.com",
	}
	row.PreferredLanguage.String, row.PreferredLanguage.Valid = "en", true

	// no way in: nothing to explain, no QR
	var message, _ = notificationEmail(row, branding.Email{Name: "LibreQuenda"}, now)
	if strings.Contains(message.Text, "check in") || len(message.Inline) != 0 {
		t.Errorf("no ways in, yet: %q, %d images", message.Text, len(message.Inline))
	}

	// only the entry code: one way, explained, and no QR
	row.EntryCode = true
	message, _ = notificationEmail(row, branding.Email{Name: "LibreQuenda"}, now)
	if !strings.Contains(message.Text, "To check in:") || !strings.Contains(message.Text, "Entry code:") || strings.Contains(message.Text, "QR reader") || len(message.Inline) != 0 {
		t.Errorf("entry code only: %q, %d images", message.Text, len(message.Inline))
	}

	// a reader and staff too: every way, and the QR for the reader
	row.HasReader, row.StaffConfirm = true, true
	message, _ = notificationEmail(row, branding.Email{Name: "LibreQuenda"}, now)
	for _, want := range []string{"several ways", "QR reader:", "Entry code:", "Staff:"} {
		if !strings.Contains(message.Text, want) {
			t.Errorf("several ways: %q lacks %q", message.Text, want)
		}
	}
	if strings.Contains(message.Text, "QR at the door") || len(message.Inline) != 1 || !strings.Contains(message.HTML, "cid:booking-qr") {
		t.Errorf("several ways: %q, %d images", message.Text, len(message.Inline))
	}
}

func TestEmailLinksDirectionsAndContacts(t *testing.T) {
	helpers.GetI18nInstance(helpers.SetDefaultLocale("gl"), helpers.SetDirectory("../i18n"))
	var now = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	var row = model.ListPendingEmailsRow{
		Type: model.NotificationTypeBookingConfirmed, Status: model.BookingStatusConfirmed,
		StartsAt: now.Add(48 * time.Hour), EndsAt: now.Add(49 * time.Hour),
		FacilityName: "Pavillón", TenantName: "Concello de Lalín", Timezone: "Europe/Madrid", Email: "a@example.com",
		EntryCode:         true,
		FacilityLatitude:  sql.NullFloat64{Float64: 42.66453, Valid: true},
		FacilityLongitude: sql.NullFloat64{Float64: -8.10815, Valid: true},
		FacilityPhone:     sql.NullString{String: "986 787 060", Valid: true},
		TenantEmail:       sql.NullString{String: "lalin@lalin.org", Valid: true},
	}
	row.PreferredLanguage.String, row.PreferredLanguage.Valid = "en", true
	var message, err = notificationEmail(row, branding.Email{Name: "LibreQuenda"}, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`href="https://www.google.com/maps/dir/?api=1&amp;destination=42.66453,-8.10815"`, // how to get there
		`>Check in</a>`,                 // the entry code's link, clickable
		`href="tel:986787060"`,          // the facility's phone
		`href="mailto:lalin@lalin.org"`, // the council's email
		"Facility contact (Pavillón):", "Concello de Lalín contact:",
	} {
		if !strings.Contains(message.HTML, want) {
			t.Errorf("HTML lacks %s:\n%s", want, message.HTML)
		}
	}
	if !strings.Contains(message.Text, "How to get there: https://www.google.com/maps/dir/") || !strings.Contains(message.Text, "986 787 060") {
		t.Errorf("text lacks directions or phone: %q", message.Text)
	}

	// a cancellation still says who to ask, but not how to get there
	row.Type, row.Status = model.NotificationTypeBookingCancelledByCouncil, model.BookingStatusCancelled
	message, _ = notificationEmail(row, branding.Email{Name: "LibreQuenda"}, now)
	if strings.Contains(message.HTML, "google.com/maps") || !strings.Contains(message.HTML, "mailto:lalin@lalin.org") {
		t.Errorf("cancellation: %s", message.HTML)
	}
}

func TestJoinedEmailNamesWhoJoined(t *testing.T) {
	helpers.GetI18nInstance(helpers.SetDefaultLocale("gl"), helpers.SetDirectory("../i18n"))
	var now = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	var row = model.ListPendingEmailsRow{
		Type: model.NotificationTypeBookingJoined, Status: model.BookingStatusConfirmed, EmailNotifications: true,
		StartsAt: now.Add(48 * time.Hour), EndsAt: now.Add(49 * time.Hour),
		FacilityName: "Pista de tenis", TenantName: "Concello de Lalín", Timezone: "Europe/Madrid", Email: "a@example.com",
		Payload: []byte(`{"name":"Uxía","guest":false}`),
	}
	row.PreferredLanguage.String, row.PreferredLanguage.Valid = "en", true
	var message, err = notificationEmail(row, branding.Email{Name: "LibreQuenda"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(message.Subject, "Someone joined your booking") || !strings.Contains(message.Text, "Uxía joined your booking at Pista de tenis") {
		t.Errorf("joined email: %q / %q", message.Subject, message.Text)
	}
	row.Payload = []byte(`{}`)
	message, _ = notificationEmail(row, branding.Email{Name: "LibreQuenda"}, now)
	if !strings.Contains(message.Text, "Someone joined your booking") {
		t.Errorf("without a name: %q", message.Text)
	}
}

func TestEmailHeaderShowsTheInstallationLogo(t *testing.T) {
	helpers.GetI18nInstance(helpers.SetDefaultLocale("gl"), helpers.SetDirectory("../i18n"))
	var row = model.ListPendingInvitationEmailsRow{Email: "nova@example.com", TenantName: "Concello de Lalín", TenantSlug: "lalin"}
	var logo = branding.Image{PNG: []byte{1, 2, 3}, Width: 128, Height: 64}

	// its own logo: the mark at 32 px high with the name beside it, sent inside the email
	var message = invitationEmail(row, branding.Email{Name: "Reservas Lalín", Logo: logo})
	if len(message.Inline) != 1 || message.Inline[0].CID != "brand-logo" || !bytes.Equal(message.Inline[0].Data, logo.PNG) {
		t.Fatalf("logo not attached: %+v", message.Inline)
	}
	if !strings.Contains(message.HTML, `src="cid:brand-logo" width="64" height="32" alt=""`) || !strings.Contains(message.HTML, ">Reservas Lalín</span>") {
		t.Errorf("header: %s", message.HTML)
	}

	// LibreQuenda's lockup carries the name: the image says it, no text beside
	message = invitationEmail(row, branding.Email{Name: "LibreQuenda", Lockup: true, Logo: logo})
	if !strings.Contains(message.HTML, `alt="LibreQuenda"`) || strings.Contains(message.HTML, ">LibreQuenda</span>") {
		t.Errorf("lockup header: %s", message.HTML)
	}

	// a booking with the reader's QR carries both images
	var now = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	var booking = model.ListPendingEmailsRow{
		Type: model.NotificationTypeBookingConfirmed, Status: model.BookingStatusConfirmed,
		StartsAt: now.Add(48 * time.Hour), EndsAt: now.Add(49 * time.Hour),
		FacilityName: "Pavillón", TenantName: "Concello de Lalín", Timezone: "Europe/Madrid", Email: "a@example.com",
		HasReader: true, CheckinToken: uuid.New(),
	}
	var withQR, err = notificationEmail(booking, branding.Email{Name: "Reservas Lalín", Logo: logo}, now)
	if err != nil || len(withQR.Inline) != 2 {
		t.Errorf("booking with QR and logo: %d images, %v", len(withQR.Inline), err)
	}
}
