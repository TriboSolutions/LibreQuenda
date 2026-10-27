package counciladmin

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Deactivating keeps the bookings already made unless the council asks to cancel them.
type ActiveInput struct {
	Active         bool   `json:"active"`
	CancelBookings bool   `json:"cancel_bookings"` // only when deactivating
	Reason         string `json:"reason"`          // shown to whoever booked; required to cancel
}

func (in *ActiveInput) validate() string {
	in.Reason = strings.TrimSpace(in.Reason)
	if in.Active {
		in.CancelBookings = false
	}
	if in.CancelBookings && !validReason(in.Reason) {
		return "errors.invalid_cancel_reason"
	}
	return ""
}

type ActiveResultDto struct {
	Facility  *FacilityDetailDto `json:"facility"`
	Cancelled int64              `json:"cancelled"`
}

// A closure: dates in the council's time zone, as the admin typed them.
type ClosureInput struct {
	StartsAt string `json:"starts_at"` // "2026-11-15T00:00"
	EndsAt   string `json:"ends_at"`
	Reason   string `json:"reason"` // shown on the facility page and in the "sorry" notice
}

const localDateTime = "2006-01-02T15:04"
const maxClosureDays = 366

// parse answers the closure in absolute time, or the i18n key of the problem.
func (in *ClosureInput) parse(loc *time.Location, now time.Time) (time.Time, time.Time, string) {
	in.Reason = strings.TrimSpace(in.Reason)
	var start, startErr = time.ParseInLocation(localDateTime, in.StartsAt, loc)
	var end, endErr = time.ParseInLocation(localDateTime, in.EndsAt, loc)
	if startErr != nil || endErr != nil || !end.After(start) || !end.After(now) || end.Sub(start) > maxClosureDays*24*time.Hour {
		return time.Time{}, time.Time{}, "errors.invalid_closure_dates"
	}
	if !validReason(in.Reason) {
		return time.Time{}, time.Time{}, "errors.invalid_closure_reason"
	}
	return start, end, ""
}

func validReason(reason string) bool {
	return reason != "" && utf8.RuneCountInString(reason) <= 255
}

type ClosureDto struct {
	Id       string    `json:"id"`
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
	Reason   string    `json:"reason"`
}

type ClosurePreviewDto struct {
	BookingsToCancel int64 `json:"bookings_to_cancel"`
}

type ClosureResultDto struct {
	Closure   ClosureDto `json:"closure"`
	Cancelled int64      `json:"cancelled"`
}
