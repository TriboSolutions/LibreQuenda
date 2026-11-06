package audit

import (
	"context"
	"database/sql"
	"encoding/csv"
	"io"
	"librequenda/db"
	"librequenda/model"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/angelbarreiros/Penguin/router/helpers"
	"github.com/google/uuid"
)

const (
	pageSize   = 50
	csvMaxRows = 10000
)

// EntryDto is one line of the log as administrators read it. Names come from
// the accounts as they are now: an erased account has none (ActorDeleted).
type EntryDto struct {
	Id           string    `json:"id"`
	At           time.Time `json:"at"`
	ActorName    string    `json:"actor_name"`
	ActorRole    string    `json:"actor_role"`    // superadmin, council_admin, resident, system
	ActorDeleted bool      `json:"actor_deleted"` // the account was erased
	TenantSlug   string    `json:"tenant_slug"`
	TenantName   string    `json:"tenant_name"`
	Action       string    `json:"action"`
	TargetType   string    `json:"target_type"`
	TargetId     string    `json:"target_id"`
	TargetName   string    `json:"target_name"`   // a facility's or tag's name, or the person's account name
	TargetExists bool      `json:"target_exists"` // a facility that still exists, to link to
	Summary      string    `json:"summary"`
}

type ListResponse struct {
	Entries []EntryDto `json:"entries"`
	Next    *time.Time `json:"next"` // pass as ?before= for the next page; null at the end
}

// Filters: what a reader of the log asks for, from the query string.
type Filters struct {
	TenantID uuid.NullUUID
	ActorID  uuid.NullUUID
	Action   string // a code or a prefix (facility, facility.updated)
	Since    sql.NullTime
	Until    sql.NullTime
	Before   sql.NullTime
}

// actionPattern: an action code (facility.updated) or its prefix, as long as the column takes.
var actionPattern = regexp.MustCompile(`^[a-z_.]{1,64}$`)

// ParseFilters reads ?action=&actor=&from=YYYY-MM-DD&to=YYYY-MM-DD&before=RFC3339; dates in loc.
func ParseFilters(q url.Values, loc *time.Location) Filters {
	var f = Filters{}
	if action := q.Get("action"); actionPattern.MatchString(action) { // a code or a prefix; like the dates, anything else is no filter
		f.Action = action
	}
	if id, err := uuid.Parse(q.Get("actor")); err == nil {
		f.ActorID = uuid.NullUUID{UUID: id, Valid: true}
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

// List reads a page of the log. With a council, inside db.WithTenant: row
// security then shows only its entries, whoever asks.
func List(ctx context.Context, conn *sql.DB, council uuid.NullUUID, f Filters, limit int) (*ListResponse, error) {
	var rows []model.ListAuditEntriesRow
	var read = func(queries *model.Queries) error {
		var err error
		rows, err = queries.ListAuditEntries(ctx, model.ListAuditEntriesParams{
			TenantID: f.TenantID,
			ActorID:  f.ActorID,
			Action:   sql.NullString{String: f.Action, Valid: f.Action != ""},
			Since:    f.Since,
			Until:    f.Until,
			Before:   f.Before,
			MaxRows:  int32(limit),
		})
		return err
	}
	var err error
	if council.Valid {
		f.TenantID = council
		err = db.WithTenant(ctx, conn, council.UUID, read)
	} else {
		err = read(model.New(conn))
	}
	if err != nil {
		return nil, err
	}
	var response = &ListResponse{Entries: make([]EntryDto, 0, len(rows))}
	for _, row := range rows {
		response.Entries = append(response.Entries, mapEntry(row))
	}
	if len(rows) == limit && limit > 0 {
		var last = rows[len(rows)-1].At
		response.Next = &last
	}
	return response, nil
}

func mapEntry(row model.ListAuditEntriesRow) EntryDto {
	var entry = EntryDto{
		Id:           row.ID.String(),
		At:           row.At,
		ActorName:    row.ActorName,
		ActorRole:    row.ActorRole,
		ActorDeleted: !row.ActorID.Valid && row.ActorRole != "system",
		TenantSlug:   row.TenantSlug,
		TenantName:   row.TenantName,
		Action:       row.Action,
		TargetType:   row.TargetType,
		TargetName:   row.TargetLabel,
		TargetExists: row.TargetExists,
		Summary:      row.Summary,
	}
	if row.TargetID.Valid {
		entry.TargetId = row.TargetID.UUID.String()
	}
	if row.TargetType == "user" {
		entry.TargetName = row.TargetUserName // "" once that account is erased
	}
	return entry
}

// Serve answers a page as JSON, or with ?format=csv the filtered entries (up
// to csvMaxRows) as a CSV file, with the same columns the page shows.
func Serve(w http.ResponseWriter, r *http.Request, conn *sql.DB, council uuid.NullUUID, f Filters) {
	var csvWanted = r.URL.Query().Get("format") == "csv"
	var limit = pageSize
	if csvWanted {
		limit = csvMaxRows
	}
	var response, err = List(r.Context(), conn, council, f, limit)
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
	w.Header().Set("Content-Disposition", `attachment; filename="audit-log.csv"`)
	_ = WriteCSV(w, response.Entries)
}

// WriteCSV writes entries with a header row; an erased account is "(deleted account)".
func WriteCSV(w io.Writer, entries []EntryDto) error {
	var out = csv.NewWriter(w)
	_ = out.Write([]string{"at", "actor", "actor_role", "council", "action", "target_type", "target", "summary"})
	for _, e := range entries {
		var actor = e.ActorName
		if e.ActorDeleted {
			actor = "(deleted account)"
		}
		_ = out.Write([]string{e.At.UTC().Format(time.RFC3339), cell(actor), e.ActorRole, cell(e.TenantName), e.Action, e.TargetType, cell(e.TargetName), cell(e.Summary)})
	}
	out.Flush()
	return out.Error()
}

// Retention: how long entries are kept, from AUDIT_RETENTION_DAYS (default two years).
func Retention(env string) int {
	if days, err := strconv.Atoi(env); err == nil && days > 0 {
		return days
	}
	return 730
}

// cell keeps a spreadsheet from reading a value as a formula (CSV injection).
func cell(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

// Zone: the dates of the filters are days in Galicia, like the councils'.
func Zone() *time.Location {
	if loc, err := time.LoadLocation("Europe/Madrid"); err == nil {
		return loc
	}
	return time.UTC
}
