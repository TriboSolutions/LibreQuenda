package counciladmin

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/model"
	"librequenda/shared"
	"librequenda/validate"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// likePattern escapes what LIKE reads as wildcards, so a search is taken literally.
func likePattern(q string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
}

// ListResidents: the council's verified residents, narrowed by name or email,
// sorted by name or by the latest verification, page 1 first.
func (s *counciladminService) ListResidents(ctx context.Context, tenantId uuid.UUID, q string, sort string, page int) (*ResidentsResponse, *shared.ErrorResponse) {
	q = strings.TrimSpace(q)
	if sort == "" {
		sort = "name"
	}
	if !residentSorts[sort] || page < 1 || len(q) > 255 {
		return nil, &shared.ErrorResponse{Code: http.StatusBadRequest, Message: "errors.invalid_query_params"}
	}
	var rows, err = model.New(s.db).ListTenantResidents(ctx, model.ListTenantResidentsParams{
		TenantID:   tenantId,
		Q:          likePattern(q),
		Sort:       sort,
		PageSize:   residentsPageSize,
		PageOffset: int32((page - 1) * residentsPageSize),
	})
	if err != nil {
		return nil, failed("errors.residents_load_failed")
	}
	var response = &ResidentsResponse{Residents: make([]ResidentDto, 0, len(rows)), Page: page, PageSize: residentsPageSize}
	for _, row := range rows {
		response.Total = row.Total
		response.Residents = append(response.Residents, ResidentDto{
			UserId:     row.UserID.String(),
			Email:      row.Email,
			FullName:   row.FullName,
			VerifiedAt: row.VerifiedAt,
			VerifiedBy: row.VerifiedByName,
		})
	}
	return response, nil
}

// AddResident verifies the person with that email as the council's resident.
// They need an account with a verified email: the council checks who they are
// (at the desk, against the census) before verifying them here.
func (s *counciladminService) AddResident(ctx context.Context, tenantId uuid.UUID, staffId uuid.UUID, email string) (uuid.UUID, *shared.ErrorResponse) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !validate.Email(email) {
		return uuid.Nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.invalid_email"}
	}
	var queries = model.New(s.db)
	var user, err = queries.GetVerifiedUserByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.resident_user_not_found"}
	}
	if err != nil {
		return uuid.Nil, failed("errors.resident_save_failed")
	}
	var added bool
	added, err = queries.AddTenantResident(ctx, model.AddTenantResidentParams{
		TenantID:   tenantId,
		UserID:     user.ID,
		VerifiedBy: uuid.NullUUID{UUID: staffId, Valid: true},
	})
	if err != nil {
		return uuid.Nil, failed("errors.resident_save_failed")
	}
	if !added {
		return user.ID, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.resident_already_verified"}
	}
	// a request they had sent has done its job: its identity data goes
	if err = queries.DeleteResidentRequestOfUser(ctx, model.DeleteResidentRequestOfUserParams{TenantID: tenantId, UserID: user.ID}); err != nil {
		return user.ID, failed("errors.resident_save_failed")
	}
	return user.ID, nil
}

// RemoveResident takes the verification back. Their bookings stay: staff
// cancel them from the calendar if they must.
func (s *counciladminService) RemoveResident(ctx context.Context, tenantId uuid.UUID, userId uuid.UUID) *shared.ErrorResponse {
	var removed, err = model.New(s.db).RemoveTenantResident(ctx, model.RemoveTenantResidentParams{TenantID: tenantId, UserID: userId})
	if err != nil {
		return failed("errors.resident_save_failed")
	}
	if removed == 0 {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.resident_not_found"}
	}
	return nil
}
