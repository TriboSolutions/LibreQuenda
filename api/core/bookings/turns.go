package bookings

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/checkin"
	"librequenda/model"
	"librequenda/schedule"
	"librequenda/shared"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Turns of a facility for the turn codes: the daily sheet prints a day of them,
// and the entrance checks a typed code against the ones open for entry now.

// DayTurns: the facility's turns on `day` (YYYY-MM-DD in the council's zone), booked or not.
func (s *bookingsService) DayTurns(ctx context.Context, facilityId uuid.UUID, day string) ([]schedule.Slot, *time.Location, *shared.ErrorResponse) {
	var queries = model.New(s.db)
	var facility, loc, err = s.facilityZone(ctx, queries, facilityId)
	if err != nil {
		return nil, nil, err
	}
	var date, parseErr = time.ParseInLocation(time.DateOnly, day, loc)
	if parseErr != nil {
		return nil, nil, &shared.ErrorResponse{Code: http.StatusBadRequest, Message: "errors.invalid_query_params"}
	}
	var slots, slotsErr = s.daySlots(ctx, queries, facility, loc, date, date.AddDate(0, 0, 1))
	return slots, loc, slotsErr
}

// EntranceTurns: today's turns, and tomorrow's when the early window reaches past midnight.
func (s *bookingsService) EntranceTurns(ctx context.Context, facilityId uuid.UUID, now time.Time) ([]schedule.Slot, *shared.ErrorResponse) {
	var queries = model.New(s.db)
	var facility, loc, err = s.facilityZone(ctx, queries, facilityId)
	if err != nil {
		return nil, err
	}
	var today = startOfDay(now, loc)
	var slots, slotsErr = s.daySlots(ctx, queries, facility, loc, today, today.AddDate(0, 0, 1))
	if slotsErr != nil {
		return nil, slotsErr
	}
	if tomorrow := startOfDay(now.Add(checkin.EarlyWindow), loc); tomorrow.After(today) {
		var more []schedule.Slot
		if more, slotsErr = s.daySlots(ctx, queries, facility, loc, tomorrow, tomorrow.AddDate(0, 0, 1)); slotsErr != nil {
			return nil, slotsErr
		}
		slots = append(slots, more...)
	}
	return slots, nil
}

func (s *bookingsService) facilityZone(ctx context.Context, queries *model.Queries, facilityId uuid.UUID) (model.GetFacilityForBookingRow, *time.Location, *shared.ErrorResponse) {
	var facility, err = queries.GetFacilityForBooking(ctx, facilityId)
	if errors.Is(err, sql.ErrNoRows) {
		return facility, nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	}
	if err != nil {
		return facility, nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.booking_failed"}
	}
	var loc, locErr = time.LoadLocation(facility.Timezone)
	if locErr != nil {
		return facility, nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.booking_failed"}
	}
	return facility, loc, nil
}
