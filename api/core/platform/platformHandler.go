package platform

import (
	"database/sql"
	"encoding/json"
	"librequenda/auth"
	"librequenda/core/me"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router/helpers"
	"github.com/google/uuid"
)

type platformHandler struct {
	db *sql.DB
}

var hPlatformOnce *sync.Once = new(sync.Once)
var hPlatformInstance *platformHandler

func PlatformHandler(db *sql.DB) *platformHandler {
	hPlatformOnce.Do(func() {
		hPlatformInstance = &platformHandler{
			db: db,
		}
	})
	return hPlatformInstance
}

// superadmin returns the current user's id, or answers 403 and returns false.
func (h *platformHandler) superadmin(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	var session, _ = auth.SessionFromContext(r.Context())
	var userId, userErr = me.CurrentUserId(r.Context(), h.db, session)
	if userErr != nil {
		helpers.SendI18NErrorResponse(w, r, userErr.Code, userErr.Message)
		return uuid.Nil, false
	}
	var allowed, err = model.New(h.db).IsPlatformAdmin(r.Context(), userId)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusInternalServerError, "errors.forbidden")
		return uuid.Nil, false
	}
	if !allowed {
		helpers.SendI18NErrorResponse(w, r, http.StatusForbidden, "errors.forbidden")
		return uuid.Nil, false
	}
	if !me.AdminSecondFactor(w, r, session) {
		return uuid.Nil, false
	}
	return userId, true
}

// councilAdmin lets in a superadministrator or an administrator of the council in
// the path, for that council only. It returns the user's id and whether they are a
// superadministrator, or answers and returns false.
func (h *platformHandler) councilAdmin(w http.ResponseWriter, r *http.Request, slug string) (uuid.UUID, bool, bool) {
	var session, _ = auth.SessionFromContext(r.Context())
	var userId, userErr = me.CurrentUserId(r.Context(), h.db, session)
	if userErr != nil {
		helpers.SendI18NErrorResponse(w, r, userErr.Code, userErr.Message)
		return uuid.Nil, false, false
	}
	var queries = model.New(h.db)
	var isSuperadmin, err = queries.IsPlatformAdmin(r.Context(), userId)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusInternalServerError, "errors.forbidden")
		return uuid.Nil, false, false
	}
	if isSuperadmin {
		if !me.AdminSecondFactor(w, r, session) {
			return uuid.Nil, false, false
		}
		return userId, true, true
	}
	var tenant model.GetTenantForPlatformRow
	tenant, err = queries.GetTenantForPlatform(r.Context(), slug)
	var allowed bool
	if err == nil {
		allowed, err = queries.IsTenantAdmin(r.Context(), model.IsTenantAdminParams{UserID: userId, TenantID: tenant.ID})
	}
	if err != nil || !allowed {
		// a council that does not exist answers the same as one that is not theirs
		helpers.SendI18NErrorResponse(w, r, http.StatusForbidden, "errors.forbidden")
		return uuid.Nil, false, false
	}
	if !me.AdminSecondFactor(w, r, session) {
		return uuid.Nil, false, false
	}
	return userId, false, true
}

func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(r.Body).Decode(into); err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_request_body")
		return false
	}
	return true
}

func pathSlug(w http.ResponseWriter, r *http.Request) (string, bool) {
	var slug, err = helpers.GetStringPathValue("slug", r)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return "", false
	}
	return slug, true
}

func pathUUID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	var id, err = helpers.GetUUidPathValue(name, r)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return uuid.Nil, false
	}
	return id, true
}

func send(w http.ResponseWriter, r *http.Request, data any, err *shared.ErrorResponse) {
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	if data == nil {
		helpers.SendNoContentResponse(w)
		return
	}
	helpers.SendSuccessResponse(w, data)
}

func (h *platformHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.superadmin(w, r); !ok {
		return
	}
	var response, err = GetPlatformService(h.db).ListTenants(r.Context())
	send(w, r, response, err)
}

func (h *platformHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var userId, ok = h.superadmin(w, r)
	if !ok {
		return
	}
	var input TenantInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetPlatformService(h.db).CreateTenant(r.Context(), userId, input)
	send(w, r, response, err)
}

// GetTenant, and the council's administrators below: its own administrators too.
func (h *platformHandler) GetTenant(w http.ResponseWriter, r *http.Request) {
	var slug, ok = pathSlug(w, r)
	if !ok {
		return
	}
	if _, _, ok = h.councilAdmin(w, r, slug); !ok {
		return
	}
	var response, err = GetPlatformService(h.db).GetTenant(r.Context(), slug)
	send(w, r, response, err)
}

// UpdateTenant: a council's administrators edit its details; only a
// superadministrator switches the whole council on or off.
func (h *platformHandler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	var slug, ok = pathSlug(w, r)
	if !ok {
		return
	}
	var isSuperadmin bool
	if _, isSuperadmin, ok = h.councilAdmin(w, r, slug); !ok {
		return
	}
	var input TenantInput
	if !decode(w, r, &input) {
		return
	}
	if !isSuperadmin {
		input.IsActive = nil
	}
	var response, err = GetPlatformService(h.db).UpdateTenant(r.Context(), slug, input)
	send(w, r, response, err)
}

func (h *platformHandler) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.superadmin(w, r); !ok {
		return
	}
	var slug, ok = pathSlug(w, r)
	if !ok {
		return
	}
	send(w, r, nil, GetPlatformService(h.db).DeleteTenant(r.Context(), slug))
}

func (h *platformHandler) InviteTenantAdmin(w http.ResponseWriter, r *http.Request) {
	var slug, ok = pathSlug(w, r)
	if !ok {
		return
	}
	var userId uuid.UUID
	if userId, _, ok = h.councilAdmin(w, r, slug); !ok {
		return
	}
	var input EmailInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetPlatformService(h.db).InviteTenantAdmin(r.Context(), userId, slug, input.Email)
	send(w, r, response, err)
}

func (h *platformHandler) DeleteInvitation(w http.ResponseWriter, r *http.Request) {
	var slug, ok = pathSlug(w, r)
	if !ok {
		return
	}
	if _, _, ok = h.councilAdmin(w, r, slug); !ok {
		return
	}
	var id uuid.UUID
	if id, ok = pathUUID(w, r, "id"); !ok {
		return
	}
	send(w, r, nil, GetPlatformService(h.db).DeleteInvitation(r.Context(), slug, id))
}

func (h *platformHandler) RemoveTenantAdmin(w http.ResponseWriter, r *http.Request) {
	var slug, ok = pathSlug(w, r)
	if !ok {
		return
	}
	if _, _, ok = h.councilAdmin(w, r, slug); !ok {
		return
	}
	var userId uuid.UUID
	if userId, ok = pathUUID(w, r, "userId"); !ok {
		return
	}
	send(w, r, nil, GetPlatformService(h.db).RemoveTenantAdmin(r.Context(), slug, userId))
}

func (h *platformHandler) ListSuperadmins(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.superadmin(w, r); !ok {
		return
	}
	var response, err = GetPlatformService(h.db).ListSuperadmins(r.Context())
	send(w, r, response, err)
}

func (h *platformHandler) AddSuperadmin(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.superadmin(w, r); !ok {
		return
	}
	var input EmailInput
	if !decode(w, r, &input) {
		return
	}
	send(w, r, nil, GetPlatformService(h.db).AddSuperadmin(r.Context(), input.Email))
}

func (h *platformHandler) RemoveSuperadmin(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.superadmin(w, r); !ok {
		return
	}
	var userId, ok = pathUUID(w, r, "userId")
	if !ok {
		return
	}
	send(w, r, nil, GetPlatformService(h.db).RemoveSuperadmin(r.Context(), userId))
}

// Tags: the catalogue is public; only a superadministrator changes it.

func (h *platformHandler) TagCatalogue(w http.ResponseWriter, r *http.Request) {
	var response, err = GetPlatformService(h.db).TagCatalogue(r.Context())
	send(w, r, response, err)
}

func (h *platformHandler) CreateTagCategory(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.superadmin(w, r); !ok {
		return
	}
	var input TagCategoryInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetPlatformService(h.db).CreateTagCategory(r.Context(), input)
	send(w, r, response, err)
}

func (h *platformHandler) UpdateTagCategory(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.superadmin(w, r); !ok {
		return
	}
	var id, ok = pathUUID(w, r, "id")
	if !ok {
		return
	}
	var input TagCategoryInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetPlatformService(h.db).UpdateTagCategory(r.Context(), id, input)
	send(w, r, response, err)
}

func (h *platformHandler) DeleteTagCategory(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.superadmin(w, r); !ok {
		return
	}
	var id, ok = pathUUID(w, r, "id")
	if !ok {
		return
	}
	var response, err = GetPlatformService(h.db).DeleteTagCategory(r.Context(), id)
	send(w, r, response, err)
}

func (h *platformHandler) CreateTag(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.superadmin(w, r); !ok {
		return
	}
	var input TagInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetPlatformService(h.db).CreateTag(r.Context(), input)
	send(w, r, response, err)
}

func (h *platformHandler) UpdateTag(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.superadmin(w, r); !ok {
		return
	}
	var id, ok = pathUUID(w, r, "id")
	if !ok {
		return
	}
	var input TagInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetPlatformService(h.db).UpdateTag(r.Context(), id, input)
	send(w, r, response, err)
}

func (h *platformHandler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.superadmin(w, r); !ok {
		return
	}
	var id, ok = pathUUID(w, r, "id")
	if !ok {
		return
	}
	var response, err = GetPlatformService(h.db).DeleteTag(r.Context(), id)
	send(w, r, response, err)
}

// Branding: the installation's name and logo are public; only a superadministrator changes them.

func (h *platformHandler) Branding(w http.ResponseWriter, r *http.Request) {
	var response, err = GetPlatformService(h.db).Branding(r.Context())
	send(w, r, response, err)
}

func (h *platformHandler) UpdateBranding(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.superadmin(w, r); !ok {
		return
	}
	var input BrandingInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetPlatformService(h.db).UpdateBranding(r.Context(), input)
	send(w, r, response, err)
}
