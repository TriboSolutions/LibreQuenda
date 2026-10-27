package counciladmin

import (
	"context"
	"encoding/json"
	"fmt"
	"librequenda/shared"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Address search through OpenStreetMap's Nominatim, called by the API so the
// visitor's IP never reaches a third party. Nominatim's public usage policy asks
// for an identifying User-Agent and at most one request per second; a council
// can point GEOCODER_URL at its own Nominatim instead.
var geocodeClient = &http.Client{Timeout: 8 * time.Second}

var geocodeMu sync.Mutex
var lastGeocode time.Time

type nominatimPlace struct {
	DisplayName string `json:"display_name"`
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
}

func geocode(ctx context.Context, query string, language string) ([]GeocodeResult, error) {
	query = strings.TrimSpace(query)
	if len(query) < 3 || len(query) > 300 {
		return []GeocodeResult{}, nil
	}

	// one request per second, for the whole API
	geocodeMu.Lock()
	if wait := time.Second - time.Since(lastGeocode); wait > 0 {
		time.Sleep(wait)
	}
	lastGeocode = time.Now()
	geocodeMu.Unlock()

	var params = url.Values{
		"q":            {query},
		"format":       {"jsonv2"},
		"limit":        {"5"},
		"countrycodes": {"es"},
	}
	var req, err = http.NewRequestWithContext(ctx, http.MethodGet, shared.GEOCODER_URL+"/search?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "LibreQuenda/"+shared.VERSION+" (municipal sports booking; self-hosted)")
	if language != "" {
		req.Header.Set("Accept-Language", language)
	}

	var res *http.Response
	res, err = geocodeClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geocoder returned %d", res.StatusCode)
	}

	var places []nominatimPlace
	if err = json.NewDecoder(res.Body).Decode(&places); err != nil {
		return nil, err
	}
	var results = make([]GeocodeResult, 0, len(places))
	for _, p := range places {
		var lat, errLat = strconv.ParseFloat(p.Lat, 64)
		var lon, errLon = strconv.ParseFloat(p.Lon, 64)
		if errLat == nil && errLon == nil {
			results = append(results, GeocodeResult{Label: p.DisplayName, Latitude: lat, Longitude: lon})
		}
	}
	return results, nil
}
