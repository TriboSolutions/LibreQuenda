// Package opendata serves a council's open data, once it turned it on
// (council settings): a DCAT-AP-ES catalog for datos.gob.es to harvest and the
// files it lists. Public, with no sign-in, and nothing personal in it.
package opendata

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"librequenda/branding"
	"librequenda/middlewares"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/angelbarreiros/Penguin/router"
	"github.com/angelbarreiros/Penguin/router/helpers"
)

// how many finished months the monthly use covers
const monthsOfUse = 24

type openDataRoutes struct {
	db     *sql.DB
	router *router.Router
}

var routesOnce = new(sync.Once)
var routesInstance *openDataRoutes

func OpenDataRoutes(db *sql.DB, r *router.Router) *openDataRoutes {
	routesOnce.Do(func() {
		routesInstance = &openDataRoutes{db: db, router: r}
	})
	return routesInstance
}

// /tenants/{slug}/open-data/catalog (RDF/XML, or Turtle when asked for),
// catalog.rdf, catalog.ttl, facilities.csv|json, monthly-use.csv|json
func (r *openDataRoutes) RegisterRoutes() {
	var h = &handler{db: r.db}
	r.router.NewRoute(router.Route{
		Path:              "/tenants/{slug}/open-data/{file}",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
		Handler:           middlewares.VersionMiddleware(h.serve),
	})
}

type handler struct {
	db *sql.DB
}

func (h *handler) serve(w http.ResponseWriter, r *http.Request) {
	var slug, err = helpers.GetStringPathValue("slug", r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var file, _ = helpers.GetStringPathValue("file", r)
	var queries = model.New(h.db)
	var council model.GetPublishedCatalogRow
	council, err = queries.GetPublishedCatalog(r.Context(), slug)
	if errors.Is(err, sql.ErrNoRows) {
		// no such council, or it does not publish its data
		http.NotFound(w, r)
		return
	}
	if err != nil {
		shared.Logger.Error("open data of %s: %v", slug, err)
		http.Error(w, "unavailable", http.StatusInternalServerError)
		return
	}

	var contentType string
	var write func(io.Writer) error
	switch file {
	case "catalog":
		if wantsTurtle(r.Header.Get("Accept")) {
			file = "catalog.ttl"
		} else {
			file = "catalog.rdf"
		}
	}
	switch file {
	case "catalog.rdf", "catalog.ttl":
		var g *graph
		if g, err = h.catalog(r, queries, council); err != nil {
			break
		}
		if file == "catalog.ttl" {
			contentType, write = "text/turtle; charset=utf-8", g.WriteTurtle
		} else {
			contentType, write = "application/rdf+xml; charset=utf-8", g.WriteRDFXML
		}
	case "facilities.csv", "facilities.json":
		var rows []model.OpenDataFacilitiesRow
		if rows, err = queries.OpenDataFacilities(r.Context(), council.ID); err == nil {
			contentType, write = table(file, facilitiesTable(rows, webURL(r)+"/"+council.Slug))
		}
	case "monthly-use.csv", "monthly-use.json":
		var rows []model.OpenDataMonthlyUseRow
		if rows, err = monthlyUse(r.Context(), queries, council); err == nil {
			contentType, write = table(file, useTable(rows))
		}
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		shared.Logger.Error("open data %s of %s: %v", file, slug, err)
		http.Error(w, "unavailable", http.StatusInternalServerError)
		return
	}

	// rendered first: a failure is still a 500, not half a file
	var body bytes.Buffer
	if err = write(&body); err != nil {
		shared.Logger.Error("open data %s of %s: %v", file, slug, err)
		http.Error(w, "unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Last-Modified", council.ModifiedAt.UTC().Format(http.TimeFormat))
	if strings.HasSuffix(file, ".csv") {
		w.Header().Set("Content-Disposition", `attachment; filename="`+council.Slug+"-"+file+`"`)
	}
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(body.Bytes())
}

func wantsTurtle(accept string) bool {
	return strings.Contains(accept, "text/turtle") && !strings.Contains(accept, "application/rdf+xml")
}

// webURL: the web app's address, where the API answers under /api.
// PUBLIC_URL when set; otherwise the address this request came to.
func webURL(r *http.Request) string {
	if shared.PUBLIC_URL != "" {
		return shared.PUBLIC_URL
	}
	var scheme = "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	var host = r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return scheme + "://" + host
}

func (h *handler) catalog(r *http.Request, queries *model.Queries, council model.GetPublishedCatalogRow) (*graph, error) {
	var facilities, err = queries.OpenDataFacilities(r.Context(), council.ID)
	if err != nil {
		return nil, err
	}
	var seen = map[string]bool{}
	var keywords []string
	for _, f := range facilities {
		for _, tag := range f.Tags {
			if !seen[tag] {
				seen[tag] = true
				keywords = append(keywords, tag)
			}
		}
	}
	sort.Strings(keywords)
	var web = webURL(r)
	var i18n = helpers.GetI18nInstance()
	var start, end = usePeriod(council.Timezone)
	return buildCatalog(catalogInput{
		useStart: start,
		useEnd:   end.AddDate(0, 0, -1),
		council:  council,
		appName:  branding.Name(r.Context(), h.db),
		webURL:   web,
		apiURL:   web + "/api/tenants/" + council.Slug + "/open-data",
		keywords: keywords,
		translate: func(key, language string, args ...any) string {
			return i18n.TranslateWithLanguageWithVars("open_data."+key, language, args...)
		},
	}), nil
}

// usePeriod: the first day of the months the monthly use covers, and the first
// of the current month (not included), in the council's time zone.
func usePeriod(timezone string) (time.Time, time.Time) {
	var location, err = time.LoadLocation(timezone)
	if err != nil {
		location = time.UTC
	}
	var now = time.Now().In(location)
	var thisMonth = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return thisMonth.AddDate(0, -monthsOfUse, 0), thisMonth
}

// monthlyUse: the last finished months, in the council's time zone.
func monthlyUse(ctx context.Context, queries *model.Queries, council model.GetPublishedCatalogRow) ([]model.OpenDataMonthlyUseRow, error) {
	var start, end = usePeriod(council.Timezone)
	return queries.OpenDataMonthlyUse(ctx, model.OpenDataMonthlyUseParams{
		RangeStart: start,
		RangeEnd:   end,
		Timezone:   council.Timezone,
		TenantID:   council.ID,
	})
}

// dataTable: a header and rows of text, written as CSV or as JSON objects.
type dataTable struct {
	header []string
	rows   [][]any // string, int64, float64, bool or nil
}

func table(file string, t dataTable) (string, func(io.Writer) error) {
	if strings.HasSuffix(file, ".json") {
		return "application/json; charset=utf-8", func(w io.Writer) error {
			var out = make([]map[string]any, 0, len(t.rows))
			for _, row := range t.rows {
				var object = make(map[string]any, len(t.header))
				for i, name := range t.header {
					object[name] = row[i]
				}
				out = append(out, object)
			}
			var encoder = json.NewEncoder(w)
			encoder.SetIndent("", "  ")
			return encoder.Encode(out)
		}
	}
	return "text/csv; charset=utf-8", func(w io.Writer) error {
		var out = csv.NewWriter(w)
		_ = out.Write(t.header)
		for _, row := range t.rows {
			var cells = make([]string, len(row))
			for i, value := range row {
				cells[i] = cell(value)
			}
			_ = out.Write(cells)
		}
		out.Flush()
		return out.Error()
	}
}

func cell(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		// a spreadsheet would take =, +, - or @ at the start for a formula
		if v != "" && strings.ContainsRune("=+-@", rune(v[0])) {
			return "'" + v
		}
		return v
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case []string:
		return strings.Join(v, "; ")
	}
	return ""
}

func text(value sql.NullString) any {
	if !value.Valid || value.String == "" {
		return nil
	}
	return value.String
}

func facilitiesTable(rows []model.OpenDataFacilitiesRow, councilURL string) dataTable {
	var t = dataTable{header: []string{"id", "name", "description", "address", "latitude", "longitude", "capacity", "accessibility", "slot_minutes", "residents_only", "min_age", "tags", "opening_hours", "url"}}
	for _, f := range rows {
		var latitude, longitude, capacity, minAge any
		if f.Latitude.Valid {
			latitude, longitude = f.Latitude.Float64, f.Longitude.Float64
		}
		if f.Capacity.Valid {
			capacity = int64(f.Capacity.Int32)
		}
		if f.MinAge.Valid {
			minAge = int64(f.MinAge.Int32)
		}
		t.rows = append(t.rows, []any{
			f.ID.String(), f.Name, text(f.Description), text(f.Address), latitude, longitude, capacity,
			text(f.AccessibilityInfo), int64(f.SlotMinutes), f.ResidentsOnly, minAge,
			nonNil(f.Tags), nonNil(f.OpeningHours), councilURL + "/" + f.Slug,
		})
	}
	return t
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func useTable(rows []model.OpenDataMonthlyUseRow) dataTable {
	var t = dataTable{header: []string{"month", "facility_id", "facility", "bookings", "events", "cancelled", "booked_hours"}}
	for _, u := range rows {
		var hours, _ = strconv.ParseFloat(u.BookedHours, 64)
		t.rows = append(t.rows, []any{u.Month, u.FacilityID.String(), u.FacilityName, u.Bookings, u.Events, u.Cancelled, hours})
	}
	return t
}
