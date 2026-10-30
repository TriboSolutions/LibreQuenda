package counciladmin

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"librequenda/auth"
	"librequenda/core/me"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"strconv"
	"sync"

	"github.com/angelbarreiros/Penguin/router/helpers"
	"github.com/google/uuid"
)

type counciladminHandler struct {
	db *sql.DB
}

var hOnce *sync.Once = new(sync.Once)
var hInstance *counciladminHandler

func CounciladminHandler(db *sql.DB) *counciladminHandler {
	hOnce.Do(func() {
		hInstance = &counciladminHandler{db: db}
	})
	return hInstance
}

// councilAdmin resolves the council in the path and checks the current user may manage it
// (its administrator, or a superadministrator). Otherwise it answers and returns false.
func (h *counciladminHandler) councilAdmin(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	var tenantId, _, ok = h.councilAdminUser(w, r)
	return tenantId, ok
}

// councilAdminUser is councilAdmin that also returns who is acting, for what they record.
func (h *counciladminHandler) councilAdminUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	var slug, err = helpers.GetStringPathValue("slug", r)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return uuid.Nil, uuid.Nil, false
	}
	var session, _ = auth.SessionFromContext(r.Context())
	var userId, userErr = me.CurrentUserId(r.Context(), h.db, session)
	if userErr != nil {
		helpers.SendI18NErrorResponse(w, r, userErr.Code, userErr.Message)
		return uuid.Nil, uuid.Nil, false
	}
	var queries = model.New(h.db)
	var tenant model.GetTenantForPlatformRow
	tenant, err = queries.GetTenantForPlatform(r.Context(), slug)
	if errors.Is(err, sql.ErrNoRows) {
		helpers.SendI18NErrorResponse(w, r, http.StatusNotFound, "errors.tenant_not_found")
		return uuid.Nil, uuid.Nil, false
	}
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusInternalServerError, "errors.forbidden")
		return uuid.Nil, uuid.Nil, false
	}
	var allowed bool
	allowed, err = queries.IsTenantAdmin(r.Context(), model.IsTenantAdminParams{UserID: userId, TenantID: tenant.ID})
	if err == nil && !allowed {
		allowed, err = queries.IsPlatformAdmin(r.Context(), userId)
	}
	if err != nil || !allowed {
		helpers.SendI18NErrorResponse(w, r, http.StatusForbidden, "errors.forbidden")
		return uuid.Nil, uuid.Nil, false
	}
	if !me.AdminSecondFactor(w, r, session) {
		return uuid.Nil, uuid.Nil, false
	}
	return tenant.ID, userId, true
}

// anyAdmin: geocoding is a tool for whoever edits facilities.
func (h *counciladminHandler) anyAdmin(w http.ResponseWriter, r *http.Request) bool {
	var session, _ = auth.SessionFromContext(r.Context())
	var userId, userErr = me.CurrentUserId(r.Context(), h.db, session)
	if userErr != nil {
		helpers.SendI18NErrorResponse(w, r, userErr.Code, userErr.Message)
		return false
	}
	var queries = model.New(h.db)
	var allowed, err = queries.IsPlatformAdmin(r.Context(), userId)
	if err == nil && !allowed {
		var roles []model.GetUserTenantRolesRow
		roles, err = queries.GetUserTenantRoles(r.Context(), userId)
		allowed = len(roles) > 0
	}
	if err != nil || !allowed {
		helpers.SendI18NErrorResponse(w, r, http.StatusForbidden, "errors.forbidden")
		return false
	}
	if !me.AdminSecondFactor(w, r, session) {
		return false
	}
	return true
}

func facilityId(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	var id, err = helpers.GetUUidPathValue("id", r)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return uuid.Nil, false
	}
	return id, true
}

func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(r.Body).Decode(into); err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_request_body")
		return false
	}
	return true
}

func send(w http.ResponseWriter, r *http.Request, data any, err *shared.ErrorResponse) {
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	if data == nil {
		helpers.SendNoContentResponse(w)
		return
	}
	helpers.SendSuccessResponse(w, data)
}

func (h *counciladminHandler) ListFacilities(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var response, err = GetCounciladminService(h.db).ListFacilities(r.Context(), tenantId)
	send(w, r, response, err)
}

func (h *counciladminHandler) CreateFacility(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var input FacilityInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetCounciladminService(h.db).CreateFacility(r.Context(), tenantId, input)
	send(w, r, response, err)
}

func (h *counciladminHandler) GetFacility(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var id uuid.UUID
	if id, ok = facilityId(w, r); !ok {
		return
	}
	var response, err = GetCounciladminService(h.db).GetFacility(r.Context(), tenantId, id)
	send(w, r, response, err)
}

func (h *counciladminHandler) UpdateFacility(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var id uuid.UUID
	if id, ok = facilityId(w, r); !ok {
		return
	}
	var input FacilityInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetCounciladminService(h.db).UpdateFacility(r.Context(), tenantId, id, input)
	send(w, r, response, err)
}

func (h *counciladminHandler) DeleteFacility(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var id uuid.UUID
	if id, ok = facilityId(w, r); !ok {
		return
	}
	send(w, r, nil, GetCounciladminService(h.db).DeleteFacility(r.Context(), tenantId, id))
}

func (h *counciladminHandler) ReplaceSchedule(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var id uuid.UUID
	if id, ok = facilityId(w, r); !ok {
		return
	}
	var input ScheduleInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetCounciladminService(h.db).ReplaceSchedule(r.Context(), tenantId, id, input)
	send(w, r, response, err)
}

func (h *counciladminHandler) Geocode(w http.ResponseWriter, r *http.Request) {
	if !h.anyAdmin(w, r) {
		return
	}
	var results, err = geocode(r.Context(), r.URL.Query().Get("q"), r.Header.Get("Accept-Language"))
	if err != nil {
		shared.Logger.Error("geocoding failed: %v", err)
		helpers.SendI18NErrorResponse(w, r, http.StatusBadGateway, "errors.geocode_failed")
		return
	}
	helpers.SendSuccessResponse(w, map[string]any{"results": results})
}

func (h *counciladminHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var id uuid.UUID
	if id, ok = facilityId(w, r); !ok {
		return
	}
	var response, err = GetCounciladminService(h.db).ListDevices(r.Context(), tenantId, id)
	send(w, r, response, err)
}

func (h *counciladminHandler) CreateDevice(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var id uuid.UUID
	if id, ok = facilityId(w, r); !ok {
		return
	}
	var input DeviceInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetCounciladminService(h.db).CreateDevice(r.Context(), tenantId, id, input)
	send(w, r, response, err)
}

func (h *counciladminHandler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var id uuid.UUID
	if id, ok = facilityId(w, r); !ok {
		return
	}
	var deviceId, err = helpers.GetUUidPathValue("deviceId", r)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return
	}
	send(w, r, nil, GetCounciladminService(h.db).DeleteDevice(r.Context(), tenantId, id, deviceId))
}

// --- active and closures ---------------------------------------------------

func (h *counciladminHandler) SetActive(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var id uuid.UUID
	if id, ok = facilityId(w, r); !ok {
		return
	}
	var input ActiveInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetCounciladminService(h.db).SetActive(r.Context(), tenantId, id, input)
	send(w, r, response, err)
}

func (h *counciladminHandler) PreviewClosure(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var id uuid.UUID
	if id, ok = facilityId(w, r); !ok {
		return
	}
	var input ClosureInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetCounciladminService(h.db).PreviewClosure(r.Context(), tenantId, id, input)
	send(w, r, response, err)
}

func (h *counciladminHandler) CreateClosure(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var id uuid.UUID
	if id, ok = facilityId(w, r); !ok {
		return
	}
	var input ClosureInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetCounciladminService(h.db).CreateClosure(r.Context(), tenantId, id, input)
	send(w, r, response, err)
}

func (h *counciladminHandler) DeleteClosure(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var id uuid.UUID
	if id, ok = facilityId(w, r); !ok {
		return
	}
	var closureId, err = helpers.GetUUidPathValue("closureId", r)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return
	}
	send(w, r, nil, GetCounciladminService(h.db).DeleteClosure(r.Context(), tenantId, id, closureId))
}

func indicatorsQuery(r *http.Request) IndicatorsQuery {
	var params = r.URL.Query()
	var query = IndicatorsQuery{From: params.Get("from"), To: params.Get("to"), Facility: params.Get("facility"), Compare: params.Get("compare")}
	if days := params.Get("days"); days != "" {
		var n, err = strconv.Atoi(days)
		if err != nil {
			n = -1 // refused by parse
		}
		query.Days = n
	}
	return query
}

func (h *counciladminHandler) GetIndicators(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var response, err = GetCounciladminService(h.db).Indicators(r.Context(), tenantId, indicatorsQuery(r))
	send(w, r, response, err)
}

// ExportIndicators: ?kind=summary|bookings, and excel=1 for Spanish-locale Excel.
func (h *counciladminHandler) ExportIndicators(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var query = indicatorsQuery(r)
	var kind = r.URL.Query().Get("kind")
	var write, err = GetCounciladminService(h.db).ExportIndicators(r.Context(), tenantId, query, kind, csvFormat{excel: r.URL.Query().Get("excel") == "1"})
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	var slug, _ = helpers.GetStringPathValue("slug", r)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-%s.csv"`, slug, kind))
	w.Header().Set("Cache-Control", "no-store")
	write(w)
}

// ExportCouncil: the council's full data as a ZIP of CSVs (see exportService.go);
// personal=1 adds the residents' personal data.
func (h *counciladminHandler) ExportCouncil(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var personal = r.URL.Query().Get("personal") == "1"
	var data, err = GetCounciladminService(h.db).ExportCouncil(r.Context(), tenantId, personal)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	if personal {
		// accountability (GDPR art. 5.2): who took the residents' data out, and when
		var session, _ = auth.SessionFromContext(r.Context())
		var userId, _ = me.CurrentUserId(r.Context(), h.db, session)
		shared.Logger.Info("council export %s with personal data (%d people) by user %s", data.tenant.Slug, len(data.people), userId)
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="librequenda-%s-%s.zip"`, data.tenant.Slug, data.generatedAt.Format(localDate)))
	w.Header().Set("Cache-Control", "no-store")
	if writeErr := data.WriteZip(r.Context(), w); writeErr != nil {
		shared.Logger.Error("council export %s: %s", data.tenant.Slug, writeErr.Error())
	}
}

// --- SMTP relay ------------------------------------------------------------

func (h *counciladminHandler) GetSmtp(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var response, err = GetCounciladminService(h.db).GetSmtp(r.Context(), tenantId)
	send(w, r, response, err)
}

func (h *counciladminHandler) SaveSmtp(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var input SmtpInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetCounciladminService(h.db).SaveSmtp(r.Context(), tenantId, input)
	send(w, r, response, err)
}

func (h *counciladminHandler) DeleteSmtp(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	send(w, r, nil, GetCounciladminService(h.db).DeleteSmtp(r.Context(), tenantId))
}

// TestSmtp emails the administrator through the relay in use, in their language.
func (h *counciladminHandler) TestSmtp(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var session, _ = auth.SessionFromContext(r.Context())
	var userId, userErr = me.CurrentUserId(r.Context(), h.db, session)
	if userErr != nil {
		helpers.SendI18NErrorResponse(w, r, userErr.Code, userErr.Message)
		return
	}
	var language = r.Header.Get("Accept-Language")
	if len(language) > 2 {
		language = language[:2]
	}
	var response, err = GetCounciladminService(h.db).TestSmtp(r.Context(), tenantId, userId, language)
	send(w, r, response, err)
}

// --- the council's booking window, which its facilities inherit ----------------

func (h *counciladminHandler) GetBookingRules(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var response, err = GetCounciladminService(h.db).GetBookingRules(r.Context(), tenantId)
	send(w, r, response, err)
}

func (h *counciladminHandler) SaveBookingRules(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var input BookingRulesDto
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetCounciladminService(h.db).SaveBookingRules(r.Context(), tenantId, input)
	send(w, r, response, err)
}

// SaveFacilityRules: the facility's booking window, from its Reserva tab.
func (h *counciladminHandler) SaveFacilityRules(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var id uuid.UUID
	if id, ok = facilityId(w, r); !ok {
		return
	}
	var input FacilityRulesInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetCounciladminService(h.db).SaveFacilityRules(r.Context(), tenantId, id, input)
	send(w, r, response, err)
}
