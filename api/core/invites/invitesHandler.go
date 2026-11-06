package invites

import (
	"database/sql"
	"encoding/json"
	"librequenda/auth"
	"librequenda/core/me"
	"librequenda/shared"
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
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
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
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
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
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
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
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
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
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
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
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	helpers.SendNoContentResponse(w)
}

// --- public ----------------------------------------------------------------

func (h *invitesHandler) GetPublicInvite(w http.ResponseWriter, r *http.Request) {
	var token, ok = uuidPath(w, r, "token")
	if !ok {
		return
	}
	var response, err = GetInvitesService(h.db).GetPublicInvite(r.Context(), token)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	helpers.SendSuccessResponse(w, response)
}

func (h *invitesHandler) JoinAsGuest(w http.ResponseWriter, r *http.Request) {
	var token, ok = uuidPath(w, r, "token")
	if !ok {
		return
	}
	var body GuestJoinDto
	if decodeErr := json.NewDecoder(shared.JSONBody(w, r)).Decode(&body); decodeErr != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_request_body")
		return
	}
	if key := body.Validate(); key != "" {
		helpers.SendI18NErrorResponse(w, r, http.StatusUnprocessableEntity, key)
		return
	}
	if err := GetInvitesService(h.db).JoinAsGuest(r.Context(), token, body); err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	helpers.SendNoContentResponse(w)
}

// JoinPublic: a signed-in person joins a public booking found in «For you».
func (h *invitesHandler) JoinPublic(w http.ResponseWriter, r *http.Request) {
	var bookingId, ok = uuidPath(w, r, "id")
	if !ok {
		return
	}
	var userId, signedIn = h.currentUser(w, r)
	if !signedIn {
		return
	}
	if err := GetInvitesService(h.db).JoinPublic(r.Context(), userId, bookingId); err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	helpers.SendNoContentResponse(w)
}

// SetPublic: {"public": true} lets anyone signed in find and join the owner's booking.
func (h *invitesHandler) SetPublic(w http.ResponseWriter, r *http.Request) {
	var bookingId, ok = uuidPath(w, r, "id")
	if !ok {
		return
	}
	var userId, signedIn = h.currentUser(w, r)
	if !signedIn {
		return
	}
	var input struct {
		Public      bool    `json:"public"`
		Description *string `json:"public_description"`
	}
	if err := json.NewDecoder(shared.JSONBody(w, r)).Decode(&input); err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_request_body")
		return
	}
	if err := GetInvitesService(h.db).SetPublic(r.Context(), userId, bookingId, input.Public, input.Description); err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	helpers.SendNoContentResponse(w)
}
