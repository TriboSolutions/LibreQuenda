package counciladmin

import "librequenda/checkin"

// CheckinSettingsDto: the facility's ways in from the resident's phone (Entrada tab).
type CheckinSettingsDto struct {
	PrintedQr    bool   `json:"printed_qr"`
	QrToken      string `json:"qr_token"`  // in the printed QR's address
	Nfc          bool   `json:"nfc"`       // an NFC tag on the door
	NfcToken     string `json:"nfc_token"` // in the address written on the tag
	EntryCode    bool   `json:"entry_code"`
	Code         string `json:"code"` // the entry code to print
	RadiusM      int    `json:"radius_m"`
	FlashBooking bool   `json:"flash_booking"`
	StaffConfirm bool   `json:"staff_confirm"` // staff mark the entry from the bookings calendar
	QrReader     bool   `json:"qr_reader"`     // off: the readers refuse every QR, but stay set up
	// who residents ask on site (staff confirmation); not accounts
	StaffContacts []checkin.StaffContact `json:"staff_contacts"`
	HasLocation   bool                   `json:"has_location"` // both ways need the facility's pin
}

type CheckinSettingsInput struct {
	PrintedQr     bool                   `json:"printed_qr"`
	Nfc           bool                   `json:"nfc"`
	EntryCode     bool                   `json:"entry_code"`
	RadiusM       int                    `json:"radius_m"`
	FlashBooking  bool                   `json:"flash_booking"`
	StaffConfirm  bool                   `json:"staff_confirm"`
	QrReader      bool                   `json:"qr_reader"`
	StaffContacts []checkin.StaffContact `json:"staff_contacts"`
}

const (
	minRadiusMetres = 25
	maxRadiusMetres = 1000
)

func (in CheckinSettingsInput) validate(hasLocation bool) string {
	switch {
	case in.RadiusM < minRadiusMetres || in.RadiusM > maxRadiusMetres:
		return "errors.invalid_checkin_radius"
	case (in.PrintedQr || in.EntryCode || in.Nfc) && !hasLocation:
		return "errors.checkin_needs_location"
	}
	return ""
}
