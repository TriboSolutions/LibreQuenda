package cronJobs

import (
	"librequenda/model"
	"strings"
	"testing"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

func TestIssueEmails(t *testing.T) {
	helpers.GetI18nInstance(helpers.SetDefaultLocale("gl"), helpers.SetDirectory("../i18n"))
	var row = model.ListPendingIssueEmailsRow{
		Type: model.NotificationTypeIssueSolved, Email: "a@example.com", FacilityName: "Pista de tenis 1", TenantName: "Concello de Lalín",
		Payload: []byte(`{"reason":"New net fitted on Tuesday."}`),
	}
	row.PreferredLanguage.String, row.PreferredLanguage.Valid = "en", true
	row.FullName.String, row.FullName.Valid = "Lucía Gómez", true
	var message = issueEmail(row, "LibreQuenda")
	for _, want := range []string{"Hello Lucía", "Pista de tenis 1 (Concello de Lalín) as fixed", "Note from the council: New net fitted on Tuesday."} {
		if !strings.Contains(message.Text, want) {
			t.Errorf("solved email lacks %q:\n%s", want, message.Text)
		}
	}
	if !strings.HasPrefix(message.Subject, "Fixed:") || strings.Contains(message.HTML, "email.") {
		t.Errorf("subject %q or untranslated HTML", message.Subject)
	}

	for _, language := range []string{"gl", "es", "en", "ca"} {
		row.PreferredLanguage.String = language
		row.Type = model.NotificationTypeIssueRejected
		message = issueEmail(row, "LibreQuenda")
		if strings.Contains(message.Subject, "email.") || strings.Contains(message.Text, "email.") {
			t.Errorf("%s: missing translation: %q / %q", language, message.Subject, message.Text)
		}
	}
}

func TestNewIssueEmailLeadsToCouncilAdministration(t *testing.T) {
	helpers.GetI18nInstance(helpers.SetDefaultLocale("gl"), helpers.SetDirectory("../i18n"))
	var row = model.ListPendingIssueEmailsRow{
		Type: model.NotificationTypeIssueReported, TenantSlug: "lalin",
		Email: "admin@example.com", FacilityName: "Pista 1", TenantName: "Lalín",
	}
	for _, language := range []string{"gl", "es", "en", "ca"} {
		row.PreferredLanguage.String = language
		var message = issueEmail(row, "LibreQuenda")
		if !strings.Contains(message.Text, "/lalin/admin?vista=incidencias") ||
			!strings.Contains(message.Subject, "Pista 1") || strings.Contains(message.Text, "email.") {
			t.Errorf("%s: invalid new issue email: %+v", language, message)
		}
	}
}
