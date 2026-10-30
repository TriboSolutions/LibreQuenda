package bookings

import "testing"

func TestReadiness(t *testing.T) {
	var cases = []struct {
		birth, verified bool
		want            string
	}{
		{true, true, ""},
		{false, true, "errors.profile_incomplete"},
		{true, false, "errors.email_unverified"},
		{false, false, "errors.profile_incomplete"}, // the birth date first, as the profile banner asks
	}
	for _, c := range cases {
		var got = readiness(c.birth, c.verified)
		if (got == nil && c.want != "") || (got != nil && got.Message != c.want) {
			t.Errorf("readiness(%v, %v) = %v, want %q", c.birth, c.verified, got, c.want)
		}
	}
}
