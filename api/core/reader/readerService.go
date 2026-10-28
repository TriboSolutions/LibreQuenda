package reader

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/checkin"
	"librequenda/core/bookings"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type readerService struct {
	db *sql.DB
}

var sInstance *readerService
var sOnce *sync.Once = new(sync.Once)

func GetReaderService(db *sql.DB) *readerService {
	sOnce.Do(func() {
		sInstance = &readerService{db: db}
	})
	return sInstance
}

// decide applies the entrance rules to a booking scanned at a facility.
func decide(booking model.GetBookingByCheckinTokenRow, facilityId uuid.UUID, now time.Time) string {
	switch {
	case booking.FacilityID != facilityId:
		return ResultOtherFacility
	case booking.Status != model.BookingStatusConfirmed:
		return ResultCancelled
	case !now.Before(booking.EndsAt):
		return ResultEnded
	case booking.CheckedInAt.Valid:
		return ResultAlreadyCheckedIn // re-entry during the booking
	case now.Before(booking.StartsAt.Add(-checkin.EarlyWindow)):
		return ResultTooEarly
	}
	return ResultCheckedIn
}

// CheckIn validates a scanned QR at the reader's facility: a booking QR lets its
// booking in; the personal QR (quick entry) flash-books a free turn.
func (s *readerService) CheckIn(ctx context.Context, device model.GetCheckinDeviceByKeyRow, code string) (*CheckinResultDto, *shared.ErrorResponse) {
	if token, ok := checkin.ParsePersonalPayload(code); ok {
		return s.personalCheckIn(ctx, device, token)
	}
	var token, ok = checkin.ParsePayload(code)
	if !ok {
		return &CheckinResultDto{Result: ResultNotABooking}, nil
	}
	return s.bookingCheckIn(ctx, device, token)
}

// bookingCheckIn records the first entry of the booking with that check-in token.
func (s *readerService) bookingCheckIn(ctx context.Context, device model.GetCheckinDeviceByKeyRow, token uuid.UUID) (*CheckinResultDto, *shared.ErrorResponse) {
	var queries = model.New(s.db)
	var booking, err = queries.GetBookingByCheckinToken(ctx, token)
	if errors.Is(err, sql.ErrNoRows) {
		return &CheckinResultDto{Result: ResultUnknownBooking}, nil
	}
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.checkin_failed"}
	}

	var result = decide(booking, device.FacilityID, time.Now())
	if result == ResultCheckedIn {
		var checkedInAt sql.NullTime
		checkedInAt, err = queries.MarkBookingCheckedIn(ctx, model.MarkBookingCheckedInParams{
			ID:       booking.ID,
			DeviceID: uuid.NullUUID{UUID: device.ID, Valid: true},
		})
		switch {
		case errors.Is(err, sql.ErrNoRows): // another reader got there first
			result = ResultAlreadyCheckedIn
			if booking, err = queries.GetBookingByCheckinToken(ctx, token); err != nil {
				return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.checkin_failed"}
			}
		case err != nil:
			return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.checkin_failed"}
		default:
			booking.CheckedInAt = checkedInAt
		}
	}

	var response = &CheckinResultDto{
		Granted: result == ResultCheckedIn || result == ResultAlreadyCheckedIn,
		Result:  result,
	}
	if result != ResultCancelled {
		response.Booking = &CheckinBookingDto{
			FacilityName: booking.FacilityName,
			StartsAt:     booking.StartsAt,
			EndsAt:       booking.EndsAt,
		}
		if response.Granted {
			response.Booking.FirstName = firstName(booking.FullName.String)
			response.Booking.CheckedInAt = &booking.CheckedInAt.Time
		}
	}
	return response, nil
}

// personalCheckIn flash-books a free turn for the person (bookings.QuickBook) and
// checks them in. It never opens an existing booking: that one goes in with its
// own QR, so with a booking here that could go in now it is refused. The QR is
// used up only when the reader lets the person in.
func (s *readerService) personalCheckIn(ctx context.Context, device model.GetCheckinDeviceByKeyRow, qrToken uuid.UUID) (*CheckinResultDto, *shared.ErrorResponse) {
	var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.checkin_failed"}
	var queries = model.New(s.db)
	var owner, err = queries.UseUserQrToken(ctx, qrToken)
	if errors.Is(err, sql.ErrNoRows) {
		return &CheckinResultDto{Result: ResultCodeExpired}, nil
	}
	if err != nil {
		return nil, failed
	}

	_, err = queries.GetUserCheckinTokenAtFacility(ctx, model.GetUserCheckinTokenAtFacilityParams{
		UserID:     uuid.NullUUID{UUID: owner.UserID, Valid: true},
		FacilityID: device.FacilityID,
		WindowEnds: time.Now().Add(checkin.EarlyWindow),
	})
	if err == nil {
		s.release(ctx, queries, qrToken)
		return &CheckinResultDto{Result: ResultNotBooked, Reason: "use_booking_qr"}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		s.release(ctx, queries, qrToken)
		return nil, failed
	}

	var booking, refused = bookings.GetBookingsService(s.db).QuickBook(ctx, owner.UserID, device.FacilityID)
	if refused != nil {
		s.release(ctx, queries, qrToken)
		if refused.Code == http.StatusInternalServerError {
			return nil, failed
		}
		return &CheckinResultDto{Result: ResultNotBooked, Reason: strings.TrimPrefix(refused.Message, "errors.")}, nil
	}

	// booked: the entry is recorded as well, so the turn never shows as a no-show
	var checkedInAt sql.NullTime
	checkedInAt, err = queries.MarkBookingCheckedIn(ctx, model.MarkBookingCheckedInParams{
		ID:       uuid.MustParse(booking.Id),
		DeviceID: uuid.NullUUID{UUID: device.ID, Valid: true},
	})
	if err != nil {
		shared.Logger.Error("failed to check in quick booking %s: %v", booking.Id, err)
		checkedInAt = sql.NullTime{Time: time.Now(), Valid: true}
	}
	return &CheckinResultDto{
		Granted: true,
		Result:  ResultBooked,
		Booking: &CheckinBookingDto{
			FacilityName: booking.FacilityName,
			FirstName:    firstName(owner.FullName.String),
			StartsAt:     booking.StartsAt,
			EndsAt:       booking.EndsAt,
			CheckedInAt:  &checkedInAt.Time,
		},
	}, nil
}

// release gives a personal QR back after a refusal, so the person can try again with it.
func (s *readerService) release(ctx context.Context, queries *model.Queries, qrToken uuid.UUID) {
	if err := queries.ReleaseUserQrToken(ctx, qrToken); err != nil {
		shared.Logger.Error("failed to release personal QR token: %v", err)
	}
}
