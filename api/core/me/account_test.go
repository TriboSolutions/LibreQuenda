package me

import "testing"

func TestNameInputValidate(t *testing.T) {
	var ok = NameInput{GivenName: "  Uxía ", FamilyName: "Pérez   Lago"}
	if key := ok.validate(); key != "" || ok.GivenName != "Uxía" || ok.FamilyName != "Pérez Lago" {
		t.Errorf("validate() = %q, %+v", key, ok)
	}
	for _, bad := range []NameInput{{GivenName: "", FamilyName: "Lago"}, {GivenName: "Uxía", FamilyName: "  "}} {
		if key := bad.validate(); key != "errors.invalid_name" {
			t.Errorf("%+v: validate() = %q", bad, key)
		}
	}
}
