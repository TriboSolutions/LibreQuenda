package ratings

import (
	"librequenda/model"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCanRate(t *testing.T) {
	var now = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	var confirmed, cancelled = model.BookingStatusConfirmed, model.BookingStatusCancelled
	var cases = []struct {
		name           string
		status         model.BookingStatus
		endsAt         time.Time
		checkedIn      bool
		checkinAtStart bool
		want           bool
	}{
		{"just ended, no way to check in", confirmed, now, false, false, true},
		{"ended and checked in", confirmed, now.Add(-time.Hour), true, true, true},
		{"not over yet", confirmed, now.Add(time.Minute), true, true, false},
		{"no-show", confirmed, now.Add(-time.Hour), false, true, false},
		{"cancelled", cancelled, now.Add(-time.Hour), false, false, false},
		{"last moment of the window", confirmed, now.Add(-Window + time.Second), false, false, true},
		{"window closed", confirmed, now.Add(-Window), false, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CanRate(c.status, c.endsAt, c.checkedIn, c.checkinAtStart, now); got != c.want {
				t.Errorf("CanRate() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestRateInputValidate(t *testing.T) {
	var a, b = uuid.New(), uuid.New()
	var cases = []struct {
		name string
		in   RateInput
		want string
	}{
		{"facility only", RateInput{FacilityStars: 4}, ""},
		{"people only", RateInput{People: []PersonStarsInput{{UserId: a, Stars: 5}}}, ""},
		{"both, one person skipped", RateInput{FacilityStars: 1, People: []PersonStarsInput{{UserId: a, Stars: 3}, {UserId: b}}}, ""},
		{"nothing", RateInput{}, "errors.invalid_rating"},
		{"only skips", RateInput{People: []PersonStarsInput{{UserId: a}}}, "errors.invalid_rating"},
		{"six stars", RateInput{FacilityStars: 6}, "errors.invalid_rating"},
		{"negative", RateInput{FacilityStars: -1}, "errors.invalid_rating"},
		{"person six stars", RateInput{People: []PersonStarsInput{{UserId: a, Stars: 6}}}, "errors.invalid_rating"},
		{"same person twice", RateInput{People: []PersonStarsInput{{UserId: a, Stars: 2}, {UserId: a, Stars: 3}}}, "errors.invalid_rating"},
		{"no id", RateInput{People: []PersonStarsInput{{Stars: 2}}}, "errors.invalid_rating"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.in.Validate(); got != c.want {
				t.Errorf("Validate() = %q, want %q", got, c.want)
			}
		})
	}
}
