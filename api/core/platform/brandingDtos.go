package platform

import (
	"librequenda/storage"
	"strings"
	"unicode"
)

// BrandingDto is how the installation presents itself: public, every page reads it.
type BrandingDto struct {
	Name       string `json:"name"`        // what to show: the custom name or LibreQuenda
	CustomName string `json:"custom_name"` // "" = LibreQuenda
	LogoRef    string `json:"logo_ref"`    // sent back unchanged when the form is saved; "" = the LibreQuenda logo
	LogoUrl    string `json:"logo_url"`
	// support contact, from SUPPORT_EMAIL and SUPPORT_PHONE; "" = not set
	SupportEmail string `json:"support_email"`
	SupportPhone string `json:"support_phone"`
}

type BrandingInput struct {
	Name    string `json:"name"`     // "" goes back to LibreQuenda
	LogoRef string `json:"logo_ref"` // a storage reference from POST /media (folder branding); "" = the LibreQuenda logo
}

// validate normalises the input and returns the i18n key of the first invalid field, or "".
func (in *BrandingInput) validate() string {
	in.Name = strings.Join(strings.Fields(in.Name), " ")
	in.LogoRef = strings.TrimSpace(in.LogoRef)
	switch {
	case len([]rune(in.Name)) > 60 || strings.IndexFunc(in.Name, unicode.IsControl) >= 0:
		return "errors.invalid_app_name"
	case in.LogoRef != "" && (!storage.ValidRef(in.LogoRef) || !strings.HasPrefix(in.LogoRef, "branding/")):
		return "errors.invalid_image"
	}
	return ""
}
