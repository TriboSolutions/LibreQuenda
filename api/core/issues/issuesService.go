package issues

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/auth"
	"librequenda/checkin"
	"librequenda/core/me"
	"librequenda/issue"
	"librequenda/model"
	"librequenda/shared"
	"librequenda/storage"
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

// Report requires a confirmed booking held or joined by the reporter, during
// its booked interval, and the reporter at the facility (its check-in radius
// around its pin; the location is checked, never stored). It lands in the
// council's maintenance list.
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
	// only a photo uploaded for this facility's council
	if err == nil && input.PhotoRef != "" && !storage.Owned(input.PhotoRef, facility.TenantID, "issues") {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.invalid_image"}
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
	if problem := atFacility(input, facility); problem != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.issue_" + problem}
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

// atFacility: "" when the phone is at the facility, or the location problem
// (location_required, location_inaccurate, too_far). A facility without a pin
// cannot be checked, so it is not asked for.
func atFacility(input issue.Report, facility model.GetFacilityForIssueRow) string {
	if !facility.Latitude.Valid || !facility.Longitude.Valid {
		return ""
	}
	var position *checkin.Position
	if input.Latitude != nil && input.Longitude != nil && input.Accuracy != nil {
		position = &checkin.Position{Latitude: *input.Latitude, Longitude: *input.Longitude, Accuracy: *input.Accuracy}
	}
	return checkin.CheckLocation(position, facility.Latitude.Float64, facility.Longitude.Float64, int(facility.RadiusM))
}
