package checkin

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"librequenda/schedule"
	"math"
	"strings"
	"time"
)

// What the resident's phone proves at the door (printed QR or turn code): it
// is within the facility's radius. The location is checked and dropped, never stored.

// MaxAccuracyMetres: a location blurrier than this cannot place the phone at the door.
const MaxAccuracyMetres = 150

// DefaultRadiusMetres: how far from the facility's pin a phone may check in, unless the council changes it.
const DefaultRadiusMetres = 100

const earthRadiusMetres = 6371000

// DistanceMetres between two points (haversine): plenty for a few hundred metres.
func DistanceMetres(lat1, lon1, lat2, lon2 float64) float64 {
	var rad = math.Pi / 180
	var dLat = (lat2 - lat1) * rad
	var dLon = (lon2 - lon1) * rad
	var a = math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusMetres * math.Asin(math.Min(1, math.Sqrt(a)))
}

// Location results: "" when the phone is close enough.
const (
	LocationMissing    = "location_required"
	LocationInaccurate = "location_inaccurate"
	LocationTooFar     = "too_far"
)

// Position is what the browser's geolocation reports.
type Position struct {
	Latitude  float64
	Longitude float64
	Accuracy  float64 // metres, 95% confidence
}

// CheckLocation: the phone must report a position good to MaxAccuracyMetres that
// could be within radius of the facility, giving it the benefit of its accuracy.
func CheckLocation(position *Position, facilityLat, facilityLon float64, radiusMetres int) string {
	switch {
	case position == nil || !validCoordinates(position.Latitude, position.Longitude):
		return LocationMissing
	case position.Accuracy <= 0 || position.Accuracy > MaxAccuracyMetres:
		return LocationInaccurate
	case DistanceMetres(position.Latitude, position.Longitude, facilityLat, facilityLon)-position.Accuracy > float64(radiusMetres):
		return LocationTooFar
	}
	return ""
}

func validCoordinates(lat, lon float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lon) && lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180 && (lat != 0 || lon != 0)
}

// CodeDigits: how long a turn code is.
const CodeDigits = 6

// TurnCode is the code of the turn that starts at turnStart, from the facility's
// secret: HMAC-SHA256 truncated like an HOTP. Nothing is stored per turn, and a
// daily sheet tells nothing about other days.
func TurnCode(secret []byte, turnStart time.Time) string {
	var mac = hmac.New(sha256.New, secret)
	mac.Write([]byte(turnStart.UTC().Format(time.RFC3339)))
	var sum = mac.Sum(nil)
	var offset = sum[len(sum)-1] & 0x0f
	var value = binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%0*d", CodeDigits, value%uint32(math.Pow10(CodeDigits)))
}

// NormalizeCode drops the spaces and dashes people type; "" if what is left is not a code.
func NormalizeCode(input string) string {
	var code = strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' {
			return -1
		}
		return r
	}, input)
	if len(code) != CodeDigits || strings.Trim(code, "0123456789") != "" {
		return ""
	}
	return code
}

// CodeMatches: the typed code is that of one of the turns, compared in constant time.
func CodeMatches(secret []byte, turnStarts []time.Time, typed string) bool {
	var code = NormalizeCode(typed)
	if code == "" {
		return false
	}
	var found = 0
	for _, start := range turnStarts {
		found |= subtle.ConstantTimeCompare([]byte(TurnCode(secret, start)), []byte(code))
	}
	return found == 1
}

// EntryTurns: the starts of the turns someone may check in for at `now`, from
// the early window before they start until they end.
func EntryTurns(turns []schedule.Slot, now time.Time) []time.Time {
	var open []time.Time
	for _, turn := range turns {
		if !now.Before(turn.Start.Add(-EarlyWindow)) && now.Before(turn.End) {
			open = append(open, turn.Start)
		}
	}
	return open
}
