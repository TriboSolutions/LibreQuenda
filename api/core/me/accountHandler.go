package me

import (
	"encoding/json"
	"librequenda/auth"
	"librequenda/shared"
	"net/http"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

// subject: the Zitadel user of the token. Account changes always apply to it.
func subject(w http.ResponseWriter, r *http.Request) (string, bool) {
	var session, ok = auth.SessionFromContext(r.Context())
	if !ok || session.Subject == "" {
		helpers.SendI18NErrorResponse(w, r, http.StatusUnauthorized, "errors.unauthorized")
		return "", false
	}
	return session.Subject, true
}

func decodeInto(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(shared.JSONBody(w, r)).Decode(into); err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_request_body")
		return false
	}
	return true
}

func reply(w http.ResponseWriter, r *http.Request, data any, err *shared.ErrorResponse) {
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	if data == nil {
		helpers.SendNoContentResponse(w)
		return
	}
	helpers.SendSuccessResponse(w, data)
}

func (h *meHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	var userId, ok = subject(w, r)
	if !ok {
		return
	}
	var response, err = GetMeService(h.db).GetAccount(r.Context(), userId)
	reply(w, r, response, err)
}

func (h *meHandler) SetName(w http.ResponseWriter, r *http.Request) {
	var userId, ok = subject(w, r)
	var input NameInput
	if !ok || !decodeInto(w, r, &input) {
		return
	}
	reply(w, r, nil, GetMeService(h.db).SetName(r.Context(), userId, input))
}

func (h *meHandler) SetPassword(w http.ResponseWriter, r *http.Request) {
	var userId, ok = subject(w, r)
	var input PasswordInput
	if !ok || !decodeInto(w, r, &input) {
		return
	}
	reply(w, r, nil, GetMeService(h.db).SetPassword(r.Context(), userId, input))
}

func (h *meHandler) SetEmail(w http.ResponseWriter, r *http.Request) {
	var userId, ok = subject(w, r)
	var input EmailInput
	if !ok || !decodeInto(w, r, &input) {
		return
	}
	reply(w, r, nil, GetMeService(h.db).SetEmail(r.Context(), userId, input))
}

func (h *meHandler) ResendEmailCode(w http.ResponseWriter, r *http.Request) {
	var userId, ok = subject(w, r)
	if !ok {
		return
	}
	reply(w, r, nil, GetMeService(h.db).ResendEmailCode(r.Context(), userId))
}

func (h *meHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var userId, ok = subject(w, r)
	var input VerifyEmailInput
	if !ok || !decodeInto(w, r, &input) {
		return
	}
	reply(w, r, nil, GetMeService(h.db).VerifyEmail(r.Context(), userId, input))
}
