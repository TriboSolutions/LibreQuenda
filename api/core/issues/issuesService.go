package issues

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/auth"
	"librequenda/core/me"
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

// Report requires a confirmed booking held or joined by the reporter,
// during its booked interval. It lands in the council's maintenance list.
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

	var tx, txErr = s.db.BeginTx(ctx, nil)
	if txErr != nil {
		return nil, failed
	}
	defer tx.Rollback()
	queries = model.New(tx)
	if err = queries.SetTenant(ctx, facility.TenantID.String()); err != nil {
		return nil, failed
	}

	var bookingId, bookingErr = uuid.Parse(input.BookingID)
	if bookingErr != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.booking_not_found"}
	}
	// Hold the booking lock until commit so concurrent reports share the five-report limit.
	var row, getErr = queries.GetBookingForIssue(ctx, model.GetBookingForIssueParams{BookingID: bookingId, UserID: uuid.NullUUID{UUID: userId, Valid: true}})
	if errors.Is(getErr, sql.ErrNoRows) || (getErr == nil && row.FacilityID != facility.ID) {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.booking_not_found"}
	}
	if getErr != nil {
		return nil, failed
	}
	if !issue.Reportable(row.StartsAt, row.EndsAt, time.Now()) {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.issue_active_booking_required"}
	}
	var booking = uuid.NullUUID{UUID: row.ID, Valid: true}

	var total, countErr = queries.CountIssuesByBooking(ctx, booking)
	if countErr != nil {
		return nil, failed
	}
	if total >= issue.MaxReportsPerBooking {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.too_many_booking_issues"}
	}

	var id, createErr = queries.CreateFacilityIssue(ctx, model.CreateFacilityIssueParams{
		FacilityID:   facility.ID,
		ReporterID:   uuid.NullUUID{UUID: userId, Valid: true},
		BookingID:    booking,
		Category:     model.IssueCategory(input.Category),
		Description:  input.Description,
		PhotoRef:     sql.NullString{String: input.PhotoRef, Valid: input.PhotoRef != ""},
		Urgent:       input.Urgent,
		ShareContact: input.ShareContact,
	})
	if createErr != nil {
		return nil, failed
	}
	if err = queries.NotifyCouncilAdminsOfIssue(ctx, id); err != nil {
		return nil, failed
	}
	if err = tx.Commit(); err != nil {
		return nil, failed
	}
	return &ReportResponse{Id: id.String()}, nil
}
