package tenants

import "testing"

func TestEligibilityAllows(t *testing.T) {
	var anyone eligibility // signed out, not a resident anywhere, or a superadministrator
	var cambre = eligibility{"cambre": true}
	var cases = []struct {
		name          string
		eligible      eligibility
		council       string
		residentsOnly bool
		want          bool
	}{
		{"open facility, anyone", anyone, "lalin", false, true},
		{"residents only, no restriction", anyone, "lalin", true, true},
		{"open facility of another council", cambre, "lalin", false, true},
		{"residents only, their council", cambre, "cambre", true, true},
		{"residents only, another council", cambre, "lalin", true, false},
	}
	for _, c := range cases {
		if got := c.eligible.allows(c.council, c.residentsOnly); got != c.want {
			t.Errorf("%s: allows = %v, want %v", c.name, got, c.want)
		}
	}
}
