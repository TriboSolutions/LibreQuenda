package platform

import (
	"strings"
	"testing"
)

func TestBrandingInputValidate(t *testing.T) {
	var cases = []struct {
		name  string
		input BrandingInput
		want  string
	}{
		{"default name and logo", BrandingInput{}, ""},
		{"custom name and logo", BrandingInput{Name: "Reservas Lalín", LogoRef: "branding/0192f5a0-0000-7000-8000-000000000001.png"}, ""},
		{"spaces only go back to the default", BrandingInput{Name: "   "}, ""},
		{"60 characters", BrandingInput{Name: strings.Repeat("á", 60)}, ""},
		{"too long", BrandingInput{Name: strings.Repeat("a", 61)}, "errors.invalid_app_name"},
		{"control character", BrandingInput{Name: "Reservas\u0007"}, "errors.invalid_app_name"},
		{"logo from another folder", BrandingInput{LogoRef: "councils/logo.png"}, "errors.invalid_image"},
		{"logo as a URL", BrandingInput{LogoRef: "https://example.org/logo.png"}, "errors.invalid_image"},
		{"logo escaping its folder", BrandingInput{LogoRef: "branding/../users/x.png"}, "errors.invalid_image"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var in = c.input
			if got := in.validate(); got != c.want {
				t.Errorf("validate() = %q, want %q", got, c.want)
			}
		})
	}

	var in = BrandingInput{Name: "  Reservas \t do   Concello "}
	in.validate()
	if in.Name != "Reservas do Concello" {
		t.Errorf("name not normalised: %q", in.Name)
	}
}
