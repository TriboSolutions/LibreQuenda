package me

import "testing"

func TestResidentRequestClean(t *testing.T) {
	var in = ResidentRequestInput{Dni: " 12.345.678-z ", FullName: "  Ana   Pérez ", PostalCode: " 27001 ", Street: " Rúa  Nova, 3 "}
	if key := in.clean(); key != "" {
		t.Fatalf("clean() = %q, want valid", key)
	}
	if in.Dni != "12345678Z" || in.FullName != "Ana Pérez" || in.PostalCode != "27001" || in.Street != "Rúa Nova, 3" {
		t.Errorf("clean() normalised to %+v", in)
	}
	var cases = map[string]ResidentRequestInput{
		"errors.invalid_dni":              {Dni: "12345678A", FullName: "Ana", PostalCode: "27001", Street: "Rúa"},
		"errors.invalid_postal_code":      {Dni: "12345678Z", FullName: "Ana", PostalCode: "2700", Street: "Rúa"},
		"errors.resident_request_invalid": {Dni: "X1234567L", FullName: "A", PostalCode: "27001", Street: "Rúa"},
	}
	for want, input := range cases {
		if got := input.clean(); got != want {
			t.Errorf("clean(%+v) = %q, want %q", input, got, want)
		}
	}
}
