package ratings

import (
	"librequenda/model"
	"time"

	"github.com/google/uuid"
)

// Window: a booking can be rated from its end for this long. Once given, the stars are final.
const Window = 30 * 24 * time.Hour

// RatingDto: what the rating dialog shows for one booking (GET /me/bookings/{id}/rating).
type RatingDto struct {
	BookingId     string      `json:"booking_id"`
	FacilityName  string      `json:"facility_name"`
	FacilitySlug  string      `json:"facility_slug"`
	TenantSlug    string      `json:"tenant_slug"`
	TenantName    string      `json:"tenant_name"`
	StartsAt      time.Time   `json:"starts_at"`
	EndsAt        time.Time   `json:"ends_at"`
	IsPublic      bool        `json:"is_public"`
	CanRate       bool        `json:"can_rate"`       // ended, not a no-show, within the window, not rated yet
	Rated         bool        `json:"rated"`          // rated already: the stars below are final
	RateUntil     time.Time   `json:"rate_until"`     // the end of the window
	FacilityStars int16       `json:"facility_stars"` // what this person gave; 0 = not yet
	People        []PersonDto `json:"people"`         // public bookings only: the others with an account
}

// PersonDto: someone else in a public booking, and the stars this person gave them (0 = not yet).
type PersonDto struct {
	UserId  string `json:"user_id"`
	Name    string `json:"name"`
	IsOwner bool   `json:"is_owner"`
	Stars   int16  `json:"stars"`
}

// RateInput (PUT /me/bookings/{id}/rating): the facility's stars and, after a
// public booking, the others' (0: left unrated). Saved once, for good.
type RateInput struct {
	FacilityStars int16              `json:"facility_stars"`
	People        []PersonStarsInput `json:"people"`
}

type PersonStarsInput struct {
	UserId uuid.UUID `json:"user_id"`
	Stars  int16     `json:"stars"`
}

// Validate: the facility's stars from 1 to 5, required (the rating is saved
// once, so it cannot come later); the others' from 1 to 5 or 0 to skip, each once.
func (in *RateInput) Validate() string {
	if in.FacilityStars == 0 {
		return "errors.rating_facility_required"
	}
	if in.FacilityStars < 0 || in.FacilityStars > 5 {
		return "errors.invalid_rating"
	}
	var seen = map[uuid.UUID]bool{}
	for _, p := range in.People {
		if p.UserId == uuid.Nil || seen[p.UserId] || p.Stars < 0 || p.Stars > 5 {
			return "errors.invalid_rating"
		}
		seen[p.UserId] = true
	}
	return ""
}

// CanRate: a confirmed booking can be rated once it ended, until the window
// closes, unless nobody checked in where someone could have (a no-show), and
// only once: rated, the stars are final.
func CanRate(status model.BookingStatus, endsAt time.Time, checkedIn, checkinAtStart, rated bool, now time.Time) bool {
	return status == model.BookingStatusConfirmed &&
		!rated &&
		!endsAt.After(now) &&
		now.Before(endsAt.Add(Window)) &&
		(checkedIn || !checkinAtStart)
}
