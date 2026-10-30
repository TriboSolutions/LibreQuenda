package tenants

import (
	"librequenda/schedule"
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

// TenantCardDto is a council on the home page: who it is and how it uses LibreQuenda.
type TenantCardDto struct {
	TenantDto
	Stats TenantStatsDto `json:"stats"`
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
}

type FacilitiesResponse struct {
	Facilities []FacilityCardDto `json:"facilities"` // by council, then name
}

type TenantStatsDto struct {
	Facilities int32    `json:"facilities"` // active ones
	Sports     []string `json:"sports"`
	FreeToday  int      `json:"free_today"` // turns still free today, across its facilities
	Bookings   int32    `json:"bookings"`   // confirmed, past and upcoming
	Residents  int32    `json:"residents"`  // distinct accounts that have booked
}

type TenantsResponse struct {
	Tenants []TenantCardDto `json:"tenants"`
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

type FacilityDayDto struct {
	Id                string   `json:"id"`
	Slug              string   `json:"slug"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	ImageUrl          string   `json:"image_url"`
	Latitude          *float64 `json:"latitude"`
	Longitude         *float64 `json:"longitude"`
	Sport             string   `json:"sport"`
	Tags              []string `json:"tags"`
	Address           string   `json:"address"`
	ContactPhone      string   `json:"contact_phone"` // its own, "" when not given: the web shows the council's instead
	ContactEmail      string   `json:"contact_email"`
	SlotMinutes       int32    `json:"slot_minutes"`
	MaxAdvanceHours   int32    `json:"max_advance_hours"`   // the booking window, from now
	MinAdvanceMinutes int32    `json:"min_advance_minutes"` // the app books no turn starting sooner
	MinCancelHours    int32    `json:"min_cancel_hours"`
	MaxActive         int32    `json:"max_active_bookings_per_user"`
	MaxPerDay         int32    `json:"max_bookings_per_day"`        // per person and day; 0 = no limit
	MaxInARow         int32    `json:"max_turns_in_a_row"`          // back-to-back turns per person; 0 = no limit
	MaxPerWeek        int32    `json:"max_bookings_per_week"`       // per person and week; 0 = no limit
	Capacity          int32    `json:"capacity"`                    // people per booking, owner included; 0 without a limit
	CouncilMaxActive  int32    `json:"council_max_active_bookings"` // per person across the council's facilities; 0 = no limit
	// instead of max_advance_hours: next week's turns open all at once; null with a rolling window
	WeeklyOpening *WeeklyOpeningDto `json:"weekly_opening"`
	MinAge        *int32            `json:"min_age"` // age control: null = none
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
}

// WeeklyOpeningDto: next week's turns open on Weekday (ISO, 7 = Sunday) at At ("20:00", council time).
type WeeklyOpeningDto struct {
	Weekday int16  `json:"weekday"`
	At      string `json:"at"`
}
