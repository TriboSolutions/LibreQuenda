package notifications

import (
	"database/sql"
	"librequenda/auth"
	"librequenda/core/me"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

type notificationsHandler struct {
	db *sql.DB
}

var hNotificationsOnce *sync.Once = new(sync.Once)
var hNotificationsInstance *notificationsHandler

func NotificationsHandler(db *sql.DB) *notificationsHandler {
	hNotificationsOnce.Do(func() {
		hNotificationsInstance = &notificationsHandler{
			db: db,
		}
	})
	return hNotificationsInstance
}

// GetNotifications is what the web polls: the latest notices and how many are unread.
func (h *notificationsHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	var session, _ = auth.SessionFromContext(r.Context())
	var userId, userErr = me.CurrentUserId(r.Context(), h.db, session)
	if userErr != nil {
		helpers.SendI18NErrorResponse(w, r, userErr.Code, userErr.Message)
		return
	}
	var response, err = GetNotificationsService(h.db).GetNotifications(r.Context(), userId)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	helpers.SendSuccessResponse(w, response)
}

func (h *notificationsHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	var session, _ = auth.SessionFromContext(r.Context())
	var userId, userErr = me.CurrentUserId(r.Context(), h.db, session)
	if userErr != nil {
		helpers.SendI18NErrorResponse(w, r, userErr.Code, userErr.Message)
		return
	}
	if err := GetNotificationsService(h.db).MarkAllRead(r.Context(), userId); err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	helpers.SendNoContentResponse(w)
}
