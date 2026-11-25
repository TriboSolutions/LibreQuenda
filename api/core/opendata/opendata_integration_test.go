package opendata

import (
	"database/sql"
	"encoding/json"
	"librequenda/shared"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// OPEN_DATA_TEST_DATABASE_URL=postgres://… go test ./core/opendata -run Integration
// A disposable database with db/init loaded: the test adds a council and removes it.
func TestOpenDataIntegration(t *testing.T) {
	var dsn = os.Getenv("OPEN_DATA_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires a disposable PostgreSQL fixture database")
	}
	var conn, err = sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() }) // after the council is removed (cleanups run last first)
	shared.PUBLIC_URL = "https://reservas.example"

	var tenantId = uuid.New()
	var slug = "od-" + tenantId.String()[:8]
	var exec = func(query string, args ...any) {
		t.Helper()
		if _, err := conn.Exec(query, args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}
	exec(`INSERT INTO tenant (id, slug, name, contact_email) VALUES ($1, $2, 'Concello de Proba', 'datos@proba.gal')`, tenantId, slug)
	t.Cleanup(func() { _, _ = conn.Exec(`DELETE FROM tenant WHERE id = $1`, tenantId) })
	var facilityId = uuid.New()
	exec(`INSERT INTO facility (id, tenant_id, slug, name, address, latitude, longitude, capacity) VALUES ($1, $2, 'pista', '=Pista 1', 'Rúa A', 43.3, -8.4, 4)`, facilityId, tenantId)
	exec(`INSERT INTO opening_hours (facility_id, weekday, opens_at, closes_at) VALUES ($1, 1, '09:00', '14:00')`, facilityId)
	// two bookings of an hour and one cancelled in the previous month, one this month (not counted yet)
	var madrid, _ = time.LoadLocation("Europe/Madrid")
	var now = time.Now().In(madrid)
	var lastMonth = time.Date(now.Year(), now.Month()-1, 10, 10, 0, 0, 0, madrid)
	for i, status := range []string{"confirmed", "confirmed", "cancelled"} {
		var start = lastMonth.Add(time.Duration(i) * 24 * time.Hour)
		exec(`INSERT INTO booking (facility_id, starts_at, ends_at, status, cancelled_by, holder_name, channel) VALUES ($1, $2, $3, $4::booking_status, CASE WHEN $4::booking_status = 'cancelled' THEN 'user'::booking_canceller END, 'x', 'staff')`,
			facilityId, start, start.Add(time.Hour), status)
	}
	var thisMonth = time.Date(now.Year(), now.Month(), 1, 12, 0, 0, 0, madrid)
	exec(`INSERT INTO booking (facility_id, starts_at, ends_at, holder_name, channel) VALUES ($1, $2, $3, 'x', 'staff')`, facilityId, thisMonth, thisMonth.Add(time.Hour))

	var h = &handler{db: conn}
	var get = func(file, accept string) *httptest.ResponseRecorder {
		var r = httptest.NewRequest(http.MethodGet, "/tenants/"+slug+"/open-data/"+file, nil)
		r.SetPathValue("slug", slug)
		r.SetPathValue("file", file)
		if accept != "" {
			r.Header.Set("Accept", accept)
		}
		var w = httptest.NewRecorder()
		h.serve(w, r)
		return w
	}

	// not published yet
	if w := get("catalog.rdf", ""); w.Code != http.StatusNotFound {
		t.Fatalf("unpublished catalog: %d", w.Code)
	}
	exec(`INSERT INTO tenant_open_data (tenant_id, enabled, dir3_code, issued_at) VALUES ($1, TRUE, 'L01150301', NOW())`, tenantId)

	var w = get("catalog", "")
	if w.Code != http.StatusOK || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/rdf+xml") {
		t.Fatalf("catalog: %d %s\n%s", w.Code, w.Header().Get("Content-Type"), w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Organismo/L01150301") || !strings.Contains(w.Body.String(), "https://reservas.example/api/tenants/"+slug+"/open-data/facilities.csv") {
		t.Errorf("catalog without publisher or files:\n%s", w.Body.String())
	}
	if w = get("catalog", "text/turtle"); !strings.HasPrefix(w.Header().Get("Content-Type"), "text/turtle") {
		t.Errorf("catalog for text/turtle: %s", w.Header().Get("Content-Type"))
	}

	w = get("facilities.csv", "")
	var lines = strings.Split(strings.TrimSpace(w.Body.String()), "\n")
	if w.Code != http.StatusOK || len(lines) != 2 || !strings.Contains(lines[1], "'=Pista 1") || !strings.Contains(lines[1], "1 09:00-14:00") {
		t.Errorf("facilities.csv: %d\n%s", w.Code, w.Body.String())
	}

	w = get("monthly-use.json", "")
	var use []map[string]any
	if err = json.Unmarshal(w.Body.Bytes(), &use); err != nil {
		t.Fatalf("monthly-use.json: %v\n%s", err, w.Body.String())
	}
	if len(use) != monthsOfUse {
		t.Fatalf("%d months, want %d", len(use), monthsOfUse)
	}
	var last = use[len(use)-1]
	if last["month"] != lastMonth.Format("2006-01") || last["bookings"] != 2.0 || last["cancelled"] != 1.0 || last["booked_hours"] != 2.0 {
		t.Errorf("last finished month: %v", last)
	}

	if w = get("people.csv", ""); w.Code != http.StatusNotFound {
		t.Errorf("unknown file: %d", w.Code)
	}
	exec(`UPDATE tenant_open_data SET enabled = FALSE WHERE tenant_id = $1`, tenantId)
	if w = get("facilities.json", ""); w.Code != http.StatusNotFound {
		t.Errorf("turned off: %d", w.Code)
	}
}
