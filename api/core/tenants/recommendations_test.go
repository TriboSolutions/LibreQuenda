package tenants

import (
	"database/sql"
	"librequenda/model"
	"librequenda/schedule"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRecommend(t *testing.T) {
	var madrid, _ = time.LoadLocation("Europe/Madrid")
	// Friday 30 Oct 2026, 16:00 in Galicia
	var now = time.Date(2026, 10, 30, 16, 0, 0, 0, madrid)
	var at = func(hour int) time.Time { return time.Date(2026, 10, 30, hour, 0, 0, 0, madrid) }
	var turns = func(free ...int) []schedule.Slot {
		var slots []schedule.Slot
		for hour := 9; hour < 22; hour++ {
			var status = schedule.SlotBooked
			if hour <= 16 {
				status = schedule.SlotPast
			}
			for _, f := range free {
				if f == hour && hour > 16 {
					status = schedule.SlotFree
				}
			}
			slots = append(slots, schedule.Slot{Start: at(hour), End: at(hour + 1), Status: status})
		}
		return slots
	}
	var padel, pavilion, tennis, pool = uuid.New(), uuid.New(), uuid.New(), uuid.New()
	var today = []todayFacility{
		{card: FacilityCardDto{Id: padel.String(), Name: "Pádel 1", Tags: []string{"Pádel", "Cuberta"}}, slots: turns(19, 20), loc: madrid},
		{card: FacilityCardDto{Id: pavilion.String(), Name: "Pavillón", Tags: []string{"Baloncesto", "Cuberta"}}, slots: turns(18), loc: madrid},
		{card: FacilityCardDto{Id: tennis.String(), Name: "Tenis", Tags: []string{"Tenis"}}, slots: turns(17), loc: madrid},
		{card: FacilityCardDto{Id: pool.String(), Name: "Piscina", Tags: []string{"Natación", "Cuberta"}}, slots: turns(), loc: madrid},
	}
	var patterns = []model.ListUserBookingPatternsRow{
		{FacilityID: padel, Weekday: 5, StartTime: "19:00", Bookings: 5}, // Fridays at 19:00: free today
		{FacilityID: padel, Weekday: 2, StartTime: "10:00", Bookings: 2},
		{FacilityID: tennis, Weekday: 6, StartTime: "11:00", Bookings: 3}, // Saturdays: not today, but free at 17:00
	}

	var got = recommend(patterns, today, now)
	if len(got) != 3 {
		t.Fatalf("want 3 cards, got %+v", got)
	}
	if got[0].Kind != RecommendUsualToday || got[0].Facility.Name != "Pádel 1" || !got[0].StartsAt.Equal(at(19)) || got[0].Times != 5 {
		t.Errorf("first, their usual Friday 19:00: %+v", got[0])
	}
	if got[1].Kind != RecommendFavourite || got[1].Facility.Name != "Tenis" || !got[1].StartsAt.Equal(at(17)) || got[1].Times != 3 {
		t.Errorf("then a facility they book, free today: %+v", got[1])
	}
	// the pavilion shares "Cuberta" with their padel court; the pool has no free turn left
	if got[2].Kind != RecommendSimilar || got[2].Facility.Name != "Pavillón" || got[2].Because != "Pádel 1" || len(got[2].SharedTags) != 1 {
		t.Errorf("then one alike, not tried: %+v", got[2])
	}

	if got := recommend(nil, today, now); got == nil || len(got) != 0 {
		t.Errorf("no history, no cards: %+v", got)
	}
}

func TestPublicBookings(t *testing.T) {
	var madrid, _ = time.LoadLocation("Europe/Madrid")
	var at = func(hour int) time.Time { return time.Date(2026, 10, 31, hour, 0, 0, 0, madrid) }
	var lat, lon, farLat = 42.664, -8.108, 43.5
	var padel, pavilion, pool, tennis = uuid.New(), uuid.New(), uuid.New(), uuid.New()
	var today = []todayFacility{
		{card: FacilityCardDto{Id: padel.String(), Name: "Pádel", Tags: []string{"padel", "cuberta"}}, loc: madrid},
		{card: FacilityCardDto{Id: pavilion.String(), Name: "Pavillón", Tags: []string{"baloncesto", "cuberta"}}, loc: madrid},
		{card: FacilityCardDto{Id: pool.String(), Name: "Piscina", Tags: []string{"natacion"}, Latitude: &lat, Longitude: &lon}, loc: madrid},
		{card: FacilityCardDto{Id: tennis.String(), Name: "Tenis", Tags: []string{"tenis"}, Latitude: &farLat, Longitude: &lon}, loc: madrid},
	}
	var four = sql.NullInt32{Int32: 4, Valid: true}
	var rows = []model.ListJoinablePublicBookingsRow{
		{ID: uuid.New(), FacilityID: padel, StartsAt: at(10), EndsAt: at(11), Capacity: four, ParticipantCount: 1, OwnerFirstName: "Uxía"}, // theirs
		{ID: uuid.New(), FacilityID: pavilion, StartsAt: at(11), EndsAt: at(12), Capacity: four},                                           // alike: covered
		{ID: uuid.New(), FacilityID: pool, StartsAt: at(12), EndsAt: at(13), Capacity: four},                                               // near
		{ID: uuid.New(), FacilityID: tennis, StartsAt: at(13), EndsAt: at(14), Capacity: four},                                             // far, unlike
		{ID: uuid.New(), FacilityID: padel, StartsAt: at(15), EndsAt: at(16), Capacity: four, ParticipantCount: 3},                         // full
	}
	var patterns = []model.ListUserBookingPatternsRow{{FacilityID: padel, Weekday: 6, StartTime: "10:00", Bookings: 4}}

	var got = publicBookings(rows, patterns, today, Position{Latitude: &lat, Longitude: &lon})
	if len(got) != 3 || got[0].Facility.Name != "Pádel" || got[1].Facility.Name != "Pavillón" || got[2].Facility.Name != "Piscina" {
		t.Fatalf("theirs, alike and near, soonest first: %+v", got)
	}
	if got[0].Kind != RecommendPublic || got[0].Booking.SpotsLeft != 2 || got[0].Booking.OwnerFirstName != "Uxía" {
		t.Errorf("the card of a public booking: %+v", got[0].Booking)
	}
	// without the person's location, the one near them is not known
	if got := publicBookings(rows, patterns, today, Position{}); len(got) != 2 {
		t.Errorf("without a position: want theirs and alike, got %d", len(got))
	}
	// no history and no position: nothing is relevant
	if got := publicBookings(rows, nil, today, Position{}); len(got) != 0 {
		t.Errorf("no history, no position: got %+v", got)
	}
}
