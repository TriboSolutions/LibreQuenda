package invites

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"librequenda/branding"
	"librequenda/core/notice"
	"librequenda/core/rating"
	"librequenda/model"
	"librequenda/shared"
	"librequenda/validate"

	"github.com/google/uuid"
)

// StaffJoinInput: who council staff sign up for one of its events, at the desk or on
// the phone: an account (its email), or someone without one (a name and an age).
type StaffJoinInput struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Age   int    `json:"age"`
}

// Validate trims it and answers the i18n key of the first problem, or "".
func (in *StaffJoinInput) Validate() string {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Name = strings.Join(strings.Fields(in.Name), " ")
	switch {
	case (in.Email == "") == (in.Name == ""):
		return "errors.staff_booking_holder" // an account or a name, exactly one
	case in.Email != "" && !validate.Email(in.Email):
		return "errors.invalid_email"
	case in.Email != "":
		return ""
	}
	var guest = GuestJoinDto{Name: in.Name, Age: in.Age}
	if key := guest.Validate(); key != "" {
		return key
	}
	in.Name = guest.Name
	return ""
}

// StaffAddToEvent signs someone up for the council's event, with every rule of
// joining it themselves (spots, age, residents only and overlaps), but not the
// council's per-person limits, as when staff book for someone. An account is told it is in, in the app and by email.
func (s *invitesService) StaffAddToEvent(ctx context.Context, tenantId uuid.UUID, bookingId uuid.UUID, input StaffJoinInput) *shared.ErrorResponse {
	if key := input.Validate(); key != "" {
		return &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	// only this council's events: anything else is not found
	var event = func(ctx context.Context, queries *model.Queries) (model.LockBookingByInviteTokenRow, error) {
		var booking, err = lockPublic(bookingId)(ctx, queries)
		if err == nil && (booking.TenantID != tenantId || booking.Kind != model.BookingKindEvent) {
			return booking, sql.ErrNoRows
		}
		return booking, err
	}
	var err *shared.ErrorResponse
	if input.Email == "" {
		err = s.join(ctx, event, addGuest(ctx, GuestJoinDto{Name: input.Name, Age: input.Age}), joiner{guestName: input.Name})
	} else {
		var user, findErr = model.New(s.db).GetUserByEmailForStaff(ctx, input.Email)
		if errors.Is(findErr, sql.ErrNoRows) {
			return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.person_not_found", Args: []any{branding.Name(ctx, s.db)}}
		}
		if findErr != nil {
			return failed("errors.invite_failed")
		}
		var add = addAccount(ctx, user.ID, false)
		err = s.join(ctx, event, func(queries *model.Queries, booking model.LockBookingByInviteTokenRow, day time.Time) *shared.ErrorResponse {
			if addErr := add(queries, booking, day); addErr != nil {
				return addErr
			}
			// told they are in, as a booking confirmed (bell, app and email)
			var payload = notice.Payload(notice.Cancelled{FacilityName: booking.FacilityName, StartsAt: booking.StartsAt, EndsAt: booking.EndsAt}, nil)
			if tellErr := notice.Tell(ctx, queries, user.ID, booking.ID, model.NotificationTypeBookingConfirmed, payload); tellErr != nil {
				return failed("errors.invite_failed")
			}
			return nil
		}, joiner{userId: user.ID})
	}
	if err == errInviteNotFound {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.booking_not_found"}
	}
	return err
}

// EventPerson: someone in one of the council's events, as its staff see them.
type EventPerson struct {
	Id         string          `json:"id"`
	HasAccount bool            `json:"has_account"`
	Name       string          `json:"name"`        // the account's name and surnames, or the guest's name
	Email      string          `json:"email"`       // "" for a guest
	Age        *int16          `json:"age"`         // a guest's; null for an account
	Rating     *rating.Summary `json:"rating"`      // the stars others gave them; null until someone did
	IsResident bool            `json:"is_resident"` // this council verified them as its resident
	JoinedAt   time.Time       `json:"joined_at"`
}

type EventPeople struct {
	People []EventPerson `json:"people"`
}

// EventPeople: who is in one of the council's events (anything else is not found).
func (s *invitesService) EventPeople(ctx context.Context, tenantId uuid.UUID, bookingId uuid.UUID) (*EventPeople, *shared.ErrorResponse) {
	var queries = model.New(s.db)
	if _, err := queries.GetEvent(ctx, model.GetEventParams{ID: bookingId, TenantID: tenantId}); errors.Is(err, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.booking_not_found"}
	} else if err != nil {
		return nil, failed("errors.invite_failed")
	}
	var rows, err = queries.ListEventPeople(ctx, model.ListEventPeopleParams{TenantID: tenantId, BookingID: bookingId})
	if err != nil {
		return nil, failed("errors.invite_failed")
	}
	var out = &EventPeople{People: make([]EventPerson, 0, len(rows))}
	for _, row := range rows {
		var person = EventPerson{Id: row.ID.String(), HasAccount: row.UserID.Valid, IsResident: row.IsResident, JoinedAt: row.JoinedAt}
		if row.UserID.Valid {
			person.Name, person.Email = row.FullName, row.Email
			person.Rating = rating.Of(row.RatingAverage, row.RatingCount)
		} else {
			person.Name = row.GuestName
			if row.GuestAge.Valid {
				var age = row.GuestAge.Int16
				person.Age = &age
			}
		}
		out.People = append(out.People, person)
	}
	return out, nil
}

// StaffRemoveFromEvent: council staff take someone out of one of its events that has
// not ended. Someone with an account cannot join it again, and is told (in the app
// and by email) which event, where and when, and that the council did it.
func (s *invitesService) StaffRemoveFromEvent(ctx context.Context, tenantId uuid.UUID, bookingId uuid.UUID, participantId uuid.UUID) *shared.ErrorResponse {
	var tx, err = s.db.BeginTx(ctx, nil)
	if err != nil {
		return failed("errors.invite_failed")
	}
	defer tx.Rollback()
	var queries = model.New(tx)
	var removed uuid.NullUUID
	removed, err = queries.RemoveEventParticipant(ctx, model.RemoveEventParticipantParams{ID: participantId, BookingID: bookingId, TenantID: tenantId})
	if errors.Is(err, sql.ErrNoRows) {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.participant_not_found"}
	}
	if err != nil {
		return failed("errors.invite_failed")
	}
	if removed.Valid { // a guest has no account to keep out or to tell
		if err = queries.KeepOutOfBooking(ctx, model.KeepOutOfBookingParams{BookingID: bookingId, UserID: removed.UUID}); err != nil {
			return failed("errors.invite_failed")
		}
		var booking, bookingErr = queries.GetBookingForNotice(ctx, bookingId)
		if bookingErr != nil {
			return failed("errors.invite_failed")
		}
		// the event's title says it was an event; the council is who took them out
		var payload = notice.Payload(
			notice.Cancelled{FacilityID: booking.FacilityID, FacilityName: booking.FacilityName, TenantSlug: booking.TenantSlug, TenantName: booking.TenantName, Title: booking.Title, StartsAt: booking.StartsAt, EndsAt: booking.EndsAt},
			map[string]any{"name": booking.TenantName, "by_staff": true},
		)
		if err = notice.Tell(ctx, queries, removed.UUID, bookingId, model.NotificationTypeParticipantRemoved, payload); err != nil {
			return failed("errors.invite_failed")
		}
	}
	if err = tx.Commit(); err != nil {
		return failed("errors.invite_failed")
	}
	return nil
}
