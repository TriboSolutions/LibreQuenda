package schedule

import (
	"testing"
	"time"
)

func clock(h, m int) time.Time { return time.Date(0, 1, 1, h, m, 0, 0, time.UTC) }

func TestDaySlots(t *testing.T) {
	var loc, _ = time.LoadLocation("Europe/Madrid")
	var day = time.Date(2026, 10, 27, 0, 0, 0, 0, loc)
	var at = func(h, m int) time.Time { return time.Date(2026, 10, 27, h, m, 0, 0, loc) }
	var now = at(10, 15)

	var windows = []Window{{clock(9, 0), clock(12, 0)}, {clock(17, 0), clock(19, 30)}}
	var closures = []Interval{{at(17, 0), at(18, 0)}}
	var busy = []Interval{{at(11, 0), at(12, 0)}}
	var rules = Rules{SlotMinutes: 60, MaxAdvanceDays: 14}

	var slots = DaySlots(day, loc, rules, windows, closures, busy, now)

	var want = []struct {
		h      int
		status SlotStatus
	}{
		{9, SlotPast}, {10, SlotPast}, {11, SlotBooked}, // 10:00 started before 10:15
		{17, SlotClosed}, {18, SlotFree}, // 19:00-20:00 does not fit before 19:30
	}
	if len(slots) != len(want) {
		t.Fatalf("expected %d slots, got %d: %+v", len(want), len(slots), slots)
	}
	for i, w := range want {
		if slots[i].Start.Hour() != w.h || slots[i].Status != w.status {
			t.Errorf("slot %d: expected %02d:00 %s, got %s %s", i, w.h, w.status, slots[i].Start.Format("15:04"), slots[i].Status)
		}
	}
}

func TestDaySlotsBookingWindow(t *testing.T) {
	var loc, _ = time.LoadLocation("Europe/Madrid")
	var now = time.Date(2026, 10, 27, 12, 0, 0, 0, loc)
	var rules = Rules{SlotMinutes: 90, MaxAdvanceDays: 2}
	var windows = []Window{{clock(9, 0), clock(12, 0)}}

	var inWindow = DaySlots(time.Date(2026, 10, 29, 0, 0, 0, 0, loc), loc, rules, windows, nil, nil, now)
	var tooFar = DaySlots(time.Date(2026, 10, 30, 0, 0, 0, 0, loc), loc, rules, windows, nil, nil, now)

	if len(inWindow) != 2 || inWindow[0].Status != SlotFree {
		t.Errorf("expected 2 free 90-minute slots two days ahead, got %+v", inWindow)
	}
	if tooFar[0].Status != SlotTooFar {
		t.Errorf("expected too_far three days ahead, got %s", tooFar[0].Status)
	}
}

func TestISOWeekday(t *testing.T) {
	if ISOWeekday(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) != 7 {
		t.Error("Sunday must be 7")
	}
	if ISOWeekday(time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)) != 1 {
		t.Error("Monday must be 1")
	}
}

func TestOpenMinutes(t *testing.T) {
	var loc, _ = time.LoadLocation("Europe/Madrid")
	var day = time.Date(2026, 10, 27, 0, 0, 0, 0, loc)
	var at = func(h, m int) time.Time { return time.Date(2026, 10, 27, h, m, 0, 0, loc) }
	var windows = []Window{{clock(9, 0), clock(12, 0)}, {clock(17, 0), clock(19, 30)}}

	// five 60-minute slots (19:00-20:00 does not fit), one of them closed
	if got := OpenMinutes(day, loc, 60, windows, []Interval{{at(17, 0), at(18, 0)}}); got != 4*60 {
		t.Errorf("OpenMinutes() = %d, want %d", got, 4*60)
	}
	if got := OpenMinutes(day, loc, 0, windows, nil); got != 0 {
		t.Errorf("no slot length, no open time: got %d", got)
	}
}
