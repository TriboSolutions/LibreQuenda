package entrance

import (
	"librequenda/checkin"
	"librequenda/core/reader"
	"time"
)

// EntranceDto: what the facility's /entrada page shows before checking in.
type EntranceDto struct {
	FacilityName string `json:"facility_name"`
	FacilitySlug string `json:"facility_slug"`
	TenantName   string `json:"tenant_name"`
	TenantSlug   string `json:"tenant_slug"`
	Timezone     string `json:"timezone"`
	PrintedQr    bool   `json:"printed_qr"` // the QR on the door works
	EntryCode    bool   `json:"entry_code"` // the facility's entry code works
	RadiusM      int    `json:"radius_m"`
	FlashBooking bool   `json:"flash_booking"` // a free turn can be booked on the spot
	// the person's own booking here that can go in now, if any
	Booking *EntranceBookingDto `json:"booking"`
}

type EntranceBookingDto struct {
	StartsAt    time.Time  `json:"starts_at"`
	EndsAt      time.Time  `json:"ends_at"`
	CheckedInAt *time.Time `json:"checked_in_at"`
}

// CheckinInput: the proof of being at the door (the printed QR's token or the
// entry code) and where the phone is. The location is checked, never stored.
type CheckinInput struct {
	QrToken   string   `json:"qr_token"`
	Code      string   `json:"code"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	Accuracy  *float64 `json:"accuracy"` // metres
	Book      bool     `json:"book"`     // without a booking: book a free turn and check in
}

// Results of the phone at the door, besides the reader's (checked_in, booked, not_booked…).
const (
	ResultMethodOff          = "method_off"           // the facility does not use that way in
	ResultInvalidQr          = "invalid_qr"           // an old print, or not this facility's QR
	ResultWrongCode          = "wrong_code"           // not the code of a turn open for entry now
	ResultNoFacilityLocation = "no_facility_location" // the facility has no pin to compare with
	ResultNoBooking          = "no_booking"           // nothing to let in; see CanBook
)

type CheckinResultDto struct {
	reader.CheckinResultDto
	CanBook bool `json:"can_book"` // no_booking only: a free turn can be booked on the spot
}

// NearbyInput: where the phone is, to find the facilities it could book at the
// door with their entry code. Checked, never stored.
type NearbyInput struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	Accuracy  *float64 `json:"accuracy"` // metres
}

// NearbyDto: the facilities the phone is at (by entry code or with staff to ask), nearest first; Result is a location
// problem (location_required, location_inaccurate) when it cannot tell.
type NearbyDto struct {
	Result     string              `json:"result"`
	Facilities []NearbyFacilityDto `json:"facilities"`
}

type NearbyFacilityDto struct {
	FacilityName string `json:"facility_name"`
	FacilitySlug string `json:"facility_slug"`
	TenantName   string `json:"tenant_name"`
	TenantSlug   string `json:"tenant_slug"`
	EntryCode    bool   `json:"entry_code"` // books with its entry code
	PrintedQr    bool   `json:"printed_qr"` // has a QR on the door
	RadiusM      int    `json:"radius_m"`   // how close the phone must be
	// staff to ask on site, when the facility has staff confirmation
	StaffContacts []checkin.StaffContact `json:"staff_contacts"`
}
