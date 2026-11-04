package invites

import (
	"database/sql"
	"librequenda/model"
	"strings"
	"testing"
	"time"
)

func TestInviteState(t *testing.T) {
	var now = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	var later = now.Add(time.Hour)
	var ten = sql.NullInt32{Int32: 10, Valid: true}
	var cases = []struct {
		name         string
		status       model.BookingStatus
		startsAt     time.Time
		capacity     sql.NullInt32
		kind         model.BookingKind
		participants int64
		want         string
		wantSpots    int
	}{
		{"open", model.BookingStatusConfirmed, later, ten, model.BookingKindPrivate, 0, StateOpen, 9},
		{"last spot", model.BookingStatusConfirmed, later, ten, model.BookingKindPrivate, 8, StateOpen, 1},
		{"full: owner + 9", model.BookingStatusConfirmed, later, ten, model.BookingKindPrivate, 9, StateFull, 0},
		{"cancelled", model.BookingStatusCancelled, later, ten, model.BookingKindPrivate, 0, StateClosed, 9},
		{"started", model.BookingStatusConfirmed, now, ten, model.BookingKindPrivate, 0, StateClosed, 9},
		{"no capacity", model.BookingStatusConfirmed, later, sql.NullInt32{}, model.BookingKindPrivate, 0, StateClosed, 0},
		{"capacity one", model.BookingStatusConfirmed, later, sql.NullInt32{Int32: 1, Valid: true}, model.BookingKindPrivate, 0, StateClosed, 0},
		{"event: no owner's place", model.BookingStatusConfirmed, later, ten, model.BookingKindEvent, 9, StateOpen, 1},
		{"event full", model.BookingStatusConfirmed, later, ten, model.BookingKindEvent, 10, StateFull, 0},
		{"event for one", model.BookingStatusConfirmed, later, sql.NullInt32{Int32: 1, Valid: true}, model.BookingKindEvent, 0, StateOpen, 1},
		{"event without capacity", model.BookingStatusConfirmed, later, sql.NullInt32{}, model.BookingKindEvent, 0, StateClosed, 0},
		{"capacity lowered below participants", model.BookingStatusConfirmed, later, sql.NullInt32{Int32: 3, Valid: true}, model.BookingKindPrivate, 5, StateFull, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := inviteState(c.status, c.startsAt, c.capacity, c.kind, c.participants, now); got != c.want {
				t.Errorf("inviteState() = %q, want %q", got, c.want)
			}
			if got := spotsLeft(c.capacity, c.kind, c.participants); got != c.wantSpots {
				t.Errorf("spotsLeft() = %d, want %d", got, c.wantSpots)
			}
		})
	}
}

func TestGuestJoinValidate(t *testing.T) {
	var cases = []struct {
		name     string
		in       GuestJoinDto
		want     string
		wantName string
	}{
		{"ok", GuestJoinDto{Name: "  Uxía   Pérez ", Age: 14}, "", "Uxía Pérez"},
		{"blank name", GuestJoinDto{Name: "   ", Age: 30}, "errors.invalid_guest_name", ""},
		{"long name", GuestJoinDto{Name: strings.Repeat("á", 81), Age: 30}, "errors.invalid_guest_name", ""},
		{"80 letters", GuestJoinDto{Name: strings.Repeat("á", 80), Age: 30}, "", strings.Repeat("á", 80)},
		{"age zero", GuestJoinDto{Name: "Iago", Age: 0}, "errors.invalid_guest_age", ""},
		{"age too high", GuestJoinDto{Name: "Iago", Age: 121}, "errors.invalid_guest_age", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var in = c.in
			if got := in.Validate(); got != c.want {
				t.Errorf("Validate() = %q, want %q", got, c.want)
			}
			if c.want == "" && in.Name != c.wantName {
				t.Errorf("name = %q, want %q", in.Name, c.wantName)
			}
		})
	}
}

func TestFirstName(t *testing.T) {
	for in, want := range map[string]string{"Uxía Pérez Lago": "Uxía", " Iago ": "Iago", "": ""} {
		if got := firstName(in); got != want {
			t.Errorf("firstName(%q) = %q, want %q", in, got, want)
		}
	}
}
