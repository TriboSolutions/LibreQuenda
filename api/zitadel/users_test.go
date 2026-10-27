package zitadel

import (
	"errors"
	"net/http"
	"testing"
)

func TestPasswordRulesMeets(t *testing.T) {
	var rules = PasswordRules{MinLength: 8, Uppercase: true, Lowercase: true, Number: true, Symbol: true}
	var cases = map[string]bool{
		"Pista2026!":  true,
		"pista2026!":  false, // no uppercase
		"PISTA2026!":  false, // no lowercase
		"PistaPista!": false, // no number
		"Pista20266":  false, // no symbol
		"Pi2!":        false, // too short
		"Piñeiro2026": true,  // ñ counts as a symbol, as in Zitadel ([^A-Za-z0-9])
	}
	for password, want := range cases {
		if got := rules.Meets(password); got != want {
			t.Errorf("Meets(%q) = %v, want %v", password, got, want)
		}
	}
	if !(&PasswordRules{MinLength: 4}).Meets("abcd") {
		t.Error("a policy without character classes only checks the length")
	}
}

func TestClassify(t *testing.T) {
	var cases = []struct {
		status  int
		message string
		want    error
	}{
		{http.StatusBadRequest, "Password is invalid (COMMAND-3M0fs)", ErrWrongPassword},
		{http.StatusBadRequest, "Code is invalid (COMMAND-eis9R)", ErrInvalidCode},
		{http.StatusBadRequest, "Password doesn't meet the complexity (DOMAIN-VoaRj)", ErrRejected},
		{http.StatusInternalServerError, "boom", errUnexpectedReply},
	}
	for _, c := range cases {
		if got := classify(c.status, apiError{Message: c.message}); !errors.Is(got, c.want) {
			t.Errorf("classify(%d, %q) = %v, want %v", c.status, c.message, got, c.want)
		}
	}
}
