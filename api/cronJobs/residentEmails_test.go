package cronJobs

import (
	"librequenda/branding"
	"librequenda/model"
	"strings"
	"testing"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

func TestResidentEmails(t *testing.T) {
	helpers.GetI18nInstance(helpers.SetDefaultLocale("gl"), helpers.SetDirectory("../i18n"))
	var row = model.ListPendingResidentEmailsRow{
		Type: model.NotificationTypeResidentRequested, Email: "admin@example.com", TenantName: "Concello de Lalín", TenantSlug: "lalin",
		Payload: []byte(`{"tenant_slug":"lalin","name":"Ana Pérez"}`),
	}
	row.PreferredLanguage.String, row.PreferredLanguage.Valid = "gl", true
	var message = residentEmail(row, branding.Email{Name: "LibreQuenda"})
	if message.Subject != "Ana Pérez quere ser veciño de Concello de Lalín" || !strings.Contains(message.Text, "/lalin/admin?vista=vecinos") {
		t.Errorf("request email: %q\n%s", message.Subject, message.Text)
	}
	for _, kind := range []model.NotificationType{model.NotificationTypeResidentRequested, model.NotificationTypeResidentVerified, model.NotificationTypeResidentRejected} {
		for _, language := range []string{"gl", "es", "en", "ca"} {
			row.Type, row.PreferredLanguage.String = kind, language
			row.TenantPhone.String, row.TenantPhone.Valid = "986 000 000", true
			message = residentEmail(row, branding.Email{Name: "LibreQuenda"})
			if strings.Contains(message.Subject, "email.") || strings.Contains(message.Text, "email.") || strings.Contains(message.HTML, "email.") {
				t.Errorf("%s %s: missing translation: %q / %q", kind, language, message.Subject, message.Text)
			}
			if strings.Contains(message.Text, "12345678Z") {
				t.Errorf("%s: the email carries identity data", kind)
			}
		}
	}
}
