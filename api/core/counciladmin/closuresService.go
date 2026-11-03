package counciladmin

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/db"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// farFuture: "from now on" for a deactivated facility.
var farFuture = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

// SetActive switches a facility on or off. Switching it off keeps the bookings
// already made, unless the council asks to cancel them (with a reason for residents).
func (s *counciladminService) SetActive(ctx context.Context, tenantId uuid.UUID, id uuid.UUID, input ActiveInput) (*ActiveResultDto, *shared.ErrorResponse) {
	if key := input.validate(); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	var cancelled int64
	var err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		var updated, updateErr = queries.SetFacilityActive(ctx, model.SetFacilityActiveParams{ID: id, TenantID: tenantId, IsActive: input.Active})
		if updateErr != nil {
			return updateErr
		}
		if updated == 0 {
			return sql.ErrNoRows
		}
		if !input.CancelBookings {
			return nil
		}
		var cancelErr error
		cancelled, cancelErr = queries.CancelBookingsInRange(ctx, model.CancelBookingsInRangeParams{
			FacilityID: id, RangeStart: time.Now(), RangeEnd: farFuture, Reason: input.Reason,
		})
		return cancelErr
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	}
	if err != nil {
		return nil, failed("errors.facility_save_failed")
	}
	var detail, detailErr = s.GetFacility(ctx, tenantId, id)
	if detailErr != nil {
		return nil, detailErr
	}
	return &ActiveResultDto{Facility: detail, Cancelled: cancelled}, nil
}

func (s *counciladminService) closureRange(ctx context.Context, tenantId uuid.UUID, id uuid.UUID, input *ClosureInput) (time.Time, time.Time, *shared.ErrorResponse) {
	var timezone, err = model.New(s.db).GetFacilityTimezone(ctx, model.GetFacilityTimezoneParams{FacilityID: id, TenantID: tenantId})
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, time.Time{}, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	}
	if err != nil {
		return time.Time{}, time.Time{}, failed("errors.closure_save_failed")
	}
	var loc, locErr = time.LoadLocation(timezone)
	if locErr != nil {
		return time.Time{}, time.Time{}, failed("errors.closure_save_failed")
	}
	var start, end, key = input.parse(loc, time.Now())
	if key != "" {
		return time.Time{}, time.Time{}, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	return start, end, nil
}

// PreviewClosure: how many bookings the closure would cancel, for the confirmation.
func (s *counciladminService) PreviewClosure(ctx context.Context, tenantId uuid.UUID, id uuid.UUID, input ClosureInput) (*ClosurePreviewDto, *shared.ErrorResponse) {
	var start, end, rangeErr = s.closureRange(ctx, tenantId, id, &input)
	if rangeErr != nil {
		return nil, rangeErr
	}
	var count, err = model.New(s.db).CountBookingsToCancel(ctx, model.CountBookingsToCancelParams{FacilityID: id, RangeStart: start, RangeEnd: end})
	if err != nil {
		return nil, failed("errors.closure_save_failed")
	}
	return &ClosurePreviewDto{BookingsToCancel: count}, nil
}

// CreateClosure closes the facility for a while: the slots show as closed and the
// bookings inside are cancelled, each owner told why and until when.
func (s *counciladminService) CreateClosure(ctx context.Context, tenantId uuid.UUID, id uuid.UUID, input ClosureInput) (*ClosureResultDto, *shared.ErrorResponse) {
	var start, end, rangeErr = s.closureRange(ctx, tenantId, id, &input)
	if rangeErr != nil {
		return nil, rangeErr
	}
	var row model.CreateClosureRow
	var cancelled int64
	var err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		var createErr error
		row, createErr = queries.CreateClosure(ctx, model.CreateClosureParams{
			FacilityID: id, StartsAt: start, EndsAt: end, Reason: sql.NullString{String: input.Reason, Valid: true},
		})
		if createErr != nil {
			return createErr
		}
		cancelled, createErr = queries.CancelBookingsInRange(ctx, model.CancelBookingsInRangeParams{
			FacilityID:  id,
			RangeStart:  start,
			RangeEnd:    end,
			Reason:      input.Reason,
			ClosedFrom:  sql.NullTime{Time: start, Valid: true},
			ClosedUntil: sql.NullTime{Time: end, Valid: true},
		})
		return createErr
	})
	if err != nil {
		return nil, failed("errors.closure_save_failed")
	}
	var result = &ClosureResultDto{Closure: mapClosure(row.ID, row.StartsAt, row.EndsAt, row.Reason), Cancelled: cancelled, Closures: []ClosureDto{}}
	// the whole list, in the database's order, so the page does not sort it
	if upcoming, listErr := model.New(s.db).ListUpcomingClosures(ctx, id); listErr == nil {
		for _, c := range upcoming {
			result.Closures = append(result.Closures, mapClosure(c.ID, c.StartsAt, c.EndsAt, c.Reason))
		}
	}
	return result, nil
}

// DeleteClosure reopens the slots. Bookings it cancelled stay cancelled.
func (s *counciladminService) DeleteClosure(ctx context.Context, tenantId uuid.UUID, id uuid.UUID, closureId uuid.UUID) *shared.ErrorResponse {
	var deleted int64
	var err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		var deleteErr error
		deleted, deleteErr = queries.DeleteClosure(ctx, model.DeleteClosureParams{ID: closureId, FacilityID: id, TenantID: tenantId})
		return deleteErr
	})
	if err != nil {
		return failed("errors.closure_delete_failed")
	}
	if deleted == 0 {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.closure_not_found"}
	}
	return nil
}

func mapClosure(id uuid.UUID, startsAt, endsAt time.Time, reason sql.NullString) ClosureDto {
	return ClosureDto{Id: id.String(), StartsAt: startsAt, EndsAt: endsAt, Reason: reason.String}
}
