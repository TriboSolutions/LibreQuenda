package reader

import (
	"librequenda/model"
	"strings"
	"time"
)

type CheckinInput struct {
	Code string `json:"code"` // the scanned QR text
}

// ReaderDto: what the reader shows on its screen while waiting for a QR.
type ReaderDto struct {
	Name         string    `json:"name"`
	FacilityName string    `json:"facility_name"`
	FacilitySlug string    `json:"facility_slug"`
	TenantName   string    `json:"tenant_name"`
	TenantSlug   string    `json:"tenant_slug"`
	Timezone     string    `json:"timezone"`
	Now          time.Time `json:"now"`
}

func MapReader(device model.GetCheckinDeviceByKeyRow) ReaderDto {
	return ReaderDto{
		Name:         device.Name,
		FacilityName: device.FacilityName,
		FacilitySlug: device.FacilitySlug,
		TenantName:   device.TenantName,
		TenantSlug:   device.TenantSlug,
		Timezone:     device.Timezone,
		Now:          time.Now(),
	}
}

// Results of a scan. Granted: checked_in, already_checked_in, booked. Every other one is refused.
const (
	ResultCheckedIn        = "checked_in"
	ResultAlreadyCheckedIn = "already_checked_in"
	ResultBooked           = "booked"          // personal QR: the turn about to start was booked, and the entry recorded
	ResultNotBooked        = "not_booked"      // personal QR: no booking to let in, and none could be made (see Reason)
	ResultCodeExpired      = "code_expired"    // personal QR: expired, already used, or not issued by LibreQuenda
	ResultNotABooking      = "not_a_booking"   // the QR is not a LibreQuenda booking
	ResultUnknownBooking   = "unknown_booking" // well formed, but no booking has that token
	ResultOtherFacility    = "other_facility"
	ResultCancelled        = "cancelled"
	ResultTooEarly         = "too_early"
	ResultEnded            = "ended"
	ResultReaderOff        = "reader_off" // the council turned this facility's readers off
)

// ReasonFlashOff: not_booked because the council does not let a free turn be booked at the entrance.
const ReasonFlashOff = "flash_off"

type CheckinResultDto struct {
	Granted bool               `json:"granted"`
	Result  string             `json:"result"`
	Reason  string             `json:"reason,omitempty"` // not_booked only: why, e.g. "no_turn_soon", "slot_booked", "use_booking_qr", "flash_off"
	Booking *CheckinBookingDto `json:"booking"`          // nil for an unknown or cancelled booking
}

type CheckinBookingDto struct {
	FacilityName string     `json:"facility_name"`
	FirstName    string     `json:"first_name"` // only when granted: enough to greet, nothing more
	StartsAt     time.Time  `json:"starts_at"`
	EndsAt       time.Time  `json:"ends_at"`
	CheckedInAt  *time.Time `json:"checked_in_at"`
}

func firstName(fullName string) string {
	var words = strings.Fields(fullName)
	if len(words) == 0 {
		return ""
	}
	return words[0]
}
