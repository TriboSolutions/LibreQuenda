package reader

import (
	"database/sql"
	"librequenda/model"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDecide(t *testing.T) {
	var facility = uuid.New()
	var start = time.Date(2026, 11, 3, 18, 0, 0, 0, time.UTC)
	var booking = model.GetBookingByCheckinTokenRow{
		FacilityID: facility,
		Status:     model.BookingStatusConfirmed,
		StartsAt:   start,
		EndsAt:     start.Add(time.Hour),
	}
	var checkedIn = booking
	checkedIn.CheckedInAt = sql.NullTime{Time: start, Valid: true}
	var cancelled = booking
	cancelled.Status = model.BookingStatusCancelled

	var cases = []struct {
		name     string
		booking  model.GetBookingByCheckinTokenRow
		facility uuid.UUID
		now      time.Time
		want     string
	}{
		{"on time", booking, facility, start.Add(5 * time.Minute), ResultCheckedIn},
		{"15 min early", booking, facility, start.Add(-15 * time.Minute), ResultCheckedIn},
		{"16 min early", booking, facility, start.Add(-16 * time.Minute), ResultTooEarly},
		{"last minute", booking, facility, start.Add(59 * time.Minute), ResultCheckedIn},
		{"at the end", booking, facility, start.Add(time.Hour), ResultEnded},
		{"re-entry", checkedIn, facility, start.Add(30 * time.Minute), ResultAlreadyCheckedIn},
		{"other facility", booking, uuid.New(), start, ResultOtherFacility},
		{"cancelled", cancelled, facility, start, ResultCancelled},
	}
	for _, c := range cases {
		if got := decide(c.booking, c.facility, c.now); got != c.want {
			t.Errorf("%s: decide = %s, want %s", c.name, got, c.want)
		}
	}
}

func TestFirstName(t *testing.T) {
	for in, want := range map[string]string{"Uxía Proba": "Uxía", "  Ana  ": "Ana", "": ""} {
		if got := firstName(in); got != want {
			t.Errorf("firstName(%q) = %q, want %q", in, got, want)
		}
	}
}
