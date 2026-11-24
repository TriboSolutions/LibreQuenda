package cronJobs

import (
	"bytes"
	"librequenda/branding"
	"librequenda/mailer"
	"librequenda/shared"
	"strconv"
	"strings"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

// PhoneCodeEmail: the code a caller on the voice line tells the assistant to sign
// in with their account, sent at once to the account's email (not from the queue:
// they are waiting on the line), in the account's language.
func PhoneCodeEmail(brand branding.Email, to, fullName, language, code string, minutes int) mailer.Message {
	var appName = brand.Name
	var i18n = helpers.GetI18nInstance()
	var t = func(key string, args ...any) string {
		return i18n.TranslateWithLanguageWithVars("email."+key, language, args...)
	}
	var view = emailView{
		Greeting:  t("greeting_anonymous"),
		Intro:     t("phone_code.intro", appName, strconv.Itoa(minutes)),
		Heading:   code,
		Manage:    t("phone_code.manage", appName),
		ManageURL: shared.PUBLIC_URL,
		Footer:    t("phone_code.footer"),
	}
	if name := strings.Fields(fullName); len(name) > 0 {
		view.Greeting = t("greeting", name[0])
	}
	var html bytes.Buffer
	brandHeader(&view, brand)
	_ = emailTemplate.Execute(&html, view) // the template is fixed and checked by the tests
	var message = mailer.Message{
		To:      to,
		Subject: t("phone_code.subject", appName, code),
		Text:    strings.Join([]string{view.Greeting, "", view.Intro, "", code, "", view.Manage + ": " + view.ManageURL, "", "--", view.Footer}, "\n"),
		HTML:    html.String(),
		Kind:    "phone_code",
	}
	attachLogo(&message, brand)
	return message
}
