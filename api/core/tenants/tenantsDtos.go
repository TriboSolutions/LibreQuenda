package tenants

import (
	"librequenda/core/rating"
	"librequenda/schedule"
	"librequenda/shared"
	"time"
)

type TenantDto struct {
	Slug    string `json:"slug"`
	Name    string `json:"name"`
	LogoUrl string `json:"logo_url"`
	// public: where residents send accessibility feedback and complaints (RD 1112/2018)
	ContactEmail string `json:"contact_email"`
	ContactPhone string `json:"contact_phone"`
}

// FacilityCardDto is one facility on the home page's list and map, from any council.
type FacilityCardDto struct {
	Id        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Sport     string    `json:"sport"`
	Tags      []string  `json:"tags"`
	ImageUrl  string    `json:"image_url"`
	Address   string    `json:"address"`
	Latitude  *float64  `json:"latitude"` // null without a map pin
	Longitude *float64  `json:"longitude"`
	FreeToday int       `json:"free_today"` // turns still free today
	Council   TenantDto `json:"council"`
	// only the council's verified residents book or join there (also its events)
	ResidentsOnly bool `json:"residents_only"`
	// what can be joined there, upcoming and with spots left: "public" bookings, "event"s
	Joinable []string `json:"joinable"`
	// with ?lat=&lon=: from there, in km (0.1); null without a pin or a position
	DistanceKm *float64 `json:"distance_km"`
}

type FacilitiesResponse struct {
	// by council, then name; with ?lat=&lon=, nearest first and the ones without a pin last
	Facilities []FacilityCardDto `json:"facilities"`
	Pagination shared.PageDto    `json:"pagination"`
}

// FacilityAnyDayDto: a facility's day as /tenants/{slug}/day has it, with its
// council and how far it is, for the search across every council.
type FacilityAnyDayDto struct {
	FacilityDayDto
	Council    TenantDto `json:"council"`
	DistanceKm *float64  `json:"distance_km"`
}

// FacilitiesDayResponse: GET /facilities/day, every council's facilities for one day.
type FacilitiesDayResponse struct {
	Date string `json:"date"` // YYYY-MM-DD
	// by council, then name; with ?lat=&lon=, nearest first and the ones without a pin last
	Facilities []FacilityAnyDayDto `json:"facilities"`
	Pagination shared.PageDto      `json:"pagination"`
}

type TenantsResponse struct {
	Tenants []TenantDto `json:"tenants"`
}

type WindowDto struct {
	OpensAt  string `json:"opens_at"` // "09:00", council local time
	ClosesAt string `json:"closes_at"`
}

type ClosureDto struct {
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
	Reason   string    `json:"reason"`
}

// PublicTurnDto: a public booking, or one of the council's events, on the facility's day.
type PublicTurnDto struct {
	Id          string    `json:"id"`
	Kind        string    `json:"kind"`  // "public" or "event"
	Title       string    `json:"title"` // events only
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	Description string    `json:"description"`
	Capacity    int32     `json:"capacity"` // people in it at most: with the owner for a public booking, only who joins for an event
	// first names: the owner first (not for an event), then who joined; signed out, only the owner
	People []string `json:"people"`
	// everyone in it, owner included; what to count spots with, since people may leave names out
	PeopleCount int `json:"people_count"`
	// who made the public booking and answers for it; nil for an event
	Owner *PublicPersonDto `json:"owner"`
	// who joined, in order, each with their public rating; empty when signed out
	Joined []PublicPersonDto `json:"joined"`
}

// HidePeople leaves only who answers for each public booking: someone signed out
// sees the owner's name and how many are in it, never who joined.
func (d *FacilityDayDto) HidePeople() {
	for i := range d.PublicBookings {
		var b = &d.PublicBookings[i]
		b.Joined = []PublicPersonDto{}
		b.People = []string{}
		if b.Owner != nil {
			b.People = []string{b.Owner.Name}
		}
	}
}

// PublicPersonDto: someone in a public booking, by first name, with the stars others gave them.
type PublicPersonDto struct {
	Name   string          `json:"name"`
	Rating *rating.Summary `json:"rating"` // nil until someone rated them, and for guests
}

type FacilityDayDto struct {
	PublicBookings    []PublicTurnDto `json:"public_bookings"`
	ShowWeather       bool            `json:"show_weather"`
	Id                string          `json:"id"`
	Slug              string          `json:"slug"`
	Name              string          `json:"name"`
	Description       string          `json:"description"`
	ImageUrl          string          `json:"image_url"`
	Latitude          *float64        `json:"latitude"`
	Longitude         *float64        `json:"longitude"`
	Sport             string          `json:"sport"`
	Tags              []string        `json:"tags"`
	Address           string          `json:"address"`
	ContactPhone      string          `json:"contact_phone"` // its own, "" when not given: the web shows the council's instead
	ContactEmail      string          `json:"contact_email"`
	SlotMinutes       int32           `json:"slot_minutes"`
	MaxAdvanceHours   int32           `json:"max_advance_hours"`   // the booking window, from now
	MinAdvanceMinutes int32           `json:"min_advance_minutes"` // the app books no turn starting sooner
	MinCancelHours    int32           `json:"min_cancel_hours"`
	MaxActive         int32           `json:"max_active_bookings_per_user"`
	MaxPerDay         int32           `json:"max_bookings_per_day"`        // per person and day; 0 = no limit
	MaxInARow         int32           `json:"max_turns_in_a_row"`          // back-to-back turns per person; 0 = no limit
	MaxPerWeek        int32           `json:"max_bookings_per_week"`       // per person and week; 0 = no limit
	Capacity          int32           `json:"capacity"`                    // people per booking, owner included; 0 without a limit
	CouncilMaxActive  int32           `json:"council_max_active_bookings"` // per person across the council's facilities; 0 = no limit
	// instead of max_advance_hours: next week's turns open all at once; null with a rolling window
	WeeklyOpening *WeeklyOpeningDto `json:"weekly_opening"`
	MinAge        *int32            `json:"min_age"`        // age control: null = none
	ResidentsOnly bool              `json:"residents_only"` // only the council's verified residents book or join
	UsageRules    string            `json:"usage_rules"`
	Windows       []WindowDto       `json:"windows"`
	Closures      []ClosureDto      `json:"closures"`
	Slots         []schedule.Slot   `json:"slots"`
}

type DayResponse struct {
	Tenant     TenantDto        `json:"tenant"`
	Timezone   string           `json:"timezone"`
	Date       string           `json:"date"` // YYYY-MM-DD
	Now        time.Time        `json:"now"`
	Facilities []FacilityDayDto `json:"facilities"`
}

type WeeklyWindowDto struct {
	Weekday  int16  `json:"weekday"` // ISO: 1 = Monday
	OpensAt  string `json:"opens_at"`
	ClosesAt string `json:"closes_at"`
}

// FacilityPageResponse: one facility's page, with the slots of the chosen day.
type FacilityPageResponse struct {
	Tenant   TenantDto         `json:"tenant"`
	Timezone string            `json:"timezone"`
	Date     string            `json:"date"`
	Now      time.Time         `json:"now"`
	Facility FacilityDayDto    `json:"facility"`
	Schedule []WeeklyWindowDto `json:"schedule"`
	Rating   *rating.Summary   `json:"rating"` // residents' stars after their bookings; null until someone rated
}

// WeeklyOpeningDto: next week's turns open on Weekday (ISO, 7 = Sunday) at At ("20:00", council time).
type WeeklyOpeningDto struct {
	Weekday int16  `json:"weekday"`
	At      string `json:"at"`
}
