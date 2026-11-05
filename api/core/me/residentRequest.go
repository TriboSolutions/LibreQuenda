package me

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"librequenda/auth"
	"librequenda/model"
	"librequenda/shared"
	"librequenda/validate"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

// ResidentRequestRetentionDays: a request nobody decided on is deleted after
// this many days (cronJobs.purgeResidentRequests), with the identity data in it.
const ResidentRequestRetentionDays = 15

// ResidentRequestInterval: a person sends one request a week, across councils
// (app_user.resident_requested_at; a withdrawn one still counts).
const ResidentRequestInterval = 7 * 24 * time.Hour

// madrid: the error says when the next request can go, in Spanish peninsular time.
var madrid = func() *time.Location {
	var loc, err = time.LoadLocation("Europe/Madrid")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// ResidentRequestInput: what the council needs to find the person in its own
// census before verifying them. Kept only until it decides.
type ResidentRequestInput struct {
	Dni        string `json:"dni"`
	FullName   string `json:"full_name"`
	PostalCode string `json:"postal_code"`
	Street     string `json:"street"`
}

type ResidentRequestResponse struct {
	RequestedAt time.Time `json:"requested_at"`
}

// ResidentRequestDto: a pending request, on /me (no identity data there).
type ResidentRequestDto struct {
	TenantName  string    `json:"tenant_name"`
	TenantSlug  string    `json:"tenant_slug"`
	RequestedAt time.Time `json:"requested_at"`
}

// clean normalises the input and returns the error key of the first bad field, or "".
func (in *ResidentRequestInput) clean() string {
	var dni = strings.ToUpper(in.Dni)
	dni = strings.NewReplacer(" ", "", "-", "", ".", "").Replace(dni)
	in.Dni = dni
	in.FullName = strings.Join(strings.Fields(in.FullName), " ")
	in.PostalCode = strings.TrimSpace(in.PostalCode)
	in.Street = strings.Join(strings.Fields(in.Street), " ")
	if !validate.DNI(in.Dni) {
		return "errors.invalid_dni"
	}
	if !validate.PostalCode(in.PostalCode) {
		return "errors.invalid_postal_code"
	}
	if n := utf8.RuneCountInString(in.FullName); n < 2 || n > 200 {
		return "errors.resident_request_invalid"
	}
	if n := utf8.RuneCountInString(in.Street); n < 2 || n > 200 {
		return "errors.resident_request_invalid"
	}
	return ""
}

func mapResidentRequests(rows []model.ListResidentRequestsByUserRow) []ResidentRequestDto {
	var out = make([]ResidentRequestDto, 0, len(rows))
	for _, r := range rows {
		out = append(out, ResidentRequestDto{TenantName: r.Name, TenantSlug: r.Slug, RequestedAt: r.CreatedAt})
	}
	return out
}

// RequestResidency asks the council to verify the person as its resident. Its
// administrators are told (bell), and decide from their own census.
func (s *meService) RequestResidency(ctx context.Context, session auth.Session, slug string, input ResidentRequestInput) (*ResidentRequestResponse, *shared.ErrorResponse) {
	if key := input.clean(); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	var userId, userErr = CurrentUserId(ctx, s.db, session)
	if userErr != nil {
		return nil, userErr
	}
	var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.resident_request_failed"}
	var tenant, err = model.New(s.db).GetTenantBySlug(ctx, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.tenant_not_found"}
	}
	if err != nil {
		return nil, failed
	}

	var tx, txErr = s.db.BeginTx(ctx, nil)
	if txErr != nil {
		return nil, failed
	}
	defer tx.Rollback()
	var queries = model.New(tx)
	var verified bool
	if verified, err = queries.IsTenantResident(ctx, model.IsTenantResidentParams{TenantID: tenant.ID, UserID: userId}); err != nil {
		return nil, failed
	}
	if verified {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.resident_already_verified_self"}
	}
	// one a week, across councils: a pending one answers first (409 below)
	var _, claimErr = queries.ClaimWeeklyResidentRequest(ctx, userId)
	var limited = errors.Is(claimErr, sql.ErrNoRows)
	if claimErr != nil && !limited {
		return nil, failed
	}
	var row, createErr = queries.CreateResidentRequest(ctx, model.CreateResidentRequestParams{
		TenantID:   tenant.ID,
		UserID:     userId,
		Dni:        input.Dni,
		FullName:   input.FullName,
		PostalCode: input.PostalCode,
		Street:     input.Street,
	})
	if errors.Is(createErr, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.resident_request_pending"}
	}
	if createErr != nil {
		return nil, failed
	}
	if limited {
		var last, lastErr = model.New(s.db).GetResidentRequestedAt(ctx, userId)
		if lastErr != nil || !last.Valid {
			return nil, failed
		}
		var next = last.Time.Add(ResidentRequestInterval).In(madrid)
		return nil, &shared.ErrorResponse{Code: http.StatusTooManyRequests, Message: "errors.resident_request_weekly", Args: []any{next.Format("02/01/2006 15:04")}}
	}
	if err = queries.NotifyCouncilAdminsOfResidentRequest(ctx, model.NotifyCouncilAdminsOfResidentRequestParams{TenantID: tenant.ID, Name: input.FullName}); err != nil {
		return nil, failed
	}
	if err = tx.Commit(); err != nil {
		return nil, failed
	}
	return &ResidentRequestResponse{RequestedAt: row.CreatedAt}, nil
}

// WithdrawResidency deletes the person's own pending request.
func (s *meService) WithdrawResidency(ctx context.Context, session auth.Session, slug string) *shared.ErrorResponse {
	var userId, userErr = CurrentUserId(ctx, s.db, session)
	if userErr != nil {
		return userErr
	}
	var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.resident_request_failed"}
	var queries = model.New(s.db)
	var tenant, err = queries.GetTenantBySlug(ctx, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.tenant_not_found"}
	}
	if err != nil {
		return failed
	}
	var deleted int64
	if deleted, err = queries.DeleteOwnResidentRequest(ctx, model.DeleteOwnResidentRequestParams{TenantID: tenant.ID, UserID: userId}); err != nil {
		return failed
	}
	if deleted == 0 {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.resident_request_not_found"}
	}
	return nil
}

// RequestResidency: POST /tenants/{slug}/resident-request.
func (h *meHandler) RequestResidency(w http.ResponseWriter, r *http.Request) {
	var session, ok = auth.SessionFromContext(r.Context())
	if !ok {
		helpers.SendI18NErrorResponse(w, r, http.StatusUnauthorized, "errors.unauthorized")
		return
	}
	var slug, slugErr = helpers.GetStringPathValue("slug", r)
	if slugErr != nil || !validate.Slug(slug) {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return
	}
	var input ResidentRequestInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&input); err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_request_body")
		return
	}
	var response, err = GetMeService(h.db).RequestResidency(r.Context(), session, slug, input)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	helpers.SendSuccessResponse(w, response)
}

// WithdrawResidency: DELETE /tenants/{slug}/resident-request.
func (h *meHandler) WithdrawResidency(w http.ResponseWriter, r *http.Request) {
	var session, ok = auth.SessionFromContext(r.Context())
	if !ok {
		helpers.SendI18NErrorResponse(w, r, http.StatusUnauthorized, "errors.unauthorized")
		return
	}
	var slug, slugErr = helpers.GetStringPathValue("slug", r)
	if slugErr != nil || !validate.Slug(slug) {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return
	}
	if err := GetMeService(h.db).WithdrawResidency(r.Context(), session, slug); err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	helpers.SendNoContentResponse(w)
}
