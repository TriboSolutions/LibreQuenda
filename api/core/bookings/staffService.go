package bookings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"librequenda/branding"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// StaffDay: one facility's turns on one day (dateParam YYYY-MM-DD, "" = today) and its bookings.
func (s *bookingsService) StaffDay(ctx context.Context, tenantId uuid.UUID, facilityId uuid.UUID, dateParam string) (*StaffDayDto, *shared.ErrorResponse) {
	var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.day_load_failed"}
	var queries = model.New(s.db)
	var facility, loc, facilityErr = s.councilFacility(ctx, queries, tenantId, facilityId)
	if facilityErr != nil {
		return nil, facilityErr
	}
	var now = time.Now()
	var day = startOfDay(now, loc)
	if dateParam != "" {
		var err error
		if day, err = time.ParseInLocation(time.DateOnly, dateParam, loc); err != nil {
			return nil, &shared.ErrorResponse{Code: http.StatusBadRequest, Message: "errors.invalid_query_params"}
		}
	}
	var dayEnd = day.AddDate(0, 0, 1)

	var slots, slotsErr = s.daySlots(ctx, queries, facility, loc, day, dayEnd, false)
	if slotsErr != nil {
		return nil, slotsErr
	}
	var staffCheckin, err = queries.GetFacilityStaffConfirm(ctx, facility.ID)
	if err != nil {
		return nil, failed
	}
	var rows []model.ListFacilityBookingsForStaffRow
	rows, err = queries.ListFacilityBookingsForStaff(ctx, model.ListFacilityBookingsForStaffParams{
		FacilityID: facility.ID, TenantID: tenantId, RangeStart: day, RangeEnd: dayEnd,
	})
	if err != nil {
		return nil, failed
	}
	var bookings = make([]StaffBookingDto, 0, len(rows))
	for _, row := range rows {
		bookings = append(bookings, mapStaffBooking(row, now, staffCheckin))
	}

	var dto = &StaffDayDto{
		Date:         day.Format(time.DateOnly),
		Timezone:     facility.Timezone,
		FacilityId:   facility.ID.String(),
		SlotMinutes:  facility.SlotMinutes,
		Slots:        staffSlots(slots, bookings),
		Bookings:     bookings,
		StaffCheckin: staffCheckin,
	}
	if facility.MinAge.Valid {
		dto.MinAge = &facility.MinAge.Int32
	}
	return dto, nil
}

// StaffBook books a free turn for someone who phoned or came to the desk. The
// turn must be free on the same schedule as in the app (open, no closure, within
// the booking window). Someone with an account cannot be in two places at once,
// and gets the usual confirmation; the per-person limit and the minimum age are
// left to staff, who are talking to the person.
func (s *bookingsService) StaffBook(ctx context.Context, tenantId uuid.UUID, staffId uuid.UUID, input StaffBookingInput) (*StaffBookingDto, *shared.ErrorResponse) {
	var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.booking_failed"}
	if key := input.validate(); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	var queries = model.New(s.db)
	var facility, loc, facilityErr = s.councilFacility(ctx, queries, tenantId, input.FacilityId)
	if facilityErr != nil {
		return nil, facilityErr
	}
	var start = input.StartsAt.In(loc)
	var day = startOfDay(start, loc)
	var slot, slotErr = s.findSlot(ctx, queries, facility, loc, day, day.AddDate(0, 0, 1), start, false)
	if slotErr != nil {
		return nil, slotErr
	}

	var params = model.CreateStaffBookingParams{
		FacilityID: facility.ID,
		StartsAt:   slot.Start,
		EndsAt:     slot.End,
		BookedBy:   uuid.NullUUID{UUID: staffId, Valid: true},
	}
	var holder = StaffBookingDto{Name: input.Name, Phone: input.Phone}
	if input.Email != "" {
		var user, err = queries.GetUserByEmailForStaff(ctx, input.Email)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.person_not_found", Args: []any{branding.Name(ctx, s.db)}}
		}
		if err != nil {
			return nil, failed
		}
		params.UserID = uuid.NullUUID{UUID: user.ID, Valid: true}
		holder = StaffBookingDto{HasAccount: true, Name: user.FullName.String, Email: user.Email}
		if holder.Name == "" {
			holder.Name = user.Email
		}
	} else {
		params.HolderName = sql.NullString{String: input.Name, Valid: true}
		params.HolderPhone = sql.NullString{String: input.Phone, Valid: input.Phone != ""}
	}

	var booking, conflict, err = s.insertStaffBooking(ctx, params)
	if conflict != nil {
		return nil, conflict
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23P01" { // exclusion_violation: booked in the meantime
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.slot_booked"}
	}
	if err != nil {
		return nil, failed
	}
	if params.UserID.Valid {
		s.notify(ctx, queries, params.UserID.UUID, booking.ID, model.NotificationTypeBookingConfirmed, facility.Name, booking.StartsAt, booking.EndsAt)
	}

	var row = model.ListFacilityBookingsForStaffRow{
		ID:           booking.ID,
		StartsAt:     booking.StartsAt,
		EndsAt:       booking.EndsAt,
		Channel:      booking.Channel,
		CheckinToken: booking.CheckinToken,
		UserID:       booking.UserID,
		FullName:     sql.NullString{String: holder.Name, Valid: holder.HasAccount},
		Email:        sql.NullString{String: holder.Email, Valid: holder.HasAccount},
		HolderName:   booking.HolderName,
		HolderPhone:  booking.HolderPhone,
	}
	var staffCheckin, _ = queries.GetFacilityStaffConfirm(ctx, facility.ID) // only hides the button until the next load
	var dto = mapStaffBooking(row, time.Now(), staffCheckin)
	return &dto, nil
}

// StaffBooking: one confirmed booking of the council as the calendar shows it,
// for example after staff marked the entry.
func (s *bookingsService) StaffBooking(ctx context.Context, tenantId uuid.UUID, bookingId uuid.UUID) (*StaffBookingDto, *shared.ErrorResponse) {
	var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.day_load_failed"}
	var notFound = &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.booking_not_found"}
	var queries = model.New(s.db)
	var booking, err = queries.GetBookingForCouncil(ctx, model.GetBookingForCouncilParams{ID: bookingId, TenantID: tenantId})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, notFound
	}
	if err != nil {
		return nil, failed
	}
	var staffCheckin bool
	if staffCheckin, err = queries.GetFacilityStaffConfirm(ctx, booking.FacilityID); err != nil {
		return nil, failed
	}
	var rows []model.ListFacilityBookingsForStaffRow
	rows, err = queries.ListFacilityBookingsForStaff(ctx, model.ListFacilityBookingsForStaffParams{
		FacilityID: booking.FacilityID, TenantID: tenantId, RangeStart: booking.StartsAt, RangeEnd: booking.EndsAt,
	})
	if err != nil {
		return nil, failed
	}
	for _, row := range rows {
		if row.ID == bookingId {
			var dto = mapStaffBooking(row, time.Now(), staffCheckin)
			return &dto, nil
		}
	}
	return nil, notFound // cancelled
}

// insertStaffBooking: for an account, only if the person is not somewhere else then (see insertBooking).
func (s *bookingsService) insertStaffBooking(ctx context.Context, params model.CreateStaffBookingParams) (model.Booking, *shared.ErrorResponse, error) {
	var tx, err = s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Booking{}, nil, err
	}
	defer tx.Rollback()
	var queries = model.New(tx)

	if params.UserID.Valid {
		if err = queries.LockUserBookings(ctx, params.UserID.UUID); err != nil {
			return model.Booking{}, nil, err
		}
		var overlaps bool
		overlaps, err = queries.UserHasOverlappingBooking(ctx, model.UserHasOverlappingBookingParams{UserID: params.UserID, StartsAt: params.StartsAt, EndsAt: params.EndsAt})
		if err != nil {
			return model.Booking{}, nil, err
		}
		if overlaps {
			return model.Booking{}, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.person_overlaps"}, nil
		}
	}

	var booking model.Booking
	if booking, err = queries.CreateStaffBooking(ctx, params); err != nil {
		return model.Booking{}, nil, err
	}
	return booking, nil, tx.Commit()
}

// StaffCancel cancels any booking of the council before it starts, for example
// when the resident phones to say they cannot come, or an event. Someone with an
// account is told, with the reason if staff gave one, and so is everyone who
// joined an event; it counts as cancelled by the council.
func (s *bookingsService) StaffCancel(ctx context.Context, tenantId uuid.UUID, bookingId uuid.UUID, input StaffCancelInput) *shared.ErrorResponse {
	var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.cancel_failed"}
	var queries = model.New(s.db)
	var booking, err = queries.GetBookingForCouncil(ctx, model.GetBookingForCouncilParams{ID: bookingId, TenantID: tenantId})
	if errors.Is(err, sql.ErrNoRows) {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.booking_not_found"}
	}
	if err != nil {
		return failed
	}
	if booking.Status != model.BookingStatusConfirmed {
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.booking_already_cancelled"}
	}
	if !booking.StartsAt.After(time.Now()) {
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.booking_started"}
	}

	var cancelled int64
	if cancelled, err = queries.CancelBookingByCouncil(ctx, bookingId); err != nil {
		return failed
	}
	if cancelled == 0 { // cancelled or started in the meantime
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.booking_already_cancelled"}
	}

	var payload, _ = json.Marshal(map[string]any{
		"facility_id":   booking.FacilityID,
		"facility_name": booking.FacilityName,
		"tenant_slug":   booking.TenantSlug,
		"tenant_name":   booking.TenantName,
		"starts_at":     booking.StartsAt,
		"ends_at":       booking.EndsAt,
		"reason":        input.Reason,
		"by_staff":      true,          // one booking, not a closure: the notice says so
		"title":         booking.Title, // an event's name, "" otherwise
	})
	if booking.UserID.Valid {
		var notifyErr = queries.CreateNotification(ctx, model.CreateNotificationParams{
			UserID:    booking.UserID.UUID,
			BookingID: uuid.NullUUID{UUID: booking.ID, Valid: true},
			Type:      model.NotificationTypeBookingCancelledByCouncil,
			Payload:   payload,
		})
		if notifyErr != nil {
			shared.Logger.Error("failed to record notification for cancelled booking %s: %v", booking.ID, notifyErr)
		}
	}
	// an event has nobody else to pass the news on: everyone who joined is told
	if booking.Kind == model.BookingKindEvent {
		s.notifyParticipants(ctx, queries, booking.ID, model.NotificationTypeBookingCancelledByCouncil, payload)
	}
	return nil
}

// councilFacility: an active facility of this council, and its time zone.
func (s *bookingsService) councilFacility(ctx context.Context, queries *model.Queries, tenantId uuid.UUID, facilityId uuid.UUID) (model.GetFacilityForBookingRow, *time.Location, *shared.ErrorResponse) {
	var notFound = &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	var facility, err = queries.GetFacilityForBooking(ctx, facilityId)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && facility.TenantID != tenantId) {
		return facility, nil, notFound
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
