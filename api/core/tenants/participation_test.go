package tenants

import "testing"

func TestAverageDays(t *testing.T) {
	for days, want := range map[float64]float64{2.345: 2.3, 0.01: 0.1, 10: 10, 0.96: 1} {
		if got := *averageDays(days); got != want {
			t.Errorf("%v days: %v, want %v", days, got, want)
		}
	}
}

func TestBBox(t *testing.T) {
	var box, ok = parseBBox("-9.3,41.8,-6.7,43.8")
	if !ok || box.West != -9.3 || box.North != 43.8 {
		t.Fatalf("%+v %v", box, ok)
	}
	for _, bad := range []string{"", "1,2,3", "a,b,c,d", "-6,41,-9,43", "-9,44,-6,43", "-200,0,10,10", "NaN,0,1,1"} {
		if _, ok = parseBBox(bad); ok {
			t.Errorf("%q accepted", bad)
		}
	}
	var lat, lon = 42.66, -8.11
	var f = Filters{BBox: &box}
	if !f.inside(&lat, &lon) || f.inside(nil, nil) {
		t.Error("inside")
	}
	var west = -10.0
	if f.inside(&lat, &west) {
		t.Error("a pin west of the box is inside")
	}
	if !(Filters{}).inside(nil, nil) {
		t.Error("without a box everything is inside")
	}
}
