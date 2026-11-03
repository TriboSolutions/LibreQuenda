package tenants

import (
	"database/sql"
	"librequenda/middlewares"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router"
)

type tenantsRoutes struct {
	db     *sql.DB
	router *router.Router
}

var routesOnce *sync.Once = new(sync.Once)
var tenantsRoutesInstance *tenantsRoutes

func TenantsRoutes(db *sql.DB, router *router.Router) *tenantsRoutes {
	routesOnce.Do(func() {
		tenantsRoutesInstance = &tenantsRoutes{
			db:     db,
			router: router,
		}
	})
	return tenantsRoutesInstance
}

// Public: residents can look up councils and availability without logging in.
func (r *tenantsRoutes) RegisterRoutes() {
	var h = TenantsHandler(r.db)
	r.router.NewRoute(router.Route{
		Path:              "/tenants",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
		Handler:           middlewares.VersionMiddleware(h.GetTenants),
	})
	r.router.NewRoute(router.Route{
		Path:              "/facilities",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
		Handler:           middlewares.VersionMiddleware(h.GetFacilities),
	})
	// the search: every council's facilities for a day (?date=), nearest first with ?lat=&lon=
	r.router.NewRoute(router.Route{
		Path:              "/facilities/day",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
		Handler:           middlewares.VersionMiddleware(h.GetFacilitiesDay),
	})
	r.router.NewRoute(router.Route{
		Path:              "/tenants/{slug}/day",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
		Handler:           middlewares.VersionMiddleware(h.GetDay),
	})
	r.router.NewRoute(router.Route{
		Path:              "/tenants/{slug}/facilities/{facility}",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
		Handler:           middlewares.VersionMiddleware(h.GetFacility),
	})
	// an outdoor facility's forecast (MeteoGalicia), public like the day view
	r.router.NewRoute(router.Route{
		Path:              "/tenants/{slug}/facilities/{facility}/weather",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
		Handler:           middlewares.VersionMiddleware(h.Weather),
	})
	// signed in: "For you" on the home page, from the person's own bookings
	r.router.NewRoute(router.Route{
		Path:              "/me/recommendations",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.OPTIONS},
		Handler:           middlewares.AuthMiddleware(h.Recommendations),
	})
	r.router.NewRoute(router.Route{
		Path:              "/me/recommendations",
		Method:            http.MethodPost,
		AdditionalMethods: []router.HTTPMethod{router.OPTIONS},
		Handler:           middlewares.AuthMiddleware(h.Recommendations),
	})
}
