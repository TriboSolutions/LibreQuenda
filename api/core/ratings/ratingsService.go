package ratings

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type ratingsService struct {
	db *sql.DB
}

var sRatingsInstance *ratingsService
var sRatingsOnce *sync.Once = new(sync.Once)

func GetRatingsService(db *sql.DB) *ratingsService {
	sRatingsOnce.Do(func() {
		sRatingsInstance = &ratingsService{db: db}
	})
	return sRatingsInstance
}

var (
	errNotFound   = &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.booking_not_found"}
	errLoadFailed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.rating_load_failed"}
	errSaveFailed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.rating_save_failed"}
	errClosed     = &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.rating_closed"}
	errRated      = &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.rating_already_given"}
	errNotPlayer  = &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.rating_not_in_booking"}
)

// GetRating: the booking the user was in, the stars they gave, and who else they can rate.
func (s *ratingsService) GetRating(ctx context.Context, userId, bookingId uuid.UUID) (*RatingDto, *shared.ErrorResponse) {
	var queries = model.New(s.db)
	var booking, err = queries.GetBookingToRate(ctx, model.GetBookingToRateParams{BookingID: bookingId, UserID: uuid.NullUUID{UUID: userId, Valid: true}})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errNotFound
	}
	if err != nil {
		return nil, errLoadFailed
	}
	var stars int16
	stars, err = queries.GetMyFacilityRating(ctx, model.GetMyFacilityRatingParams{BookingID: bookingId, UserID: uuid.NullUUID{UUID: userId, Valid: true}})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, errLoadFailed
	}
	var response = &RatingDto{
		BookingId:     booking.ID.String(),
		FacilityName:  booking.FacilityName,
		FacilitySlug:  booking.FacilitySlug,
		TenantSlug:    booking.TenantSlug,
		TenantName:    booking.TenantName,
		StartsAt:      booking.StartsAt,
		EndsAt:        booking.EndsAt,
		IsPublic:      booking.IsPublic,
		CanRate:       CanRate(booking.Status, booking.EndsAt, booking.CheckedInAt.Valid, booking.CheckinAtStart, booking.Rated, time.Now()),
		Rated:         booking.Rated,
		RateUntil:     booking.EndsAt.Add(Window),
		FacilityStars: stars,
		People:        []PersonDto{},
	}
	if !booking.IsPublic {
		return response, nil
	}
	var people []model.ListPeopleToRateRow
	if people, err = queries.ListPeopleToRate(ctx, model.ListPeopleToRateParams{BookingID: bookingId, RaterID: userId}); err != nil {
		return nil, errLoadFailed
	}
	for _, p := range people {
		response.People = append(response.People, PersonDto{UserId: p.UserID.UUID.String(), Name: p.Name, IsOwner: p.IsOwner, Stars: p.Stars})
	}
	return response, nil
}

// Rate saves the stars, all or none: the facility's, and the others' after a public booking.
func (s *ratingsService) Rate(ctx context.Context, userId, bookingId uuid.UUID, input RateInput) (*RatingDto, *shared.ErrorResponse) {
	var tx, err = s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errSaveFailed
	}
	defer tx.Rollback()
	var queries = model.New(tx)

	var booking model.GetBookingToRateRow
	booking, err = queries.GetBookingToRate(ctx, model.GetBookingToRateParams{BookingID: bookingId, UserID: uuid.NullUUID{UUID: userId, Valid: true}})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errNotFound
	}
	if err != nil {
		return nil, errSaveFailed
	}
	if booking.Rated {
		return nil, errRated
	}
	if !CanRate(booking.Status, booking.EndsAt, booking.CheckedInAt.Valid, booking.CheckinAtStart, false, time.Now()) {
		return nil, errClosed
	}
	if len(input.People) > 0 {
		// only the others with an account, after a public booking
		if !booking.IsPublic {
			return nil, errNotPlayer
		}
		var people []model.ListPeopleToRateRow
		if people, err = queries.ListPeopleToRate(ctx, model.ListPeopleToRateParams{BookingID: bookingId, RaterID: userId}); err != nil {
			return nil, errSaveFailed
		}
		var players = make(map[uuid.UUID]bool, len(people))
		for _, p := range people {
			players[p.UserID.UUID] = true
		}
		for _, p := range input.People {
			if !players[p.UserId] {
				return nil, errNotPlayer
			}
		}
	}

	// plain inserts: the unique keys refuse a second rating, even one sent at the same time
	err = queries.InsertFacilityRating(ctx, model.InsertFacilityRatingParams{
		FacilityID: booking.FacilityID,
		BookingID:  bookingId,
		UserID:     uuid.NullUUID{UUID: userId, Valid: true},
		Stars:      input.FacilityStars,
	})
	if isUniqueViolation(err) {
		return nil, errRated
	}
	if err != nil {
		return nil, errSaveFailed
	}
	for _, p := range input.People {
		if p.Stars == 0 {
			continue
		}
		err = queries.InsertUserRating(ctx, model.InsertUserRatingParams{BookingID: bookingId, RaterID: userId, RatedID: p.UserId, Stars: p.Stars})
		if isUniqueViolation(err) {
			return nil, errRated
		}
		if err != nil {
			return nil, errSaveFailed
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, errSaveFailed
	}
	return s.GetRating(ctx, userId, bookingId)
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
