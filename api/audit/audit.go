// Package audit keeps the audit log: what administrators did, for
// accountability (GDPR Art. 5(2) and 32, ENS). Route wraps the administrative
// routes, so every change made through them is logged once it succeeded; the
// services add what changed (Describe) and what it was done to (Target).
//
// Data minimisation: an entry holds the actor's account id (names are read
// from the account when shown, and vanish when it is erased), the council,
// a stable action code, the target and a short summary built from field names
// and non-personal values. Never passwords, keys, tokens, residents' contact
// details or free text, nor locations.
package audit

import (
	"context"
	"database/sql"
	"fmt"
	"librequenda/auth"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// Entry is one line of the log, for Record.
type Entry struct {
	TenantID     uuid.NullUUID
	TenantLabel  string // the council's name; read from the council when empty
	Action       string // e.g. "facility.updated"
	TargetType   string
	TargetID     uuid.NullUUID
	TargetUserID uuid.NullUUID // when the target is a person
	TargetLabel  string        // a non-personal name (a facility's, a tag's)
	Summary      string
}

// details is what a service adds to the entry of the request it handles.
type details struct {
	tenantID     uuid.NullUUID
	targetType   string
	targetID     uuid.NullUUID
	targetUserID uuid.NullUUID
	targetLabel  string
	summary      []string
	skip         bool
}

type detailsKey struct{}

func from(ctx context.Context) *details {
	var d, _ = ctx.Value(detailsKey{}).(*details)
	return d
}

// Describe adds a line to what changed: field names and non-personal values only.
func Describe(ctx context.Context, format string, args ...any) {
	if d := from(ctx); d != nil {
		d.summary = append(d.summary, fmt.Sprintf(format, args...))
	}
}

// Target says what the action was done to, when the route's path does not.
func Target(ctx context.Context, targetType string, id uuid.UUID, label string) {
	if d := from(ctx); d != nil {
		d.targetType, d.targetID, d.targetLabel = targetType, uuid.NullUUID{UUID: id, Valid: true}, label
	}
}

// TargetUser: the action was done to a person's account (an administrator added or removed).
func TargetUser(ctx context.Context, id uuid.UUID) {
	if d := from(ctx); d != nil {
		d.targetType, d.targetUserID = "user", uuid.NullUUID{UUID: id, Valid: true}
	}
}

// Council: the council the action concerns, when the route's path does not name it (a new council).
func Council(ctx context.Context, id uuid.UUID) {
	if d := from(ctx); d != nil {
		d.tenantID = uuid.NullUUID{UUID: id, Valid: true}
	}
}

// Skip: this request changed nothing worth logging.
func Skip(ctx context.Context) {
	if d := from(ctx); d != nil {
		d.skip = true
	}
}

// Changed describes a field that changed, from → to; nothing when equal.
func Changed[T comparable](ctx context.Context, field string, before, after T) {
	if before != after {
		Describe(ctx, "%s: %v → %v", field, before, after)
	}
}

// record is how Route writes; tests replace it.
var record = func(ctx context.Context, db *sql.DB, entry Entry) error { return Record(ctx, db, entry) }

// Record writes one entry now, as the signed-in person of ctx (none: the system).
func Record(ctx context.Context, db model.DBTX, entry Entry) error {
	var subject sql.NullString
	if session, ok := auth.SessionFromContext(ctx); ok {
		subject = sql.NullString{String: session.Subject, Valid: true}
	}
	return model.New(db).InsertAuditEntry(ctx, model.InsertAuditEntryParams{
		Subject:      subject,
		TenantID:     entry.TenantID,
		TenantLabel:  sql.NullString{String: entry.TenantLabel, Valid: entry.TenantLabel != ""},
		Action:       entry.Action,
		TargetType:   sql.NullString{String: entry.TargetType, Valid: entry.TargetType != ""},
		TargetID:     entry.TargetID,
		TargetUserID: entry.TargetUserID,
		TargetLabel:  sql.NullString{String: entry.TargetLabel, Valid: entry.TargetLabel != ""},
		Summary:      truncate(entry.Summary, 2000),
	})
}

// statusWriter remembers the status the handler answered.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// Route logs a route's successful requests: the changes (POST, PUT, PATCH,
// DELETE) and the reads listed in actions (data exports). It goes inside the
// authentication middleware, which puts the session in the context. An entry
// is written once the action succeeded; if writing it fails the action stands
// and the failure is logged, never shown to the person.
func Route(db *sql.DB, method, path string, handler http.HandlerFunc) http.HandlerFunc {
	var known, listed = actions[method+" "+path]
	if !listed && method == http.MethodGet {
		return handler // plain reads are not logged
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var d = &details{}
		var ctx = context.WithValue(r.Context(), detailsKey{}, d)
		r = r.WithContext(ctx)

		// the council and the target as the path names them, read before the
		// action in case it deletes them
		var tenantID, tenantLabel = tenantOf(ctx, db, r)
		if listed && known.param != "" {
			if id, err := uuid.Parse(r.PathValue(known.param)); err == nil {
				if known.target == "user" {
					d.targetUserID = uuid.NullUUID{UUID: id, Valid: true}
				} else {
					d.targetID = uuid.NullUUID{UUID: id, Valid: true}
					if known.target == "facility" && db != nil {
						d.targetLabel, _ = model.New(db).GetFacilityNameForAudit(ctx, id)
					}
				}
				d.targetType = known.target
			}
		}

		var sw = &statusWriter{ResponseWriter: w}
		handler(sw, r)
		if sw.status == 0 {
			sw.status = http.StatusOK
		}
		if sw.status >= 300 || d.skip || (listed && known.code == "") {
			return
		}

		var entry = Entry{
			TenantID:     tenantID,
			TenantLabel:  tenantLabel,
			TargetType:   d.targetType,
			TargetID:     d.targetID,
			TargetUserID: d.targetUserID,
			TargetLabel:  d.targetLabel,
			Summary:      strings.Join(d.summary, "; "),
		}
		if d.tenantID.Valid && d.tenantID != tenantID {
			entry.TenantID, entry.TenantLabel = d.tenantID, ""
		}
		if listed {
			entry.Action = known.code
		} else {
			// a change through a route nobody listed yet: logged all the same
			entry.Action = "other"
			if entry.Summary == "" {
				entry.Summary = method + " " + path
			} else {
				entry.Summary = method + " " + path + "; " + entry.Summary
			}
		}
		if err := record(ctx, db, entry); err != nil {
			shared.Logger.Error("audit: %s %s not logged: %v", method, r.URL.Path, err)
		}
	}
}

// tenantOf: the council the path names ({slug}), if any.
func tenantOf(ctx context.Context, db *sql.DB, r *http.Request) (uuid.NullUUID, string) {
	var slug = r.PathValue("slug")
	if slug == "" || db == nil {
		return uuid.NullUUID{}, ""
	}
	var row, err = model.New(db).GetTenantIdBySlug(ctx, slug)
	if err != nil {
		return uuid.NullUUID{}, ""
	}
	return uuid.NullUUID{UUID: row.ID, Valid: true}, row.Name
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// Recording gives a context that collects what Describe, Target and the rest
// add, outside Route (tests, and callers that write with Record themselves).
func Recording(ctx context.Context) (context.Context, func() Entry) {
	var d = &details{}
	return context.WithValue(ctx, detailsKey{}, d), func() Entry {
		return Entry{TenantID: d.tenantID, TargetType: d.targetType, TargetID: d.targetID, TargetUserID: d.targetUserID, TargetLabel: d.targetLabel, Summary: strings.Join(d.summary, "; ")}
	}
}
