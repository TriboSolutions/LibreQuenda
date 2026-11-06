package tenants

import (
	"math"
	"net/http"
	"sort"
	"strconv"

	"librequenda/core/checkin"
)

// Near: where the person is, from ?lat=&lon=, to sort facilities by distance,
// and ?radius_km= to keep only those within it. Never stored, and the API does
// not log query strings.
type Near struct {
	Latitude  float64
	Longitude float64
	RadiusKm  float64 // 0 = any distance
}

// MaxRadiusKm: the widest «within» the search offers (Galicia fits in it).
const MaxRadiusKm = 500

// nearFromQuery reads ?lat=&lon=: nil when absent; ok false when given but not valid.
func nearFromQuery(r *http.Request) (*Near, bool) {
	var q = r.URL.Query()
	var latText, lonText = q.Get("lat"), q.Get("lon")
	if latText == "" && lonText == "" {
		return nil, q.Get("radius_km") == "" // a radius needs a position
	}
	var lat, latErr = strconv.ParseFloat(latText, 64)
	var lon, lonErr = strconv.ParseFloat(lonText, 64)
	if latErr != nil || lonErr != nil || math.IsNaN(lat) || math.IsNaN(lon) || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return nil, false
	}
	var near = &Near{Latitude: lat, Longitude: lon}
	if radiusText := q.Get("radius_km"); radiusText != "" {
		var radius, radiusErr = strconv.ParseFloat(radiusText, 64)
		if radiusErr != nil || math.IsNaN(radius) || radius <= 0 || radius > MaxRadiusKm {
			return nil, false
		}
		near.RadiusKm = radius
	}
	return near, true
}

// within: the facility is inside the radius asked for (any, without one; never without a pin).
func (n *Near) within(km *float64) bool {
	return n == nil || n.RadiusKm == 0 || (km != nil && *km <= n.RadiusKm)
}

// distanceKm from near to a facility's pin, rounded to 0.1 km; nil without either.
func distanceKm(near *Near, lat, lon *float64) *float64 {
	if near == nil || lat == nil || lon == nil {
		return nil
	}
	var km = math.Round(checkin.DistanceMetres(near.Latitude, near.Longitude, *lat, *lon)/100) / 10
	return &km
}

// byDistance sorts nearest first, the ones without a pin last; the order is kept
// among equals, and as it was when nobody shared where they are.
func byDistance[T any](items []T, distance func(T) *float64) {
	sort.SliceStable(items, func(i, j int) bool {
		var a, b = distance(items[i]), distance(items[j])
		switch {
		case a == nil:
			return false
		case b == nil:
			return true
		default:
			return *a < *b
		}
	})
}
