package bookings

import (
	"librequenda/model"
	"testing"
	"time"
)

func TestChangeable(t *testing.T) {
	var now = time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)
	var cases = []struct {
		name     string
		status   model.BookingStatus
		startsAt time.Time
		minHours int32
		want     string // "" when it can be changed
	}{
		{"well ahead", model.BookingStatusConfirmed, now.Add(48 * time.Hour), 24, ""},
		{"no deadline, not started", model.BookingStatusConfirmed, now.Add(time.Minute), 0, ""},
		{"past the deadline", model.BookingStatusConfirmed, now.Add(23 * time.Hour), 24, "errors.change_too_late"},
		{"started", model.BookingStatusConfirmed, now.Add(-time.Minute), 0, "errors.change_too_late"},
		{"cancelled", model.BookingStatusCancelled, now.Add(48 * time.Hour), 24, "errors.booking_already_cancelled"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got = changeable(c.status, c.startsAt, c.minHours, now)
			var message string
			if got != nil {
				message = got.Message
			}
			if message != c.want {
				t.Errorf("changeable() = %q, want %q", message, c.want)
			}
		})
	}
}
