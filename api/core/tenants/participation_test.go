package tenants

import "testing"

func TestAverageDays(t *testing.T) {
	for days, want := range map[float64]float64{2.345: 2.3, 0.01: 0.1, 10: 10, 0.96: 1} {
		if got := *averageDays(days); got != want {
			t.Errorf("%v days: %v, want %v", days, got, want)
		}
	}
}
