package tenants

import (
	"librequenda/schedule"
	"time"
)

type TenantDto struct {
	Slug    string `json:"slug"`
	Name    string `json:"name"`
	LogoUrl string `json:"logo_url"`
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

type FacilityDayDto struct {
	Id             string          `json:"id"`
	Slug           string          `json:"slug"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	ImageUrl       string          `json:"image_url"`
	Latitude       *float64        `json:"latitude"`
	Longitude      *float64        `json:"longitude"`
	Sport          string          `json:"sport"`
	Address        string          `json:"address"`
	SlotMinutes    int32           `json:"slot_minutes"`
	MaxAdvanceDays int32           `json:"max_advance_days"`
	MinCancelHours int32           `json:"min_cancel_hours"`
	MaxActive      int32           `json:"max_active_bookings_per_user"`
	MinAge         *int32          `json:"min_age"` // age control: null = none
	UsageRules     string          `json:"usage_rules"`
	Windows        []WindowDto     `json:"windows"`
	Closures       []ClosureDto    `json:"closures"`
	Slots          []schedule.Slot `json:"slots"`
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
