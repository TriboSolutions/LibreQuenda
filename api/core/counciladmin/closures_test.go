package counciladmin

import (
	"strings"
	"testing"
	"time"
)

func TestClosureInputParse(t *testing.T) {
	var madrid, _ = time.LoadLocation("Europe/Madrid")
	var now = time.Date(2026, 11, 2, 10, 0, 0, 0, madrid)
	var cases = []struct {
		name string
		in   ClosureInput
		want string
	}{
		{"two weeks of works", ClosureInput{StartsAt: "2026-11-15T00:00", EndsAt: "2026-11-30T23:59", Reason: " Obras na cuberta "}, ""},
		{"an evening event", ClosureInput{StartsAt: "2026-11-05T18:00", EndsAt: "2026-11-05T22:00", Reason: "Concerto"}, ""},
		{"already started, still on", ClosureInput{StartsAt: "2026-11-01T00:00", EndsAt: "2026-11-03T00:00", Reason: "Avaría"}, ""},
		{"end before start", ClosureInput{StartsAt: "2026-11-05T18:00", EndsAt: "2026-11-05T17:00", Reason: "x"}, "errors.invalid_closure_dates"},
		{"all in the past", ClosureInput{StartsAt: "2026-10-01T00:00", EndsAt: "2026-10-02T00:00", Reason: "x"}, "errors.invalid_closure_dates"},
		{"longer than a year", ClosureInput{StartsAt: "2026-11-05T00:00", EndsAt: "2027-12-05T00:00", Reason: "x"}, "errors.invalid_closure_dates"},
		{"bad date", ClosureInput{StartsAt: "05/11/2026", EndsAt: "2026-11-06T00:00", Reason: "x"}, "errors.invalid_closure_dates"},
		{"no reason", ClosureInput{StartsAt: "2026-11-05T18:00", EndsAt: "2026-11-05T22:00", Reason: "  "}, "errors.invalid_closure_reason"},
		{"reason too long", ClosureInput{StartsAt: "2026-11-05T18:00", EndsAt: "2026-11-05T22:00", Reason: strings.Repeat("a", 256)}, "errors.invalid_closure_reason"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var in = c.in
			var start, _, got = in.parse(madrid, now)
			if got != c.want {
				t.Fatalf("parse() = %q, want %q", got, c.want)
			}
			if c.name == "an evening event" && start.UTC().Hour() != 17 {
				t.Errorf("18:00 in Madrid is 17:00 UTC in November, got %v", start.UTC())
			}
		})
	}
}

func TestActiveInputValidate(t *testing.T) {
	var cases = []struct {
		in   ActiveInput
		want string
	}{
		{ActiveInput{Active: false}, ""},                                        // keep the bookings
		{ActiveInput{Active: false, CancelBookings: true, Reason: "Peche"}, ""}, // cancel them, with a reason
		{ActiveInput{Active: false, CancelBookings: true}, "errors.invalid_cancel_reason"},
		{ActiveInput{Active: true, CancelBookings: true}, ""}, // activating never cancels
	}
	for _, c := range cases {
		var in = c.in
		if got := in.validate(); got != c.want {
			t.Errorf("%+v: validate() = %q, want %q", c.in, got, c.want)
		}
		if c.in.Active && in.CancelBookings {
			t.Errorf("activating must not cancel bookings")
		}
	}
}
