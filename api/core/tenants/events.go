package tenants

import (
	"context"
	"librequenda/core/bookings"
	"librequenda/model"
	"librequenda/shared"
	"librequenda/storage"
	"net/http"
	"time"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

// UpcomingEventDto: one of the councils' events, as the app's home lists them:
// what it is, when, how many places are left, and the facility to open it in.
type UpcomingEventDto struct {
	Id          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	StartsAt    time.Time       `json:"starts_at"`
	EndsAt      time.Time       `json:"ends_at"`
	Capacity    int             `json:"capacity"`   // who can join
	SpotsLeft   int             `json:"spots_left"` // 0: full
	Facility    FacilityCardDto `json:"facility"`
}

type UpcomingEventsResponse struct {
	Events []UpcomingEventDto `json:"events"` // not started yet, soonest first
}

// GetEvents: GET /events (?lat=&lon= for the distance), every council's upcoming events.
func (h *tenantsHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	var near, ok = nearFromQuery(r)
	if !ok {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return
	}
	var response, err = GetTenantsService(h.db).UpcomingEvents(r.Context(), near)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	helpers.SendSuccessResponse(w, response)
}

func (s *tenantsService) UpcomingEvents(ctx context.Context, near *Near) (*UpcomingEventsResponse, *shared.ErrorResponse) {
	var rows, err = model.New(s.db).ListUpcomingEvents(ctx)
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.events_load_failed"}
	}
	var response = &UpcomingEventsResponse{Events: make([]UpcomingEventDto, 0, len(rows))}
	for _, row := range rows {
		response.Events = append(response.Events, mapUpcomingEvent(row, near))
	}
	return response, nil
}

func mapUpcomingEvent(row model.ListUpcomingEventsRow, near *Near) UpcomingEventDto {
	var places = bookings.Places(row.Capacity, model.BookingKindEvent)
	var card = FacilityCardDto{
		Id:            row.FacilityID.String(),
		Slug:          row.FacilitySlug,
		Name:          row.FacilityName,
		Tags:          []string{},
		ImageUrl:      storage.URL(row.ImageRef.String),
		Council:       TenantDto{Slug: row.TenantSlug, Name: row.TenantName},
		ResidentsOnly: row.ResidentsOnly,
		Joinable:      []string{string(model.BookingKindEvent)},
	}
	if row.Latitude.Valid && row.Longitude.Valid {
		card.Latitude, card.Longitude = &row.Latitude.Float64, &row.Longitude.Float64
	}
	card.DistanceKm = distanceKm(near, card.Latitude, card.Longitude)
	return UpcomingEventDto{
		Id:          row.ID.String(),
		Title:       row.Title,
		Description: row.PublicDescription,
		StartsAt:    row.StartsAt,
		EndsAt:      row.EndsAt,
		Capacity:    places,
		SpotsLeft:   max(places-int(row.ParticipantCount), 0),
		Facility:    card,
	}
}
