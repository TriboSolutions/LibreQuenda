package tenants

import (
	"librequenda/model"
	"librequenda/schedule"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Tuesday 3 Nov 2026 (ISO weekday 2), in UTC so local hours are the slot hours.
func at(hour int) time.Time { return time.Date(2026, 11, 3, hour, 0, 0, 0, time.UTC) }

func slot(hour int, status schedule.SlotStatus) schedule.Slot {
	return schedule.Slot{Start: at(hour), End: at(hour + 1), Status: status}
}

func coords(lat, lon float64) (*float64, *float64) { return &lat, &lon }

func TestQuietestFree(t *testing.T) {
	var id = uuid.New()
	var histories = busyHistoryByFacility([]model.ListBusyHoursByTenantRow{
		{FacilityID: id, Weekday: 2, Hour: 18, Bookings: 9},
		{FacilityID: id, Weekday: 2, Hour: 19, Bookings: 5},
		{FacilityID: id, Weekday: 2, Hour: 10, Bookings: 1},
		{FacilityID: id, Weekday: 3, Hour: 9, Bookings: 7}, // another weekday: ignored
	})
	var slots = []schedule.Slot{
		slot(9, schedule.SlotFree),
		slot(10, schedule.SlotFree),
		slot(11, schedule.SlotBooked),
		slot(18, schedule.SlotFree),
		slot(19, schedule.SlotFree),
		slot(20, schedule.SlotPast),
	}
	var got = quietestFree(slots, histories[id], time.UTC, 4)
	var want = []struct {
		hour  int
		level BusyLevel
	}{{9, BusyQuiet}, {10, BusyQuiet}, {19, BusyModerate}, {18, BusyBusy}}
	if len(got) != len(want) {
		t.Fatalf("quietestFree returned %d slots, want %d", len(got), len(want))
	}
	for i, w := range want {
		if !got[i].StartsAt.Equal(at(w.hour)) || got[i].Level != w.level {
			t.Errorf("slot %d = %s %s, want %02d:00 %s", i, got[i].StartsAt.Format("15:04"), got[i].Level, w.hour, w.level)
		}
	}
	if n := len(quietestFree(slots, histories[id], time.UTC, 2)); n != 2 {
		t.Errorf("quietestFree with n=2 returned %d slots", n)
	}
}

func TestSuggest(t *testing.T) {
	var full = []schedule.Slot{slot(18, schedule.SlotBooked)}
	var open = []schedule.Slot{slot(18, schedule.SlotFree), slot(19, schedule.SlotFree)}
	var facility = func(name, sport string, slots []schedule.Slot, lat, lon *float64) FacilityDayDto {
		return FacilityDayDto{Id: uuid.NewString(), Slug: name, Name: name, Sport: sport, Slots: slots, Latitude: lat, Longitude: lon}
	}
	var target = facility("target", "Pádel", full, nil, nil)
	target.Latitude, target.Longitude = coords(43.0, -7.55)
	var farLat, farLon = coords(43.1, -7.55)
	var nearLat, nearLon = coords(43.01, -7.55)
	var facilities = []FacilityDayDto{
		target,
		facility("far", "pádel ", open, farLat, farLon),
		facility("no-pin", "Pádel", open, nil, nil),
		facility("near", "Pádel", open, nearLat, nearLon),
		facility("tennis", "Tenis", open, nearLat, nearLon),    // another sport
		facility("also-full", "Pádel", full, nearLat, nearLon), // nothing free
	}

	var got = suggest(target, facilities, map[uuid.UUID]*busyHistory{}, time.UTC)
	if len(got.Quiet) != 0 {
		t.Errorf("no history should mean no quiet turns, got %d", len(got.Quiet))
	}
	var names []string
	for _, a := range got.Alternatives {
		names = append(names, a.Name)
	}
	if want := []string{"near", "far", "no-pin"}; len(names) != 3 || names[0] != want[0] || names[1] != want[1] || names[2] != want[2] {
		t.Errorf("alternatives = %v, want %v", names, want)
	}
	if d := got.Alternatives[0].DistanceKm; d == nil || *d != 1.1 {
		t.Errorf("near distance = %v, want 1.1 km", d)
	}
	if got.Alternatives[0].FreeSlots != 2 {
		t.Errorf("near free slots = %d, want 2", got.Alternatives[0].FreeSlots)
	}

	target.Sport = ""
	if got := suggest(target, facilities, nil, time.UTC); len(got.Alternatives) != 0 {
		t.Errorf("a facility with no sport should get no alternatives, got %d", len(got.Alternatives))
	}
}
