package me

import (
	"testing"
	"time"
)

func TestValidateProfile(t *testing.T) {
	var today = time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	var cases = []struct {
		name string
		in   ProfileInput
		want string
	}{
		{"valid", ProfileInput{BirthDate: "1990-05-17", Phone: "+34 600 123 456", PreferredLanguage: "gl"}, ""},
		{"birth date required", ProfileInput{}, "errors.invalid_birth_date"},
		{"birth date in the future", ProfileInput{BirthDate: "2027-01-01"}, "errors.invalid_birth_date"},
		{"birth date over 120 years ago", ProfileInput{BirthDate: "1900-01-01"}, "errors.invalid_birth_date"},
		{"wrong date format", ProfileInput{BirthDate: "17/05/1990"}, "errors.invalid_birth_date"},
		{"phone with letters", ProfileInput{BirthDate: "1990-05-17", Phone: "call me"}, "errors.invalid_phone_format"},
		{"photo of a facility", ProfileInput{BirthDate: "1990-05-17", AvatarRef: "facilities/a.png"}, "errors.invalid_image"},
		{"own photo", ProfileInput{BirthDate: "1990-05-17", AvatarRef: "users/a.png"}, ""},
		{"unknown language", ProfileInput{BirthDate: "1990-05-17", PreferredLanguage: "fr"}, "errors.invalid_language"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var in = c.in
			if _, got := validateProfile(&in, today); got != c.want {
				t.Errorf("validateProfile() = %q, want %q", got, c.want)
			}
		})
	}
}
