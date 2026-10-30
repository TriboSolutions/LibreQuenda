package tenants

import (
	"database/sql"
	"encoding/json"
	"librequenda/auth"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

type tenantsHandler struct {
	db *sql.DB
}

var hTenantsOnce *sync.Once = new(sync.Once)
var hTenantsInstance *tenantsHandler

func TenantsHandler(db *sql.DB) *tenantsHandler {
	hTenantsOnce.Do(func() {
		hTenantsInstance = &tenantsHandler{
			db: db,
		}
	})
	return hTenantsInstance
}

func (h *tenantsHandler) GetTenants(w http.ResponseWriter, r *http.Request) {
	var response, err = GetTenantsService(h.db).GetTenants(r.Context())
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	helpers.SendSuccessResponse(w, response)
}

// GetFacilities returns every active facility of every council, for the home page.
func (h *tenantsHandler) GetFacilities(w http.ResponseWriter, r *http.Request) {
	var response, err = GetTenantsService(h.db).GetFacilities(r.Context())
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	helpers.SendSuccessResponse(w, response)
}

// GetFacility returns one facility's page: details, the day's slots (?date=) and its weekly schedule.
func (h *tenantsHandler) GetFacility(w http.ResponseWriter, r *http.Request) {
	var slug, slugErr = helpers.GetStringPathValue("slug", r)
	var facility, facilityErr = helpers.GetStringPathValue("facility", r)
	if slugErr != nil || facilityErr != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return
	}
	var response, err = GetTenantsService(h.db).GetFacilityPage(r.Context(), slug, facility, r.URL.Query().Get("date"))
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	helpers.SendSuccessResponse(w, response)
}

// GetDay returns every active facility of the council with its slots for one
// day (?date=YYYY-MM-DD, default today in the council's timezone).
func (h *tenantsHandler) GetDay(w http.ResponseWriter, r *http.Request) {
	var slug, slugErr = helpers.GetStringPathValue("slug", r)
	if slugErr != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return
	}

	var response, err = GetTenantsService(h.db).GetDay(r.Context(), slug, r.URL.Query().Get("date"))
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	helpers.SendSuccessResponse(w, response)
}

// Recommendations: the signed-in resident's "For you" on the home page.
func (h *tenantsHandler) Recommendations(w http.ResponseWriter, r *http.Request) {
	var session, ok = auth.SessionFromContext(r.Context())
	if !ok {
		helpers.SendI18NErrorResponse(w, r, http.StatusUnauthorized, "errors.unauthorized")
		return
	}
	// POST may carry where the person is (the home page's «near me»), in the body so it stays out of the logs
	var position Position
	if r.Method == http.MethodPost {
		if decodeErr := json.NewDecoder(r.Body).Decode(&position); decodeErr != nil {
			helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_request_body")
			return
		}
	}
	var response, err = GetTenantsService(h.db).Recommendations(r.Context(), session, position)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	helpers.SendSuccessResponse(w, response)
}
