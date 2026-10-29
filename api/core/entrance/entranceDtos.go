package entrance

import (
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
	TurnCodes    bool   `json:"turn_codes"` // a code from the daily sheet works
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

// CheckinInput: the proof of being at the door (the printed QR's token or a
// turn code) and where the phone is. The location is checked, never stored.
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
