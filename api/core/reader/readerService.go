package reader

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/checkin"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
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

// CheckIn validates a scanned QR at the reader's facility and records the first entry.
func (s *readerService) CheckIn(ctx context.Context, device model.GetCheckinDeviceByKeyRow, code string) (*CheckinResultDto, *shared.ErrorResponse) {
	var token, ok = checkin.ParsePayload(code)
	if !ok {
		return &CheckinResultDto{Result: ResultNotABooking}, nil
	}
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
