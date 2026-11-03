package bookings

import (
	"testing"
	"time"

	"librequenda/model"
)

// A participant can leave until 15 minutes before the start, never after.
func TestCanLeaveAt(t *testing.T) {
	var start = time.Date(2026, 11, 3, 18, 0, 0, 0, time.UTC)
	var cases = []struct {
		name   string
		status model.BookingStatus
		now    time.Time
		want   bool
	}{
		{"an hour before", model.BookingStatusConfirmed, start.Add(-time.Hour), true},
		{"16 minutes before", model.BookingStatusConfirmed, start.Add(-16 * time.Minute), true},
		{"15 minutes before", model.BookingStatusConfirmed, start.Add(-15 * time.Minute), false},
		{"5 minutes before", model.BookingStatusConfirmed, start.Add(-5 * time.Minute), false},
		{"started", model.BookingStatusConfirmed, start.Add(time.Minute), false},
		{"cancelled", model.BookingStatusCancelled, start.Add(-time.Hour), false},
	}
	for _, c := range cases {
		if got := CanLeaveAt(c.status, start, c.now); got != c.want {
			t.Errorf("%s: CanLeaveAt = %v, want %v", c.name, got, c.want)
		}
	}
}
