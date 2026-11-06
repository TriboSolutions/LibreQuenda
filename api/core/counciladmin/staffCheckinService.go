package counciladmin

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/core/bookings"
	"librequenda/core/checkin"
	"librequenda/core/reader"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// staffCheckinRefusal: the error for a booking staff cannot mark as in, from the
// reader's result; nil when it went in.
func staffCheckinRefusal(result string) *shared.ErrorResponse {
	var conflict = func(key string) *shared.ErrorResponse {
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: key}
	}
	switch result {
	case reader.ResultCheckedIn:
		return nil
	case reader.ResultAlreadyCheckedIn:
		return conflict("errors.already_checked_in")
	case reader.ResultTooEarly:
		return conflict("errors.staff_checkin_too_early")
	case reader.ResultEnded:
		return conflict("errors.staff_checkin_ended")
	case reader.ResultCancelled:
		return conflict("errors.booking_already_cancelled")
	}
	return failed("errors.checkin_failed")
}

// StaffCheckIn: staff mark that the person came in, from the bookings calendar,
// when the facility allows it (Entrada tab). It goes through the reader's own
// check-in, with the same window, so the indicators stay the same; it works for
// bookings by name too, and records who marked it.
func (s *counciladminService) StaffCheckIn(ctx context.Context, tenantId uuid.UUID, staffId uuid.UUID, bookingId uuid.UUID) (*bookings.StaffBookingDto, *shared.ErrorResponse) {
	var queries = model.New(s.db)
	var booking, err = queries.GetBookingForCouncil(ctx, model.GetBookingForCouncilParams{ID: bookingId, TenantID: tenantId})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.booking_not_found"}
	}
	if err != nil {
		return nil, failed("errors.checkin_failed")
	}
	var allowed bool
	if allowed, err = queries.GetFacilityStaffConfirm(ctx, booking.FacilityID); err != nil {
		return nil, failed("errors.checkin_failed")
	}
	if !allowed {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.staff_checkin_off"}
	}

	var entry = reader.Entry{Method: model.CheckinMethodStaff, StaffID: uuid.NullUUID{UUID: staffId, Valid: true}}
	var result, checkinErr = reader.GetReaderService(s.db).CheckInBooking(ctx, booking.FacilityID, entry, booking.CheckinToken)
	if checkinErr != nil {
		return nil, checkinErr
	}
	if refused := staffCheckinRefusal(result.Result); refused != nil {
		return nil, refused
	}
	return bookings.GetBookingsService(s.db).StaffBooking(ctx, tenantId, bookingId)
}

// UndoStaffCheckIn takes back an entry staff marked by mistake, within
// checkin.StaffUndoWindow. Entries from a reader or a phone stay.
func (s *counciladminService) UndoStaffCheckIn(ctx context.Context, tenantId uuid.UUID, bookingId uuid.UUID) (*bookings.StaffBookingDto, *shared.ErrorResponse) {
	var queries = model.New(s.db)
	var undone, err = queries.UndoStaffCheckIn(ctx, model.UndoStaffCheckInParams{
		ID:       bookingId,
		TenantID: tenantId,
		Since:    time.Now().Add(-checkin.StaffUndoWindow),
	})
	if err != nil {
		return nil, failed("errors.checkin_failed")
	}
	if undone == 0 {
		if _, err = queries.GetBookingForCouncil(ctx, model.GetBookingForCouncilParams{ID: bookingId, TenantID: tenantId}); errors.Is(err, sql.ErrNoRows) {
			return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.booking_not_found"}
		}
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.staff_checkin_undo_expired"}
	}
	return bookings.GetBookingsService(s.db).StaffBooking(ctx, tenantId, bookingId)
}
