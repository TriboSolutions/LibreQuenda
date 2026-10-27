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
