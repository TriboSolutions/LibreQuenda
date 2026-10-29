package checkin

import (
	"librequenda/schedule"
	"math"
	"testing"
	"time"
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

func TestTurnCode(t *testing.T) {
	var secret = []byte("a facility secret of 32 bytes...")
	var start = time.Date(2026, 11, 3, 18, 0, 0, 0, time.UTC)
	var code = TurnCode(secret, start)
	if len(code) != CodeDigits || NormalizeCode(code) != code {
		t.Fatalf("code %q is not %d digits", code, CodeDigits)
	}
	if TurnCode(secret, start) != code {
		t.Fatal("the same turn must give the same code")
	}
	// the same instant in the council's zone is the same turn
	var madrid, _ = time.LoadLocation("Europe/Madrid")
	if TurnCode(secret, start.In(madrid)) != code {
		t.Fatal("the code must not depend on the zone the time is in")
	}
	if TurnCode(secret, start.Add(time.Hour)) == code && TurnCode(secret, start.Add(2*time.Hour)) == code {
		t.Fatal("every turn gets its own code")
	}
	if TurnCode([]byte("another facility's secret......."), start) == code {
		t.Fatal("another facility gets another code")
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
	var six = time.Date(2026, 11, 3, 18, 0, 0, 0, time.UTC)
	var seven = six.Add(time.Hour)
	var code = TurnCode(secret, seven)
	if !CodeMatches(secret, []time.Time{six, seven}, code[:3]+" "+code[3:]) {
		t.Fatal("the code of an open turn, typed with a space, must match")
	}
	if CodeMatches(secret, []time.Time{six}, code) {
		t.Fatal("the code of a turn that is not open must not match")
	}
	if CodeMatches(secret, nil, code) {
		t.Fatal("with no open turn, nothing matches")
	}
	if CodeMatches(secret, []time.Time{seven}, "") {
		t.Fatal("an empty code must not match")
	}
}

func TestEntryTurns(t *testing.T) {
	var six = time.Date(2026, 11, 3, 18, 0, 0, 0, time.UTC)
	var turns = []schedule.Slot{
		{Start: six, End: six.Add(time.Hour)},
		{Start: six.Add(time.Hour), End: six.Add(2 * time.Hour)},
	}
	var cases = []struct {
		name string
		now  time.Time
		want int
	}{
		{"before the early window", six.Add(-EarlyWindow - time.Minute), 0},
		{"early window of the first", six.Add(-EarlyWindow), 1},
		{"during the first", six.Add(30 * time.Minute), 1},
		{"end of the first, early for the second", six.Add(50 * time.Minute), 2},
		{"after both", six.Add(2 * time.Hour), 0},
	}
	for _, c := range cases {
		if got := EntryTurns(turns, c.now); len(got) != c.want {
			t.Errorf("%s: %d open turns, want %d", c.name, len(got), c.want)
		}
	}
}
