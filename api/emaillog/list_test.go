package emaillog

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestParseFilters(t *testing.T) {
	var f = ParseFilters(url.Values{
		"q": {"  example.com "}, "status": {"failed"}, "kind": {"booking_reminder"},
		"from": {"2026-11-01"}, "to": {"2026-11-07"}, "before": {"2026-11-07T10:00:00Z"},
	}, time.UTC)
	if f.Query != "example.com" || f.Status != "failed" || f.Kind != "booking_reminder" {
		t.Fatalf("got %+v", f)
	}
	if !f.Since.Valid || !f.Until.Valid || f.Until.Time.Day() != 8 || !f.Before.Valid {
		t.Fatalf("dates: %+v", f)
	}
	var bad = ParseFilters(url.Values{"q": {strings.Repeat("a", 101)}, "status": {"lost"}, "kind": {"DROP TABLE"}, "from": {"ayer"}}, time.UTC)
	if bad.Query != "" || bad.Status != "" || bad.Kind != "" || bad.Since.Valid {
		t.Fatalf("invalid values are no filter: %+v", bad)
	}
}

func TestEscapeLike(t *testing.T) {
	if got := escapeLike(`50%_off\`); got != `50\%\_off\\` {
		t.Fatalf("got %q", got)
	}
}
