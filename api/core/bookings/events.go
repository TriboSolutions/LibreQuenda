package bookings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Events: the council's own activities («Fiesta del baloncesto», «Torneo de
// fútbol 5v5»). An event is a booking with no owner, created by the council's
// administrators: any start and end (not the turns, nor the opening hours) and
// none of the per-person limits. It never takes the place of a resident's
// booking: when one is in the way, staff cancel it first (StaffCancel tells the
// person), then create the event. Residents join it like a public booking.

const (
	maxEventTitle       = 120
	maxEventDescription = 1000
	maxEventLength      = 7 * 24 * time.Hour // a typo must not block a facility for a year
	maxEventCapacity    = 10000
)

// EventInput: what staff fill in, the times in the council's time zone
// ("2026-11-15T10:00", like closures). Capacity nil: the facility's.
type EventInput struct {
	FacilityId  uuid.UUID `json:"facility_id"`
	Start       string    `json:"starts_at"`
	End         string    `json:"ends_at"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Capacity    *int32    `json:"capacity"`
	// read from Start and End by validate
	StartsAt time.Time `json:"-"`
	EndsAt   time.Time `json:"-"`
}

const localDateTime = "2006-01-02T15:04"

// validate trims the input, reads the times in loc and returns an error key, or "".
func (d *EventInput) validate(now time.Time, loc *time.Location) string {
	d.Title = strings.Join(strings.Fields(d.Title), " ")
	d.Description = strings.TrimSpace(d.Description)
	var startErr, endErr error
	d.StartsAt, startErr = time.ParseInLocation(localDateTime, d.Start, loc)
	d.EndsAt, endErr = time.ParseInLocation(localDateTime, d.End, loc)
	switch {
	case d.FacilityId == uuid.Nil || startErr != nil || endErr != nil:
		return "errors.invalid_request_body"
	case d.Title == "" || utf8.RuneCountInString(d.Title) > maxEventTitle:
		return "errors.event_title"
	case utf8.RuneCountInString(d.Description) > maxEventDescription:
		return "errors.event_description_too_long"
	case !d.EndsAt.After(d.StartsAt):
		return "errors.event_ends_before_start"
	case d.EndsAt.Sub(d.StartsAt) > maxEventLength:
		return "errors.event_too_long"
	case !d.StartsAt.After(now):
		return "errors.event_in_past"
	case d.Capacity != nil && (*d.Capacity < 1 || *d.Capacity > maxEventCapacity):
		return "errors.event_capacity"
	}
	return ""
}

func (d *EventInput) capacity() sql.NullInt32 {
	if d.Capacity == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: *d.Capacity, Valid: true}
}

// EventDto: an event as the "Eventos" tab lists it.
type EventDto struct {
	Id               string    `json:"id"`
	FacilityId       string    `json:"facility_id"`
	FacilityName     string    `json:"facility_name"`
	StartsAt         time.Time `json:"starts_at"`
	EndsAt           time.Time `json:"ends_at"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	OwnCapacity      *int32    `json:"own_capacity"` // null: the facility's
	Capacity         int       `json:"capacity"`     // who can join
	ParticipantCount int64     `json:"participant_count"`
	CanChange        bool      `json:"can_change"` // until it starts
	Timezone         string    `json:"timezone"`   // the council's, to show and edit the times in
}

func mapEvent(row model.GetEventRow, now time.Time) EventDto {
	var dto = EventDto{
		Id:               row.ID.String(),
		FacilityId:       row.FacilityID.String(),
		FacilityName:     row.FacilityName,
		StartsAt:         row.StartsAt,
		EndsAt:           row.EndsAt,
		Title:            row.Title.String,
		Description:      row.PublicDescription,
		Capacity:         Places(row.Capacity, model.BookingKindEvent),
		ParticipantCount: row.ParticipantCount,
		CanChange:        row.StartsAt.After(now),
		Timezone:         row.Timezone,
	}
	if row.EventCapacity.Valid {
		dto.OwnCapacity = &row.EventCapacity.Int32
	}
	return dto
}

type EventsResponse struct {
	Timezone string     `json:"timezone"` // the council's: new events are typed in it
	Events   []EventDto `json:"events"`   // not ended yet, soonest first
}

// InTheWayDto: what stops an event at that time: confirmed bookings, which staff
// can cancel from Reservas, and closures.
type InTheWayDto struct {
	Bookings []BookingInTheWayDto `json:"bookings"`
	Closures []ClosureInTheWayDto `json:"closures"`
}

type BookingInTheWayDto struct {
	Id       string    `json:"id"`
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
	Holder   string    `json:"holder"` // the person, or another event's title; "" once forgotten
	IsEvent  bool      `json:"is_event"`
}

type ClosureInTheWayDto struct {
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
	Reason   string    `json:"reason"`
}

func (d *InTheWayDto) empty() bool {
	return len(d.Bookings) == 0 && len(d.Closures) == 0
}

func (s *bookingsService) ListEvents(ctx context.Context, tenantId uuid.UUID) (*EventsResponse, *shared.ErrorResponse) {
	var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.events_load_failed"}
	var queries = model.New(s.db)
	var zone, err = queries.GetTenantTimezone(ctx, tenantId)
	if err != nil {
		return nil, failed
	}
	var rows []model.ListEventsByTenantRow
	if rows, err = queries.ListEventsByTenant(ctx, tenantId); err != nil {
		return nil, failed
	}
	var now = time.Now()
	var response = &EventsResponse{Timezone: zone, Events: make([]EventDto, 0, len(rows))}
	for _, row := range rows {
		response.Events = append(response.Events, mapEvent(model.GetEventRow(row), now))
	}
	return response, nil
}

// PreviewEvent: what is in the way of an event at that time, before staff create
// or move it. except: the event being moved.
func (s *bookingsService) PreviewEvent(ctx context.Context, tenantId uuid.UUID, input EventInput, except uuid.NullUUID) (*InTheWayDto, *shared.ErrorResponse) {
	var queries = model.New(s.db)
	var _, loc, facilityErr = s.councilFacility(ctx, queries, tenantId, input.FacilityId)
	if facilityErr != nil {
		return nil, facilityErr
	}
	if key := input.validate(time.Now(), loc); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	var inTheWay, err = inTheWay(ctx, queries, input.FacilityId, input.StartsAt, input.EndsAt, except)
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.events_load_failed"}
	}
	return inTheWay, nil
}

func inTheWay(ctx context.Context, queries *model.Queries, facilityId uuid.UUID, startsAt, endsAt time.Time, except uuid.NullUUID) (*InTheWayDto, error) {
	var bookings, err = queries.ListBookingsInTheWay(ctx, model.ListBookingsInTheWayParams{FacilityID: facilityId, StartsAt: startsAt, EndsAt: endsAt, ExceptID: except})
	if err != nil {
		return nil, err
	}
	var closures []model.ListClosuresInTheWayRow
	if closures, err = queries.ListClosuresInTheWay(ctx, model.ListClosuresInTheWayParams{FacilityID: facilityId, StartsAt: startsAt, EndsAt: endsAt}); err != nil {
		return nil, err
	}
	var out = &InTheWayDto{Bookings: make([]BookingInTheWayDto, 0, len(bookings)), Closures: make([]ClosureInTheWayDto, 0, len(closures))}
	for _, b := range bookings {
		var holder = b.Holder
		if b.Kind == model.BookingKindEvent {
			holder = b.Title
		}
		out.Bookings = append(out.Bookings, BookingInTheWayDto{Id: b.ID.String(), StartsAt: b.StartsAt, EndsAt: b.EndsAt, Holder: holder, IsEvent: b.Kind == model.BookingKindEvent})
	}
	for _, c := range closures {
		out.Closures = append(out.Closures, ClosureInTheWayDto{StartsAt: c.StartsAt, EndsAt: c.EndsAt, Reason: c.Reason.String})
	}
	return out, nil
}

// errInTheWay: the first thing that stops the event, with how many bookings.
func errInTheWay(found *InTheWayDto) *shared.ErrorResponse {
	if len(found.Closures) > 0 {
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.event_closed"}
	}
	return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.event_bookings_in_the_way", Args: []any{len(found.Bookings)}}
}

// eventCapacity: the event's own, else the facility's; without either nobody could join.
func eventCapacity(own sql.NullInt32, facility sql.NullInt32) *shared.ErrorResponse {
	if !own.Valid && !facility.Valid {
		return &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.event_needs_capacity"}
	}
	return nil
}

func (s *bookingsService) CreateEvent(ctx context.Context, tenantId uuid.UUID, staffId uuid.UUID, input EventInput) (*EventDto, *shared.ErrorResponse) {
	var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.event_failed"}
	var facility, loc, facilityErr = s.councilFacility(ctx, model.New(s.db), tenantId, input.FacilityId)
	if facilityErr != nil {
		return nil, facilityErr
	}
	if key := input.validate(time.Now(), loc); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	if capErr := eventCapacity(input.capacity(), facility.Capacity); capErr != nil {
		return nil, capErr
	}

	var tx, err = s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, failed
	}
	defer tx.Rollback()
	var queries = model.New(tx)
	var found *InTheWayDto
	if found, err = inTheWay(ctx, queries, facility.ID, input.StartsAt, input.EndsAt, uuid.NullUUID{}); err != nil {
		return nil, failed
	}
	if !found.empty() {
		return nil, errInTheWay(found)
	}
	var id uuid.UUID
	id, err = queries.CreateEvent(ctx, model.CreateEventParams{
		FacilityID:    facility.ID,
		StartsAt:      input.StartsAt,
		EndsAt:        input.EndsAt,
		BookedBy:      uuid.NullUUID{UUID: staffId, Valid: true},
		Title:         sql.NullString{String: input.Title, Valid: true},
		Description:   input.Description,
		EventCapacity: input.capacity(),
	})
	if overlapped(err) { // booked in the meantime
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.event_bookings_in_the_way", Args: []any{1}}
	}
	if err != nil || tx.Commit() != nil {
		return nil, failed
	}
	return s.event(ctx, tenantId, id)
}

// UpdateEvent: staff change an event before it starts. Its capacity never drops
// below who already joined; moved, those people are told the new time.
func (s *bookingsService) UpdateEvent(ctx context.Context, tenantId uuid.UUID, eventId uuid.UUID, input EventInput) (*EventDto, *shared.ErrorResponse) {
	var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.event_failed"}
	var tx, err = s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, failed
	}
	defer tx.Rollback()
	var queries = model.New(tx)
	var event model.LockEventRow
	event, err = queries.LockEvent(ctx, model.LockEventParams{ID: eventId, TenantID: tenantId})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.booking_not_found"}
	}
	if err != nil {
		return nil, failed
	}
	var loc, locErr = time.LoadLocation(event.Timezone)
	if locErr != nil {
		return nil, failed
	}
	if key := input.validate(time.Now(), loc); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	switch {
	case event.Status != model.BookingStatusConfirmed:
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.booking_already_cancelled"}
	case !event.StartsAt.After(time.Now()):
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.booking_started"}
	case input.FacilityId != event.FacilityID:
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.event_facility_fixed"}
	}
	if capErr := eventCapacity(input.capacity(), event.FacilityCapacity); capErr != nil {
		return nil, capErr
	}
	var capacity = input.capacity()
	if !capacity.Valid {
		capacity = event.FacilityCapacity
	}
	if int64(capacity.Int32) < event.ParticipantCount {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.event_capacity_below_joined", Args: []any{event.ParticipantCount}}
	}
	var moved = !input.StartsAt.Equal(event.StartsAt) || !input.EndsAt.Equal(event.EndsAt)
	if moved {
		var found *InTheWayDto
		if found, err = inTheWay(ctx, queries, event.FacilityID, input.StartsAt, input.EndsAt, uuid.NullUUID{UUID: eventId, Valid: true}); err != nil {
			return nil, failed
		}
		if !found.empty() {
			return nil, errInTheWay(found)
		}
	}
	_, err = queries.UpdateEvent(ctx, model.UpdateEventParams{
		ID:            eventId,
		Title:         sql.NullString{String: input.Title, Valid: true},
		Description:   input.Description,
		EventCapacity: input.capacity(),
		StartsAt:      input.StartsAt,
		EndsAt:        input.EndsAt,
	})
	if overlapped(err) {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.event_bookings_in_the_way", Args: []any{1}}
	}
	if err != nil || tx.Commit() != nil {
		return nil, failed
	}
	var dto, dtoErr = s.event(ctx, tenantId, eventId)
	if dtoErr == nil && moved {
		s.notifyParticipants(ctx, model.New(s.db), eventId, model.NotificationTypeBookingChanged,
			NotificationPayload(dto.FacilityName, dto.StartsAt, dto.EndsAt))
	}
	return dto, dtoErr
}

func (s *bookingsService) event(ctx context.Context, tenantId uuid.UUID, id uuid.UUID) (*EventDto, *shared.ErrorResponse) {
	var row, err = model.New(s.db).GetEvent(ctx, model.GetEventParams{ID: id, TenantID: tenantId})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.booking_not_found"}
	}
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.events_load_failed"}
	}
	var dto = mapEvent(row, time.Now())
	return &dto, nil
}

// notifyParticipants tells everyone with an account who joined the booking; a
// failure is logged and never undoes the change.
func (s *bookingsService) notifyParticipants(ctx context.Context, queries *model.Queries, bookingId uuid.UUID, kind model.NotificationType, payload json.RawMessage) {
	var users, err = queries.ListParticipantAccounts(ctx, bookingId)
	if err != nil {
		shared.Logger.Error("failed to list who joined booking %s: %v", bookingId, err)
		return
	}
	for _, user := range users {
		if err = queries.CreateNotification(ctx, model.CreateNotificationParams{
			UserID:    user.UUID,
			BookingID: uuid.NullUUID{UUID: bookingId, Valid: true},
			Type:      kind,
			Payload:   payload,
		}); err != nil {
			shared.Logger.Error("failed to record notification %s for booking %s: %v", kind, bookingId, err)
		}
	}
}

// overlapped: the exclusion constraint on booking refused the time.
func overlapped(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23P01"
}
