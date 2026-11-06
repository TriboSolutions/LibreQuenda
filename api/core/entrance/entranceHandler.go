package entrance

import (
	"database/sql"
	"encoding/json"
	"librequenda/auth"
	"librequenda/shared"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

type entranceHandler struct {
	db *sql.DB
}

var hOnce *sync.Once = new(sync.Once)
var hInstance *entranceHandler

func EntranceHandler(db *sql.DB) *entranceHandler {
	hOnce.Do(func() {
		hInstance = &entranceHandler{db: db}
	})
	return hInstance
}

// place reads the council and facility slugs and the signed-in session; it answers the error itself.
func place(w http.ResponseWriter, r *http.Request) (auth.Session, string, string, bool) {
	var session, ok = auth.SessionFromContext(r.Context())
	if !ok {
		helpers.SendI18NErrorResponse(w, r, http.StatusUnauthorized, "errors.unauthorized")
		return session, "", "", false
	}
	var slug, slugErr = helpers.GetStringPathValue("slug", r)
	var facility, facilityErr = helpers.GetStringPathValue("facility", r)
	if slugErr != nil || facilityErr != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return session, "", "", false
	}
	return session, slug, facility, true
}

func (h *entranceHandler) Get(w http.ResponseWriter, r *http.Request) {
	var session, slug, facility, ok = place(w, r)
	if !ok {
		return
	}
	var response, err = GetEntranceService(h.db).Get(r.Context(), session, slug, facility)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	helpers.SendSuccessResponse(w, response)
}

func (h *entranceHandler) CheckIn(w http.ResponseWriter, r *http.Request) {
	var session, slug, facility, ok = place(w, r)
	if !ok {
		return
	}
	var body CheckinInput
	if err := json.NewDecoder(shared.JSONBody(w, r)).Decode(&body); err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_request_body")
		return
	}
	var response, err = GetEntranceService(h.db).CheckIn(r.Context(), session, slug, facility, body)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	helpers.SendSuccessResponse(w, response)
}

// Nearby: the location goes in the body, so it stays out of the access logs.
func (h *entranceHandler) Nearby(w http.ResponseWriter, r *http.Request) {
	var body NearbyInput
	if err := json.NewDecoder(shared.JSONBody(w, r)).Decode(&body); err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_request_body")
		return
	}
	var response, err = GetEntranceService(h.db).Nearby(r.Context(), body)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	helpers.SendSuccessResponse(w, response)
}
