package counciladmin

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/audit"
	"librequenda/core/me"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"strconv"
	"time"

	"github.com/angelbarreiros/Penguin/router/helpers"
	"github.com/google/uuid"
)

// requestRetention: when a request nobody decided on is deleted (expires_at).
const requestRetention = me.ResidentRequestRetentionDays * 24 * time.Hour

// ResidentRequestDto: someone asking to be verified, with what the council
// needs to look them up in its census. Shown to its administrators only,
// deleted once they decide.
type ResidentRequestDto struct {
	Id          string    `json:"id"`
	UserId      string    `json:"user_id"`
	Email       string    `json:"email"`        // their account's
	AccountName string    `json:"account_name"` // their account's, "" when not set
	Dni         string    `json:"dni"`
	FullName    string    `json:"full_name"` // as they wrote it in the request
	PostalCode  string    `json:"postal_code"`
	Street      string    `json:"street"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"` // deleted then if nobody decided
}

type ResidentRequestsResponse struct {
	Requests []ResidentRequestDto `json:"requests"` // the oldest first
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

type ResidentRequestCount struct {
	Pending int64 `json:"pending"`
}

func (s *counciladminService) ListResidentRequests(ctx context.Context, tenantId uuid.UUID, page int) (*ResidentRequestsResponse, *shared.ErrorResponse) {
	if page < 1 || page > shared.MaxPage {
		return nil, &shared.ErrorResponse{Code: http.StatusBadRequest, Message: "errors.invalid_query_params"}
	}
	var rows, err = model.New(s.db).ListResidentRequests(ctx, model.ListResidentRequestsParams{
		TenantID:   tenantId,
		PageSize:   residentsPageSize,
		PageOffset: int32((page - 1) * residentsPageSize),
	})
	if err != nil {
		return nil, failed("errors.resident_requests_load_failed")
	}
	var response = &ResidentRequestsResponse{Requests: make([]ResidentRequestDto, 0, len(rows)), Page: page, PageSize: residentsPageSize}
	for _, row := range rows {
		response.Total = row.Total
		response.Requests = append(response.Requests, ResidentRequestDto{
			Id:          row.ID.String(),
			UserId:      row.UserID.String(),
			Email:       row.Email,
			AccountName: row.AccountName,
			Dni:         row.Dni,
			FullName:    row.FullName,
			PostalCode:  row.PostalCode,
			Street:      row.Street,
			CreatedAt:   row.CreatedAt,
			ExpiresAt:   row.CreatedAt.Add(requestRetention),
		})
	}
	return response, nil
}

func (s *counciladminService) CountResidentRequests(ctx context.Context, tenantId uuid.UUID) (*ResidentRequestCount, *shared.ErrorResponse) {
	var count, err = model.New(s.db).CountResidentRequests(ctx, tenantId)
	if err != nil {
		return nil, failed("errors.resident_requests_load_failed")
	}
	return &ResidentRequestCount{Pending: count}, nil
}

// DecideResidentRequest verifies the person (verify) or discards the request.
// Either way the request, with its identity data, is deleted, and the person
// is told. Returns whose it was, for the audit log.
func (s *counciladminService) DecideResidentRequest(ctx context.Context, tenantId uuid.UUID, staffId uuid.UUID, requestId uuid.UUID, verify bool) (uuid.UUID, *shared.ErrorResponse) {
	var tx, err = s.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, failed("errors.resident_save_failed")
	}
	defer tx.Rollback()
	var queries = model.New(tx)
	var userId uuid.UUID
	userId, err = queries.TakeResidentRequest(ctx, model.TakeResidentRequestParams{ID: requestId, TenantID: tenantId})
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.resident_request_not_found"}
	}
	if err != nil {
		return uuid.Nil, failed("errors.resident_save_failed")
	}
	var notice = model.NotificationTypeResidentRejected
	if verify {
		notice = model.NotificationTypeResidentVerified
		// already verified by email meanwhile: the request is simply done
		if _, err = queries.AddTenantResident(ctx, model.AddTenantResidentParams{
			TenantID:   tenantId,
			UserID:     userId,
			VerifiedBy: uuid.NullUUID{UUID: staffId, Valid: true},
		}); isUniqueViolation(err) { // verified by another council meanwhile: one council per person
			return uuid.Nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.resident_of_another_council"}
		} else if err != nil {
			return uuid.Nil, failed("errors.resident_save_failed")
		}
	}
	if err = queries.NotifyResidentRequestDecided(ctx, model.NotifyResidentRequestDecidedParams{UserID: userId, Type: notice, TenantID: tenantId}); err != nil {
		return uuid.Nil, failed("errors.resident_save_failed")
	}
	if err = tx.Commit(); err != nil {
		return uuid.Nil, failed("errors.resident_save_failed")
	}
	return userId, nil
}

// ListResidentRequests: ?page=<n>, optional.
func (h *counciladminHandler) ListResidentRequests(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var page = 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		var parsed, err = strconv.Atoi(raw)
		if err != nil {
			helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
			return
		}
		page = parsed
	}
	var response, err = GetCounciladminService(h.db).ListResidentRequests(r.Context(), tenantId, page)
	send(w, r, response, err)
}

func (h *counciladminHandler) CountResidentRequests(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var response, err = GetCounciladminService(h.db).CountResidentRequests(r.Context(), tenantId)
	send(w, r, response, err)
}

func (h *counciladminHandler) VerifyResidentRequest(w http.ResponseWriter, r *http.Request) {
	h.decideResidentRequest(w, r, true)
}

func (h *counciladminHandler) DiscardResidentRequest(w http.ResponseWriter, r *http.Request) {
	h.decideResidentRequest(w, r, false)
}

func (h *counciladminHandler) decideResidentRequest(w http.ResponseWriter, r *http.Request, verify bool) {
	var tenantId, staffId, ok = h.councilAdminUser(w, r)
	if !ok {
		return
	}
	var raw, err = helpers.GetStringPathValue("requestId", r)
	var requestId uuid.UUID
	if err == nil {
		requestId, err = uuid.Parse(raw)
	}
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return
	}
	var userId, decideErr = GetCounciladminService(h.db).DecideResidentRequest(r.Context(), tenantId, staffId, requestId, verify)
	if decideErr == nil {
		audit.TargetUser(r.Context(), userId)
	}
	send(w, r, nil, decideErr)
}
