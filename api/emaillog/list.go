// Package emaillog reads the email registry (email_log) for the superadministrators.
package emaillog

import (
	"context"
	"database/sql"
	"encoding/csv"
	"librequenda/model"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/angelbarreiros/Penguin/router/helpers"
	"github.com/google/uuid"
)

const (
	pageSize   = 50
	csvMaxRows = 10000
	maxQuery   = 100
)

type EntryDto struct {
	Id         string    `json:"id"`
	At         time.Time `json:"at"`
	Status     string    `json:"status"` // sent | failed
	Kind       string    `json:"kind"`
	TenantSlug string    `json:"tenant_slug"`
	TenantName string    `json:"tenant_name"`
	Recipient  string    `json:"recipient"`
	Subject    string    `json:"subject"`
	Sender     string    `json:"sender"`
	Relay      string    `json:"relay"`
	Error      string    `json:"error"`
}

type ListResponse struct {
	Entries []EntryDto `json:"entries"`
	Next    *time.Time `json:"next"`  // pass as ?before= for the next page; null at the end
	Kinds   []string   `json:"kinds"` // every kind in the log, for the filter
	Sent24h int64      `json:"sent_24h"`
	Failed  int64      `json:"failed_24h"`
}

// Filters: what the superadministrator asks for, from the query string.
type Filters struct {
	Query    string
	Status   string
	Kind     string
	TenantID uuid.NullUUID
	Since    sql.NullTime
	Until    sql.NullTime
	Before   sql.NullTime
}

var kindPattern = regexp.MustCompile(`^[a-z_]{1,64}$`)

// ParseFilters reads ?q=&status=sent|failed&kind=&from=YYYY-MM-DD&to=YYYY-MM-DD&before=RFC3339;
// a value that is not valid is no filter. Days in loc.
func ParseFilters(q url.Values, loc *time.Location) Filters {
	var f = Filters{}
	if query := strings.TrimSpace(q.Get("q")); query != "" && utf8.RuneCountInString(query) <= maxQuery {
		f.Query = query
	}
	if status := q.Get("status"); status == "sent" || status == "failed" {
		f.Status = status
	}
	if kind := q.Get("kind"); kindPattern.MatchString(kind) {
		f.Kind = kind
	}
	if day, err := time.ParseInLocation(time.DateOnly, q.Get("from"), loc); err == nil {
		f.Since = sql.NullTime{Time: day, Valid: true}
	}
	if day, err := time.ParseInLocation(time.DateOnly, q.Get("to"), loc); err == nil {
		f.Until = sql.NullTime{Time: day.AddDate(0, 0, 1), Valid: true}
	}
	if at, err := time.Parse(time.RFC3339Nano, q.Get("before")); err == nil {
		f.Before = sql.NullTime{Time: at, Valid: true}
	}
	return f
}

// escapeLike: % and _ in what was typed are themselves, not wildcards.
var escapeLike = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace

func List(ctx context.Context, conn *sql.DB, f Filters, limit int) (*ListResponse, error) {
	var queries = model.New(conn)
	var rows, err = queries.ListEmailLog(ctx, model.ListEmailLogParams{
		Q:        sql.NullString{String: escapeLike(f.Query), Valid: f.Query != ""},
		Status:   sql.NullString{String: f.Status, Valid: f.Status != ""},
		Kind:     sql.NullString{String: f.Kind, Valid: f.Kind != ""},
		TenantID: f.TenantID,
		Since:    f.Since,
		Until:    f.Until,
		Before:   f.Before,
		MaxRows:  int32(limit),
	})
	if err != nil {
		return nil, err
	}
	var response = &ListResponse{Entries: make([]EntryDto, 0, len(rows)), Kinds: []string{}}
	for _, row := range rows {
		response.Entries = append(response.Entries, EntryDto{
			Id:         row.ID.String(),
			At:         row.At,
			Status:     row.Status,
			Kind:       row.Kind,
			TenantSlug: row.TenantSlug,
			TenantName: row.TenantName,
			Recipient:  row.Recipient,
			Subject:    row.Subject,
			Sender:     row.Sender,
			Relay:      row.Relay,
			Error:      row.Error,
		})
	}
	if len(rows) == limit && limit > 0 {
		var last = rows[len(rows)-1].At
		response.Next = &last
	}
	if kinds, err := queries.ListEmailLogKinds(ctx); err == nil {
		response.Kinds = kinds
	}
	if counts, err := queries.CountEmailLogSince(ctx, time.Now().Add(-24*time.Hour)); err == nil {
		response.Sent24h, response.Failed = counts.Sent, counts.Failed
	}
	return response, nil
}

// Serve answers a page as JSON, or with ?format=csv the filtered emails (up to csvMaxRows) as CSV.
func Serve(w http.ResponseWriter, r *http.Request, conn *sql.DB, f Filters) {
	var csvWanted = r.URL.Query().Get("format") == "csv"
	var limit = pageSize
	if csvWanted {
		limit = csvMaxRows
	}
	var response, err = List(r.Context(), conn, f, limit)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if !csvWanted {
		helpers.SendSuccessResponse(w, response)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="email-log.csv"`)
	var out = csv.NewWriter(w)
	_ = out.Write([]string{"at", "status", "kind", "council", "to", "subject", "from", "relay", "error"})
	for _, e := range response.Entries {
		_ = out.Write([]string{e.At.UTC().Format(time.RFC3339), e.Status, e.Kind, cell(e.TenantName), cell(e.Recipient), cell(e.Subject), cell(e.Sender), e.Relay, cell(e.Error)})
	}
	out.Flush()
}

// Retention: how long emails stay in the log, from EMAIL_LOG_RETENTION_DAYS (default a year).
func Retention(env string) int {
	if days, err := strconv.Atoi(env); err == nil && days > 0 {
		return days
	}
	return 365
}

// cell keeps a spreadsheet from reading a value as a formula (CSV injection).
func cell(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}
