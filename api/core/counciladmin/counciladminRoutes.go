package counciladmin

import (
	"database/sql"
	"librequenda/middlewares"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router"
)

type counciladminRoutes struct {
	db     *sql.DB
	router *router.Router
}

var routesOnce *sync.Once = new(sync.Once)
var counciladminRoutesInstance *counciladminRoutes

func CounciladminRoutes(db *sql.DB, router *router.Router) *counciladminRoutes {
	routesOnce.Do(func() {
		counciladminRoutesInstance = &counciladminRoutes{
			db:     db,
			router: router,
		}
	})
	return counciladminRoutesInstance
}

// Council administration: every handler checks "admin of this council, or superadmin".
func (r *counciladminRoutes) RegisterRoutes() {
	var h = CounciladminHandler(r.db)
	var routes = []router.Route{
		{Path: "/tenants/{slug}/admin/facilities", Method: http.MethodGet, Handler: h.ListFacilities},
		{Path: "/tenants/{slug}/admin/facilities", Method: http.MethodPost, Handler: h.CreateFacility},
		{Path: "/tenants/{slug}/admin/facilities/{id}", Method: http.MethodGet, Handler: h.GetFacility},
		{Path: "/tenants/{slug}/admin/facilities/{id}", Method: http.MethodPut, Handler: h.UpdateFacility},
		{Path: "/tenants/{slug}/admin/facilities/{id}", Method: http.MethodDelete, Handler: h.DeleteFacility},
		{Path: "/tenants/{slug}/admin/facilities/{id}/schedule", Method: http.MethodPut, Handler: h.ReplaceSchedule},
		{Path: "/tenants/{slug}/admin/facilities/{id}/devices", Method: http.MethodGet, Handler: h.ListDevices},
		{Path: "/tenants/{slug}/admin/facilities/{id}/devices", Method: http.MethodPost, Handler: h.CreateDevice},
		{Path: "/tenants/{slug}/admin/facilities/{id}/devices/{deviceId}", Method: http.MethodDelete, Handler: h.DeleteDevice},
		{Path: "/tenants/{slug}/admin/facilities/{id}/active", Method: http.MethodPut, Handler: h.SetActive},
		{Path: "/tenants/{slug}/admin/facilities/{id}/closures/preview", Method: http.MethodPost, Handler: h.PreviewClosure},
		{Path: "/tenants/{slug}/admin/facilities/{id}/closures", Method: http.MethodPost, Handler: h.CreateClosure},
		{Path: "/tenants/{slug}/admin/facilities/{id}/closures/{closureId}", Method: http.MethodDelete, Handler: h.DeleteClosure},
		{Path: "/tenants/{slug}/admin/indicators", Method: http.MethodGet, Handler: h.GetIndicators},
		{Path: "/tenants/{slug}/admin/indicators/export", Method: http.MethodGet, Handler: h.ExportIndicators},
		{Path: "/tenants/{slug}/admin/export", Method: http.MethodGet, Handler: middlewares.WithStrictRateLimit(h.ExportCouncil)},
		{Path: "/tenants/{slug}/admin/smtp", Method: http.MethodGet, Handler: h.GetSmtp},
		{Path: "/tenants/{slug}/admin/smtp", Method: http.MethodPut, Handler: h.SaveSmtp},
		{Path: "/tenants/{slug}/admin/smtp", Method: http.MethodDelete, Handler: h.DeleteSmtp},
		{Path: "/tenants/{slug}/admin/smtp/test", Method: http.MethodPost, Handler: middlewares.WithStrictRateLimit(h.TestSmtp)},
		{Path: "/geocode", Method: http.MethodGet, Handler: h.Geocode},
	}
	for _, route := range routes {
		route.Handler = middlewares.AuthMiddleware(route.Handler)
		route.AdditionalMethods = []router.HTTPMethod{router.OPTIONS}
		r.router.NewRoute(route)
	}
}
