package checkin

import (
	"math"
	"testing"
)

func TestDistanceMetres(t *testing.T) {
	// two points in Lalín, about 500 m apart
	var d = DistanceMetres(42.66117, -8.11204, 42.66453, -8.10815)
	if d < 450 || d > 550 {
		t.Fatalf("distance = %.0f m, want about 500 m", d)
	}
	if d := DistanceMetres(42.66, -8.11, 42.66, -8.11); d != 0 {
		t.Fatalf("same point: %f", d)
	}
	// one thousandth of a degree of latitude is about 111 m everywhere
	if d := DistanceMetres(42, -8, 42.001, -8); math.Abs(d-111.2) > 1 {
		t.Fatalf("0.001° of latitude = %.1f m", d)
	}
}

func TestCheckLocation(t *testing.T) {
	const lat, lon = 42.66453, -8.10815
	var at = func(dLat, accuracy float64) *Position {
		return &Position{Latitude: lat + dLat, Longitude: lon, Accuracy: accuracy}
	}
	var cases = []struct {
		name     string
		position *Position
		want     string
	}{
		{"at the door", at(0, 10), ""},
		{"no location", nil, LocationMissing},
		{"zero coordinates", &Position{Accuracy: 10}, LocationMissing},
		{"out of range", &Position{Latitude: 91, Longitude: 0, Accuracy: 10}, LocationMissing},
		{"no accuracy", at(0, 0), LocationInaccurate},
		{"too blurry", at(0, MaxAccuracyMetres+1), LocationInaccurate},
		{"inside the radius", at(0.0008, 5), ""},                 // ~89 m
		{"outside, but within its accuracy", at(0.0011, 30), ""}, // ~122 m ± 30
		{"too far", at(0.0011, 5), LocationTooFar},
		{"in town", at(0.01, 20), LocationTooFar}, // ~1.1 km
	}
	for _, c := range cases {
		if got := CheckLocation(c.position, lat, lon, 100); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestEntryCode(t *testing.T) {
	var secret = []byte("a facility secret of 32 bytes...")
	var code = EntryCode(secret)
	if len(code) != CodeDigits || NormalizeCode(code) != code {
		t.Fatalf("code %q is not %d digits", code, CodeDigits)
	}
	if EntryCode(secret) != code {
		t.Fatal("the code is fixed while the secret stays")
	}
	if EntryCode([]byte("another facility's secret.......")) == code {
		t.Fatal("another secret gives another code")
	}
}

func TestNormalizeCode(t *testing.T) {
	var cases = map[string]string{
		"123456":   "123456",
		"123 456":  "123456",
		" 123-456": "123456",
		"12345":    "",
		"1234567":  "",
		"12a456":   "",
		"":         "",
	}
	for in, want := range cases {
		if got := NormalizeCode(in); got != want {
			t.Errorf("NormalizeCode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCodeMatches(t *testing.T) {
	var secret = []byte("a facility secret of 32 bytes...")
	var code = EntryCode(secret)
	if !CodeMatches(secret, code[:3]+" "+code[3:]) {
		t.Fatal("the code, typed with a space, must match")
	}
	if CodeMatches([]byte("a new secret, after a new code..."), code) {
		t.Fatal("the old code must not match after a new one")
	}
	if CodeMatches(secret, "") {
		t.Fatal("an empty code must not match")
	}
}
