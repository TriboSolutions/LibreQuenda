package bookings

import (
	"librequenda/schedule"
	"testing"
	"time"
)

func TestNextTurn(t *testing.T) {
	var at = func(hour, minute int) time.Time { return time.Date(2026, 11, 3, hour, minute, 0, 0, time.UTC) }
	var slots = []schedule.Slot{
		{Start: at(17, 0), End: at(18, 0), Status: schedule.SlotPast},
		{Start: at(18, 0), End: at(19, 0), Status: schedule.SlotFree},
		{Start: at(19, 0), End: at(20, 0), Status: schedule.SlotBooked},
	}
	var cases = []struct {
		name  string
		now   time.Time
		want  time.Time
		found bool
	}{
		{"15 min before", at(17, 45), at(18, 0), true},
		{"1 min before", at(17, 59), at(18, 0), true},
		{"right at the start", at(18, 0), at(18, 0), true},
		{"16 min before", at(17, 44), time.Time{}, false},
		{"once it has started", at(18, 1), time.Time{}, false},
		{"booked turn", at(18, 50), at(19, 0), true},
		{"after the last turn", at(19, 50), time.Time{}, false},
	}
	for _, c := range cases {
		var slot, found = NextTurn(slots, c.now)
		if found != c.found || !slot.Start.Equal(c.want) {
			t.Errorf("%s: NextTurn = %s, %v; want %s, %v", c.name, slot.Start, found, c.want, c.found)
		}
	}
}
