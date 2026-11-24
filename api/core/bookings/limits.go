package bookings

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/model"
	"librequenda/schedule"
	"librequenda/shared"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// The booking window's limits per person, for a resident booking from the app,
// at the door or on the voice line: bookings per day and per week, turns in a row
// and the no-show block. Staff booking for someone skip them. The cap on upcoming
// bookings is checked in book.

// Booker: whom a booking is for and counts against: an account, or on the voice
// line a caller without one, by their name and number (E.164). The rules of an
// account apply to the number: its bookings are the ones with that holder_phone.
type Booker struct {
	User  uuid.UUID // uuid.Nil: a caller without an account
	Name  string
	Phone string
}

func Account(userId uuid.UUID) Booker { return Booker{User: userId} }

func (b Booker) account() bool { return b.User != uuid.Nil }

func (b Booker) user() uuid.NullUUID { return uuid.NullUUID{UUID: b.User, Valid: b.account()} }

// phone: what the queries match holder_phone with, only without an account.
func (b Booker) phone() sql.NullString {
	return sql.NullString{String: b.Phone, Valid: !b.account() && b.Phone != ""}
}

// personalLimits refuses the turn when it would break one of them; exclude is
// the booking being changed, which does not count against its own new time.
func (s *bookingsService) personalLimits(ctx context.Context, queries *model.Queries, who Booker, facility model.GetFacilityForBookingRow, slot *schedule.Slot, exclude uuid.NullUUID) *shared.ErrorResponse {
	var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.booking_failed"}
	var user, phone = who.user(), who.phone()
	var loc, err = time.LoadLocation(facility.Timezone)
	if err != nil {
		return failed
	}

	if facility.NoShowPenalty {
		var since = time.Now().AddDate(0, 0, -int(facility.NoShowWindowDays))
		var noShows, listErr = queries.ListUserNoShows(ctx, model.ListUserNoShowsParams{UserID: user, HolderPhone: phone, FacilityID: facility.ID, Since: since})
		if listErr != nil {
			return failed
		}
		if until, blocked := noShowBlock(noShows, int(facility.NoShowLimit), int(facility.NoShowBlockDays), time.Now()); blocked {
			return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.no_show_blocked", Args: []any{until.In(loc).Format("02/01/2006")}}
		}
	}

	if facility.MaxBookingsPerDay > 0 {
		var day = startOfDay(slot.Start, loc)
		var count, countErr = queries.CountUserBookingsInRange(ctx, model.CountUserBookingsInRangeParams{
			UserID: user, HolderPhone: phone, FacilityID: facility.ID, RangeStart: day, RangeEnd: day.AddDate(0, 0, 1), Exclude: exclude,
		})
		if countErr != nil {
			return failed
		}
		if count >= int64(facility.MaxBookingsPerDay) {
			return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.daily_limit_reached"}
		}
	}

	if facility.MaxBookingsPerWeek > 0 {
		var day = startOfDay(slot.Start, loc)
		var monday = day.AddDate(0, 0, 1-int(schedule.ISOWeekday(day))) // the ISO week, in the council's zone
		var count, countErr = queries.CountUserBookingsInRange(ctx, model.CountUserBookingsInRangeParams{
			UserID: user, HolderPhone: phone, FacilityID: facility.ID, RangeStart: monday, RangeEnd: monday.AddDate(0, 0, 7), Exclude: exclude,
		})
		if countErr != nil {
			return failed
		}
		if count >= int64(facility.MaxBookingsPerWeek) {
			return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.weekly_limit_reached"}
		}
	}

	if facility.MaxTurnsInARow > 0 {
		var reach = time.Duration(facility.MaxTurnsInARow) * slot.End.Sub(slot.Start)
		var near, nearErr = queries.ListUserBookingsNear(ctx, model.ListUserBookingsNearParams{
			UserID: user, HolderPhone: phone, FacilityID: facility.ID, RangeStart: slot.Start.Add(-reach), RangeEnd: slot.End.Add(reach), Exclude: exclude,
		})
		if nearErr != nil {
			return failed
		}
		if turnsInARow(near, *slot) > int(facility.MaxTurnsInARow) {
			return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.turns_in_a_row_limit"}
		}
	}
	return nil
}

// turnsInARow: how many back-to-back turns the person would have with this one.
func turnsInARow(theirs []model.ListUserBookingsNearRow, slot schedule.Slot) int {
	var count = 1
	for start := slot.Start; ; count++ {
		var found = false
		for _, b := range theirs {
			if b.EndsAt.Equal(start) {
				start, found = b.StartsAt, true
				break
			}
		}
		if !found {
			break
		}
	}
	for end := slot.End; ; count++ {
		var found = false
		for _, b := range theirs {
			if b.StartsAt.Equal(end) {
				end, found = b.EndsAt, true
				break
			}
		}
		if !found {
			break
		}
	}
	return count
}

// noShowBlock: with `limit` no-shows in the window counted (latest first), the
// person cannot book until blockDays after the latest of them.
func noShowBlock(noShows []time.Time, limit, blockDays int, now time.Time) (time.Time, bool) {
	if len(noShows) < limit {
		return time.Time{}, false
	}
	var until = noShows[0].AddDate(0, 0, blockDays)
	return until, now.Before(until)
}

// councilQueries: what CouncilLimits reads (*model.Queries, or a fake in tests).
type councilQueries interface {
	GetCouncilBookingPolicy(ctx context.Context, tenantID uuid.UUID) (model.GetCouncilBookingPolicyRow, error)
	GetUserBookingInOtherCouncil(ctx context.Context, arg model.GetUserBookingInOtherCouncilParams) (string, error)
	CountUserUpcomingInCouncil(ctx context.Context, arg model.CountUserUpcomingInCouncilParams) (int64, error)
}

// CouncilLimits: the rules beyond one facility, checked inside the transaction
// that locks the person's bookings (LockUserBookings), so two requests at once
// cannot both pass:
//   - one council at a time (a platform setting, on by default): while they hold
//     an upcoming booking of their own in another council, they cannot book here.
//     Joining someone's game is not booking: it neither counts nor is refused by
//     this rule (only by overlapping their own times, checked before);
//   - own: the council's cap on their own upcoming bookings across its facilities.
//     Joining someone's booking does not count towards it, like the facility's cap.
//
// Staff booking for someone skip both, like the other per-person limits.
func CouncilLimits(ctx context.Context, queries councilQueries, userId, tenantId uuid.UUID, own bool) *shared.ErrorResponse {
	return councilLimits(ctx, queries, Account(userId), tenantId, own)
}

func councilLimits(ctx context.Context, queries councilQueries, who Booker, tenantId uuid.UUID, own bool) *shared.ErrorResponse {
	var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.booking_failed"}
	var user, phone = who.user(), who.phone()
	var policy, err = queries.GetCouncilBookingPolicy(ctx, tenantId)
	if err != nil {
		return failed
	}
	if own && policy.OneCouncilPerAccount {
		var other, otherErr = queries.GetUserBookingInOtherCouncil(ctx, model.GetUserBookingInOtherCouncilParams{TenantID: tenantId, UserID: user, HolderPhone: phone})
		switch {
		case otherErr == nil:
			return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.other_council_booking", Args: []any{other}}
		case !errors.Is(otherErr, sql.ErrNoRows):
			return failed
		}
	}
	if own && policy.CouncilMaxActive > 0 {
		var count, countErr = queries.CountUserUpcomingInCouncil(ctx, model.CountUserUpcomingInCouncilParams{UserID: user, HolderPhone: phone, TenantID: tenantId})
		if countErr != nil {
			return failed
		}
		if count >= int64(policy.CouncilMaxActive) {
			return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.council_limit_reached", Args: []any{policy.CouncilMaxActive}}
		}
	}
	return nil
}
