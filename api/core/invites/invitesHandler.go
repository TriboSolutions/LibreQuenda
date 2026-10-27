package invites

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

type invitesHandler struct {
	db *sql.DB
}

var hInvitesOnce *sync.Once = new(sync.Once)
var hInvitesInstance *invitesHandler

func InvitesHandler(db *sql.DB) *invitesHandler {
	hInvitesOnce.Do(func() {
		hInvitesInstance = &invitesHandler{
			db: db,
		}
	})
	return hInvitesInstance
}

// currentUser answers the signed-in user's id, or writes the error and returns false.
func (h *invitesHandler) currentUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	var session, _ = auth.SessionFromContext(r.Context())
	var userId, err = me.CurrentUserId(r.Context(), h.db, session)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return uuid.Nil, false
	}
	return userId, true
}

func uuidPath(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	var id, err = helpers.GetUUidPathValue(name, r)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return uuid.Nil, false
	}
	return id, true
}

// --- owner -----------------------------------------------------------------

func (h *invitesHandler) GetInvite(w http.ResponseWriter, r *http.Request) {
	var bookingId, ok = uuidPath(w, r, "id")
	if !ok {
		return
	}
	var userId, signedIn = h.currentUser(w, r)
	if !signedIn {
		return
	}
	var response, err = GetInvitesService(h.db).GetInvite(r.Context(), userId, bookingId)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	helpers.SendSuccessResponse(w, response)
}

func (h *invitesHandler) ShareInvite(w http.ResponseWriter, r *http.Request) {
	var bookingId, ok = uuidPath(w, r, "id")
	if !ok {
		return
	}
	var userId, signedIn = h.currentUser(w, r)
	if !signedIn {
		return
	}
	var response, err = GetInvitesService(h.db).ShareInvite(r.Context(), userId, bookingId)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	helpers.SendSuccessResponse(w, response)
}

func (h *invitesHandler) RemoveParticipant(w http.ResponseWriter, r *http.Request) {
	var bookingId, ok = uuidPath(w, r, "id")
	if !ok {
		return
	}
	var participantId, okParticipant = uuidPath(w, r, "participantId")
	if !okParticipant {
		return
	}
	var userId, signedIn = h.currentUser(w, r)
	if !signedIn {
		return
	}
	if err := GetInvitesService(h.db).RemoveParticipant(r.Context(), userId, bookingId, participantId); err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	helpers.SendNoContentResponse(w)
}

// --- participant -----------------------------------------------------------

func (h *invitesHandler) Leave(w http.ResponseWriter, r *http.Request) {
	var bookingId, ok = uuidPath(w, r, "id")
	if !ok {
		return
	}
	var userId, signedIn = h.currentUser(w, r)
	if !signedIn {
		return
	}
	if err := GetInvitesService(h.db).Leave(r.Context(), userId, bookingId); err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	helpers.SendNoContentResponse(w)
}

func (h *invitesHandler) JoinAsUser(w http.ResponseWriter, r *http.Request) {
	var token, ok = uuidPath(w, r, "token")
	if !ok {
		return
	}
	var userId, signedIn = h.currentUser(w, r)
	if !signedIn {
		return
	}
	if err := GetInvitesService(h.db).JoinAsUser(r.Context(), userId, token); err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	helpers.SendNoContentResponse(w)
}

// --- public ----------------------------------------------------------------

func (h *invitesHandler) GetPublicInvite(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}
	var token, ok = uuidPath(w, r, "token")
	if !ok {
		return
	}
	var response, err = GetInvitesService(h.db).GetPublicInvite(r.Context(), token)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	helpers.SendSuccessResponse(w, response)
}

func (h *invitesHandler) JoinAsGuest(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}
	var token, ok = uuidPath(w, r, "token")
	if !ok {
		return
	}
	var body GuestJoinDto
	if decodeErr := json.NewDecoder(r.Body).Decode(&body); decodeErr != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_request_body")
		return
	}
	if key := body.Validate(); key != "" {
		helpers.SendI18NErrorResponse(w, r, http.StatusUnprocessableEntity, key)
		return
	}
	if err := GetInvitesService(h.db).JoinAsGuest(r.Context(), token, body); err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return
	}
	helpers.SendNoContentResponse(w)
}
