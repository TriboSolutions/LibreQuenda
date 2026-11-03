package me

import "time"

// PersonalQrDto: the personal QR on /perfil. The web asks for a new one every
// RefreshSeconds; each code works once, until ExpiresAt.
type PersonalQrDto struct {
	Code           string    `json:"code"` // QR content: "lq2:" + a random token, no personal data
	ExpiresAt      time.Time `json:"expires_at"`
	RefreshSeconds int       `json:"refresh_seconds"`
}

// PersonalQrStatusDto: whether a reader used the QR (a refused try gives it back unused), and
// the booking it made then. Unknown or long gone, it is simply not used.
type PersonalQrStatusDto struct {
	Used      bool   `json:"used"`
	BookingId string `json:"booking_id,omitempty"`
}
