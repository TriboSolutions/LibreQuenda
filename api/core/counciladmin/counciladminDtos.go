package counciladmin

import (
	"librequenda/storage"
	"librequenda/validate"
	"sort"
	"strings"
	"time"
)

// Paths under /<council>/ the web app uses itself.
var reservedFacilitySlugs = map[string]bool{"admin": true, "reservas": true, "me": true}

type FacilityInput struct {
	Slug              string   `json:"slug"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Tags              []string `json:"tags"` // slugs from the platform's tag catalogue: its sports, kind of place…
	Address           string   `json:"address"`
	Latitude          *float64 `json:"latitude"`
	Longitude         *float64 `json:"longitude"`
	Capacity          *int32   `json:"capacity"`
	AccessibilityInfo string   `json:"accessibility_info"`
	ContactPhone      string   `json:"contact_phone"` // optional: without them, residents see the council's
	ContactEmail      string   `json:"contact_email"`
	ImageRef          string   `json:"image_ref"` // a storage reference from POST /media, never a URL
	IsActive          bool     `json:"is_active"`
	SlotMinutes       int32    `json:"slot_minutes"`
	// the booking window: the council's (custom_rules false: Rules is ignored) or its own
	CustomRules bool            `json:"custom_rules"`
	Rules       BookingRulesDto `json:"rules"`
	MinAge      *int32          `json:"min_age"` // age control: null = none, else 1 to 80
	UsageRules  string          `json:"usage_rules"`
	// create only: the weekly hours, saved with it, so a new facility is set up in one go
	Schedule *ScheduleInput `json:"schedule,omitempty"`
}

func (in *FacilityInput) normalise() {
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	in.Tags = uniqueTags(in.Tags)
	in.Address = strings.TrimSpace(in.Address)
	in.AccessibilityInfo = strings.TrimSpace(in.AccessibilityInfo)
	in.ContactPhone = strings.TrimSpace(in.ContactPhone)
	in.ContactEmail = strings.ToLower(strings.TrimSpace(in.ContactEmail))
	in.ImageRef = strings.TrimSpace(in.ImageRef)
	in.UsageRules = strings.TrimSpace(in.UsageRules)
}

// validate returns the i18n key of the first invalid field, or "".
func (in *FacilityInput) validate() string {
	switch {
	case in.Name == "" || len(in.Name) > 255:
		return "errors.invalid_facility_name"
	case !validate.Slug(in.Slug):
		return "errors.invalid_slug"
	case reservedFacilitySlugs[in.Slug]:
		return "errors.slug_reserved"
	case !validTags(in.Tags):
		return "errors.invalid_tags"
	case len(in.Address) > 500 || len(in.Description) > 5000 || len(in.UsageRules) > 5000 || len(in.AccessibilityInfo) > 2000:
		return "errors.invalid_text_length"
	case len(in.ContactPhone) > 50:
		return "errors.invalid_phone"
	case in.ContactEmail != "" && !validate.Email(in.ContactEmail):
		return "errors.invalid_email"
	case (in.Latitude == nil) != (in.Longitude == nil),
		in.Latitude != nil && (*in.Latitude < -90 || *in.Latitude > 90 || *in.Longitude < -180 || *in.Longitude > 180):
		return "errors.invalid_coordinates"
	case in.Capacity != nil && (*in.Capacity < 1 || *in.Capacity > 100000):
		return "errors.invalid_capacity"
	case in.ImageRef != "" && (!storage.ValidRef(in.ImageRef) || !strings.HasPrefix(in.ImageRef, "facilities/")):
		return "errors.invalid_image"
	case in.SlotMinutes < 15 || in.SlotMinutes > 480 || in.SlotMinutes%5 != 0:
		return "errors.invalid_slot_minutes"
	case in.CustomRules && in.Rules.validate() != "":
		return in.Rules.validate()
	case in.MinAge != nil && (*in.MinAge < 1 || *in.MinAge > 80):
		return "errors.invalid_min_age"
	}
	return ""
}

const maxFacilityTags = 40

// uniqueTags keeps each tag slug once, in the order given.
func uniqueTags(tags []string) []string {
	var out = []string{}
	var seen = map[string]bool{}
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if !seen[tag] {
			seen[tag] = true
			out = append(out, tag)
		}
	}
	return out
}

// validTags checks the shape only; whether each tag is in the catalogue is known when it is saved.
func validTags(tags []string) bool {
	if len(tags) > maxFacilityTags {
		return false
	}
	for _, tag := range tags {
		if !validate.Slug(tag) {
			return false
		}
	}
	return true
}

type WindowInput struct {
	Weekday  int16  `json:"weekday"` // ISO: 1 = Monday
	OpensAt  string `json:"opens_at"`
	ClosesAt string `json:"closes_at"`
}

type ScheduleInput struct {
	Windows []WindowInput `json:"windows"`
}

type parsedWindow struct {
	weekday int16
	opens   time.Time
	closes  time.Time
}

const maxWindowsPerDay = 6

// parse checks the weekly schedule: valid times, closing after opening, no overlaps on a day.
func (in *ScheduleInput) parse() ([]parsedWindow, string) {
	var windows []parsedWindow
	var perDay = map[int16][]parsedWindow{}
	for _, w := range in.Windows {
		var opens, errOpen = time.Parse("15:04", w.OpensAt)
		var closes, errClose = time.Parse("15:04", w.ClosesAt)
		if w.Weekday < 1 || w.Weekday > 7 || errOpen != nil || errClose != nil || !closes.After(opens) {
			return nil, "errors.invalid_schedule"
		}
		var p = parsedWindow{w.Weekday, opens, closes}
		perDay[w.Weekday] = append(perDay[w.Weekday], p)
		windows = append(windows, p)
	}
	for _, day := range perDay {
		if len(day) > maxWindowsPerDay {
			return nil, "errors.invalid_schedule"
		}
		sort.Slice(day, func(i, j int) bool { return day[i].opens.Before(day[j].opens) })
		for i := 1; i < len(day); i++ {
			if day[i].opens.Before(day[i-1].closes) {
				return nil, "errors.schedule_overlap"
			}
		}
	}
	return windows, ""
}

type WindowDto struct {
	Weekday  int16  `json:"weekday"`
	OpensAt  string `json:"opens_at"`
	ClosesAt string `json:"closes_at"`
}

type FacilitySummaryDto struct {
	Id                   string   `json:"id"`
	Slug                 string   `json:"slug"`
	Name                 string   `json:"name"`
	Sport                string   `json:"sport"`
	Tags                 []string `json:"tags"`
	ImageUrl             string   `json:"image_url"`
	IsActive             bool     `json:"is_active"`
	OpeningWindowCount   int32    `json:"opening_window_count"`
	UpcomingBookingCount int32    `json:"upcoming_booking_count"`
}

type FacilitiesResponse struct {
	Facilities []FacilitySummaryDto `json:"facilities"`
}

type FacilityDetailDto struct {
	Id                string          `json:"id"`
	Slug              string          `json:"slug"`
	Name              string          `json:"name"`
	Description       string          `json:"description"`
	Tags              []string        `json:"tags"`
	Address           string          `json:"address"`
	Latitude          *float64        `json:"latitude"`
	Longitude         *float64        `json:"longitude"`
	Capacity          *int32          `json:"capacity"`
	AccessibilityInfo string          `json:"accessibility_info"`
	ContactPhone      string          `json:"contact_phone"`
	ContactEmail      string          `json:"contact_email"`
	ImageRef          string          `json:"image_ref"` // sent back unchanged when the form is saved
	ImageUrl          string          `json:"image_url"`
	IsActive          bool            `json:"is_active"`
	SlotMinutes       int32           `json:"slot_minutes"`
	CustomRules       bool            `json:"custom_rules"`  // false: the council's window
	Rules             BookingRulesDto `json:"rules"`         // the window in force
	CouncilRules      BookingRulesDto `json:"council_rules"` // the council's, to show what it inherits
	MinAge            *int32          `json:"min_age"`
	UsageRules        string          `json:"usage_rules"`
	BookingCount      int64           `json:"booking_count"`
	BookingsToCancel  int64           `json:"bookings_to_cancel"` // upcoming bookings that deactivating could cancel
	Schedule          []WindowDto     `json:"schedule"`
	Closures          []ClosureDto    `json:"closures"` // current and upcoming
	Timezone          string          `json:"timezone"` // the council's: closure dates are typed in it
}

type GeocodeResult struct {
	Label     string  `json:"label"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// --- check-in readers ------------------------------------------------------

type DeviceInput struct {
	Name string `json:"name"`
}

// validate returns the i18n key of the first invalid field, or "".
func (in *DeviceInput) validate() string {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 80 {
		return "errors.invalid_device_name"
	}
	return ""
}

type DeviceDto struct {
	Id         string     `json:"id"`
	Name       string     `json:"name"`
	KeyHint    string     `json:"key_hint"` // last characters of the key
	LastSeenAt *time.Time `json:"last_seen_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type DevicesResponse struct {
	Devices []DeviceDto `json:"devices"`
}

// CreatedDeviceDto carries the key: the only time it is ever shown.
type CreatedDeviceDto struct {
	DeviceDto
	Key string `json:"key"`
}
