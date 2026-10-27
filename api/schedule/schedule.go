// Package schedule turns a facility's opening hours, closures and bookings into
// the bookable slots of one day. It is the single source of truth for both the
// public day view and the booking validation.
package schedule

import "time"

type SlotStatus string

const (
	SlotFree   SlotStatus = "free"
	SlotBooked SlotStatus = "booked"
	SlotClosed SlotStatus = "closed"  // a closure (holiday, maintenance) overlaps it
	SlotPast   SlotStatus = "past"    // already started
	SlotTooFar SlotStatus = "too_far" // beyond the facility's booking window
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

type Rules struct {
	SlotMinutes    int
	MaxAdvanceDays int
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
	var length time.Duration = time.Duration(rules.SlotMinutes) * time.Minute
	var y, m, d = day.Date()
	var lastBookableDay time.Time = startOfDay(now.In(loc), loc).AddDate(0, 0, rules.MaxAdvanceDays+1)

	for _, w := range windows {
		var start time.Time = time.Date(y, m, d, w.Opens.Hour(), w.Opens.Minute(), 0, 0, loc)
		var closes time.Time = time.Date(y, m, d, w.Closes.Hour(), w.Closes.Minute(), 0, 0, loc)
		for end := start.Add(length); !end.After(closes); start, end = end, end.Add(length) {
			slots = append(slots, Slot{
				Start:  start,
				End:    end,
				Status: status(Interval{start, end}, closures, busy, now, lastBookableDay),
			})
		}
	}
	return slots
}

func status(slot Interval, closures []Interval, busy []Interval, now time.Time, lastBookableDay time.Time) SlotStatus {
	if !slot.Start.After(now) {
		return SlotPast
	}
	if !slot.Start.Before(lastBookableDay) {
		return SlotTooFar
	}
	if overlapsAny(slot, closures) {
		return SlotClosed
	}
	if overlapsAny(slot, busy) {
		return SlotBooked
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
