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
