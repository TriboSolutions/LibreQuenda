package tenants

import (
	"database/sql"
	"librequenda/model"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUpcomingEvent(t *testing.T) {
	var at = time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	var row = model.ListUpcomingEventsRow{
		ID: uuid.New(), Title: "Torneo de fútbol 5x5", StartsAt: at, EndsAt: at.Add(4 * time.Hour),
		Capacity: sql.NullInt32{Int32: 30, Valid: true}, ParticipantCount: 8,
		FacilityID: uuid.New(), FacilitySlug: "pavillon", FacilityName: "Pavillón", TenantSlug: "lalin", TenantName: "Concello de Lalín",
		Latitude: sql.NullFloat64{Float64: 42.66, Valid: true}, Longitude: sql.NullFloat64{Float64: -8.11, Valid: true},
	}
	var dto = mapUpcomingEvent(row, &Near{Latitude: 42.66, Longitude: -8.11})
	if dto.Capacity != 30 || dto.SpotsLeft != 22 {
		t.Errorf("capacity %d, spots %d: an event has no owner taking a place", dto.Capacity, dto.SpotsLeft)
	}
	if dto.Facility.Council.Slug != "lalin" || dto.Facility.DistanceKm == nil || *dto.Facility.DistanceKm != 0 {
		t.Errorf("facility card = %+v", dto.Facility)
	}
	row.ParticipantCount = 31 // the capacity lowered under who joined, somehow: never below 0
	if got := mapUpcomingEvent(row, nil); got.SpotsLeft != 0 || got.Facility.DistanceKm != nil {
		t.Errorf("full event = %+v", got)
	}
}
