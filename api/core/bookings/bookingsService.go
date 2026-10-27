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

	var loc, locErr = time.LoadLocation(facility.Timezone)
	if locErr != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.booking_failed"}
	}
	var start = body.StartsAt.In(loc)
	var day = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)

	if facility.MinAge.Valid {
		var birth, birthErr = queries.GetUserBirthDate(ctx, userId)
		if birthErr != nil || !birth.Valid {
			return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.profile_incomplete"}
		}
		if AgeOn(birth.Time, day) < int(facility.MinAge.Int32) {
			return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.age_restricted"}
		}
	}
	var dayEnd = day.AddDate(0, 0, 1)

	var slot, slotErr = s.findSlot(ctx, queries, facility, loc, day, dayEnd, start)
	if slotErr != nil {
		return nil, slotErr
	}

	var active int64
	active, err = queries.CountActiveBookingsByUserFacility(ctx, model.CountActiveBookingsByUserFacilityParams{
		UserID:     uuid.NullUUID{UUID: userId, Valid: true},
		FacilityID: facility.ID,
	})
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.booking_failed"}
	}
	if active >= int64(facility.MaxActiveBookingsPerUser) {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.booking_limit_reached"}
	}

	var booking model.Booking
	var overlapErr *shared.ErrorResponse
	booking, overlapErr, err = s.insertBooking(ctx, userId, facility.ID, slot)
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
		CancelUntil:  cancelUntil,
		CheckinCode:  checkin.Payload(booking.CheckinToken),
		Role:         RoleOwner,
	}, nil
}

// insertBooking creates the booking unless the person is already somewhere else
// at that time, in any facility or council, as owner or invited (UserHasOverlappingBooking).
func (s *bookingsService) insertBooking(ctx context.Context, userId uuid.UUID, facilityId uuid.UUID, slot *schedule.Slot) (model.Booking, *shared.ErrorResponse, error) {
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

	var booking model.Booking
	booking, err = queries.CreateBooking(ctx, model.CreateBookingParams{FacilityID: facilityId, UserID: user, StartsAt: slot.Start, EndsAt: slot.End})
	if err != nil {
		return model.Booking{}, nil, err
	}
	return booking, nil, tx.Commit()
}

func (s *bookingsService) findSlot(ctx context.Context, queries *model.Queries, facility model.GetFacilityForBookingRow, loc *time.Location, day, dayEnd, start time.Time) (*schedule.Slot, *shared.ErrorResponse) {
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

	var rules = schedule.Rules{SlotMinutes: int(facility.SlotMinutes), MaxAdvanceDays: int(facility.MaxAdvanceDays)}
	for _, slot := range schedule.DaySlots(day, loc, rules, windows, closureIntervals, busy, time.Now()) {
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
