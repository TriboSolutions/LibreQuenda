package cronJobs

import (
	"librequenda/branding"
	"librequenda/model"
	"strings"
	"testing"
	"time"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

func TestNoticeEmail(t *testing.T) {
	helpers.GetI18nInstance(helpers.SetDefaultLocale("gl"), helpers.SetDirectory("../i18n"))
	var row = model.ListPendingNoticeEmailsRow{
		Email: "a@example.com", Title: "Piscina pechada por obras", Body: "Do 3 ao 10 de decembro.\nGrazas pola paciencia.",
		TenantName: "Concello de Lalín", TenantSlug: "lalin", FacilityName: "Piscina municipal", FacilitySlug: "piscina",
	}
	row.FullName.String, row.FullName.Valid = "Lucía Gómez", true
	var message = noticeEmail(row, branding.Email{Name: "LibreQuenda"})
	for _, want := range []string{"Ola Lucía", "Piscina pechada por obras", "Grazas pola paciencia.", "/lalin/piscina", "Piscina municipal"} {
		if !strings.Contains(message.Text, want) {
			t.Errorf("notice email lacks %q:\n%s", want, message.Text)
		}
	}
	if !strings.Contains(message.HTML, "white-space:pre-line") || !strings.Contains(message.Subject, "Piscina pechada por obras") {
		t.Errorf("subject %q, or the body without its line breaks", message.Subject)
	}
	// the council's text is escaped in the HTML
	row.Body = "<script>alert(1)</script>"
	if message = noticeEmail(row, branding.Email{Name: "LibreQuenda"}); strings.Contains(message.HTML, "<script>") {
		t.Error("the announcement's text is not escaped")
	}
	for _, language := range []string{"gl", "es", "en", "ca"} {
		row.PreferredLanguage.String, row.PreferredLanguage.Valid = language, true
		for _, facility := range []string{"", "Piscina municipal"} {
			row.FacilityName = facility
			message = noticeEmail(row, branding.Email{Name: "LibreQuenda"})
			if strings.Contains(message.Subject, "email.") || strings.Contains(message.Text, "email.") {
				t.Errorf("%s: missing translation: %q / %q", language, message.Subject, message.Text)
			}
		}
	}
}

func TestPollEmail(t *testing.T) {
	helpers.GetI18nInstance(helpers.SetDefaultLocale("gl"), helpers.SetDirectory("../i18n"))
	var row = model.ListPendingPollEmailsRow{
		Email: "a@example.com", Question: "Que horario preferides os sábados?", TenantName: "Concello de Lalín", TenantSlug: "lalin",
		Type: model.NotificationTypePollOpened, Timezone: "Europe/Madrid", ClosesAt: time.Date(2026, 11, 16, 23, 0, 0, 0, time.UTC), IsOpen: true,
	}
	var message = pollEmail(row, branding.Email{Name: "LibreQuenda"})
	for _, want := range []string{"Que horario preferides os sábados?", "17/11/2026 00:00", "/enquisas?enquisa="} {
		if !strings.Contains(message.Text, want) {
			t.Errorf("poll email lacks %q:\n%s", want, message.Text)
		}
	}
	for _, language := range []string{"gl", "es", "en", "ca"} {
		row.PreferredLanguage.String, row.PreferredLanguage.Valid = language, true
		for _, facility := range []string{"", "Piscina municipal"} {
			row.FacilityName = facility
			row.Type = model.NotificationTypePollOpened
			if facility == "" {
				row.Type = model.NotificationTypePollClosed // the results are out
			}
			message = pollEmail(row, branding.Email{Name: "LibreQuenda"})
			if strings.Contains(message.Subject, "email.") || strings.Contains(message.Text, "email.") {
				t.Errorf("%s: missing translation: %q / %q", language, message.Subject, message.Text)
			}
		}
	}
}
