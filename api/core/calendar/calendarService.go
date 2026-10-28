package calendar

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"sync"

	"github.com/google/uuid"
)

type calendarService struct {
	db *sql.DB
}

var sCalendarInstance *calendarService
var sCalendarOnce *sync.Once = new(sync.Once)

func GetCalendarService(db *sql.DB) *calendarService {
	sCalendarOnce.Do(func() {
		sCalendarInstance = &calendarService{db: db}
	})
	return sCalendarInstance
}

var errCalendarFailed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.calendar_failed"}
var errCalendarNotFound = &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.calendar_not_found"}

// feedUrl: the address calendar apps fetch, through the proxy's /api.
func feedUrl(token uuid.NullUUID) string {
	if !token.Valid {
		return ""
	}
	return shared.PUBLIC_URL + "/api/calendar/" + token.UUID.String() + ".ics"
}

func (s *calendarService) Get(ctx context.Context, userId uuid.UUID) (*CalendarFeedDto, *shared.ErrorResponse) {
	var token, err = model.New(s.db).GetCalendarToken(ctx, userId)
	if err != nil {
		return nil, errCalendarFailed
	}
	return &CalendarFeedDto{Url: feedUrl(token)}, nil
}

// Reset turns the feed on, or moves it to a new address: the old one stops working.
func (s *calendarService) Reset(ctx context.Context, userId uuid.UUID) (*CalendarFeedDto, *shared.ErrorResponse) {
	var token, err = model.New(s.db).ResetCalendarToken(ctx, userId)
	if err != nil {
		return nil, errCalendarFailed
	}
	return &CalendarFeedDto{Url: feedUrl(token)}, nil
}

func (s *calendarService) TurnOff(ctx context.Context, userId uuid.UUID) *shared.ErrorResponse {
	if err := model.New(s.db).ClearCalendarToken(ctx, userId); err != nil {
		return errCalendarFailed
	}
	return nil
}

// Feed: the bookings behind a feed token, for writeICS.
func (s *calendarService) Feed(ctx context.Context, token uuid.UUID) ([]model.ListCalendarBookingsRow, *shared.ErrorResponse) {
	var queries = model.New(s.db)
	var userId, err = queries.GetUserIdByCalendarToken(ctx, uuid.NullUUID{UUID: token, Valid: true})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errCalendarNotFound
	}
	if err != nil {
		return nil, errCalendarFailed
	}
	var rows, listErr = queries.ListCalendarBookings(ctx, uuid.NullUUID{UUID: userId, Valid: true})
	if listErr != nil {
		return nil, errCalendarFailed
	}
	return rows, nil
}
