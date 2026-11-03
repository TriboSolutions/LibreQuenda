package tenants

import (
	"net/http/httptest"
	"testing"
	"time"

	"librequenda/schedule"
)

func TestFiltersFromQuery(t *testing.T) {
	var cases = []struct {
		query   string
		withDay bool
		ok      bool
	}{
		{"", false, true},
		{"q=p%C3%A1del+lal%C3%ADn&council=lalin&sport=padel&tags=cuberta,vestiarios", false, true},
		{"council=Lal%C3%ADn", false, false},
		{"tags=ok,../x", false, false},
		{"from=16:00&until=20:00&public=true", true, true},
		{"from=16:00", false, false}, // the day's filters only on /facilities/day
		{"from=25:00", true, false},
		{"from=20:00&until=16:00", true, false},
		{"public=yes", true, false},
	}
	for _, c := range cases {
		if _, ok := filtersFromQuery(httptest.NewRequest("GET", "/f?"+c.query, nil), c.withDay); ok != c.ok {
			t.Errorf("%q: ok %v, want %v", c.query, ok, c.ok)
		}
	}
}

func TestMatches(t *testing.T) {
	var f, _ = filtersFromQuery(httptest.NewRequest("GET", "/f?q=PADEL+cuberta&sport=padel", nil), false)
	var names = map[string]string{"padel": "Pádel", "cuberta": "Cuberta"}
	if !f.matches("Pista de pádel 2", "lalin", "Concello de Lalín", "Pádel", []string{"padel", "cuberta"}, names) {
		t.Error("words in the name and a tag's name, accents aside, should match")
	}
	if f.matches("Pista de pádel 1", "lalin", "Concello de Lalín", "Pádel", []string{"padel"}, names) {
		t.Error("«cuberta» is not there")
	}
}

func TestNarrowDay(t *testing.T) {
	var loc, _ = time.LoadLocation("Europe/Madrid")
	var now = time.Date(2026, 11, 3, 12, 0, 0, 0, loc)
	var at = func(h int) time.Time { return time.Date(2026, 11, 3, h, 0, 0, 0, loc) }
	var day = FacilityDayDto{
		Slots: []schedule.Slot{{Start: at(10), Status: schedule.SlotFree}, {Start: at(17), Status: schedule.SlotFree}, {Start: at(18), Status: schedule.SlotBooked}},
		PublicBookings: []PublicTurnDto{
			{StartsAt: at(18), Capacity: 4, People: []string{"A", "B"}},
			{StartsAt: at(19), Capacity: 2, People: []string{"A", "B"}}, // full
		},
	}
	var afternoon, _ = filtersFromQuery(httptest.NewRequest("GET", "/f?from=14:00&until=20:00", nil), true)
	if got, ok := afternoon.narrowDay(day, loc, now); !ok || len(got.Slots) != 1 || !got.Slots[0].Start.Equal(at(17)) {
		t.Errorf("afternoon free turns = %+v", got.Slots)
	}
	var public, _ = filtersFromQuery(httptest.NewRequest("GET", "/f?public=true", nil), true)
	if got, ok := public.narrowDay(day, loc, now); !ok || len(got.PublicBookings) != 1 {
		t.Errorf("public with spots = %+v", got.PublicBookings)
	}
	var late, _ = filtersFromQuery(httptest.NewRequest("GET", "/f?from=21:00", nil), true)
	if _, ok := late.narrowDay(day, loc, now); ok {
		t.Error("nothing free from 21:00: the facility drops out")
	}
}

// Every word anywhere: the name, the council (name or slug), the sport, any tag (name or slug).
func TestMatchesEverywhere(t *testing.T) {
	var names = map[string]string{"futbol-sala": "Fútbol sala", "vestiarios": "Vestiarios"}
	var tags = []string{"futbol-sala", "vestiarios", "cuberta"} // «cuberta» has no name here: its slug still counts
	for _, q := range []string{"pavillon", "concello+lalin", "lalin", "f%C3%BAtbol+sala", "futbol-sala", "vestiarios", "cuberta", "baloncesto"} {
		var f, _ = filtersFromQuery(httptest.NewRequest("GET", "/f?q="+q, nil), false)
		if !f.matches("Pavillón municipal", "lalin", "Concello de Lalín", "Baloncesto", tags, names) {
			t.Errorf("%q should match", q)
		}
	}
	var f, _ = filtersFromQuery(httptest.NewRequest("GET", "/f?q=piscina", nil), false)
	if f.matches("Pavillón municipal", "lalin", "Concello de Lalín", "Baloncesto", tags, names) {
		t.Error("«piscina» is nowhere")
	}
}
