package bookings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"librequenda/checkin"
	"librequenda/model"
	"librequenda/schedule"
	"librequenda/shared"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type bookingsService struct {
	db *sql.DB
}

var sBookingsInstance *bookingsService
var sBookingsOnce *sync.Once = new(sync.Once)

func GetBookingsService(db *sql.DB) *bookingsService {
	sBookingsOnce.Do(func() {
		sBookingsInstance = &bookingsService{db: db}
	})
	return sBookingsInstance
}

func (s *bookingsService) GetMyBookings(ctx context.Context, userId uuid.UUID) (*MyBookingsResponse, *shared.ErrorResponse) {
	var loadFailed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.bookings_load_failed"}
	var queries = model.New(s.db)
	var user = uuid.NullUUID{UUID: userId, Valid: true}
	var rows, err = queries.ListBookingsByUser(ctx, user)
	if err != nil {
		return nil, loadFailed
	}
	var joined []model.ListParticipationsByUserRow
	joined, err = queries.ListParticipationsByUser(ctx, user)
	if err != nil {
		return nil, loadFailed
	}

	var ids = make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	var counts []model.CountParticipantsByBookingsRow
	counts, err = queries.CountParticipantsByBookings(ctx, ids)
	if err != nil {
		return nil, loadFailed
	}

	var allIds = ids
	for _, row := range joined {
		allIds = append(allIds, row.ID)
	}
	var joinedRows []model.ListPeopleByBookingsRow
	joinedRows, err = queries.ListPeopleByBookings(ctx, allIds)
	if err != nil {
		return nil, loadFailed
	}
	var spots = make(map[uuid.UUID]Spots, len(counts))
	for _, c := range counts {
		spots[c.ID] = Spots{Capacity: c.Capacity, Participants: c.ParticipantCount}
	}
	var joinedBy = make(map[uuid.UUID][]model.ListPeopleByBookingsRow)
	for _, row := range joinedRows {
		joinedBy[row.BookingID] = append(joinedBy[row.BookingID], row)
	}
	return MapMyBookings(rows, joined, spots, joinedBy, userId, time.Now()), nil
}

// CreateBooking accepts only a slot that the schedule reports as free right now,
// so the rules shown on the day view and the rules enforced here are the same code.
func (s *bookingsService) CreateBooking(ctx context.Context, userId uuid.UUID, body CreateBookingDto) (*BookingDto, *shared.ErrorResponse) {
	var queries = model.New(s.db)
	if complete, err := queries.HasBirthDate(ctx, userId); err != nil || !complete {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.profile_incomplete"}
	}
	var facility, err = queries.GetFacilityForBooking(ctx, body.FacilityId)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	}
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.booking_failed"}
	}

	var slot, slotErr = s.bookableSlot(ctx, queries, userId, facility, body.StartsAt)
	if slotErr != nil {
		return nil, slotErr
	}
	var booking, bookErr = s.book(ctx, queries, userId, facility, slot)
	if bookErr != nil || !body.Public {
		return booking, bookErr
	}
	// public: only where others can join (room for more than the owner); the booking stands either way
	if facility.Capacity.Valid && facility.Capacity.Int32 > 1 {
		var id = uuid.MustParse(booking.Id)
		if _, err := queries.SetBookingPublic(ctx, model.SetBookingPublicParams{ID: id, OwnerID: uuid.NullUUID{UUID: userId, Valid: true}, IsPublic: true}); err == nil {
			booking.IsPublic = true
		}
	}
	return booking, nil
}

// book creates the booking of a turn already found free, within the person's
// limit at the facility, and sends the confirmation.
func (s *bookingsService) book(ctx context.Context, queries *model.Queries, userId uuid.UUID, facility model.GetFacilityForBookingRow, slot *schedule.Slot) (*BookingDto, *shared.ErrorResponse) {
	var active, err = queries.CountActiveBookingsByUserFacility(ctx, model.CountActiveBookingsByUserFacilityParams{
		UserID:     uuid.NullUUID{UUID: userId, Valid: true},
		FacilityID: facility.ID,
	})
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.booking_failed"}
	}
	if active >= int64(facility.MaxActiveBookingsPerUser) {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.booking_limit_reached"}
	}
	if limitErr := s.personalLimits(ctx, queries, userId, facility, slot, uuid.NullUUID{}); limitErr != nil {
		return nil, limitErr
	}

	var booking model.Booking
	var overlapErr *shared.ErrorResponse
	booking, overlapErr, err = s.insertBooking(ctx, userId, facility, slot)
	if overlapErr != nil {
		return nil, overlapErr
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23P01" { // exclusion_violation: booked in the meantime
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.slot_booked"}
	}
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.booking_failed"}
	}

	s.notify(ctx, queries, userId, booking.ID, model.NotificationTypeBookingConfirmed, facility.Name, booking.StartsAt, booking.EndsAt)

	var cancelUntil = booking.StartsAt.Add(-time.Duration(facility.MinCancelHours) * time.Hour)
	return &BookingDto{
		Id:           booking.ID.String(),
		StartsAt:     booking.StartsAt,
		EndsAt:       booking.EndsAt,
		Status:       string(booking.Status),
		FacilityId:   facility.ID.String(),
		FacilityName: facility.Name,
		CanCancel:    time.Now().Before(cancelUntil),
		CanChange:    time.Now().Before(cancelUntil),
		CancelUntil:  cancelUntil,
		CheckinCode:  checkin.Payload(booking.CheckinToken),
		Role:         RoleOwner,
	}, nil
}

// insertBooking creates the booking unless the person is already somewhere else
// at that time, in any facility or council, as owner or invited (UserHasOverlappingBooking),
// or it breaks the rules beyond the facility (CouncilLimits).
func (s *bookingsService) insertBooking(ctx context.Context, userId uuid.UUID, facility model.GetFacilityForBookingRow, slot *schedule.Slot) (model.Booking, *shared.ErrorResponse, error) {
	var facilityId = facility.ID
	var tx, err = s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Booking{}, nil, err
	}
	defer tx.Rollback()
	var queries = model.New(tx)
	var user = uuid.NullUUID{UUID: userId, Valid: true}

	if err = queries.LockUserBookings(ctx, userId); err != nil {
		return model.Booking{}, nil, err
	}
	var overlaps bool
	overlaps, err = queries.UserHasOverlappingBooking(ctx, model.UserHasOverlappingBookingParams{UserID: user, StartsAt: slot.Start, EndsAt: slot.End})
	if err != nil {
		return model.Booking{}, nil, err
	}
	if overlaps {
		return model.Booking{}, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.booking_overlaps"}, nil
	}
	if limitErr := CouncilLimits(ctx, queries, userId, facility.TenantID, true); limitErr != nil {
		return model.Booking{}, limitErr, nil
	}

	var booking model.Booking
	booking, err = queries.CreateBooking(ctx, model.CreateBookingParams{FacilityID: facilityId, UserID: user, StartsAt: slot.Start, EndsAt: slot.End})
	if err != nil {
		return model.Booking{}, nil, err
	}
	return booking, nil, tx.Commit()
}

// bookableSlot: the turn starting at startsAt, if it is free right now and the
// person is old enough on that day (in the council's time zone).
func (s *bookingsService) bookableSlot(ctx context.Context, queries *model.Queries, userId uuid.UUID, facility model.GetFacilityForBookingRow, startsAt time.Time) (*schedule.Slot, *shared.ErrorResponse) {
	var loc, locErr = time.LoadLocation(facility.Timezone)
	if locErr != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.booking_failed"}
	}
	var start = startsAt.In(loc)
	var day = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)

	if ageErr := checkAge(ctx, queries, userId, facility, day); ageErr != nil {
		return nil, ageErr
	}
	return s.findSlot(ctx, queries, facility, loc, day, day.AddDate(0, 0, 1), start, true)
}

// checkAge: the person is old enough for the facility on that day, when it has a minimum age.
func checkAge(ctx context.Context, queries *model.Queries, userId uuid.UUID, facility model.GetFacilityForBookingRow, day time.Time) *shared.ErrorResponse {
	if !facility.MinAge.Valid {
		return nil
	}
	var birth, err = queries.GetUserBirthDate(ctx, userId)
	if err != nil || !birth.Valid {
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.profile_incomplete"}
	}
	if AgeOn(birth.Time, day) < int(facility.MinAge.Int32) {
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.age_restricted"}
	}
	return nil
}

func (s *bookingsService) findSlot(ctx context.Context, queries *model.Queries, facility model.GetFacilityForBookingRow, loc *time.Location, day, dayEnd, start time.Time, resident bool) (*schedule.Slot, *shared.ErrorResponse) {
	var slots, err = s.daySlots(ctx, queries, facility, loc, day, dayEnd, resident)
	if err != nil {
		return nil, err
	}
	for _, slot := range slots {
		if !slot.Start.Equal(start) {
			continue
		}
		if slot.Status != schedule.SlotFree {
			return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.slot_" + string(slot.Status)}
		}
		return &slot, nil
	}
	return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.slot_not_found"}
}

// daySlots: the facility's turns on one day, as the schedule reports them right
// now. resident: booked from the app, so the window's minimum advance applies;
// staff and the door book up to the start.
func (s *bookingsService) daySlots(ctx context.Context, queries *model.Queries, facility model.GetFacilityForBookingRow, loc *time.Location, day, dayEnd time.Time, resident bool) ([]schedule.Slot, *shared.ErrorResponse) {
	var inputs, err = s.dayInputs(ctx, queries, facility, day, dayEnd)
	if err != nil {
		return nil, err
	}
	return schedule.DaySlots(day, loc, rulesOf(facility, resident), inputs.windows, inputs.closures, inputs.busy, time.Now()), nil
}

// rulesOf: the facility's turns and booking window for the schedule.
func rulesOf(facility model.GetFacilityForBookingRow, resident bool) schedule.Rules {
	var rules = schedule.Rules{SlotMinutes: int(facility.SlotMinutes), MaxAdvance: time.Duration(facility.MaxAdvanceHours) * time.Hour}
	if facility.WeeklyOpening {
		rules.Weekly = &schedule.WeeklyOpening{Weekday: int(facility.OpensWeekday), Minute: int(facility.OpensMinute)}
	}
	if resident {
		rules.MinAdvance = time.Duration(facility.MinAdvanceMinutes) * time.Minute
	}
	return rules
}

// scheduleDay: what the schedule of one day is cut from.
type scheduleDay struct {
	windows  []schedule.Window
	closures []schedule.Interval
	busy     []schedule.Interval
}

// dayInputs loads the facility's opening hours, closures and confirmed bookings of one day.
func (s *bookingsService) dayInputs(ctx context.Context, queries *model.Queries, facility model.GetFacilityForBookingRow, day, dayEnd time.Time) (*scheduleDay, *shared.ErrorResponse) {
	var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.booking_failed"}

	var hours, err = queries.ListOpeningHoursByTenantWeekday(ctx, model.ListOpeningHoursByTenantWeekdayParams{TenantID: facility.TenantID, Weekday: schedule.ISOWeekday(day)})
	if err != nil {
		return nil, failed
	}
	var windows []schedule.Window
	for _, h := range hours {
		if h.FacilityID == facility.ID {
			windows = append(windows, schedule.Window{Opens: h.OpensAt, Closes: h.ClosesAt})
		}
	}

	var closures []model.ListClosuresByTenantRangeRow
	closures, err = queries.ListClosuresByTenantRange(ctx, model.ListClosuresByTenantRangeParams{TenantID: facility.TenantID, RangeStart: day, RangeEnd: dayEnd})
	if err != nil {
		return nil, failed
	}
	var closureIntervals []schedule.Interval
	for _, c := range closures {
		if c.FacilityID == facility.ID {
			closureIntervals = append(closureIntervals, schedule.Interval{Start: c.StartsAt, End: c.EndsAt})
		}
	}

	var bookings []model.ListConfirmedBookingsInRangeRow
	bookings, err = queries.ListConfirmedBookingsInRange(ctx, model.ListConfirmedBookingsInRangeParams{FacilityID: facility.ID, RangeStart: day, RangeEnd: dayEnd})
	if err != nil {
		return nil, failed
	}
	var busy []schedule.Interval
	for _, b := range bookings {
		busy = append(busy, schedule.Interval{Start: b.StartsAt, End: b.EndsAt})
	}

	return &scheduleDay{windows: windows, closures: closureIntervals, busy: busy}, nil
}

// QuickBook is the flash booking of someone scanning their personal QR at the
// facility's reader (FlashTurn picks the turn): the rest of a turn under way,
// from now until it ends, or the turn about to start. The profile, age, limit
// and overlap rules are the same as in the app; scanning counts as accepting the
// usage rules.
func (s *bookingsService) QuickBook(ctx context.Context, userId uuid.UUID, facilityId uuid.UUID) (*BookingDto, *shared.ErrorResponse) {
	var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.booking_failed"}
	var queries = model.New(s.db)
	if complete, err := queries.HasBirthDate(ctx, userId); err != nil || !complete {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.profile_incomplete"}
	}
	var facility, err = queries.GetFacilityForBooking(ctx, facilityId)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	}
	if err != nil {
		return nil, failed
	}
	var loc, locErr = time.LoadLocation(facility.Timezone)
	if locErr != nil {
		return nil, failed
	}

	var now = time.Now()
	var today = startOfDay(now, loc)
	var inputs, inputsErr = s.dayInputs(ctx, queries, facility, today, today.AddDate(0, 0, 1))
	if inputsErr != nil {
		return nil, inputsErr
	}
	var current, inProgress = schedule.Current(today, loc, int(facility.SlotMinutes), inputs.windows, inputs.closures, inputs.busy, now)
	var slots = schedule.DaySlots(today, loc, rulesOf(facility, false), inputs.windows, inputs.closures, inputs.busy, now)
	// near midnight the turn may start on the next day
	if tomorrow := startOfDay(now.Add(checkin.EarlyWindow), loc); tomorrow.After(today) {
		var more []schedule.Slot
		if more, inputsErr = s.daySlots(ctx, queries, facility, loc, tomorrow, tomorrow.AddDate(0, 0, 1), false); inputsErr != nil {
			return nil, inputsErr
		}
		slots = append(slots, more...)
	}

	var slot, reason = FlashTurn(current, inProgress, slots, now)
	if reason != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors." + reason}
	}
	if ageErr := checkAge(ctx, queries, userId, facility, startOfDay(slot.Start, loc)); ageErr != nil {
		return nil, ageErr
	}
	return s.book(ctx, queries, userId, facility, &slot)
}

// FlashTurn picks what a personal QR books at `now`, given the rest of the turn
// under way (schedule.Current) and the day's turns:
//  1. the rest of the turn under way, when it is free and at least checkin.EarlyWindow is left;
//  2. otherwise the turn that starts within checkin.EarlyWindow, when it is free;
//  3. otherwise whatever is left of the turn under way, when it is free.
//
// With nothing to book it returns the reason, e.g. "slot_booked" or "no_turn_soon".
func FlashTurn(current schedule.Slot, inProgress bool, slots []schedule.Slot, now time.Time) (schedule.Slot, string) {
	var currentFree = inProgress && current.Status == schedule.SlotFree
	if currentFree && current.End.Sub(current.Start) >= checkin.EarlyWindow {
		return current, ""
	}
	var next, soon = NextTurn(slots, now)
	if soon && next.Status == schedule.SlotFree {
		return next, ""
	}
	switch {
	case currentFree:
		return current, ""
	case inProgress:
		return schedule.Slot{}, "slot_" + string(current.Status)
	case soon:
		return schedule.Slot{}, "slot_" + string(next.Status)
	}
	return schedule.Slot{}, "no_turn_soon"
}

// NextTurn: the first turn that starts from now until checkin.EarlyWindow later, free or not.
func NextTurn(slots []schedule.Slot, now time.Time) (schedule.Slot, bool) {
	var latest = now.Add(checkin.EarlyWindow)
	for _, slot := range slots {
		if !slot.Start.Before(now) && !slot.Start.After(latest) {
			return slot, true
		}
	}
	return schedule.Slot{}, false
}

func startOfDay(t time.Time, loc *time.Location) time.Time {
	var local = t.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}

// ChangeBooking moves the owner's booking to another free turn of the same facility,
// on any day it can be booked. It is allowed while the booking could still be
// cancelled; the people who joined, the QR and the invite link stay with it.
func (s *bookingsService) ChangeBooking(ctx context.Context, userId uuid.UUID, bookingId uuid.UUID, body ChangeBookingDto) (*BookingDto, *shared.ErrorResponse) {
	var changeFailed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.change_failed"}
	var queries = model.New(s.db)
	var owner = uuid.NullUUID{UUID: userId, Valid: true}
	var current, err = queries.GetBookingForUser(ctx, model.GetBookingForUserParams{ID: bookingId, UserID: owner})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.booking_not_found"}
	}
	if err != nil {
		return nil, changeFailed
	}
	if changeErr := changeable(current.Status, current.StartsAt, current.MinCancelHours, time.Now()); changeErr != nil {
		return nil, changeErr
	}
	if body.StartsAt.Equal(current.StartsAt) {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.booking_same_time"}
	}

	var facility model.GetFacilityForBookingRow
	facility, err = queries.GetFacilityForBooking(ctx, current.FacilityID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	}
	if err != nil {
		return nil, changeFailed
	}
	var slot, slotErr = s.bookableSlot(ctx, queries, userId, facility, body.StartsAt)
	if slotErr != nil {
		return nil, slotErr
	}
	if limitErr := s.personalLimits(ctx, queries, userId, facility, slot, uuid.NullUUID{UUID: bookingId, Valid: true}); limitErr != nil {
		return nil, limitErr
	}

	var booking model.Booking
	var conflict *shared.ErrorResponse
	booking, conflict, err = s.moveBooking(ctx, userId, bookingId, slot)
	if conflict != nil {
		return nil, conflict
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23P01" { // exclusion_violation: booked in the meantime
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.slot_booked"}
	}
	if err != nil {
		return nil, changeFailed
	}

	s.notify(ctx, queries, userId, booking.ID, model.NotificationTypeBookingChanged, facility.Name, booking.StartsAt, booking.EndsAt)

	var cancelUntil = booking.StartsAt.Add(-time.Duration(facility.MinCancelHours) * time.Hour)
	var canCancel = time.Now().Before(cancelUntil)
	return &BookingDto{
		Id:           booking.ID.String(),
		StartsAt:     booking.StartsAt,
		EndsAt:       booking.EndsAt,
		Status:       string(booking.Status),
		FacilityId:   facility.ID.String(),
		FacilityName: facility.Name,
		CanCancel:    canCancel,
		CanChange:    canCancel,
		CancelUntil:  cancelUntil,
		CheckinCode:  checkin.Payload(booking.CheckinToken),
		Role:         RoleOwner,
	}, nil
}

// changeable: a booking can be moved on the same terms it can be cancelled.
func changeable(status model.BookingStatus, startsAt time.Time, minCancelHours int32, now time.Time) *shared.ErrorResponse {
	if status != model.BookingStatusConfirmed {
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.booking_already_cancelled"}
	}
	if now.After(startsAt.Add(-time.Duration(minCancelHours) * time.Hour)) {
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.change_too_late"}
	}
	return nil
}

// moveBooking changes the time unless the owner, or someone who joined with an
// account, is already somewhere else then. Locks: the booking row first (as a
// join does), then the people, so a join and a change cannot cross.
func (s *bookingsService) moveBooking(ctx context.Context, userId uuid.UUID, bookingId uuid.UUID, slot *schedule.Slot) (model.Booking, *shared.ErrorResponse, error) {
	var tx, err = s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Booking{}, nil, err
	}
	defer tx.Rollback()
	var queries = model.New(tx)
	var owner = uuid.NullUUID{UUID: userId, Valid: true}

	var locked model.LockOwnedBookingRow
	locked, err = queries.LockOwnedBooking(ctx, model.LockOwnedBookingParams{ID: bookingId, UserID: owner})
	if errors.Is(err, sql.ErrNoRows) {
		return model.Booking{}, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.booking_not_found"}, nil
	}
	if err != nil {
		return model.Booking{}, nil, err
	}
	// cancelled or moved in the meantime
	if changeErr := changeable(locked.Status, locked.StartsAt, locked.MinCancelHours, time.Now()); changeErr != nil {
		return model.Booking{}, changeErr, nil
	}

	if err = queries.LockUserBookings(ctx, userId); err != nil {
		return model.Booking{}, nil, err
	}
	var overlaps bool
	overlaps, err = queries.UserHasOverlappingBooking(ctx, model.UserHasOverlappingBookingParams{
		UserID:          owner,
		StartsAt:        slot.Start,
		EndsAt:          slot.End,
		ExceptBookingID: uuid.NullUUID{UUID: bookingId, Valid: true},
	})
	if err != nil {
		return model.Booking{}, nil, err
	}
	if overlaps {
		return model.Booking{}, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.booking_overlaps"}, nil
	}

	if err = queries.LockParticipantBookings(ctx, bookingId); err != nil {
		return model.Booking{}, nil, err
	}
	overlaps, err = queries.ParticipantsHaveOverlappingBooking(ctx, model.ParticipantsHaveOverlappingBookingParams{BookingID: bookingId, StartsAt: slot.Start, EndsAt: slot.End})
	if err != nil {
		return model.Booking{}, nil, err
	}
	if overlaps {
		return model.Booking{}, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.participant_overlaps"}, nil
	}

	var booking model.Booking
	booking, err = queries.ChangeBookingTime(ctx, model.ChangeBookingTimeParams{ID: bookingId, UserID: owner, StartsAt: slot.Start, EndsAt: slot.End})
	if err != nil {
		return model.Booking{}, nil, err
	}
	return booking, nil, tx.Commit()
}

func (s *bookingsService) CancelBooking(ctx context.Context, userId uuid.UUID, bookingId uuid.UUID) *shared.ErrorResponse {
	var queries = model.New(s.db)
	var owner = uuid.NullUUID{UUID: userId, Valid: true}
	var booking, err = queries.GetBookingForUser(ctx, model.GetBookingForUserParams{ID: bookingId, UserID: owner})
	if errors.Is(err, sql.ErrNoRows) {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.booking_not_found"}
	}
	if err != nil {
		return &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.cancel_failed"}
	}
	if booking.Status != model.BookingStatusConfirmed {
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.booking_already_cancelled"}
	}
	if time.Now().After(booking.StartsAt.Add(-time.Duration(booking.MinCancelHours) * time.Hour)) {
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.cancel_too_late"}
	}

	var cancelled model.Booking
	cancelled, err = queries.CancelBooking(ctx, model.CancelBookingParams{ID: bookingId, UserID: owner})
	if err != nil {
		return &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.cancel_failed"}
	}

	var facility, facilityErr = queries.GetFacilityForBooking(ctx, cancelled.FacilityID)
	var facilityName string
	if facilityErr == nil {
		facilityName = facility.Name
	}
	s.notify(ctx, queries, userId, cancelled.ID, model.NotificationTypeBookingCancelled, facilityName, cancelled.StartsAt, cancelled.EndsAt)
	return nil
}

// notify records an "aviso"; a failure is logged and never undoes the booking.
func (s *bookingsService) notify(ctx context.Context, queries *model.Queries, userId uuid.UUID, bookingId uuid.UUID, kind model.NotificationType, facilityName string, startsAt, endsAt time.Time) {
	var err = queries.CreateNotification(ctx, model.CreateNotificationParams{
		UserID:    userId,
		BookingID: uuid.NullUUID{UUID: bookingId, Valid: true},
		Type:      kind,
		Payload:   NotificationPayload(facilityName, startsAt, endsAt),
	})
	if err != nil {
		shared.Logger.Error("failed to record notification %s for booking %s: %v", kind, bookingId, err)
	}
}

// NotificationPayload: what the notification list shows about the booking.
func NotificationPayload(facilityName string, startsAt, endsAt time.Time) json.RawMessage {
	var payload, _ = json.Marshal(map[string]any{
		"facility_name": facilityName,
		"starts_at":     startsAt,
		"ends_at":       endsAt,
	})
	return payload
}

// AgeOn is the age in whole years on a given day: someone is 16 from their 16th birthday on.
// A 29 February birthday counts as 1 March in other years.
func AgeOn(birth time.Time, day time.Time) int {
	var age = day.Year() - birth.Year()
	if day.Month() < birth.Month() || (day.Month() == birth.Month() && day.Day() < birth.Day()) {
		age--
	}
	return age
}
