// Package analytics: questions about a council's data (or the whole installation's),
// answered by running read-only SQL on the analytics views (db/init/07_analytics_views.sql)
// as the read-only user librequenda_analytics (08_analytics_role.sh): from the
// indicators' assistant, which asks a language model to write that SQL, and from
// the MCP server's tools.
package analytics

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"librequenda/shared"

	"github.com/google/uuid"
)

// Scope: which councils a query sees. Everyone: a superadministrator.
type Scope struct {
	Everything bool
	Councils   []uuid.UUID
}

func (s Scope) empty() bool { return !s.Everything && len(s.Councils) == 0 }

// Result: what a query answered, at most MaxRows rows.
type Result struct {
	Columns   []string `json:"columns"`
	Rows      [][]any  `json:"rows"`
	Truncated bool     `json:"truncated"` // there were more rows than MaxRows
}

const (
	MaxRows      = 500
	maxQueryLen  = 10000
	queryTimeout = 8 * time.Second
)

var (
	ErrDisabled = errors.New("analytics queries are not set up on this installation (ANALYTICS_DB_PASSWORD)")
	ErrNoScope  = errors.New("no council to look at")
)

var (
	poolOnce sync.Once
	pool     *sql.DB
	poolErr  error
)

// connection: the analytics user's pool. No idle connection is kept, so nothing a
// query leaves in its session (a session lock, a setting) outlives it.
func connection() (*sql.DB, error) {
	poolOnce.Do(func() {
		if shared.ANALYTICS_DB_PASSWORD == "" {
			poolErr = ErrDisabled
			return
		}
		var dsn = fmt.Sprintf("host=db port=5432 user=librequenda_analytics password=%s dbname=%s sslmode=disable",
			quoteConnValue(shared.ANALYTICS_DB_PASSWORD), quoteConnValue(shared.POSTGRES_DB))
		pool, poolErr = sql.Open("postgres", dsn)
		if poolErr == nil {
			pool.SetMaxOpenConns(4)
			pool.SetMaxIdleConns(0)
		}
	})
	return pool, poolErr
}

func quoteConnValue(v string) string {
	return "'" + strings.ReplaceAll(strings.ReplaceAll(v, `\`, `\\`), `'`, `\'`) + "'"
}

var leadingComments = regexp.MustCompile(`^(\s+|--[^\n]*(\n|$)|/\*(?s:.*?)\*/)*`)

// Check: a first look at the SQL, before the database refuses anything else: one
// SELECT (or WITH) statement, of a sensible length. The guarantees are the
// database's (read-only user, views only, no set_config); this only answers
// sooner and more clearly.
func Check(query string) (string, error) {
	query = strings.TrimSpace(query)
	query = strings.TrimSpace(strings.TrimSuffix(query, ";"))
	if query == "" {
		return "", errors.New("the query is empty")
	}
	if utf8.RuneCountInString(query) > maxQueryLen {
		return "", fmt.Errorf("the query is longer than %d characters", maxQueryLen)
	}
	if strings.Contains(query, ";") {
		return "", errors.New("one statement only, without ';'")
	}
	var head = strings.ToLower(leadingComments.ReplaceAllString(query, ""))
	if !strings.HasPrefix(head, "select") && !strings.HasPrefix(head, "with") {
		return "", errors.New("only SELECT (or WITH … SELECT) queries")
	}
	return query, nil
}

// Run executes one query in a read-only transaction that sees only the scope's
// councils, wrapped so it stays a single SELECT and returns at most MaxRows rows.
func Run(ctx context.Context, scope Scope, query string) (*Result, error) {
	if scope.empty() {
		return nil, ErrNoScope
	}
	var checked, checkErr = Check(query)
	if checkErr != nil {
		return nil, checkErr
	}
	var db, err = connection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout+2*time.Second)
	defer cancel()
	var tx *sql.Tx
	if tx, err = db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, fmt.Errorf("the database did not answer: %w", err)
	}
	defer tx.Rollback()
	var settings = []string{
		fmt.Sprintf("SET LOCAL statement_timeout = '%dms'", queryTimeout.Milliseconds()),
		"SET LOCAL search_path = analytics",
	}
	if scope.Everything {
		settings = append(settings, "SET LOCAL analytics.superadmin = 'on'")
	} else {
		var ids = make([]string, len(scope.Councils))
		for i, id := range scope.Councils {
			ids[i] = id.String() // uuid.UUID's text: hex and dashes only
		}
		settings = append(settings, "SET LOCAL analytics.tenants = '{"+strings.Join(ids, ",")+"}'")
	}
	for _, statement := range settings {
		if _, err = tx.ExecContext(ctx, statement); err != nil {
			return nil, fmt.Errorf("the database did not answer: %w", err)
		}
	}
	// a subquery: no second statement can follow, and the limit is the API's
	var rows *sql.Rows
	if rows, err = tx.QueryContext(ctx, fmt.Sprintf("SELECT * FROM (\n%s\n) AS result LIMIT %d", checked, MaxRows+1)); err != nil {
		return nil, queryError(err)
	}
	defer rows.Close()
	var result = &Result{Rows: [][]any{}}
	if result.Columns, err = rows.Columns(); err != nil {
		return nil, queryError(err)
	}
	for rows.Next() {
		if len(result.Rows) == MaxRows {
			result.Truncated = true
			break
		}
		var values = make([]any, len(result.Columns))
		var pointers = make([]any, len(values))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err = rows.Scan(pointers...); err != nil {
			return nil, queryError(err)
		}
		for i, value := range values {
			values[i] = plain(value)
		}
		result.Rows = append(result.Rows, values)
	}
	if err = rows.Err(); err != nil {
		return nil, queryError(err)
	}
	return result, nil
}

// queryError: PostgreSQL's own words (what to fix in the SQL), without the driver's prefix.
func queryError(err error) error {
	return errors.New(strings.TrimPrefix(err.Error(), "pq: "))
}

// plain: a value as JSON shows it (text for bytes and numerics, RFC 3339 for times).
func plain(value any) any {
	switch v := value.(type) {
	case []byte:
		return string(v)
	case time.Time:
		return v.Format(time.RFC3339)
	default:
		return v
	}
}

// View: one analytics view, as the assistant and the MCP's tools describe it.
type View struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Columns     []Column `json:"columns"`
}

type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

var (
	viewsOnce sync.Once
	views     []View
)

// Views: the analytics views, their comments and columns (read once: they change
// only with the schema).
func Views(ctx context.Context) ([]View, error) {
	var db, err = connection()
	if err != nil {
		return nil, err
	}
	viewsOnce.Do(func() {
		var rows, queryErr = db.QueryContext(ctx, `
			SELECT c.relname, COALESCE(obj_description(c.oid, 'pg_class'), ''), a.attname, format_type(a.atttypid, a.atttypmod)
			FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = 'analytics'
			JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
			WHERE c.relkind = 'v'
			ORDER BY c.relname, a.attnum`)
		if queryErr != nil {
			err = queryErr
			return
		}
		defer rows.Close()
		var byName = map[string]*View{}
		var order []string
		for rows.Next() {
			var name, description, column, kind string
			if err = rows.Scan(&name, &description, &column, &kind); err != nil {
				return
			}
			if byName[name] == nil {
				byName[name] = &View{Name: name, Description: description}
				order = append(order, name)
			}
			byName[name].Columns = append(byName[name].Columns, Column{column, kind})
		}
		for _, name := range order {
			views = append(views, *byName[name])
		}
	})
	if err != nil {
		viewsOnce = sync.Once{} // try again next time
		return nil, err
	}
	return views, nil
}
