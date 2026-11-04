package platform

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/model"
	"librequenda/shared"
	"librequenda/storage"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type platformService struct {
	db *sql.DB
}

var sPlatformInstance *platformService
var sPlatformOnce *sync.Once = new(sync.Once)

func GetPlatformService(db *sql.DB) *platformService {
	sPlatformOnce.Do(func() {
		sPlatformInstance = &platformService{db: db}
	})
	return sPlatformInstance
}

func failed(message string) *shared.ErrorResponse {
	return &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: message}
}

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23503"
}

func (s *platformService) tenant(ctx context.Context, queries *model.Queries, slug string) (model.GetTenantForPlatformRow, *shared.ErrorResponse) {
	var tenant, err = queries.GetTenantForPlatform(ctx, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return tenant, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.tenant_not_found"}
	}
	if err != nil {
		return tenant, failed("errors.tenants_load_failed")
	}
	return tenant, nil
}

func (s *platformService) ListTenants(ctx context.Context) (*TenantsResponse, *shared.ErrorResponse) {
	var rows, err = model.New(s.db).ListTenantsForPlatform(ctx)
	if err != nil {
		return nil, failed("errors.tenants_load_failed")
	}
	var response = &TenantsResponse{Tenants: make([]TenantSummaryDto, 0, len(rows))}
	for _, t := range rows {
		response.Tenants = append(response.Tenants, TenantSummaryDto{
			Slug:                   t.Slug,
			Name:                   t.Name,
			Timezone:               t.Timezone,
			ContactEmail:           t.ContactEmail.String,
			ContactPhone:           t.ContactPhone.String,
			LogoUrl:                storage.URL(t.LogoRef.String),
			IsActive:               t.IsActive,
			CreatedAt:              t.CreatedAt,
			FacilityCount:          t.FacilityCount,
			AdminCount:             t.AdminCount,
			PendingInvitationCount: t.PendingInvitationCount,
			BookingCount:           t.BookingCount,
		})
	}
	return response, nil
}

// CreateTenant creates the council and the invitation of its first administrator together.
func (s *platformService) CreateTenant(ctx context.Context, invitedBy uuid.UUID, input TenantInput) (*TenantDetailDto, *shared.ErrorResponse) {
	input.normalise()
	if key := input.validate(true); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}

	var tx, err = s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, failed("errors.tenant_save_failed")
	}
	defer tx.Rollback()
	var queries = model.New(tx)

	var created model.CreateTenantRow
	created, err = queries.CreateTenant(ctx, model.CreateTenantParams{
		Slug:             input.Slug,
		Name:             input.Name,
		Timezone:         input.Timezone,
		ContactEmail:     nullString(input.ContactEmail),
		ContactPhone:     nullString(input.ContactPhone),
		MunicipalityCode: nullString(input.MunicipalityCode),
		LogoRef:          nullString(input.LogoRef),
	})
	if isUniqueViolation(err) {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.slug_taken"}
	}
	if err != nil {
		return nil, failed("errors.tenant_save_failed")
	}

	_, err = queries.CreateInvitation(ctx, model.CreateInvitationParams{
		TenantID:  created.ID,
		Email:     input.AdminEmail,
		Role:      model.TenantRoleAdmin,
		InvitedBy: uuid.NullUUID{UUID: invitedBy, Valid: true},
	})
	if err != nil {
		return nil, failed("errors.tenant_save_failed")
	}
	if err = tx.Commit(); err != nil {
		return nil, failed("errors.tenant_save_failed")
	}
	return s.GetTenant(ctx, created.Slug)
}

func (s *platformService) GetTenant(ctx context.Context, slug string) (*TenantDetailDto, *shared.ErrorResponse) {
	var queries = model.New(s.db)
	var tenant, notFound = s.tenant(ctx, queries, slug)
	if notFound != nil {
		return nil, notFound
	}

	var bookings, err = queries.CountBookingsByTenant(ctx, tenant.ID)
	if err != nil {
		return nil, failed("errors.tenants_load_failed")
	}
	var members []model.ListTenantMembersRow
	members, err = queries.ListTenantMembers(ctx, tenant.ID)
	if err != nil {
		return nil, failed("errors.tenants_load_failed")
	}
	var invitations []model.ListPendingInvitationsRow
	invitations, err = queries.ListPendingInvitations(ctx, tenant.ID)
	if err != nil {
		return nil, failed("errors.tenants_load_failed")
	}

	var detail = &TenantDetailDto{
		Slug:             tenant.Slug,
		Name:             tenant.Name,
		Timezone:         tenant.Timezone,
		ContactEmail:     tenant.ContactEmail.String,
		ContactPhone:     tenant.ContactPhone.String,
		MunicipalityCode: tenant.MunicipalityCode.String,
		LogoRef:          tenant.LogoRef.String,
		LogoUrl:          storage.URL(tenant.LogoRef.String),
		IsActive:         tenant.IsActive,
		BookingCount:     bookings,
		Admins:           []MemberDto{},
		Invitations:      []InvitationDto{},
	}
	for _, m := range members {
		if m.Role != model.TenantRoleAdmin {
			continue
		}
		detail.Admins = append(detail.Admins, MemberDto{
			UserId:   m.UserID.String(),
			Email:    m.Email,
			FullName: m.FullName.String,
			Role:     string(m.Role),
			Since:    m.CreatedAt,
		})
	}
	for _, i := range invitations {
		detail.Invitations = append(detail.Invitations, InvitationDto{
			Id:        i.ID.String(),
			Email:     i.Email,
			Role:      string(i.Role),
			CreatedAt: i.CreatedAt,
		})
	}
	return detail, nil
}

func (s *platformService) UpdateTenant(ctx context.Context, slug string, input TenantInput) (*TenantDetailDto, *shared.ErrorResponse) {
	input.normalise()
	if key := input.validate(false); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	var queries = model.New(s.db)
	var tenant, notFound = s.tenant(ctx, queries, slug)
	if notFound != nil {
		return nil, notFound
	}
	// only a logo uploaded for this council
	if input.LogoRef != "" && !storage.Owned(input.LogoRef, tenant.ID, "councils") {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.invalid_image"}
	}
	var isActive = tenant.IsActive
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	var updated, err = queries.UpdateTenant(ctx, model.UpdateTenantParams{
		ID:               tenant.ID,
		Slug:             input.Slug,
		Name:             input.Name,
		Timezone:         input.Timezone,
		ContactEmail:     nullString(input.ContactEmail),
		ContactPhone:     nullString(input.ContactPhone),
		MunicipalityCode: nullString(input.MunicipalityCode),
		LogoRef:          nullString(input.LogoRef),
		IsActive:         isActive,
	})
	if isUniqueViolation(err) {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.slug_taken"}
	}
	if err != nil {
		return nil, failed("errors.tenant_save_failed")
	}
	return s.GetTenant(ctx, updated.Slug)
}

// DeleteTenant removes a council created by mistake. One with bookings is
// deactivated instead, so its history and indicators are kept.
func (s *platformService) DeleteTenant(ctx context.Context, slug string) *shared.ErrorResponse {
	var queries = model.New(s.db)
	var tenant, notFound = s.tenant(ctx, queries, slug)
	if notFound != nil {
		return notFound
	}
	var bookings, err = queries.CountBookingsByTenant(ctx, tenant.ID)
	if err != nil {
		return failed("errors.tenant_delete_failed")
	}
	if bookings > 0 {
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.tenant_has_bookings"}
	}
	if _, err = queries.DeleteTenant(ctx, tenant.ID); err != nil {
		return failed("errors.tenant_delete_failed")
	}
	// its bucket goes with it; a leftover one only holds files nothing points to
	if err = storage.DeleteCouncil(ctx, tenant.ID); err != nil {
		shared.Logger.Error("council %s deleted, its files not: %v", tenant.ID, err)
	}
	return nil
}

func (s *platformService) InviteTenantAdmin(ctx context.Context, invitedBy uuid.UUID, slug string, email string) (*TenantDetailDto, *shared.ErrorResponse) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !validEmail(email) {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.invalid_email"}
	}
	var queries = model.New(s.db)
	var tenant, notFound = s.tenant(ctx, queries, slug)
	if notFound != nil {
		return nil, notFound
	}
	var _, err = queries.CreateInvitation(ctx, model.CreateInvitationParams{
		TenantID:  tenant.ID,
		Email:     email,
		Role:      model.TenantRoleAdmin,
		InvitedBy: uuid.NullUUID{UUID: invitedBy, Valid: true},
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.invitation_exists"}
	}
	if err != nil {
		return nil, failed("errors.invitation_failed")
	}
	return s.GetTenant(ctx, slug)
}

func (s *platformService) DeleteInvitation(ctx context.Context, slug string, id uuid.UUID) *shared.ErrorResponse {
	var queries = model.New(s.db)
	var tenant, notFound = s.tenant(ctx, queries, slug)
	if notFound != nil {
		return notFound
	}
	var deleted, err = queries.DeleteInvitation(ctx, model.DeleteInvitationParams{ID: id, TenantID: tenant.ID})
	if err != nil {
		return failed("errors.invitation_failed")
	}
	if deleted == 0 {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.invitation_not_found"}
	}
	return nil
}

func (s *platformService) RemoveTenantAdmin(ctx context.Context, slug string, userId uuid.UUID) *shared.ErrorResponse {
	var queries = model.New(s.db)
	var tenant, notFound = s.tenant(ctx, queries, slug)
	if notFound != nil {
		return notFound
	}
	var removed, err = queries.RemoveTenantMember(ctx, model.RemoveTenantMemberParams{TenantID: tenant.ID, UserID: userId, Role: model.TenantRoleAdmin})
	if err != nil {
		return failed("errors.member_remove_failed")
	}
	if removed == 0 {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.member_not_found"}
	}
	return nil
}

func (s *platformService) ListSuperadmins(ctx context.Context) (*SuperadminsResponse, *shared.ErrorResponse) {
	var rows, err = model.New(s.db).ListPlatformAdmins(ctx)
	if err != nil {
		return nil, failed("errors.superadmins_load_failed")
	}
	var response = &SuperadminsResponse{Superadmins: make([]SuperadminDto, 0, len(rows))}
	for _, row := range rows {
		response.Superadmins = append(response.Superadmins, SuperadminDto{
			UserId:    row.ID.String(),
			Email:     row.Email,
			FullName:  row.FullName.String,
			FromEnv:   shared.SUPERADMIN_EMAIL != "" && strings.EqualFold(row.Email, shared.SUPERADMIN_EMAIL),
			CreatedAt: row.CreatedAt,
		})
	}
	return response, nil
}

// AddSuperadmin needs a person who has already logged in with that email verified.
func (s *platformService) AddSuperadmin(ctx context.Context, email string) *shared.ErrorResponse {
	email = strings.ToLower(strings.TrimSpace(email))
	if !validEmail(email) {
		return &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.invalid_email"}
	}
	var queries = model.New(s.db)
	var user, err = queries.GetVerifiedUserByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.superadmin_user_not_found"}
	}
	if err != nil {
		return failed("errors.superadmin_save_failed")
	}
	if err = queries.AddPlatformAdmin(ctx, user.ID); err != nil {
		return failed("errors.superadmin_save_failed")
	}
	return nil
}

func (s *platformService) RemoveSuperadmin(ctx context.Context, userId uuid.UUID) *shared.ErrorResponse {
	var queries = model.New(s.db)
	var admins, err = queries.ListPlatformAdmins(ctx)
	if err != nil {
		return failed("errors.superadmin_save_failed")
	}
	if len(admins) <= 1 {
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.last_superadmin"}
	}
	for _, a := range admins {
		if a.ID == userId && shared.SUPERADMIN_EMAIL != "" && strings.EqualFold(a.Email, shared.SUPERADMIN_EMAIL) {
			return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.superadmin_from_env"}
		}
	}
	var removed int64
	removed, err = queries.RemovePlatformAdmin(ctx, userId)
	if err != nil {
		return failed("errors.superadmin_save_failed")
	}
	if removed == 0 {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.member_not_found"}
	}
	return nil
}
