package cronJobs

import (
	"librequenda/branding"
	"strings"
	"testing"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

func TestPhoneCodeEmail(t *testing.T) {
	helpers.GetI18nInstance(helpers.SetDefaultLocale("gl"), helpers.SetDirectory("../i18n"))
	var message = PhoneCodeEmail(branding.Email{Name: "LibreQuenda"}, "ana@example.com", "Ana Pérez", "gl", "042917", 10)
	if message.To != "ana@example.com" || message.Subject != "O teu código para LibreQuenda: 042917" || message.Kind != "phone_code" {
		t.Errorf("email to %s, %q, kind %s", message.To, message.Subject, message.Kind)
	}
	if !strings.Contains(message.Text, "Ola Ana,") || !strings.Contains(message.Text, "\n042917\n") || !strings.Contains(message.HTML, "042917") || !strings.Contains(message.Text, "10 minutos") {
		t.Errorf("email text:\n%s", message.Text)
	}
	for _, language := range []string{"gl", "es", "en", "ca", ""} {
		message = PhoneCodeEmail(branding.Email{Name: "LibreQuenda"}, "ana@example.com", "", language, "042917", 10)
		if strings.Contains(message.Subject+message.Text+message.HTML, "phone_code.") {
			t.Errorf("%q: missing translation: %q / %q", language, message.Subject, message.Text)
		}
	}
}
