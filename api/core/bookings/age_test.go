package bookings

import (
	"testing"
	"time"
)

func TestAgeOn(t *testing.T) {
	var date = func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }
	var cases = []struct {
		name  string
		birth time.Time
		day   time.Time
		want  int
	}{
		{"day before 16th birthday", date(2010, 11, 5), date(2026, 11, 4), 15},
		{"16th birthday", date(2010, 11, 5), date(2026, 11, 5), 16},
		{"after birthday", date(2010, 3, 1), date(2026, 10, 27), 16},
		{"earlier month", date(2010, 12, 31), date(2026, 10, 27), 15},
		{"29 Feb, non-leap year, 28 Feb", date(2008, 2, 29), date(2026, 2, 28), 17},
		{"29 Feb, non-leap year, 1 Mar", date(2008, 2, 29), date(2026, 3, 1), 18},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := AgeOn(c.birth, c.day); got != c.want {
				t.Errorf("AgeOn() = %d, want %d", got, c.want)
			}
		})
	}
}
