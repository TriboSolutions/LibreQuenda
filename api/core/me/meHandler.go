package me

import (
	"database/sql"
	"encoding/json"
	"librequenda/auth"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

type meHandler struct {
	db *sql.DB
}

var hMeOnce *sync.Once = new(sync.Once)
var hMeInstance *meHandler

func MeHandler(db *sql.DB) *meHandler {
	hMeOnce.Do(func() {
		hMeInstance = &meHandler{
			db: db,
		}
	})
	return hMeInstance
}

// GetMe returns the current user, creating or updating its app_user row from Zitadel.
func (h *meHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	var session, ok = auth.SessionFromContext(r.Context())
	if !ok {
		helpers.SendI18NErrorResponse(w, r, http.StatusUnauthorized, "errors.unauthorized")
		return
	}

	var response, err = GetMeService(h.db).SyncCurrentUser(r.Context(), session)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}

	helpers.SendSuccessResponse(w, response)
}

// UpdateProfile saves what LibreQuenda keeps about the user (date of birth, phone, photo, language, reminders).
func (h *meHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var session, ok = auth.SessionFromContext(r.Context())
	if !ok {
		helpers.SendI18NErrorResponse(w, r, http.StatusUnauthorized, "errors.unauthorized")
		return
	}
	var input ProfileInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_request_body")
		return
	}
	var response, err = GetMeService(h.db).UpdateProfile(r.Context(), session, input)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	helpers.SendSuccessResponse(w, response)
}

// GetPersonalQr: a fresh personal QR; the web asks again every minute.
func (h *meHandler) GetPersonalQr(w http.ResponseWriter, r *http.Request) {
	var session, ok = auth.SessionFromContext(r.Context())
	if !ok {
		helpers.SendI18NErrorResponse(w, r, http.StatusUnauthorized, "errors.unauthorized")
		return
	}
	var response, err = GetMeService(h.db).PersonalQr(r.Context(), session)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	helpers.SendSuccessResponse(w, response)
}

// Export: ?format=csv for a spreadsheet, JSON otherwise.
func (h *meHandler) Export(w http.ResponseWriter, r *http.Request) {
	var session, ok = auth.SessionFromContext(r.Context())
	if !ok {
		helpers.SendI18NErrorResponse(w, r, http.StatusUnauthorized, "errors.unauthorized")
		return
	}
	var response, err = GetMeService(h.db).Export(r.Context(), session)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	var name = "librequenda-" + response.GeneratedAt.Format("2006-01-02")
	w.Header().Set("Cache-Control", "no-store")
	if r.URL.Query().Get("format") == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.csv"`)
		writeExportCSV(w, response)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.json"`)
	helpers.SendSuccessResponse(w, response)
}

// DeleteAccount erases the account for good; the web signs out afterwards.
func (h *meHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	var session, ok = auth.SessionFromContext(r.Context())
	if !ok {
		helpers.SendI18NErrorResponse(w, r, http.StatusUnauthorized, "errors.unauthorized")
		return
	}
	reply(w, r, nil, GetMeService(h.db).DeleteAccount(r.Context(), session))
}
