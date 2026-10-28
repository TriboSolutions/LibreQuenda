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

func TestFlashTurn(t *testing.T) {
	var at = func(hour, minute int) time.Time { return time.Date(2026, 11, 3, hour, minute, 0, 0, time.UTC) }
	var turn = func(start, end time.Time, status schedule.SlotStatus) schedule.Slot {
		return schedule.Slot{Start: start, End: end, Status: status}
	}
	var cases = []struct {
		name       string
		now        time.Time
		current    schedule.Slot
		inProgress bool
		next       schedule.SlotStatus // of the 17:00 turn
		want       schedule.Slot
		reason     string
	}{
		{"the rest of a free turn", at(16, 15), turn(at(16, 15), at(17, 0), schedule.SlotFree), true, schedule.SlotBooked,
			turn(at(16, 15), at(17, 0), schedule.SlotFree), ""},
		{"little left: the next free turn", at(16, 50), turn(at(16, 50), at(17, 0), schedule.SlotFree), true, schedule.SlotFree,
			turn(at(17, 0), at(18, 0), schedule.SlotFree), ""},
		{"little left and the next is booked: what is left", at(16, 50), turn(at(16, 50), at(17, 0), schedule.SlotFree), true, schedule.SlotBooked,
			turn(at(16, 50), at(17, 0), schedule.SlotFree), ""},
		{"booked now, the next free", at(16, 50), turn(at(16, 50), at(17, 0), schedule.SlotBooked), true, schedule.SlotFree,
			turn(at(17, 0), at(18, 0), schedule.SlotFree), ""},
		{"booked now and later", at(16, 50), turn(at(16, 50), at(17, 0), schedule.SlotBooked), true, schedule.SlotBooked,
			schedule.Slot{}, "slot_booked"},
		{"before opening, the first turn", at(16, 50), schedule.Slot{}, false, schedule.SlotFree,
			turn(at(17, 0), at(18, 0), schedule.SlotFree), ""},
		{"closed now, nothing soon", at(16, 20), turn(at(16, 20), at(17, 0), schedule.SlotClosed), true, schedule.SlotFree,
			schedule.Slot{}, "slot_closed"},
		{"nothing open", at(14, 0), schedule.Slot{}, false, schedule.SlotFree,
			schedule.Slot{}, "no_turn_soon"},
	}
	for _, c := range cases {
		var slots = []schedule.Slot{turn(at(16, 0), at(17, 0), schedule.SlotPast), turn(at(17, 0), at(18, 0), c.next)}
		var slot, reason = FlashTurn(c.current, c.inProgress, slots, c.now)
		if reason != c.reason || slot != c.want {
			t.Errorf("%s: FlashTurn = %s-%s %q; want %s-%s %q", c.name, slot.Start.Format("15:04"), slot.End.Format("15:04"), reason, c.want.Start.Format("15:04"), c.want.End.Format("15:04"), c.reason)
		}
	}
}
