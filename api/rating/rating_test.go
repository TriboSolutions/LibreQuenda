package rating

import "testing"

func TestOf(t *testing.T) {
	if got := Of(0, 0); got != nil {
		t.Errorf("Of(0, 0) = %+v, want nil", got)
	}
	for _, c := range []struct {
		average float64
		count   int64
		want    float64
	}{
		{5, 1, 5},
		{4.25, 4, 4.3},
		{4.24, 50, 4.2},
		{1, 3, 1},
		{11.0 / 3, 3, 3.7},
	} {
		var got = Of(c.average, c.count)
		if got == nil || got.Average != c.want || got.Count != c.count {
			t.Errorf("Of(%v, %d) = %+v, want %v", c.average, c.count, got, c.want)
		}
	}
}
