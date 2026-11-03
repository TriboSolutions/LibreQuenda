package tenants

import (
	"net/http/httptest"
	"testing"
)

func TestNearFromQuery(t *testing.T) {
	var cases = []struct {
		query string
		near  bool
		ok    bool
	}{
		{"", false, true},
		{"lat=43.36&lon=-8.41", true, true},
		{"lat=43.36", false, false},
		{"lat=abc&lon=1", false, false},
		{"lat=95&lon=1", false, false},
		{"lat=43.36&lon=-8.41&radius_km=10", true, true},
		{"lat=43.36&lon=-8.41&radius_km=0", false, false},
		{"lat=43.36&lon=-8.41&radius_km=900", false, false},
		{"lat=43.36&lon=-8.41&radius_km=x", false, false},
		{"radius_km=10", false, false},
	}
	for _, c := range cases {
		var near, ok = nearFromQuery(httptest.NewRequest("GET", "/facilities?"+c.query, nil))
		if ok != c.ok || (near != nil) != c.near {
			t.Errorf("%q: near %v ok %v, want near %v ok %v", c.query, near != nil, ok, c.near, c.ok)
		}
	}
}

// Nearest first; the ones without a pin last, in their order.
func TestByDistance(t *testing.T) {
	var a, b, c = 5.0, 0.4, 61.0
	type item struct {
		name string
		km   *float64
	}
	var items = []item{{"no pin 1", nil}, {"a", &a}, {"no pin 2", nil}, {"c", &c}, {"b", &b}}
	byDistance(items, func(i item) *float64 { return i.km })
	var got = ""
	for _, i := range items {
		got += i.name + ","
	}
	if want := "b,a,c,no pin 1,no pin 2,"; got != want {
		t.Errorf("order = %s, want %s", got, want)
	}
}

func TestDistanceKm(t *testing.T) {
	var lat, lon = 43.3623, -8.4115 // A Coruña
	var lalinLat, lalinLon = 42.66453, -8.10815
	var km = distanceKm(&Near{Latitude: lat, Longitude: lon}, &lalinLat, &lalinLon)
	if km == nil || *km < 75 || *km > 85 {
		t.Errorf("A Coruña to Lalín = %v km, want about 80", km)
	}
	if distanceKm(nil, &lalinLat, &lalinLon) != nil || distanceKm(&Near{Latitude: lat, Longitude: lon}, nil, nil) != nil {
		t.Error("no position or no pin: no distance")
	}
}

func TestWithin(t *testing.T) {
	var near, far = 4.0, 40.0
	var r = &Near{RadiusKm: 10}
	if !r.within(&near) || r.within(&far) || r.within(nil) {
		t.Error("10 km: 4 in, 40 out, no pin out")
	}
	var any *Near
	if !any.within(nil) || !(&Near{}).within(&far) {
		t.Error("no radius keeps everything")
	}
}
