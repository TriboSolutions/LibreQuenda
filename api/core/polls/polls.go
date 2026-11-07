// Package polls: the councils' quick polls, as residents see them and vote.
// One vote per account with a verified email (and, for residents-only polls,
// a verified resident of the council), which can change while the poll is
// open. Nobody sees who voted what, and the counts only once it closed.
package polls

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"librequenda/auth"
	"librequenda/core/me"
	"librequenda/middlewares"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"sync"
	"time"

	"github.com/angelbarreiros/Penguin/router"
	"github.com/angelbarreiros/Penguin/router/helpers"
	"github.com/google/uuid"
)

// OptionDto: one answer. Votes is null for residents while the poll is open.
type OptionDto struct {
	Id    string `json:"id"`
	Label string `json:"label"`
	Votes *int32 `json:"votes"`
}

// Options: each poll's options in order, with their votes.
func Options(ctx context.Context, queries *model.Queries, polls []uuid.UUID) (map[uuid.UUID][]OptionDto, error) {
	var byPoll = map[uuid.UUID][]OptionDto{}
	if len(polls) == 0 {
		return byPoll, nil
	}
	var rows, err = queries.ListPollOptions(ctx, polls)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		var votes = row.Votes
		byPoll[row.PollID] = append(byPoll[row.PollID], OptionDto{Id: row.ID.String(), Label: row.Label, Votes: &votes})
	}
	return byPoll, nil
}

type PollDto struct {
	Id            string      `json:"id"`
	Question      string      `json:"question"`
	TenantName    string      `json:"tenant_name"`
	TenantSlug    string      `json:"tenant_slug"`
	FacilityName  string      `json:"facility_name"` // "" when it is about the whole council
	FacilitySlug  string      `json:"facility_slug"`
	ResidentsOnly bool        `json:"residents_only"`
	Open          bool        `json:"open"`
	ClosesAt      time.Time   `json:"closes_at"` // when it closed, if it closed early
	MyOption      string      `json:"my_option"` // the person's vote; "" before voting
	Votes         *int32      `json:"votes"`     // null while open
	Options       []OptionDto `json:"options"`
}

type PollsResponse struct {
	Polls []PollDto `json:"polls"` // the open ones first, those closing soonest first
}

type VoteInput struct {
	OptionId string `json:"option_id"`
}

type pollsService struct {
	db *sql.DB
}

// ForUser: the polls of the councils the person is part of, open and recently closed.
func (s *pollsService) ForUser(ctx context.Context, userId uuid.UUID) (*PollsResponse, *shared.ErrorResponse) {
	var queries = model.New(s.db)
	var rows, err = queries.ListPollsForUser(ctx, uuid.NullUUID{UUID: userId, Valid: true})
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.participation_failed"}
	}
	var ids = make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	var options, optionsErr = Options(ctx, queries, ids)
	if optionsErr != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.participation_failed"}
	}
	var response = &PollsResponse{Polls: make([]PollDto, 0, len(rows))}
	for _, row := range rows {
		var dto = PollDto{
			Id: row.ID.String(), Question: row.Question, TenantName: row.TenantName, TenantSlug: row.TenantSlug,
			FacilityName: row.FacilityName, FacilitySlug: row.FacilitySlug, ResidentsOnly: row.ResidentsOnly,
			Open: row.IsOpen, ClosesAt: row.ClosesAt, Options: options[row.ID],
		}
		if row.ClosedAt.Valid {
			dto.ClosesAt = row.ClosedAt.Time
		}
		if row.MyOption.Valid {
			dto.MyOption = row.MyOption.UUID.String()
		}
		if dto.Options == nil {
			dto.Options = []OptionDto{}
		}
		if dto.Open {
			for i := range dto.Options {
				dto.Options[i].Votes = nil // the results wait for the close, so nobody follows the crowd
			}
		} else {
			var votes = row.Votes
			dto.Votes = &votes
		}
		response.Polls = append(response.Polls, dto)
	}
	return response, nil
}

// Vote casts, or changes, the person's vote.
func (s *pollsService) Vote(ctx context.Context, userId, pollId uuid.UUID, input VoteInput) *shared.ErrorResponse {
	var optionId, parseErr = uuid.Parse(input.OptionId)
	if parseErr != nil {
		return &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.poll_option_not_found"}
	}
	var queries = model.New(s.db)
	var poll, err = queries.GetPollForVote(ctx, model.GetPollForVoteParams{UserID: userId, PollID: pollId})
	if errors.Is(err, sql.ErrNoRows) {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.poll_not_found"}
	}
	if err != nil {
		return &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.participation_failed"}
	}
	if key := mayVote(poll); key != "" {
		var code = http.StatusConflict
		if key == "errors.poll_residents_only" {
			code = http.StatusForbidden
		}
		return &shared.ErrorResponse{Code: code, Message: key}
	}
	var cast, castErr = queries.CastPollVote(ctx, model.CastPollVoteParams{UserID: uuid.NullUUID{UUID: userId, Valid: true}, OptionID: optionId, PollID: pollId})
	if castErr != nil {
		return &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.participation_failed"}
	}
	if cast == 0 {
		return &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.poll_option_not_found"}
	}
	return nil
}

// mayVote: "" when the person may vote, the error key otherwise.
func mayVote(poll model.GetPollForVoteRow) string {
	switch {
	case !poll.IsOpen:
		return "errors.poll_closed"
	case !poll.EmailVerified:
		return "errors.email_unverified"
	case poll.ResidentsOnly && !poll.IsResident:
		return "errors.poll_residents_only"
	}
	return ""
}

// --- handlers and routes ---

type pollsRoutes struct {
	db     *sql.DB
	router *router.Router
}

var routesOnce = new(sync.Once)
var routesInstance *pollsRoutes

func PollsRoutes(db *sql.DB, r *router.Router) *pollsRoutes {
	routesOnce.Do(func() {
		routesInstance = &pollsRoutes{db: db, router: r}
	})
	return routesInstance
}

func (r *pollsRoutes) RegisterRoutes() {
	var s = &pollsService{db: r.db}
	r.router.NewRoute(router.Route{
		Path:              "/me/polls",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
		Handler: middlewares.AuthMiddleware(func(w http.ResponseWriter, req *http.Request) {
			var userId, ok = currentUser(w, req, r.db)
			if !ok {
				return
			}
			var response, err = s.ForUser(req.Context(), userId)
			if err != nil {
				helpers.SendI18NErrorResponse(w, req, err.Code, err.Message)
				return
			}
			helpers.SendSuccessResponse(w, response)
		}),
	})
	r.router.NewRoute(router.Route{
		Path:              "/me/polls/{pollId}/vote",
		Method:            http.MethodPut,
		AdditionalMethods: []router.HTTPMethod{router.OPTIONS},
		Handler: middlewares.AuthMiddleware(func(w http.ResponseWriter, req *http.Request) {
			var pollId, pathErr = helpers.GetUUidPathValue("pollId", req)
			if pathErr != nil {
				helpers.SendI18NErrorResponse(w, req, http.StatusBadRequest, "errors.invalid_query_params")
				return
			}
			var userId, ok = currentUser(w, req, r.db)
			if !ok {
				return
			}
			var input VoteInput
			if err := json.NewDecoder(shared.JSONBody(w, req)).Decode(&input); err != nil {
				helpers.SendI18NErrorResponse(w, req, http.StatusBadRequest, "errors.invalid_request_body")
				return
			}
			if err := s.Vote(req.Context(), userId, pollId, input); err != nil {
				helpers.SendI18NErrorResponse(w, req, err.Code, err.Message)
				return
			}
			helpers.SendNoContentResponse(w)
		}),
	})
}

func currentUser(w http.ResponseWriter, r *http.Request, db *sql.DB) (uuid.UUID, bool) {
	var session, _ = auth.SessionFromContext(r.Context())
	var userId, err = me.CurrentUserId(r.Context(), db, session)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message)
		return uuid.Nil, false
	}
	return userId, true
}
