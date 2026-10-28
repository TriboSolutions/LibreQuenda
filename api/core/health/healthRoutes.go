package health

import (
	"database/sql"
	"librequenda/middlewares"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router"
)

type healthRoutes struct {
	db     *sql.DB
	router *router.Router
}

var healthRoutesInstance *healthRoutes = nil
var onceHealthRoutes sync.Once

func HealthRoutes(db *sql.DB, r *router.Router) *healthRoutes {
	onceHealthRoutes.Do(func() {
		healthRoutesInstance = &healthRoutes{
			db:     db,
			router: r,
		}
	})
	return healthRoutesInstance
}
func (hr *healthRoutes) CreateHealthRoutes() {
	hr.router.NewRoute(router.Route{
		Path:              "/health/api",
		Method:            "GET",
		Handler:           middlewares.VersionMiddleware(apiHealthHandler),
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
	})

	hr.router.NewRoute(router.Route{
		Path:              "/health/db",
		Method:            "GET",
		Handler:           middlewares.WithRateLimit(databaseHealthHandler(hr.db)),
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
	})
}

func apiHealthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func databaseHealthHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		var err error = db.Ping()
		if err != nil {
			http.Error(w, `{"status":"db down"}`, http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
