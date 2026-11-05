package zitadel

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"reflect"
	"strings"
)

// The installation's name and logo (branding) also reach Zitadel, so its sign-in
// pages and the emails it sends (verify the address, reset the password…) look
// like the rest of the app. They are set on the organisation residents register
// in, as its own label policy and message texts, which the service account may
// change as ORG_SETTINGS_MANAGER (zitadel/setup.sh); the instance's defaults,
// LibreQuenda's, stay as zitadel-setup left them.

// messageTypes: every email Zitadel can send, as the management API names them.
var messageTypes = []string{
	"init", "passwordreset", "verifyemail", "verifyphone", "verifysmsotp", "verifyemailotp",
	"domainclaimed", "passwordless_registration", "password_change", "invite_user",
}

// textLanguages: the app's languages Zitadel has texts for; Galician and Catalan
// it lacks, and their speakers get the instance's default language.
var textLanguages = []string{"es", "en"}

// SyncBranding puts the installation's name in Zitadel's emails, where they say
// "Zitadel", and its logo (a PNG; nil for LibreQuenda's) on the sign-in pages and
// those emails. It changes only what differs, so it can run on every start.
func SyncBranding(ctx context.Context, name string, logo []byte) error {
	if serviceToken() == "" {
		return ErrUnavailable
	}
	for _, language := range textLanguages {
		for _, kind := range messageTypes {
			if err := syncMessageText(ctx, kind, language, name); err != nil {
				return fmt.Errorf("%s/%s text: %w", kind, language, err)
			}
		}
	}
	if err := syncLabelPolicy(ctx, logo); err != nil {
		return fmt.Errorf("label policy: %w", err)
	}
	return nil
}

// message texts are flat objects of strings; details and isDefault are not texts
type messageText map[string]any

func (m messageText) texts() map[string]string {
	var out = map[string]string{}
	for key, value := range m {
		if text, ok := value.(string); ok && text != "" {
			out[key] = text
		}
	}
	return out
}

func syncMessageText(ctx context.Context, kind, language, name string) error {
	var path = "/management/v1/text/message/" + kind + "/" + language
	var defaults, current struct {
		CustomText messageText `json:"customText"`
	}
	if err := call(ctx, http.MethodGet, "/management/v1/text/default/message/"+kind+"/"+language, nil, &defaults); err != nil {
		return err
	}
	var want = defaults.CustomText.texts()
	for key, text := range want {
		want[key] = strings.NewReplacer("ZITADEL", name, "Zitadel", name).Replace(text)
	}
	if err := call(ctx, http.MethodGet, path, nil, &current); err != nil {
		return err
	}
	if reflect.DeepEqual(current.CustomText.texts(), want) {
		return nil
	}
	return call(ctx, http.MethodPut, path, want, nil)
}

type labelPolicy struct {
	IsDefault           bool   `json:"isDefault"`
	PrimaryColor        string `json:"primaryColor,omitempty"`
	WarnColor           string `json:"warnColor,omitempty"`
	BackgroundColor     string `json:"backgroundColor,omitempty"`
	FontColor           string `json:"fontColor,omitempty"`
	PrimaryColorDark    string `json:"primaryColorDark,omitempty"`
	WarnColorDark       string `json:"warnColorDark,omitempty"`
	BackgroundColorDark string `json:"backgroundColorDark,omitempty"`
	FontColorDark       string `json:"fontColorDark,omitempty"`
	HideLoginNameSuffix bool   `json:"hideLoginNameSuffix"`
	DisableWatermark    bool   `json:"disableWatermark"`
	ThemeMode           string `json:"themeMode,omitempty"`
}

// syncLabelPolicy: with a logo, the organisation gets its own label policy (the
// instance's colours, that logo); without one it goes back to the instance's.
func syncLabelPolicy(ctx context.Context, logo []byte) error {
	var current struct {
		Policy labelPolicy `json:"policy"`
	}
	if err := call(ctx, http.MethodGet, "/management/v1/policies/label", nil, &current); err != nil {
		return err
	}
	if len(logo) == 0 {
		if current.Policy.IsDefault {
			return nil
		}
		return call(ctx, http.MethodDelete, "/management/v1/policies/label", nil, nil)
	}
	if current.Policy.IsDefault {
		var instance struct {
			Policy labelPolicy `json:"policy"`
		}
		if err := call(ctx, http.MethodGet, "/management/v1/policies/default/label", nil, &instance); err != nil {
			return err
		}
		var policy = instance.Policy
		policy.IsDefault = false
		if err := call(ctx, http.MethodPost, "/management/v1/policies/label", policy, nil); err != nil {
			return err
		}
	}
	// the installation's logo is one image, for light and dark and as the icon
	for _, asset := range []string{"logo", "logo/dark", "icon", "icon/dark"} {
		if err := uploadLabelAsset(ctx, asset, logo); err != nil {
			return fmt.Errorf("%s: %w", asset, err)
		}
	}
	return call(ctx, http.MethodPost, "/management/v1/policies/label/_activate", map[string]any{}, nil)
}

func uploadLabelAsset(ctx context.Context, asset string, image []byte) error {
	var body bytes.Buffer
	var form = multipart.NewWriter(&body)
	var header = make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="logo.png"`)
	header.Set("Content-Type", "image/png")
	var part, err = form.CreatePart(header)
	if err != nil {
		return err
	}
	if _, err = part.Write(image); err != nil {
		return err
	}
	if err = form.Close(); err != nil {
		return err
	}
	// the same request as the profile photo's, with the service account's token
	return own(ctx, serviceToken(), http.MethodPost, "/assets/v1/org/policy/label/"+asset, &body, form.FormDataContentType())
}
