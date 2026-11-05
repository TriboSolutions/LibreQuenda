package counciladmin

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestLikePatternIsLiteral(t *testing.T) {
	var cases = map[string]string{
		"lucia":      "lucia",
		"100%":       `100\%`,
		"ana_b":      `ana\_b`,
		`back\slash`: `back\\slash`,
	}
	for in, want := range cases {
		if got := likePattern(in); got != want {
			t.Errorf("likePattern(%q) = %q, want %q", in, got, want)
		}
	}
}

// Bad query parameters are refused before the database is read.
func TestListResidentsValidatesTheQuery(t *testing.T) {
	var s = &counciladminService{}
	var cases = []struct {
		name string
		sort string
		page int
	}{
		{"unknown sort", "email", 1},
		{"page 0", "name", 0},
		{"negative page", "recent", -2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var _, err = s.ListResidents(context.Background(), uuid.New(), "", c.sort, c.page)
			if err == nil || err.Code != http.StatusBadRequest {
				t.Errorf("got %+v, want 400", err)
			}
		})
	}
}

func TestAddResidentNeedsAnEmail(t *testing.T) {
	var s = &counciladminService{}
	for _, email := range []string{"", "  ", "not-an-email"} {
		if _, err := s.AddResident(context.Background(), uuid.New(), uuid.New(), email); err == nil || err.Message != "errors.invalid_email" {
			t.Errorf("AddResident(%q) = %+v, want errors.invalid_email", email, err)
		}
	}
}
