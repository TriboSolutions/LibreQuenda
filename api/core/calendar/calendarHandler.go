package calendar

import (
	"database/sql"
	"librequenda/auth"
	"librequenda/branding"
	"librequenda/core/me"
	"librequenda/shared"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/angelbarreiros/Penguin/router/helpers"
	"github.com/google/uuid"
)

type calendarHandler struct {
	db *sql.DB
}

var hCalendarOnce *sync.Once = new(sync.Once)
var hCalendarInstance *calendarHandler

func CalendarHandler(db *sql.DB) *calendarHandler {
	hCalendarOnce.Do(func() {
		hCalendarInstance = &calendarHandler{
			db: db,
		}
	})
	return hCalendarInstance
}

// currentUser answers the signed-in user's id, or writes the error and returns false.
func (h *calendarHandler) currentUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	var session, _ = auth.SessionFromContext(r.Context())
	var userId, err = me.CurrentUserId(r.Context(), h.db, session)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return uuid.Nil, false
	}
	return userId, true
}

func reply(w http.ResponseWriter, r *http.Request, data *CalendarFeedDto, err *shared.ErrorResponse) {
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	w.Header().Set("Cache-Control", "no-store") // the address is a secret
	helpers.SendSuccessResponse(w, data)
}

func (h *calendarHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	var userId, ok = h.currentUser(w, r)
	if !ok {
		return
	}
	var response, err = GetCalendarService(h.db).Get(r.Context(), userId)
	reply(w, r, response, err)
}

func (h *calendarHandler) ResetFeed(w http.ResponseWriter, r *http.Request) {
	var userId, ok = h.currentUser(w, r)
	if !ok {
		return
	}
	var response, err = GetCalendarService(h.db).Reset(r.Context(), userId)
	reply(w, r, response, err)
}

func (h *calendarHandler) TurnOffFeed(w http.ResponseWriter, r *http.Request) {
	var userId, ok = h.currentUser(w, r)
	if !ok {
		return
	}
	if err := GetCalendarService(h.db).TurnOff(r.Context(), userId); err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	helpers.SendNoContentResponse(w)
}

// Ics serves /calendar/<token>.ics to calendar apps, which cannot sign in: the
// token in the address is the only key.
func (h *calendarHandler) Ics(w http.ResponseWriter, r *http.Request) {
	var file, _ = helpers.GetStringPathValue("file", r)
	var token, parseErr = uuid.Parse(strings.TrimSuffix(file, ".ics"))
	if parseErr != nil || !strings.HasSuffix(file, ".ics") {
		helpers.SendI18NErrorResponse(w, r, http.StatusNotFound, errCalendarNotFound.Message)
		return
	}
	var rows, err = GetCalendarService(h.db).Feed(r.Context(), token)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", `inline; filename="librequenda.ics"`)
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if r.Method == http.MethodHead {
		return
	}
	if writeErr := writeICS(w, rows, branding.Name(r.Context(), h.db), shared.PUBLIC_URL+"/reservas", time.Now()); writeErr != nil {
		shared.Logger.Error("calendar feed write failed: %v", writeErr)
	}
}
