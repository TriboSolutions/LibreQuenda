package counciladmin

import "testing"

func TestCheckinSettingsValidate(t *testing.T) {
	var cases = []struct {
		name        string
		input       CheckinSettingsInput
		hasLocation bool
		want        string
	}{
		{"defaults", CheckinSettingsInput{RadiusM: 100, FlashBooking: true}, false, ""},
		{"printed QR with a pin", CheckinSettingsInput{PrintedQr: true, RadiusM: 100}, true, ""},
		{"entry code without a pin", CheckinSettingsInput{EntryCode: true, RadiusM: 100}, false, "errors.checkin_needs_location"},
		{"radius too small", CheckinSettingsInput{RadiusM: 10}, true, "errors.invalid_checkin_radius"},
		{"radius too large", CheckinSettingsInput{RadiusM: 5000}, true, "errors.invalid_checkin_radius"},
	}
	for _, c := range cases {
		if got := c.input.validate(c.hasLocation); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
