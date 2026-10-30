package weather

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseMeteoGalicia(t *testing.T) {
	var body, _ = os.ReadFile("testdata/lalin.json")
	var days, err = Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(days) != 4 {
		t.Fatalf("want 4 days, got %d", len(days))
	}
	var today = days["2026-10-30"]
	if today.Sky != "partly_cloudy" || *today.TempMax != 19 || *today.TempMin != 9 || len(today.Periods) != 3 {
		t.Fatalf("today: %+v", today)
	}
	var morning = today.Periods[0]
	if morning.Period != Morning || morning.Sky != "partly_cloudy" || *morning.RainChance != 5 || morning.Wind != "calm" || *morning.Temperature != 17 {
		t.Errorf("morning: %+v", morning)
	}
	if night := today.Periods[2]; night.Sky != "partly_cloudy" || night.Wind != "light" || night.WindFrom != "ne" || *night.Temperature != 11 {
		t.Errorf("night (203, 302, min 11): %+v", night)
	}
	if len(today.Warnings) != 0 {
		t.Errorf("a fine day has no warnings: %v", today.Warnings)
	}
	// -9999: not available yet
	if days["2026-11-01"].Periods[0].Temperature != nil {
		t.Error("-9999 must read as unknown")
	}
}

func TestBadWeatherRule(t *testing.T) {
	var n = func(v int) *int { return &v }
	var cases = []struct {
		name string
		p    PeriodForecast
		want []string
	}{
		{"showers", PeriodForecast{Sky: "showers", RainChance: n(80)}, []string{WarnRain}},
		{"cloudy but 70% rain", PeriodForecast{Sky: "cloudy", RainChance: n(70)}, []string{WarnRain}},
		{"storm", PeriodForecast{Sky: "storm"}, []string{WarnStorm}},
		{"hail", PeriodForecast{Sky: "hail"}, []string{WarnSnow}},
		{"strong wind", PeriodForecast{Sky: "clear", Wind: "strong"}, []string{WarnWind}},
		{"heat", PeriodForecast{Sky: "clear", Temperature: n(36)}, []string{WarnHeat}},
		{"frost", PeriodForecast{Sky: "clear", Temperature: n(0)}, []string{WarnCold}},
		{"fine", PeriodForecast{Sky: "clear", RainChance: n(10), Wind: "light", Temperature: n(20)}, []string{}},
	}
	for _, c := range cases {
		var got = warnings(c.p)
		if len(got) != len(c.want) || (len(got) > 0 && got[0] != c.want[0]) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	if s, d := wind(325); s != "very_strong" || d != "n" {
		t.Errorf("325: %s %s", s, d)
	}
	if sky(213) != "storm" || sky(111) != "rain" || sky(9999) != "" {
		t.Error("sky codes")
	}
	if PeriodOf(time.Date(2026, 1, 1, 13, 59, 0, 0, time.UTC)) != Morning || PeriodOf(time.Date(2026, 1, 1, 18, 0, 0, 0, time.UTC)) != Afternoon || PeriodOf(time.Date(2026, 1, 1, 21, 0, 0, 0, time.UTC)) != Night {
		t.Error("periods")
	}
}

func TestCache(t *testing.T) {
	var body, _ = os.ReadFile("testdata/lalin.json")
	var calls atomic.Int32
	var fail atomic.Bool
	var server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if fail.Load() {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write(body)
	}))
	defer server.Close()
	var now = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	var c = New(server.URL)
	c.now = func() time.Time { return now }

	if _, ok := c.Forecast(context.Background(), "36024", "2026-10-30"); !ok {
		t.Fatal("first: no forecast")
	}
	c.Forecast(context.Background(), "36024", "2026-10-31")
	if calls.Load() != 1 {
		t.Fatalf("within the hour it is cached: %d calls", calls.Load())
	}
	now = now.Add(61 * time.Minute)
	fail.Store(true)
	if _, ok := c.Forecast(context.Background(), "36024", "2026-10-30"); ok {
		t.Fatal("a failing source gives no forecast")
	}
	c.Forecast(context.Background(), "36024", "2026-10-30")
	if calls.Load() != 2 {
		t.Fatalf("a failure is remembered for a while: %d calls", calls.Load())
	}
	if _, ok := c.Forecast(context.Background(), "", "2026-10-30"); ok {
		t.Fatal("no municipality, no forecast")
	}
}
