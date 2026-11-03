package ratings

import (
	"database/sql"
	"encoding/json"
	"librequenda/auth"
	"librequenda/core/me"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router/helpers"
	"github.com/google/uuid"
)

type ratingsHandler struct {
	db *sql.DB
}

var hRatingsOnce *sync.Once = new(sync.Once)
var hRatingsInstance *ratingsHandler

func RatingsHandler(db *sql.DB) *ratingsHandler {
	hRatingsOnce.Do(func() {
		hRatingsInstance = &ratingsHandler{db: db}
	})
	return hRatingsInstance
}

// target answers the signed-in user and the booking in the path, or writes the error.
func (h *ratingsHandler) target(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	var bookingId, pathErr = helpers.GetUUidPathValue("id", r)
	if pathErr != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return uuid.Nil, uuid.Nil, false
	}
	var session, _ = auth.SessionFromContext(r.Context())
	var userId, err = me.CurrentUserId(r.Context(), h.db, session)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return uuid.Nil, uuid.Nil, false
	}
	return userId, bookingId, true
}

func (h *ratingsHandler) GetRating(w http.ResponseWriter, r *http.Request) {
	var userId, bookingId, ok = h.target(w, r)
	if !ok {
		return
	}
	var response, err = GetRatingsService(h.db).GetRating(r.Context(), userId, bookingId)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	helpers.SendSuccessResponse(w, response)
}

func (h *ratingsHandler) Rate(w http.ResponseWriter, r *http.Request) {
	var userId, bookingId, ok = h.target(w, r)
	if !ok {
		return
	}
	var input RateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_request_body")
		return
	}
	if key := input.Validate(); key != "" {
		helpers.SendI18NErrorResponse(w, r, http.StatusUnprocessableEntity, key)
		return
	}
	var response, err = GetRatingsService(h.db).Rate(r.Context(), userId, bookingId, input)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	helpers.SendSuccessResponse(w, response)
}
