// Package weather reads MeteoGalicia's open forecast for a municipality: up to
// four days (today first), each in three periods (morning, afternoon, night),
// with the sky, the chance of rain, the wind and temperatures. Only the API
// calls MeteoGalicia, so residents' addresses never reach a third party; the
// answers are cached for an hour per municipality and any failure means "no
// forecast", never an error for the page.
//
// Source: https://servizos.meteogalicia.gal/mgrss/predicion/jsonPredConcellos.action?idConc=<INE code>
// (keyless; codes in MeteoGalicia's "JSON_Pred_Concello" documentation). The
// data is MeteoGalicia's and is shown with its attribution.
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// BaseURL can point elsewhere (WEATHER_URL), e.g. for tests or a mirror.
var BaseURL = func() string {
	if url := os.Getenv("WEATHER_URL"); url != "" {
		return url
	}
	return "https://servizos.meteogalicia.gal/mgrss/predicion/jsonPredConcellos.action"
}()

const (
	cacheFor     = time.Hour
	failedRetry  = 5 * time.Minute // after a failure, ask again only this much later
	notAvailable = -9999
)

// Period of the day as MeteoGalicia splits it.
type Period string

const (
	Morning   Period = "morning"
	Afternoon Period = "afternoon"
	Night     Period = "night"
)

// PeriodOf: the period a local time falls in (morning until 14:00, afternoon until 21:00).
func PeriodOf(t time.Time) Period {
	switch h := t.Hour(); {
	case h < 14:
		return Morning
	case h < 21:
		return Afternoon
	}
	return Night
}

// Warnings for a period or a day: what makes an outdoor turn a bad idea.
const (
	WarnRain  = "rain"
	WarnStorm = "storm"
	WarnSnow  = "snow" // snow, sleet or hail
	WarnWind  = "wind"
	WarnHeat  = "heat"
	WarnCold  = "cold"
	WarnAlert = "alert" // MeteoGalicia's own adverse-weather warning for the municipality
)

// Thresholds of the warnings.
const (
	rainChanceWarn = 60 // % chance of rain
	heatWarn       = 33 // ºC
	coldWarn       = 2  // ºC
)

type PeriodForecast struct {
	Period      Period   `json:"period"`
	Sky         string   `json:"sky"`         // clear, partly_cloudy, cloudy, overcast, fog, haze, drizzle, rain, showers, storm, snow, sleet, hail; "" unknown
	RainChance  *int     `json:"rain_chance"` // %, null when unknown
	Wind        string   `json:"wind"`        // calm, variable, light, moderate, strong, very_strong; "" unknown
	WindFrom    string   `json:"wind_from"`   // n, ne, e, se, s, sw, w, nw; "" unknown or none
	Temperature *int     `json:"temperature"` // ºC, the period's maximum (or minimum at night); null when unknown
	Warnings    []string `json:"warnings"`
}

type DayForecast struct {
	Date     string           `json:"date"` // YYYY-MM-DD
	Sky      string           `json:"sky"`
	TempMin  *int             `json:"temp_min"`
	TempMax  *int             `json:"temp_max"`
	Periods  []PeriodForecast `json:"periods"`
	Warnings []string         `json:"warnings"` // every period's, plus MeteoGalicia's alert
}

// --- MeteoGalicia's JSON ---------------------------------------------------

type franxas struct {
	Manha int `json:"manha"`
	Tarde int `json:"tarde"`
	Noite int `json:"noite"`
}

type mgDay struct {
	Data       string   `json:"dataPredicion"` // "2026-10-30T00:00:00"
	CeoDia     int      `json:"ceoDia"`
	Ceo        franxas  `json:"ceo"`
	PChoiva    franxas  `json:"pchoiva"`
	Vento      franxas  `json:"vento"`
	TMax       int      `json:"tMax"`
	TMin       int      `json:"tMin"`
	TMaxFranxa franxas  `json:"tmaxFranxa"`
	TMinFranxa franxas  `json:"tminFranxa"`
	NivelAviso *float64 `json:"nivelAviso"` // null without a warning
}

type mgResponse struct {
	PredConcello struct {
		Dias []mgDay `json:"listaPredDiaConcello"`
	} `json:"predConcello"`
}

// Parse turns MeteoGalicia's answer into forecasts by date.
func Parse(body []byte) (map[string]DayForecast, error) {
	var raw mgResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	var days = make(map[string]DayForecast, len(raw.PredConcello.Dias))
	for _, d := range raw.PredConcello.Dias {
		if len(d.Data) < 10 {
			continue
		}
		var day = DayForecast{Date: d.Data[:10], Sky: sky(d.CeoDia), TempMin: known(d.TMin), TempMax: known(d.TMax), Warnings: []string{}}
		for _, p := range []struct {
			period          Period
			ceo, rain, wind int
			temp            int
		}{
			{Morning, d.Ceo.Manha, d.PChoiva.Manha, d.Vento.Manha, d.TMaxFranxa.Manha},
			{Afternoon, d.Ceo.Tarde, d.PChoiva.Tarde, d.Vento.Tarde, d.TMaxFranxa.Tarde},
			{Night, d.Ceo.Noite, d.PChoiva.Noite, d.Vento.Noite, d.TMinFranxa.Noite},
		} {
			var strength, from = wind(p.wind)
			var period = PeriodForecast{Period: p.period, Sky: sky(p.ceo), RainChance: known(p.rain), Wind: strength, WindFrom: from, Temperature: known(p.temp)}
			period.Warnings = warnings(period)
			day.Periods = append(day.Periods, period)
			day.Warnings = merge(day.Warnings, period.Warnings)
		}
		// the day's extremes count too, when the periods do not say
		if day.TempMax != nil && *day.TempMax >= heatWarn {
			day.Warnings = merge(day.Warnings, []string{WarnHeat})
		}
		if day.TempMin != nil && *day.TempMin <= coldWarn {
			day.Warnings = merge(day.Warnings, []string{WarnCold})
		}
		if d.NivelAviso != nil && *d.NivelAviso > 0 {
			day.Warnings = merge(day.Warnings, []string{WarnAlert})
		}
		days[day.Date] = day
	}
	return days, nil
}

func known(v int) *int {
	if v == notAvailable {
		return nil
	}
	return &v
}

// sky: MeteoGalicia's sky codes (1xx by day, 2xx by night) as a few kinds.
func sky(code int) string {
	if code == notAvailable || code < 100 {
		return ""
	}
	switch code % 100 {
	case 1:
		return "clear"
	case 2, 3, 16:
		return "partly_cloudy"
	case 4:
		return "cloudy"
	case 5, 25:
		return "overcast"
	case 6, 14, 15:
		return "fog"
	case 24:
		return "haze"
	case 10:
		return "drizzle"
	case 11, 17:
		return "rain"
	case 7, 8, 18, 22:
		return "showers"
	case 13, 19:
		return "storm"
	case 9, 12, 23:
		return "snow"
	case 20:
		return "sleet"
	case 21:
		return "hail"
	}
	return ""
}

var directions = []string{"n", "ne", "e", "se", "s", "sw", "w", "nw"}

// wind: MeteoGalicia's wind codes (299 calm, 300 variable, then 8 directions
// per strength: light 301-308, moderate 309-316, strong 317-324, very strong 325-332).
func wind(code int) (string, string) {
	switch {
	case code == 299:
		return "calm", ""
	case code == 300:
		return "variable", ""
	case code >= 301 && code <= 332:
		var i = code - 301
		return []string{"light", "moderate", "strong", "very_strong"}[i/8], directions[i%8]
	}
	return "", ""
}

// warnings: the bad-weather rule for an outdoor turn.
func warnings(p PeriodForecast) []string {
	var out = []string{}
	switch p.Sky {
	case "storm":
		out = append(out, WarnStorm)
	case "snow", "sleet", "hail":
		out = append(out, WarnSnow)
	case "rain", "showers":
		out = append(out, WarnRain)
	}
	if p.RainChance != nil && *p.RainChance >= rainChanceWarn && p.Sky != "storm" && p.Sky != "rain" && p.Sky != "showers" {
		out = append(out, WarnRain)
	}
	if p.Wind == "strong" || p.Wind == "very_strong" {
		out = append(out, WarnWind)
	}
	if p.Temperature != nil && *p.Temperature >= heatWarn {
		out = append(out, WarnHeat)
	}
	if p.Temperature != nil && *p.Temperature <= coldWarn {
		out = append(out, WarnCold)
	}
	return out
}

func merge(a, b []string) []string {
	for _, w := range b {
		var seen = false
		for _, x := range a {
			seen = seen || x == w
		}
		if !seen {
			a = append(a, w)
		}
	}
	return a
}

// --- fetching, with a cache per municipality -------------------------------

type entry struct {
	days    map[string]DayForecast
	expires time.Time
}

// Client fetches and caches forecasts; the zero value is not usable, use New.
type Client struct {
	http  *http.Client
	base  string
	mu    sync.Mutex
	cache map[string]entry
	now   func() time.Time
}

func New(base string) *Client {
	return &Client{http: &http.Client{Timeout: 5 * time.Second}, base: base, cache: map[string]entry{}, now: time.Now}
}

var defaultClient = New(BaseURL)

// Forecast for a municipality (INE code) and a local date; false when there is none.
func Forecast(ctx context.Context, municipality, date string) (DayForecast, bool) {
	return defaultClient.Forecast(ctx, municipality, date)
}

func (c *Client) Forecast(ctx context.Context, municipality, date string) (DayForecast, bool) {
	if municipality == "" {
		return DayForecast{}, false
	}
	var days = c.days(ctx, municipality)
	var day, ok = days[date]
	return day, ok
}

func (c *Client) days(ctx context.Context, municipality string) map[string]DayForecast {
	c.mu.Lock()
	if e, ok := c.cache[municipality]; ok && c.now().Before(e.expires) {
		c.mu.Unlock()
		return e.days
	}
	c.mu.Unlock()

	var days, err = c.fetch(ctx, municipality)
	var e = entry{days: days, expires: c.now().Add(cacheFor)}
	if err != nil {
		// a failure is remembered briefly, so a slow MeteoGalicia does not slow every page
		e = entry{days: map[string]DayForecast{}, expires: c.now().Add(failedRetry)}
	}
	c.mu.Lock()
	c.cache[municipality] = e
	c.mu.Unlock()
	return e.days
}

func (c *Client) fetch(ctx context.Context, municipality string) (map[string]DayForecast, error) {
	var req, err = http.NewRequestWithContext(ctx, http.MethodGet, c.base+"?idConc="+municipality, nil)
	if err != nil {
		return nil, err
	}
	var res *http.Response
	if res, err = c.http.Do(req); err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("meteogalicia: %d", res.StatusCode)
	}
	var body []byte
	if body, err = readAll(res); err != nil {
		return nil, err
	}
	return Parse(body)
}

// readAll reads at most 1 MB: a forecast is a couple of kilobytes.
func readAll(res *http.Response) ([]byte, error) {
	return io.ReadAll(io.LimitReader(res.Body, 1<<20))
}
