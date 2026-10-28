// Package checkin holds what the booking QR and the entrance readers share:
// the QR payload, the QR image and the reader (device) keys.
package checkin

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"time"

	"github.com/google/uuid"
	qrcode "github.com/skip2/go-qrcode"
)

// payloadPrefix versions the QR content: "lq1:" + the booking's check-in token.
const payloadPrefix = "lq1:"

// EarlyWindow: a reader accepts a booking from this long before it starts until it ends.
const EarlyWindow = 15 * time.Minute

// Payload is the text inside a booking QR. It carries no personal data.
func Payload(token uuid.UUID) string {
	return payloadPrefix + token.String()
}

// ParsePayload reads a scanned QR; ok is false for anything that is not a booking QR.
func ParsePayload(text string) (uuid.UUID, bool) {
	var raw, found = strings.CutPrefix(strings.TrimSpace(text), payloadPrefix)
	if !found {
		return uuid.Nil, false
	}
	var token, err = uuid.Parse(raw)
	return token, err == nil
}

// personalPrefix marks the personal QR on /perfil: "lq2:" + a token that lasts about a minute.
const personalPrefix = "lq2:"

// PersonalRefresh: how often /perfil shows a new personal QR. PersonalValidFor is
// longer, so the code on screen still works while the next one loads.
const (
	PersonalRefresh  = time.Minute
	PersonalValidFor = 90 * time.Second
)

// PersonalPayload is the text inside a personal QR. Like the booking QR, it carries no personal data.
func PersonalPayload(token uuid.UUID) string {
	return personalPrefix + token.String()
}

// ParsePersonalPayload reads a scanned QR; ok is false for anything that is not a personal QR.
func ParsePersonalPayload(text string) (uuid.UUID, bool) {
	var raw, found = strings.CutPrefix(strings.TrimSpace(text), personalPrefix)
	if !found {
		return uuid.Nil, false
	}
	var token, err = uuid.Parse(raw)
	return token, err == nil
}

// PNG renders the booking QR (for emails; the web draws its own).
func PNG(token uuid.UUID, size int) ([]byte, error) {
	return qrcode.Encode(Payload(token), qrcode.Medium, size)
}

// keyPrefix tells a reader key apart from a Zitadel access token in the Authorization header.
const keyPrefix = "lqd_"

// NewDeviceKey returns a reader key (shown once), the hash that is stored and a short hint.
func NewDeviceKey() (key string, hash []byte, hint string, err error) {
	var secret = make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		return "", nil, "", err
	}
	key = keyPrefix + base64.RawURLEncoding.EncodeToString(secret)
	return key, HashDeviceKey(key), key[len(key)-4:], nil
}

// HashDeviceKey: keys are random, so a plain SHA-256 is enough to store them.
func HashDeviceKey(key string) []byte {
	var sum = sha256.Sum256([]byte(key))
	return sum[:]
}

// IsDeviceKey: the shape of a reader key, checked before touching the database.
func IsDeviceKey(key string) bool {
	return strings.HasPrefix(key, keyPrefix) && len(key) == len(keyPrefix)+43
}
