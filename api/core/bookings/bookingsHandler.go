package bookings

import (
	"database/sql"
	"encoding/json"
	"librequenda/auth"
	"librequenda/core/me"
	"librequenda/shared"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

type bookingsHandler struct {
	db *sql.DB
}

var hBookingsOnce *sync.Once = new(sync.Once)
var hBookingsInstance *bookingsHandler

func BookingsHandler(db *sql.DB) *bookingsHandler {
	hBookingsOnce.Do(func() {
		hBookingsInstance = &bookingsHandler{
			db: db,
		}
	})
	return hBookingsInstance
}

func (h *bookingsHandler) GetMyBookings(w http.ResponseWriter, r *http.Request) {
	var session, _ = auth.SessionFromContext(r.Context())
	var userId, userErr = me.CurrentUserId(r.Context(), h.db, session)
	if userErr != nil {
		helpers.SendI18NErrorResponse(w, r, userErr.Code, userErr.Message, userErr.Args...)
		return
	}

	// a page of the list (?page=&pageSize=, 100 by default); a bad value is a 400
	var page, pageErr = shared.Page(r)
	if pageErr != nil {
		helpers.SendI18NErrorResponse(w, r, pageErr.Code, pageErr.Message)
		return
	}
	var response, err = GetBookingsService(h.db).GetMyBookings(r.Context(), userId, page)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	helpers.SendSuccessResponse(w, response)
}

func (h *bookingsHandler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	var body CreateBookingDto
	if decodeErr := json.NewDecoder(shared.JSONBody(w, r)).Decode(&body); decodeErr != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_request_body")
		return
	}
	if errs := body.Validate(); len(errs) > 0 {
		helpers.SendValidationErrorResponse(w, errs)
		return
	}
	if !body.AcceptRules {
		helpers.SendI18NErrorResponse(w, r, http.StatusUnprocessableEntity, "errors.rules_not_accepted")
		return
	}

	var session, _ = auth.SessionFromContext(r.Context())
	var userId, userErr = me.CurrentUserId(r.Context(), h.db, session)
	if userErr != nil {
		helpers.SendI18NErrorResponse(w, r, userErr.Code, userErr.Message, userErr.Args...)
		return
	}

	var response, err = GetBookingsService(h.db).CreateBooking(r.Context(), userId, body)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	helpers.SendSuccessResponse(w, response)
}

func (h *bookingsHandler) ChangeBooking(w http.ResponseWriter, r *http.Request) {
	var bookingId, idErr = helpers.GetUUidPathValue("id", r)
	if idErr != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return
	}
	var body ChangeBookingDto
	if decodeErr := json.NewDecoder(shared.JSONBody(w, r)).Decode(&body); decodeErr != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_request_body")
		return
	}
	if errs := body.Validate(); len(errs) > 0 {
		helpers.SendValidationErrorResponse(w, errs)
		return
	}

	var session, _ = auth.SessionFromContext(r.Context())
	var userId, userErr = me.CurrentUserId(r.Context(), h.db, session)
	if userErr != nil {
		helpers.SendI18NErrorResponse(w, r, userErr.Code, userErr.Message, userErr.Args...)
		return
	}

	var response, err = GetBookingsService(h.db).ChangeBooking(r.Context(), userId, bookingId, body)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	helpers.SendSuccessResponse(w, response)
}

func (h *bookingsHandler) CancelBooking(w http.ResponseWriter, r *http.Request) {
	var bookingId, idErr = helpers.GetUUidPathValue("id", r)
	if idErr != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return
	}

	var session, _ = auth.SessionFromContext(r.Context())
	var userId, userErr = me.CurrentUserId(r.Context(), h.db, session)
	if userErr != nil {
		helpers.SendI18NErrorResponse(w, r, userErr.Code, userErr.Message, userErr.Args...)
		return
	}

	var err = GetBookingsService(h.db).CancelBooking(r.Context(), userId, bookingId)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	helpers.SendNoContentResponse(w)
}
