package platform

import "testing"

func TestTenantInputValidate(t *testing.T) {
	var valid = func() TenantInput {
		return TenantInput{Slug: "lalin", Name: "Concello de Lalín", Timezone: "Europe/Madrid", AdminEmail: "deportes@concello.example"}
	}

	var cases = []struct {
		name     string
		change   func(*TenantInput)
		creating bool
		want     string
	}{
		{"valid", func(*TenantInput) {}, true, ""},
		{"slug with hyphen", func(in *TenantInput) { in.Slug = "a-estrada" }, true, ""},
		{"slug normalised", func(in *TenantInput) { in.Slug = "  Lalin " }, true, ""},
		{"slug with accent", func(in *TenantInput) { in.Slug = "lalín" }, true, "errors.invalid_slug"},
		{"slug double hyphen", func(in *TenantInput) { in.Slug = "a--estrada" }, true, "errors.invalid_slug"},
		{"slug too short", func(in *TenantInput) { in.Slug = "a" }, true, "errors.invalid_slug"},
		{"reserved slug", func(in *TenantInput) { in.Slug = "admin" }, true, "errors.slug_reserved"},
		{"empty name", func(in *TenantInput) { in.Name = "  " }, true, "errors.invalid_name"},
		{"unknown time zone", func(in *TenantInput) { in.Timezone = "Europe/Lalin" }, true, "errors.invalid_timezone"},
		{"bad contact email", func(in *TenantInput) { in.ContactEmail = "deportes" }, true, "errors.invalid_email"},
		{"missing admin email on create", func(in *TenantInput) { in.AdminEmail = "" }, true, "errors.invalid_admin_email"},
		{"admin email not needed on update", func(in *TenantInput) { in.AdminEmail = "" }, false, ""},
		{"display name instead of address", func(in *TenantInput) { in.AdminEmail = "Ana <ana@concello.example>" }, true, "errors.invalid_admin_email"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var in = valid()
			c.change(&in)
			in.normalise()
			if got := in.validate(c.creating); got != c.want {
				t.Errorf("validate() = %q, want %q", got, c.want)
			}
		})
	}
}
