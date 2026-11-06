package analytics

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"librequenda/shared"

	"github.com/google/uuid"
)

// AskInput: the question, from the indicators' assistant.
type AskInput struct {
	Question string `json:"question"`
}

// Answering: Ask for a handler, its refusals as the API answers them (the
// language model's own failures are passed on as they come, in English).
func Answering(ctx context.Context, db *sql.DB, scope Scope, who string, userId uuid.UUID, question string) (*Answer, *shared.ErrorResponse) {
	var answer, err = Ask(ctx, db, scope, who, question)
	switch {
	case err == nil:
		for _, step := range answer.Steps {
			shared.Logger.Info("analytics: user %s ran: %s", userId, step.SQL)
		}
		return answer, nil
	case errors.Is(err, ErrQuestion):
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.ai_question"}
	case errors.Is(err, ErrOffTopic):
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.ai_off_topic"}
	case errors.Is(err, ErrNotConfigured):
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.ai_not_configured"}
	case errors.Is(err, ErrDisabled):
		return nil, &shared.ErrorResponse{Code: http.StatusServiceUnavailable, Message: "errors.analytics_disabled"}
	}
	shared.Logger.Error("analytics: %v", err)
	return nil, &shared.ErrorResponse{Code: http.StatusBadGateway, Message: err.Error()}
}
