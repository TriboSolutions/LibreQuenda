package counciladmin

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"librequenda/core/issue"
	"librequenda/db"
	"librequenda/model"
	"librequenda/shared"
	"librequenda/storage"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// --- the maintenance list: problems residents reported at the council's facilities ---

type IssueDto struct {
	ReporterPhone    string     `json:"reporter_phone"` // only when the reporter shared contact details
	FacilityImageURL string     `json:"facility_image_url"`
	Id               string     `json:"id"`
	FacilityId       string     `json:"facility_id"`
	FacilityName     string     `json:"facility_name"`
	Category         string     `json:"category"`
	Description      string     `json:"description"`
	PhotoUrl         string     `json:"photo_url"`
	Urgent           bool       `json:"urgent"`
	Status           string     `json:"status"` // new, in_progress, solved, rejected
	StaffNote        string     `json:"staff_note"`
	CreatedAt        time.Time  `json:"created_at"`
	ResolvedAt       *time.Time `json:"resolved_at"`
	BookingStartsAt  *time.Time `json:"booking_starts_at"` // the booking it was reported from
	ReporterName     string     `json:"reporter_name"`     // first name only; "" after account erasure
	ReporterEmail    string     `json:"reporter_email"`    // only when the reporter shared it
}

type IssuesResponse struct {
	Issues []IssueDto `json:"issues"`
}

type IssueCountDto struct {
	New int64 `json:"new"`
}

type IssueChangeInput struct {
	Status    string `json:"status"`
	StaffNote string `json:"staff_note"` // for the reporter; required to reject
}

func mapIssue(row model.ListCouncilIssuesRow) IssueDto {
	var dto = IssueDto{
		Id: row.ID.String(), FacilityId: row.FacilityID.String(), FacilityName: row.FacilityName,
		Category: string(row.Category), Description: row.Description, PhotoUrl: storage.URL(row.PhotoRef.String),
		Urgent: row.Urgent, Status: string(row.Status), StaffNote: row.StaffNote.String, CreatedAt: row.CreatedAt,
		ReporterName:     row.ReporterFirstName,
		FacilityImageURL: storage.URL(row.FacilityImageRef.String),
	}
	if row.ShareContact {
		dto.ReporterEmail = row.ReporterEmail.String
		dto.ReporterPhone = row.ReporterPhone.String
	}
	if row.ResolvedAt.Valid {
		dto.ResolvedAt = &row.ResolvedAt.Time
	}
	if row.BookingStartsAt.Valid {
		dto.BookingStartsAt = &row.BookingStartsAt.Time
	}
	return dto
}

// ListIssues: newest first; facility and status ("" for all) narrow it.
func (s *counciladminService) ListIssues(ctx context.Context, tenantId uuid.UUID, facility string, status string) (*IssuesResponse, *shared.ErrorResponse) {
	var params = model.ListCouncilIssuesParams{TenantID: tenantId}
	if facility != "" {
		var id, err = uuid.Parse(facility)
		if err != nil {
			return nil, &shared.ErrorResponse{Code: http.StatusBadRequest, Message: "errors.invalid_query_params"}
		}
		params.FacilityID = uuid.NullUUID{UUID: id, Valid: true}
	}
	if status != "" {
		if issue.Change(model.IssueStatus(status), "-") != "" {
			return nil, &shared.ErrorResponse{Code: http.StatusBadRequest, Message: "errors.invalid_query_params"}
		}
		params.Status = model.NullIssueStatus{IssueStatus: model.IssueStatus(status), Valid: true}
	}
	var rows, err = model.New(s.db).ListCouncilIssues(ctx, params)
	if err != nil {
		return nil, failed("errors.issue_failed")
	}
	var response = &IssuesResponse{Issues: make([]IssueDto, 0, len(rows))}
	for _, row := range rows {
		response.Issues = append(response.Issues, mapIssue(row))
	}
	return response, nil
}

func (s *counciladminService) CountNewIssues(ctx context.Context, tenantId uuid.UUID) (*IssueCountDto, *shared.ErrorResponse) {
	var count, err = model.New(s.db).CountNewCouncilIssues(ctx, tenantId)
	if err != nil {
		return nil, failed("errors.issue_failed")
	}
	return &IssueCountDto{New: count}, nil
}

// issueNotice is the reporter's notification: what the bell and the email show.
type issueNotice struct {
	FacilityName string `json:"facility_name"`
	FacilitySlug string `json:"facility_slug"`
	TenantName   string `json:"tenant_name"`
	TenantSlug   string `json:"tenant_slug"`
	Reason       string `json:"reason"` // the council's note
}

// ChangeIssue: staff move a problem along and leave a note for the reporter,
// who is told (bell and email) when it is solved or rejected.
func (s *counciladminService) ChangeIssue(ctx context.Context, tenantId, staffId, issueId uuid.UUID, input IssueChangeInput) (*IssueDto, *shared.ErrorResponse) {
	var to = model.IssueStatus(input.Status)
	var note = strings.TrimSpace(input.StaffNote)
	if key := issue.Change(to, note); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	var current, err = model.New(s.db).GetCouncilIssue(ctx, model.GetCouncilIssueParams{ID: issueId, TenantID: tenantId})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.issue_not_found"}
	}
	if err != nil {
		return nil, failed("errors.issue_failed")
	}
	err = db.WithTenant(ctx, s.db, tenantId, func(q *model.Queries) error {
		if updateErr := q.UpdateFacilityIssue(ctx, model.UpdateFacilityIssueParams{
			ID: issueId, Status: to, StaffNote: sql.NullString{String: note, Valid: note != ""},
			StaffID: uuid.NullUUID{UUID: staffId, Valid: true},
		}); updateErr != nil {
			return updateErr
		}
		var kind = issue.Notice(current.Status, to)
		if kind == "" || !current.ReporterID.Valid {
			return nil
		}
		var payload, _ = json.Marshal(issueNotice{
			FacilityName: current.FacilityName, FacilitySlug: current.FacilitySlug,
			TenantName: current.TenantName, TenantSlug: current.TenantSlug, Reason: note,
		})
		return q.CreateIssueNotification(ctx, model.CreateIssueNotificationParams{
			UserID: current.ReporterID.UUID, IssueID: uuid.NullUUID{UUID: issueId, Valid: true}, Type: kind, Payload: payload,
		})
	})
	if err != nil {
		return nil, failed("errors.issue_failed")
	}
	var list, listErr = model.New(s.db).ListCouncilIssues(ctx, model.ListCouncilIssuesParams{TenantID: tenantId})
	if listErr != nil {
		return nil, failed("errors.issue_failed")
	}
	for _, row := range list {
		if row.ID == issueId {
			var dto = mapIssue(row)
			return &dto, nil
		}
	}
	return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.issue_not_found"}
}
