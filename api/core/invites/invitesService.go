package invites

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/core/bookings"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type invitesService struct {
	db *sql.DB
}

var sInvitesInstance *invitesService
var sInvitesOnce *sync.Once = new(sync.Once)

func GetInvitesService(db *sql.DB) *invitesService {
	sInvitesOnce.Do(func() {
		sInvitesInstance = &invitesService{db: db}
	})
	return sInvitesInstance
}

func failed(key string) *shared.ErrorResponse {
	return &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: key}
}

var errBookingNotFound = &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.booking_not_found"}
var errInviteNotFound = &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.invite_not_found"}

// --- owner -----------------------------------------------------------------

func (s *invitesService) GetInvite(ctx context.Context, ownerId uuid.UUID, bookingId uuid.UUID) (*InviteDto, *shared.ErrorResponse) {
	var queries = model.New(s.db)
	var booking, err = queries.GetOwnedBookingForInvite(ctx, model.GetOwnedBookingForInviteParams{ID: bookingId, OwnerID: uuid.NullUUID{UUID: ownerId, Valid: true}})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errBookingNotFound
	}
	if err != nil {
		return nil, failed("errors.invite_load_failed")
	}
	var rows []model.ListParticipantsByBookingRow
	rows, err = queries.ListParticipantsByBooking(ctx, bookingId)
	if err != nil {
		return nil, failed("errors.invite_load_failed")
	}

	var invite = &InviteDto{
		Capacity:     int(booking.Capacity.Int32),
		SpotsLeft:    spotsLeft(booking.Capacity, int64(len(rows))),
		State:        inviteState(booking.Status, booking.StartsAt, booking.Capacity, int64(len(rows)), time.Now()),
		Participants: mapParticipants(rows),
	}
	if booking.InviteToken.Valid {
		invite.Token = booking.InviteToken.UUID.String()
	}
	return invite, nil
}

// ShareInvite creates the invite link, or replaces it so the old one stops working.
func (s *invitesService) ShareInvite(ctx context.Context, ownerId uuid.UUID, bookingId uuid.UUID) (*InviteDto, *shared.ErrorResponse) {
	var invite, err = s.GetInvite(ctx, ownerId, bookingId)
	if err != nil {
		return nil, err
	}
	if invite.State == StateClosed {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.invite_closed"}
	}
	var _, setErr = model.New(s.db).SetBookingInviteToken(ctx, model.SetBookingInviteTokenParams{ID: bookingId, OwnerID: uuid.NullUUID{UUID: ownerId, Valid: true}})
	if setErr != nil {
		return nil, failed("errors.invite_failed")
	}
	return s.GetInvite(ctx, ownerId, bookingId)
}

func (s *invitesService) RemoveParticipant(ctx context.Context, ownerId uuid.UUID, bookingId uuid.UUID, participantId uuid.UUID) *shared.ErrorResponse {
	var removed, err = model.New(s.db).RemoveParticipant(ctx, model.RemoveParticipantParams{
		ID:        participantId,
		BookingID: bookingId,
		OwnerID:   uuid.NullUUID{UUID: ownerId, Valid: true},
	})
	if err != nil {
		return failed("errors.invite_failed")
	}
	if removed == 0 {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.participant_not_found"}
	}
	return nil
}

// --- invite page -----------------------------------------------------------

func (s *invitesService) GetPublicInvite(ctx context.Context, token uuid.UUID) (*PublicInviteDto, *shared.ErrorResponse) {
	var row, err = model.New(s.db).GetInviteByToken(ctx, uuid.NullUUID{UUID: token, Valid: true})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errInviteNotFound
	}
	if err != nil {
		return nil, failed("errors.invite_load_failed")
	}
	var invite = &PublicInviteDto{
		FacilityName: row.FacilityName,
		FacilitySlug: row.FacilitySlug,
		TenantSlug:   row.TenantSlug,
		TenantName:   row.TenantName,
		Timezone:     row.Timezone,
		OwnerName:    firstName(row.OwnerName.String),
		StartsAt:     row.StartsAt,
		EndsAt:       row.EndsAt,
		Capacity:     int(row.Capacity.Int32),
		SpotsLeft:    spotsLeft(row.Capacity, row.ParticipantCount),
		State:        inviteState(row.Status, row.StartsAt, row.Capacity, row.ParticipantCount, time.Now()),
		Facility:     bookings.FacilityDetails(row.Facility, row.FacilitySport),
	}
	return invite, nil
}

// JoinAsUser adds a signed-in person. It does not count towards their own booking limit.
func (s *invitesService) JoinAsUser(ctx context.Context, userId uuid.UUID, token uuid.UUID) *shared.ErrorResponse {
	return s.join(ctx, lockByToken(token), addUser(ctx, userId))
}

// JoinPublic adds a signed-in person to a public booking, found in «For you»:
// the same rules as the invite link, without one.
func (s *invitesService) JoinPublic(ctx context.Context, userId uuid.UUID, bookingId uuid.UUID) *shared.ErrorResponse {
	var err = s.join(ctx, lockPublic(bookingId), addUser(ctx, userId))
	if err == errInviteNotFound { // not public (any more), or gone
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.public_booking_not_found"}
	}
	return err
}

// SetPublic: the owner makes an upcoming booking public, or private again.
// Only a facility that takes invites (room for more than the owner) can have public bookings.
func (s *invitesService) SetPublic(ctx context.Context, ownerId uuid.UUID, bookingId uuid.UUID, public bool) *shared.ErrorResponse {
	var queries = model.New(s.db)
	var owner = uuid.NullUUID{UUID: ownerId, Valid: true}
	var booking, err = queries.GetOwnedBookingForInvite(ctx, model.GetOwnedBookingForInviteParams{ID: bookingId, OwnerID: owner})
	if errors.Is(err, sql.ErrNoRows) {
		return errBookingNotFound
	}
	if err != nil {
		return failed("errors.invite_failed")
	}
	if public && !invitable(booking.Capacity) {
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.public_needs_capacity"}
	}
	var updated int64
	if updated, err = queries.SetBookingPublic(ctx, model.SetBookingPublicParams{ID: bookingId, OwnerID: owner, IsPublic: public}); err != nil {
		return failed("errors.invite_failed")
	}
	if updated == 0 {
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.invite_closed"}
	}
	return nil
}

// lock finds the booking to join and locks its row; sql.ErrNoRows when there is none.
type lock func(context.Context, *model.Queries) (model.LockBookingByInviteTokenRow, error)

func lockByToken(token uuid.UUID) lock {
	return func(ctx context.Context, queries *model.Queries) (model.LockBookingByInviteTokenRow, error) {
		return queries.LockBookingByInviteToken(ctx, uuid.NullUUID{UUID: token, Valid: true})
	}
}

func lockPublic(bookingId uuid.UUID) lock {
	return func(ctx context.Context, queries *model.Queries) (model.LockBookingByInviteTokenRow, error) {
		var row, err = queries.LockPublicBooking(ctx, bookingId)
		return model.LockBookingByInviteTokenRow(row), err
	}
}

// addUser: a signed-in person joins, as themselves.
func addUser(ctx context.Context, userId uuid.UUID) func(*model.Queries, model.LockBookingByInviteTokenRow, time.Time) *shared.ErrorResponse {
	return func(queries *model.Queries, booking model.LockBookingByInviteTokenRow, day time.Time) *shared.ErrorResponse {
		if booking.OwnerID.Valid && booking.OwnerID.UUID == userId {
			return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.invite_own_booking"}
		}
		// one place at a time: same rule as booking (bookings.insertBooking)
		if err := queries.LockUserBookings(ctx, userId); err != nil {
			return failed("errors.invite_failed")
		}
		var overlaps, overlapErr = queries.UserHasOverlappingBooking(ctx, model.UserHasOverlappingBookingParams{
			UserID:   uuid.NullUUID{UUID: userId, Valid: true},
			StartsAt: booking.StartsAt,
			EndsAt:   booking.EndsAt,
		})
		if overlapErr != nil {
			return failed("errors.invite_failed")
		}
		if overlaps {
			return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.booking_overlaps"}
		}
		if booking.MinAge.Valid {
			var birth, err = queries.GetUserBirthDate(ctx, userId)
			if err != nil || !birth.Valid {
				return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.profile_incomplete"}
			}
			if bookings.AgeOn(birth.Time, day) < int(booking.MinAge.Int32) {
				return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.age_restricted"}
			}
		}
		var _, err = queries.AddUserParticipant(ctx, model.AddUserParticipantParams{BookingID: booking.ID, UserID: uuid.NullUUID{UUID: userId, Valid: true}})
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" { // unique_violation: uq_booking_participant_user
			return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.invite_already_joined"}
		}
		if err != nil {
			return failed("errors.invite_failed")
		}
		return nil
	}
}

// JoinAsGuest adds someone without an account: only a name and an age are kept.
func (s *invitesService) JoinAsGuest(ctx context.Context, token uuid.UUID, guest GuestJoinDto) *shared.ErrorResponse {
	return s.join(ctx, lockByToken(token), func(queries *model.Queries, booking model.LockBookingByInviteTokenRow, _ time.Time) *shared.ErrorResponse {
		if booking.MinAge.Valid && guest.Age < int(booking.MinAge.Int32) {
			return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.guest_age_restricted"}
		}
		var _, err = queries.AddGuestParticipant(ctx, model.AddGuestParticipantParams{
			BookingID: booking.ID,
			GuestName: sql.NullString{String: guest.Name, Valid: true},
			GuestAge:  sql.NullInt16{Int16: int16(guest.Age), Valid: true},
		})
		if err != nil {
			return failed("errors.invite_failed")
		}
		return nil
	})
}

// join locks the booking row, so two people cannot both take the last spot.
func (s *invitesService) join(ctx context.Context, find lock, add func(*model.Queries, model.LockBookingByInviteTokenRow, time.Time) *shared.ErrorResponse) *shared.ErrorResponse {
	var tx, err = s.db.BeginTx(ctx, nil)
	if err != nil {
		return failed("errors.invite_failed")
	}
	defer tx.Rollback()
	var queries = model.New(tx)

	var booking model.LockBookingByInviteTokenRow
	booking, err = find(ctx, queries)
	if errors.Is(err, sql.ErrNoRows) {
		return errInviteNotFound
	}
	if err != nil {
		return failed("errors.invite_failed")
	}
	var participants int64
	participants, err = queries.CountParticipants(ctx, booking.ID)
	if err != nil {
		return failed("errors.invite_failed")
	}
	switch inviteState(booking.Status, booking.StartsAt, booking.Capacity, participants, time.Now()) {
	case StateClosed:
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.invite_closed"}
	case StateFull:
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.invite_full"}
	}

	// age limits apply on the day of the booking, in the council's time zone
	var loc, locErr = time.LoadLocation(booking.Timezone)
	if locErr != nil {
		return failed("errors.invite_failed")
	}
	var start = booking.StartsAt.In(loc)
	var day = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)

	if addErr := add(queries, booking, day); addErr != nil {
		return addErr
	}
	if err = tx.Commit(); err != nil {
		return failed("errors.invite_failed")
	}
	return nil
}

// --- participant -----------------------------------------------------------

func (s *invitesService) Leave(ctx context.Context, userId uuid.UUID, bookingId uuid.UUID) *shared.ErrorResponse {
	var queries = model.New(s.db)
	var user = uuid.NullUUID{UUID: userId, Valid: true}
	var booking, err = queries.GetParticipation(ctx, model.GetParticipationParams{BookingID: bookingId, UserID: user})
	if errors.Is(err, sql.ErrNoRows) {
		return errBookingNotFound
	}
	if err != nil {
		return failed("errors.leave_failed")
	}
	if booking.Status == model.BookingStatusConfirmed && !time.Now().Before(booking.StartsAt) {
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.leave_too_late"}
	}
	if _, err = queries.LeaveBooking(ctx, model.LeaveBookingParams{BookingID: bookingId, UserID: user}); err != nil {
		return failed("errors.leave_failed")
	}
	return nil
}
