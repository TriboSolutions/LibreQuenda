package audit

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// written collects the entries Route writes, instead of the database.
func written(t *testing.T) *[]Entry {
	t.Helper()
	var entries []Entry
	var original = record
	record = func(_ context.Context, _ *sql.DB, e Entry) error {
		entries = append(entries, e)
		return nil
	}
	t.Cleanup(func() { record = original })
	return &entries
}

func serve(handler http.HandlerFunc, method, pattern, url string) {
	var mux = http.NewServeMux()
	mux.HandleFunc(method+" "+pattern, handler)
	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(method, url, nil))
}

func TestRouteLogsSuccessfulChangesOnly(t *testing.T) {
	var entries = written(t)
	var facility = uuid.New()
	var ok = Route(nil, http.MethodPut, "/tenants/{slug}/admin/facilities/{id}", func(w http.ResponseWriter, r *http.Request) {
		Changed(r.Context(), "slot_minutes", 60, 90)
		Changed(r.Context(), "name", "Pista", "Pista") // unchanged: not described
		w.WriteHeader(http.StatusOK)
	})
	serve(ok, http.MethodPut, "/tenants/{slug}/admin/facilities/{id}", "/tenants/lalin/admin/facilities/"+facility.String())
	if len(*entries) != 1 {
		t.Fatalf("want one entry, got %+v", *entries)
	}
	var e = (*entries)[0]
	if e.Action != "facility.updated" || e.TargetType != "facility" || e.TargetID.UUID != facility || e.Summary != "slot_minutes: 60 → 90" {
		t.Errorf("entry: %+v", e)
	}

	var refused = Route(nil, http.MethodPut, "/tenants/{slug}/admin/facilities/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	})
	serve(refused, http.MethodPut, "/tenants/{slug}/admin/facilities/{id}", "/tenants/lalin/admin/facilities/"+facility.String())
	if len(*entries) != 1 {
		t.Errorf("a refused change is not logged: %+v", *entries)
	}
}

func TestRouteSkipsReadsAndPreviews(t *testing.T) {
	var entries = written(t)
	var called = false
	var handler = func(w http.ResponseWriter, r *http.Request) { called = true }
	serve(Route(nil, http.MethodGet, "/tenants/{slug}/admin/facilities", handler), http.MethodGet, "/tenants/{slug}/admin/facilities", "/tenants/lalin/admin/facilities")
	serve(Route(nil, http.MethodPost, "/tenants/{slug}/admin/facilities/{id}/closures/preview", handler), http.MethodPost, "/tenants/{slug}/admin/facilities/{id}/closures/preview", "/tenants/lalin/admin/facilities/"+uuid.NewString()+"/closures/preview")
	if !called || len(*entries) != 0 {
		t.Errorf("reads and previews are not logged: %+v", *entries)
	}
}

func TestRouteLogsUnlistedChangesAsOther(t *testing.T) {
	var entries = written(t)
	serve(Route(nil, http.MethodPost, "/tenants/{slug}/admin/something-new", func(http.ResponseWriter, *http.Request) {}),
		http.MethodPost, "/tenants/{slug}/admin/something-new", "/tenants/lalin/admin/something-new")
	if len(*entries) != 1 || (*entries)[0].Action != "other" || !strings.Contains((*entries)[0].Summary, "POST /tenants/{slug}/admin/something-new") {
		t.Errorf("an unlisted change is still logged: %+v", *entries)
	}
}

func TestRouteLogsExportsWithPersonalData(t *testing.T) {
	var entries = written(t)
	serve(Route(nil, http.MethodGet, "/tenants/{slug}/admin/export", func(w http.ResponseWriter, r *http.Request) {
		Describe(r.Context(), "personal data: %v", true)
	}), http.MethodGet, "/tenants/{slug}/admin/export", "/tenants/lalin/admin/export?personal=1")
	if len(*entries) != 1 || (*entries)[0].Action != "council.exported" || (*entries)[0].Summary != "personal data: true" {
		t.Errorf("the export is logged: %+v", *entries)
	}
}

func TestSkip(t *testing.T) {
	var entries = written(t)
	serve(Route(nil, http.MethodDelete, "/admin/tags/{id}", func(w http.ResponseWriter, r *http.Request) { Skip(r.Context()) }),
		http.MethodDelete, "/admin/tags/{id}", "/admin/tags/"+uuid.NewString())
	if len(*entries) != 0 {
		t.Errorf("skipped: %+v", *entries)
	}
}

func TestRecordingCollects(t *testing.T) {
	var ctx, entry = Recording(context.Background())
	var person = uuid.New()
	TargetUser(ctx, person)
	Describe(ctx, "a")
	Describe(ctx, "b")
	if e := entry(); e.TargetType != "user" || e.TargetUserID.UUID != person || e.Summary != "a; b" {
		t.Errorf("collected: %+v", e)
	}
	// without a recording context, nothing breaks
	Describe(context.Background(), "nobody listens")
}

func TestCSVKeepsFormulasOut(t *testing.T) {
	var out strings.Builder
	var err = WriteCSV(&out, []EntryDto{
		{At: time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC), ActorDeleted: true, ActorRole: "council_admin", Action: "tag.created", Summary: `=HYPERLINK("x")`},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "(deleted account)") || !strings.Contains(out.String(), `'=HYPERLINK`) {
		t.Errorf("csv: %s", out.String())
	}
}

func TestRetention(t *testing.T) {
	if Retention("") != 730 || Retention("x") != 730 || Retention("-5") != 730 || Retention("365") != 365 {
		t.Error("retention defaults to two years and reads a positive number of days")
	}
}
