package issues

import (
	"database/sql"
	"encoding/json"
	"librequenda/auth"
	"librequenda/core/issue"
	"librequenda/shared"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

type issuesHandler struct {
	db *sql.DB
}

var hOnce *sync.Once = new(sync.Once)
var hInstance *issuesHandler

func IssuesHandler(db *sql.DB) *issuesHandler {
	hOnce.Do(func() {
		hInstance = &issuesHandler{db: db}
	})
	return hInstance
}

func (h *issuesHandler) Report(w http.ResponseWriter, r *http.Request) {
	var session, ok = auth.SessionFromContext(r.Context())
	if !ok {
		helpers.SendI18NErrorResponse(w, r, http.StatusUnauthorized, "errors.unauthorized")
		return
	}
	var body issue.Report
	if err := json.NewDecoder(shared.JSONBody(w, r)).Decode(&body); err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_request_body")
		return
	}
	var response, err = GetIssuesService(h.db).Report(r.Context(), session, body)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	helpers.SendSuccessResponse(w, response)
}
