package counciladmin

// CheckinSettingsDto: the facility's ways in from the resident's phone (Entrada tab).
type CheckinSettingsDto struct {
	PrintedQr    bool   `json:"printed_qr"`
	QrToken      string `json:"qr_token"` // in the printed QR's address
	EntryCode    bool   `json:"entry_code"`
	Code         string `json:"code"` // the entry code to print
	RadiusM      int    `json:"radius_m"`
	FlashBooking bool   `json:"flash_booking"`
	StaffConfirm bool   `json:"staff_confirm"` // staff mark the entry from the bookings calendar
	HasLocation  bool   `json:"has_location"`  // both ways need the facility's pin
}

type CheckinSettingsInput struct {
	PrintedQr    bool `json:"printed_qr"`
	EntryCode    bool `json:"entry_code"`
	RadiusM      int  `json:"radius_m"`
	FlashBooking bool `json:"flash_booking"`
	StaffConfirm bool `json:"staff_confirm"`
}

const (
	minRadiusMetres = 25
	maxRadiusMetres = 1000
)

func (in CheckinSettingsInput) validate(hasLocation bool) string {
	switch {
	case in.RadiusM < minRadiusMetres || in.RadiusM > maxRadiusMetres:
		return "errors.invalid_checkin_radius"
	case (in.PrintedQr || in.EntryCode) && !hasLocation:
		return "errors.checkin_needs_location"
	}
	return ""
}
