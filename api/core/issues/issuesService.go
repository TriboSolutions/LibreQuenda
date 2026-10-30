package issues

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/auth"
	"librequenda/core/me"
	"librequenda/db"
	"librequenda/issue"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type issuesService struct {
	db *sql.DB
}

var sInstance *issuesService
var sOnce *sync.Once = new(sync.Once)

func GetIssuesService(db *sql.DB) *issuesService {
	sOnce.Do(func() {
		sInstance = &issuesService{db: db}
	})
	return sInstance
}

var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.issue_failed"}

type ReportResponse struct {
	Id string `json:"id"`
}

// Report: a signed-in resident tells the facility's council about a problem,
// from the facility's page or from one of their bookings there (until a week
// after it ended). It lands in the council's maintenance list.
func (s *issuesService) Report(ctx context.Context, session auth.Session, input issue.Report) (*ReportResponse, *shared.ErrorResponse) {
	if key := input.Clean(); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	var facilityId, parseErr = uuid.Parse(input.FacilityID)
	if parseErr != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.facility_not_found"}
	}
	var userId, userErr = me.CurrentUserId(ctx, s.db, session)
	if userErr != nil {
		return nil, userErr
	}
	var queries = model.New(s.db)
	var facility, err = queries.GetFacilityForIssue(ctx, facilityId)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	}
	if err != nil {
		return nil, failed
	}

	var booking uuid.NullUUID
	if input.BookingID != "" {
		var bookingId, bookingErr = uuid.Parse(input.BookingID)
		if bookingErr != nil {
			return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.booking_not_found"}
		}
		var row, getErr = queries.GetBookingForIssue(ctx, model.GetBookingForIssueParams{BookingID: bookingId, UserID: uuid.NullUUID{UUID: userId, Valid: true}})
		if errors.Is(getErr, sql.ErrNoRows) || (getErr == nil && row.FacilityID != facility.ID) {
			return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.booking_not_found"}
		}
		if getErr != nil {
			return nil, failed
		}
		if !issue.Reportable(row.EndsAt, time.Now()) {
			return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.issue_booking_too_old"}
		}
		booking = uuid.NullUUID{UUID: row.ID, Valid: true}
	}

	var open, countErr = queries.CountOpenIssuesByReporter(ctx, uuid.NullUUID{UUID: userId, Valid: true})
	if countErr != nil {
		return nil, failed
	}
	if open >= issue.MaxOpenReports {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.too_many_open_issues"}
	}

	var id uuid.UUID
	err = db.WithTenant(ctx, s.db, facility.TenantID, func(q *model.Queries) error {
		var createErr error
		id, createErr = q.CreateFacilityIssue(ctx, model.CreateFacilityIssueParams{
			FacilityID:   facility.ID,
			ReporterID:   uuid.NullUUID{UUID: userId, Valid: true},
			BookingID:    booking,
			Category:     model.IssueCategory(input.Category),
			Description:  input.Description,
			PhotoRef:     sql.NullString{String: input.PhotoRef, Valid: input.PhotoRef != ""},
			Urgent:       input.Urgent,
			ShareContact: input.ShareContact,
		})
		return createErr
	})
	if err != nil {
		return nil, failed
	}
	return &ReportResponse{Id: id.String()}, nil
}
