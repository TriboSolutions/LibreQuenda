package counciladmin

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/core/notice"
	"librequenda/db"
	"librequenda/model"
	"librequenda/shared"
	"librequenda/validate"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

// likePattern escapes what LIKE reads as wildcards, so a search is taken literally.
func likePattern(q string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
}

// maxResidentSearch: the longest search in the Veciños tab, as the facilities search.
const maxResidentSearch = 100

// ListResidents: the council's verified residents, narrowed by name or email,
// sorted by name or by the latest verification, page 1 first.
func (s *counciladminService) ListResidents(ctx context.Context, tenantId uuid.UUID, q string, sort string, page int) (*ResidentsResponse, *shared.ErrorResponse) {
	q = strings.TrimSpace(q)
	if sort == "" {
		sort = "name"
	}
	if !residentSorts[sort] || page < 1 || page > shared.MaxPage || utf8.RuneCountInString(q) > maxResidentSearch {
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
	if isUniqueViolation(err) { // verified by another council: one council per person
		return uuid.Nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.resident_of_another_council"}
	}
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
	// told like a request accepted (bell, the app and email), whether or not they had sent one
	if err = queries.NotifyResidentRequestDecided(ctx, model.NotifyResidentRequestDecidedParams{UserID: user.ID, Type: model.NotificationTypeResidentVerified, TenantID: tenantId}); err != nil {
		return user.ID, failed("errors.resident_save_failed")
	}
	return user.ID, nil
}

// RemoveResident takes the verification back. A person is a resident of one
// council only, so their upcoming bookings at its residents-only facilities go
// too and they leave the games they joined there. Each one is told to them (where
// and when) and to whoever had joined, by notice and email, and a last notice says
// the verification was taken back and how many bookings went with it.
func (s *counciladminService) RemoveResident(ctx context.Context, tenantId uuid.UUID, userId uuid.UUID) *shared.ErrorResponse {
	var removed int64
	var err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		var removeErr error
		if removed, removeErr = queries.RemoveTenantResident(ctx, model.RemoveTenantResidentParams{TenantID: tenantId, UserID: userId}); removeErr != nil || removed == 0 {
			return removeErr
		}
		var cancelled, cancelErr = queries.CancelBookingsOfFormerResident(ctx, model.CancelBookingsOfFormerResidentParams{TenantID: tenantId, UserID: uuid.NullUUID{UUID: userId, Valid: true}})
		if cancelErr != nil {
			return cancelErr
		}
		for _, booking := range cancelled {
			var c = notice.Cancelled{FacilityID: booking.FacilityID, FacilityName: booking.FacilityName, TenantSlug: booking.TenantSlug, TenantName: booking.TenantName, Title: booking.Title, StartsAt: booking.StartsAt, EndsAt: booking.EndsAt}
			// they are told each booking that went, where and when; who had joined it, that the council cancelled it
			if tellErr := notice.Tell(ctx, queries, userId, booking.ID, model.NotificationTypeBookingCancelledByCouncil, notice.Payload(c, map[string]any{"cause": notice.CauseResidentRevoked})); tellErr != nil {
				return tellErr
			}
			if tellErr := notice.TellParticipants(ctx, queries, booking.ID, model.NotificationTypeBookingCancelledByCouncil, notice.Payload(c, map[string]any{"by_staff": true})); tellErr != nil {
				return tellErr
			}
		}
		var left, leaveErr = queries.LeaveGamesOfFormerResident(ctx, model.LeaveGamesOfFormerResidentParams{TenantID: tenantId, UserID: uuid.NullUUID{UUID: userId, Valid: true}})
		if leaveErr != nil {
			return leaveErr
		}
		for _, game := range left {
			var c = notice.Cancelled{FacilityID: game.FacilityID, FacilityName: game.FacilityName, TenantSlug: game.TenantSlug, TenantName: game.TenantName, Title: game.Title, StartsAt: game.StartsAt, EndsAt: game.EndsAt}
			if tellErr := notice.Tell(ctx, queries, userId, game.ID, model.NotificationTypeBookingCancelledByCouncil, notice.Payload(c, map[string]any{"cause": notice.CauseResidentRevokedLeft})); tellErr != nil {
				return tellErr
			}
		}
		return queries.NotifyResidentRevoked(ctx, model.NotifyResidentRevokedParams{UserID: userId, TenantID: tenantId, Cancelled: int32(len(cancelled))})
	})
	if err != nil {
		return failed("errors.resident_save_failed")
	}
	if removed == 0 {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.resident_not_found"}
	}
	return nil
}
