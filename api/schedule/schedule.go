// Package schedule turns a facility's opening hours, closures and bookings into
// the bookable slots of one day. It is the single source of truth for both the
// public day view and the booking validation.
package schedule

import "time"

type SlotStatus string

const (
	SlotFree    SlotStatus = "free"
	SlotBooked  SlotStatus = "booked"
	SlotClosed  SlotStatus = "closed"   // a closure (holiday, maintenance) overlaps it
	SlotPast    SlotStatus = "past"     // already started
	SlotTooFar  SlotStatus = "too_far"  // beyond the facility's booking window
	SlotTooSoon SlotStatus = "too_soon" // starts sooner than the window lets the app book it
)

// Window is a local opening range, e.g. 09:00-22:00. Times only use hour and minute.
type Window struct {
	Opens  time.Time
	Closes time.Time
}

type Interval struct {
	Start time.Time
	End   time.Time
}

// Rules: the turns' length and the booking window, counted from now.
type Rules struct {
	SlotMinutes int
	MaxAdvance  time.Duration // a turn starting later than now + MaxAdvance is too far
	MinAdvance  time.Duration // one starting sooner is too soon; 0 for staff and the door
}

type Slot struct {
	Start  time.Time  `json:"starts_at"`
	End    time.Time  `json:"ends_at"`
	Status SlotStatus `json:"status"`
}

// DaySlots cuts each opening window of `day` (a date in `loc`) into slots of
// SlotMinutes. A trailing piece shorter than a slot is not bookable and is dropped.
func DaySlots(day time.Time, loc *time.Location, rules Rules, windows []Window, closures []Interval, busy []Interval, now time.Time) []Slot {
	var slots []Slot
	if rules.SlotMinutes <= 0 {
		return slots
	}
	for _, i := range cut(day, loc, rules.SlotMinutes, windows) {
		slots = append(slots, Slot{
			Start:  i.Start,
			End:    i.End,
			Status: status(i, closures, busy, now, rules),
		})
	}
	return slots
}

// Current is the rest of the turn under way at `now`: from now (to the minute)
// until the turn ends, free unless a closure or a booking overlaps that part.
// Used to book a turn that has already started (a walk-in at the reader).
func Current(day time.Time, loc *time.Location, slotMinutes int, windows []Window, closures []Interval, busy []Interval, now time.Time) (Slot, bool) {
	if slotMinutes <= 0 {
		return Slot{}, false
	}
	var start = now.Truncate(time.Minute)
	for _, i := range cut(day, loc, slotMinutes, windows) {
		if now.Before(i.Start) || !now.Before(i.End) {
			continue
		}
		var rest = Interval{Start: start, End: i.End}
		var status = SlotFree
		switch {
		case overlapsAny(rest, closures):
			status = SlotClosed
		case overlapsAny(rest, busy):
			status = SlotBooked
		}
		return Slot{Start: rest.Start, End: rest.End, Status: status}, true
	}
	return Slot{}, false
}

// OpenMinutes is how long a facility can be booked on `day`: its slots that no
// closure overlaps, booked or not. The occupancy indicator divides by it.
func OpenMinutes(day time.Time, loc *time.Location, slotMinutes int, windows []Window, closures []Interval) int {
	var minutes int
	if slotMinutes <= 0 {
		return minutes
	}
	for _, i := range cut(day, loc, slotMinutes, windows) {
		if !overlapsAny(i, closures) {
			minutes += slotMinutes
		}
	}
	return minutes
}

// cut splits each opening window of `day` into slots of slotMinutes (> 0).
func cut(day time.Time, loc *time.Location, slotMinutes int, windows []Window) []Interval {
	var slots []Interval
	var length time.Duration = time.Duration(slotMinutes) * time.Minute
	var y, m, d = day.Date()
	for _, w := range windows {
		var start time.Time = time.Date(y, m, d, w.Opens.Hour(), w.Opens.Minute(), 0, 0, loc)
		var closes time.Time = time.Date(y, m, d, w.Closes.Hour(), w.Closes.Minute(), 0, 0, loc)
		for end := start.Add(length); !end.After(closes); start, end = end, end.Add(length) {
			slots = append(slots, Interval{start, end})
		}
	}
	return slots
}

func status(slot Interval, closures []Interval, busy []Interval, now time.Time, rules Rules) SlotStatus {
	switch {
	case !slot.Start.After(now):
		return SlotPast
	case slot.Start.After(now.Add(rules.MaxAdvance)):
		return SlotTooFar
	case overlapsAny(slot, closures):
		return SlotClosed
	case overlapsAny(slot, busy):
		return SlotBooked
	case slot.Start.Before(now.Add(rules.MinAdvance)):
		return SlotTooSoon
	}
	return SlotFree
}

func overlapsAny(slot Interval, intervals []Interval) bool {
	for _, i := range intervals {
		if slot.Start.Before(i.End) && i.Start.Before(slot.End) {
			return true
		}
	}
	return false
}

func startOfDay(t time.Time, loc *time.Location) time.Time {
	var y, m, d = t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

// ISOWeekday returns 1 (Monday) to 7 (Sunday), as stored in opening_hours.weekday.
func ISOWeekday(day time.Time) int16 {
	var wd = day.Weekday()
	if wd == time.Sunday {
		return 7
	}
	return int16(wd)
}
