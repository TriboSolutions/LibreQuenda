package counciladmin

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/audit"
	"librequenda/core/polls"
	"librequenda/db"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/angelbarreiros/Penguin/router/helpers"
	"github.com/google/uuid"
)

// --- participation: the council's announcements and its quick polls ---

const (
	noticeTitleMax  = 120
	noticeBodyMax   = 2000
	noticeMaxDays   = 366
	pollQuestionMax = 200
	pollOptionMax   = 120
	pollMinOptions  = 2
	pollMaxOptions  = 6
	pollMaxDays     = 90
)

type NoticeDto struct {
	Id           string    `json:"id"`
	FacilityId   string    `json:"facility_id"` // "" for the whole council
	FacilityName string    `json:"facility_name"`
	Title        string    `json:"title"`
	Body         string    `json:"body"`
	ExpiresAt    time.Time `json:"expires_at"`
	Active       bool      `json:"active"` // still shown on the pages
	Notify       bool      `json:"notify"`
	Recipients   int32     `json:"recipients"` // how many were told in the bell and by email
	CreatedAt    time.Time `json:"created_at"`
}

type NoticesResponse struct {
	Timezone string      `json:"timezone"`
	Notices  []NoticeDto `json:"notices"`
}

// NoticeInput: shown until the end of ExpiresOn (YYYY-MM-DD, council time).
type NoticeInput struct {
	FacilityId string `json:"facility_id"` // "" for the whole council
	Title      string `json:"title"`
	Body       string `json:"body"`
	ExpiresOn  string `json:"expires_on"`
	Notify     bool   `json:"notify"` // also in the bell and by email to who books there
}

type PollDto struct {
	Id            string            `json:"id"`
	FacilityId    string            `json:"facility_id"`
	FacilityName  string            `json:"facility_name"`
	Question      string            `json:"question"`
	ResidentsOnly bool              `json:"residents_only"`
	ClosesAt      time.Time         `json:"closes_at"`
	ClosedAt      *time.Time        `json:"closed_at"` // closed early
	Open          bool              `json:"open"`
	Votes         int32             `json:"votes"`
	Recipients    int32             `json:"recipients"` // asked in the bell and by email when it opened
	Options       []polls.OptionDto `json:"options"`
	CreatedAt     time.Time         `json:"created_at"`
}

type PollsResponse struct {
	Timezone string    `json:"timezone"`
	Polls    []PollDto `json:"polls"`
}

// PollInput: open until the end of ClosesOn (YYYY-MM-DD, council time).
type PollInput struct {
	FacilityId    string   `json:"facility_id"`
	Question      string   `json:"question"`
	Options       []string `json:"options"`
	ResidentsOnly bool     `json:"residents_only"`
	ClosesOn      string   `json:"closes_on"`
}

func invalid(key string) *shared.ErrorResponse {
	return &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
}

// endOfDay: the first instant after the given council day, which must be from
// today to maxDays ahead.
func endOfDay(day string, zone *time.Location, now time.Time, maxDays int) (time.Time, bool) {
	var date, err = time.ParseInLocation(time.DateOnly, day, zone)
	if err != nil {
		return time.Time{}, false
	}
	var local = now.In(zone)
	var today = time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, zone)
	if date.Before(today) || date.After(today.AddDate(0, 0, maxDays)) {
		return time.Time{}, false
	}
	return date.AddDate(0, 0, 1), true
}

func (s *counciladminService) councilZone(ctx context.Context, tenantId uuid.UUID) (*time.Location, string) {
	var name, err = model.New(s.db).GetTenantTimezone(ctx, tenantId)
	if err != nil || name == "" {
		name = "Europe/Madrid"
	}
	var zone, zoneErr = time.LoadLocation(name)
	if zoneErr != nil {
		return time.UTC, "UTC"
	}
	return zone, name
}

// councilFacility: "" is the whole council; otherwise one of its facilities, with its name.
func (s *counciladminService) councilFacility(ctx context.Context, tenantId uuid.UUID, id string) (uuid.NullUUID, string, *shared.ErrorResponse) {
	if id == "" {
		return uuid.NullUUID{}, "", nil
	}
	var parsed, err = uuid.Parse(id)
	if err != nil {
		return uuid.NullUUID{}, "", &shared.ErrorResponse{Code: http.StatusBadRequest, Message: "errors.invalid_request_body"}
	}
	var facility, getErr = model.New(s.db).GetFacilityForAdmin(ctx, model.GetFacilityForAdminParams{ID: parsed, TenantID: tenantId})
	if errors.Is(getErr, sql.ErrNoRows) {
		return uuid.NullUUID{}, "", &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	}
	if getErr != nil {
		return uuid.NullUUID{}, "", failed("errors.participation_failed")
	}
	return uuid.NullUUID{UUID: parsed, Valid: true}, facility.Name, nil
}

func (s *counciladminService) ListNotices(ctx context.Context, tenantId uuid.UUID) (*NoticesResponse, *shared.ErrorResponse) {
	var rows, err = model.New(s.db).ListCouncilNotices(ctx, tenantId)
	if err != nil {
		return nil, failed("errors.participation_failed")
	}
	var _, zone = s.councilZone(ctx, tenantId)
	var now = time.Now()
	var response = &NoticesResponse{Timezone: zone, Notices: make([]NoticeDto, 0, len(rows))}
	for _, row := range rows {
		var dto = NoticeDto{
			Id: row.ID.String(), FacilityName: row.FacilityName.String, Title: row.Title, Body: row.Body,
			ExpiresAt: row.ExpiresAt, Active: row.ExpiresAt.After(now), Notify: row.Notify, Recipients: row.Recipients, CreatedAt: row.CreatedAt,
		}
		if row.FacilityID.Valid {
			dto.FacilityId = row.FacilityID.UUID.String()
		}
		response.Notices = append(response.Notices, dto)
	}
	return response, nil
}

// CreateNotice publishes an announcement and, when asked, tells who books
// there, in the bell and by email (the email job sends it).
func (s *counciladminService) CreateNotice(ctx context.Context, tenantId, staffId uuid.UUID, input NoticeInput) (*NoticeDto, *shared.ErrorResponse) {
	var title = strings.TrimSpace(input.Title)
	var body = strings.TrimSpace(input.Body)
	if title == "" || utf8.RuneCountInString(title) > noticeTitleMax {
		return nil, invalid("errors.notice_title_invalid")
	}
	if body == "" || utf8.RuneCountInString(body) > noticeBodyMax {
		return nil, invalid("errors.notice_body_invalid")
	}
	var zone, _ = s.councilZone(ctx, tenantId)
	var expires, ok = endOfDay(input.ExpiresOn, zone, time.Now(), noticeMaxDays)
	if !ok {
		return nil, invalid("errors.notice_expiry_invalid")
	}
	var facilityId, facilityName, facilityErr = s.councilFacility(ctx, tenantId, input.FacilityId)
	if facilityErr != nil {
		return nil, facilityErr
	}
	var dto = &NoticeDto{
		FacilityName: facilityName, Title: title, Body: body, ExpiresAt: expires, Active: true, Notify: input.Notify,
	}
	var err = db.WithTenant(ctx, s.db, tenantId, func(q *model.Queries) error {
		var created, createErr = q.CreateCouncilNotice(ctx, model.CreateCouncilNoticeParams{
			TenantID: tenantId, FacilityID: facilityId, Title: title, Body: body, ExpiresAt: expires,
			Notify: input.Notify, CreatedBy: uuid.NullUUID{UUID: staffId, Valid: true},
		})
		if createErr != nil {
			return createErr
		}
		dto.Id, dto.CreatedAt = created.ID.String(), created.CreatedAt
		if !input.Notify {
			return nil
		}
		var told, notifyErr = q.NotifyNoticeRecipients(ctx, created.ID)
		if notifyErr != nil {
			return notifyErr
		}
		dto.Recipients = int32(told)
		return q.SetNoticeRecipients(ctx, model.SetNoticeRecipientsParams{ID: created.ID, Recipients: dto.Recipients})
	})
	if err != nil {
		return nil, failed("errors.participation_failed")
	}
	if facilityId.Valid {
		dto.FacilityId = facilityId.UUID.String()
	}
	audit.Target(ctx, "notice", uuid.MustParse(dto.Id), "")
	audit.Describe(ctx, "facility: %s; expires: %s; notify: %t; recipients: %d", orCouncil(facilityName), input.ExpiresOn, input.Notify, dto.Recipients)
	return dto, nil
}

func orCouncil(facility string) string {
	if facility == "" {
		return "all"
	}
	return facility
}

// DeleteNotice withdraws it from the pages and the bells.
func (s *counciladminService) DeleteNotice(ctx context.Context, tenantId, noticeId uuid.UUID) *shared.ErrorResponse {
	var deleted int64
	var err = db.WithTenant(ctx, s.db, tenantId, func(q *model.Queries) error {
		var deleteErr error
		deleted, deleteErr = q.DeleteCouncilNotice(ctx, model.DeleteCouncilNoticeParams{ID: noticeId, TenantID: tenantId})
		return deleteErr
	})
	if err != nil {
		return failed("errors.participation_failed")
	}
	if deleted == 0 {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.notice_not_found"}
	}
	audit.Target(ctx, "notice", noticeId, "")
	return nil
}

func (s *counciladminService) ListPolls(ctx context.Context, tenantId uuid.UUID) (*PollsResponse, *shared.ErrorResponse) {
	var queries = model.New(s.db)
	var rows, err = queries.ListCouncilPolls(ctx, tenantId)
	if err != nil {
		return nil, failed("errors.participation_failed")
	}
	var ids = make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	var options, optionsErr = polls.Options(ctx, queries, ids)
	if optionsErr != nil {
		return nil, failed("errors.participation_failed")
	}
	var _, zone = s.councilZone(ctx, tenantId)
	var response = &PollsResponse{Timezone: zone, Polls: make([]PollDto, 0, len(rows))}
	for _, row := range rows {
		var dto = PollDto{
			Id: row.ID.String(), FacilityName: row.FacilityName.String, Question: row.Question, ResidentsOnly: row.ResidentsOnly,
			ClosesAt: row.ClosesAt, Open: row.IsOpen, Votes: row.Votes, Recipients: row.Recipients, Options: options[row.ID], CreatedAt: row.CreatedAt,
		}
		if row.FacilityID.Valid {
			dto.FacilityId = row.FacilityID.UUID.String()
		}
		if row.ClosedAt.Valid {
			dto.ClosedAt = &row.ClosedAt.Time
		}
		if dto.Options == nil {
			dto.Options = []polls.OptionDto{}
		}
		response.Polls = append(response.Polls, dto)
	}
	return response, nil
}

// CleanPollInput checks a poll before it is created: the question, 2 to 6
// distinct options, and returns them trimmed.
func CleanPollInput(input PollInput) (string, []string, *shared.ErrorResponse) {
	var question = strings.TrimSpace(input.Question)
	if question == "" || utf8.RuneCountInString(question) > pollQuestionMax {
		return "", nil, invalid("errors.poll_question_invalid")
	}
	var options = make([]string, 0, len(input.Options))
	var seen = map[string]bool{}
	for _, option := range input.Options {
		var label = strings.TrimSpace(option)
		if label == "" {
			continue
		}
		if utf8.RuneCountInString(label) > pollOptionMax || seen[strings.ToLower(label)] {
			return "", nil, invalid("errors.poll_options_invalid")
		}
		seen[strings.ToLower(label)] = true
		options = append(options, label)
	}
	if len(options) < pollMinOptions || len(options) > pollMaxOptions {
		return "", nil, invalid("errors.poll_options_invalid")
	}
	return question, options, nil
}

func (s *counciladminService) CreatePoll(ctx context.Context, tenantId, staffId uuid.UUID, input PollInput) (*PollDto, *shared.ErrorResponse) {
	var question, options, cleanErr = CleanPollInput(input)
	if cleanErr != nil {
		return nil, cleanErr
	}
	var zone, _ = s.councilZone(ctx, tenantId)
	var closes, ok = endOfDay(input.ClosesOn, zone, time.Now(), pollMaxDays)
	if !ok {
		return nil, invalid("errors.poll_close_invalid")
	}
	var facilityId, facilityName, facilityErr = s.councilFacility(ctx, tenantId, input.FacilityId)
	if facilityErr != nil {
		return nil, facilityErr
	}
	var dto = &PollDto{
		FacilityName: facilityName, Question: question, ResidentsOnly: input.ResidentsOnly,
		ClosesAt: closes, Open: true, Options: make([]polls.OptionDto, 0, len(options)), CreatedAt: time.Now(),
	}
	var err = db.WithTenant(ctx, s.db, tenantId, func(q *model.Queries) error {
		var id, createErr = q.CreatePoll(ctx, model.CreatePollParams{
			TenantID: tenantId, FacilityID: facilityId, Question: question, ResidentsOnly: input.ResidentsOnly,
			ClosesAt: closes, CreatedBy: uuid.NullUUID{UUID: staffId, Valid: true},
		})
		if createErr != nil {
			return createErr
		}
		dto.Id = id.String()
		for i, label := range options {
			if optionErr := q.CreatePollOption(ctx, model.CreatePollOptionParams{PollID: id, Position: int16(i), Label: label}); optionErr != nil {
				return optionErr
			}
			dto.Options = append(dto.Options, polls.OptionDto{Label: label, Votes: new(int32)})
		}
		// the council's people are asked in the bell, and by email (the email job)
		var asked, notifyErr = q.NotifyPollAudience(ctx, id)
		if notifyErr != nil {
			return notifyErr
		}
		dto.Recipients = int32(asked)
		return q.SetPollRecipients(ctx, model.SetPollRecipientsParams{ID: id, Recipients: dto.Recipients})
	})
	if err != nil {
		return nil, failed("errors.participation_failed")
	}
	if facilityId.Valid {
		dto.FacilityId = facilityId.UUID.String()
	}
	audit.Target(ctx, "poll", uuid.MustParse(dto.Id), "")
	audit.Describe(ctx, "facility: %s; options: %d; residents_only: %t; closes: %s; recipients: %d", orCouncil(facilityName), len(options), input.ResidentsOnly, input.ClosesOn, dto.Recipients)
	return dto, nil
}

// ClosePoll ends it now: the results show to everyone who could vote.
func (s *counciladminService) ClosePoll(ctx context.Context, tenantId, pollId uuid.UUID) *shared.ErrorResponse {
	var closed int64
	var err = db.WithTenant(ctx, s.db, tenantId, func(q *model.Queries) error {
		var closeErr error
		closed, closeErr = q.ClosePoll(ctx, model.ClosePollParams{ID: pollId, TenantID: tenantId})
		return closeErr
	})
	if err != nil {
		return failed("errors.participation_failed")
	}
	if closed == 0 {
		var _, getErr = model.New(s.db).GetCouncilPoll(ctx, model.GetCouncilPollParams{ID: pollId, TenantID: tenantId})
		if errors.Is(getErr, sql.ErrNoRows) {
			return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.poll_not_found"}
		}
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.poll_closed"}
	}
	audit.Target(ctx, "poll", pollId, "")
	return nil
}

func (s *counciladminService) DeletePoll(ctx context.Context, tenantId, pollId uuid.UUID) *shared.ErrorResponse {
	var deleted int64
	var err = db.WithTenant(ctx, s.db, tenantId, func(q *model.Queries) error {
		var deleteErr error
		deleted, deleteErr = q.DeletePoll(ctx, model.DeletePollParams{ID: pollId, TenantID: tenantId})
		return deleteErr
	})
	if err != nil {
		return failed("errors.participation_failed")
	}
	if deleted == 0 {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.poll_not_found"}
	}
	audit.Target(ctx, "poll", pollId, "")
	return nil
}

// --- handlers ---

func pathId(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	var id, err = helpers.GetUUidPathValue(name, r)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return uuid.Nil, false
	}
	return id, true
}

func (h *counciladminHandler) ListNotices(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var response, err = GetCounciladminService(h.db).ListNotices(r.Context(), tenantId)
	send(w, r, response, err)
}

func (h *counciladminHandler) CreateNotice(w http.ResponseWriter, r *http.Request) {
	var tenantId, staffId, ok = h.councilAdminUser(w, r)
	if !ok {
		return
	}
	var input NoticeInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetCounciladminService(h.db).CreateNotice(r.Context(), tenantId, staffId, input)
	send(w, r, response, err)
}

func (h *counciladminHandler) DeleteNotice(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var noticeId, idOk = pathId(w, r, "noticeId")
	if !idOk {
		return
	}
	send(w, r, nil, GetCounciladminService(h.db).DeleteNotice(r.Context(), tenantId, noticeId))
}

func (h *counciladminHandler) ListPolls(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var response, err = GetCounciladminService(h.db).ListPolls(r.Context(), tenantId)
	send(w, r, response, err)
}

func (h *counciladminHandler) CreatePoll(w http.ResponseWriter, r *http.Request) {
	var tenantId, staffId, ok = h.councilAdminUser(w, r)
	if !ok {
		return
	}
	var input PollInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetCounciladminService(h.db).CreatePoll(r.Context(), tenantId, staffId, input)
	send(w, r, response, err)
}

func (h *counciladminHandler) ClosePoll(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var pollId, idOk = pathId(w, r, "pollId")
	if !idOk {
		return
	}
	send(w, r, nil, GetCounciladminService(h.db).ClosePoll(r.Context(), tenantId, pollId))
}

func (h *counciladminHandler) DeletePoll(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var pollId, idOk = pathId(w, r, "pollId")
	if !idOk {
		return
	}
	send(w, r, nil, GetCounciladminService(h.db).DeletePoll(r.Context(), tenantId, pollId))
}
