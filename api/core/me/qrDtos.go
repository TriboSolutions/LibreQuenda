package me

import "time"

// PersonalQrDto: the personal QR on /perfil. The web asks for a new one every
// RefreshSeconds; each code works once, until ExpiresAt.
type PersonalQrDto struct {
	Code           string    `json:"code"` // QR content: "lq2:" + a random token, no personal data
	ExpiresAt      time.Time `json:"expires_at"`
	RefreshSeconds int       `json:"refresh_seconds"`
}
