package platform

import (
	"database/sql"
	"librequenda/audit"
	"librequenda/middlewares"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router"
)

type platformRoutes struct {
	db     *sql.DB
	router *router.Router
}

var routesOnce *sync.Once = new(sync.Once)
var platformRoutesInstance *platformRoutes

func PlatformRoutes(db *sql.DB, router *router.Router) *platformRoutes {
	routesOnce.Do(func() {
		platformRoutesInstance = &platformRoutes{
			db:     db,
			router: router,
		}
	})
	return platformRoutesInstance
}

// Superadministrator area: every handler checks the platform_admin role itself.
// One council's details and administrators are also open to that council's
// administrators (councilAdmin).
func (r *platformRoutes) RegisterRoutes() {
	var h = PlatformHandler(r.db)
	var routes = []router.Route{
		{Path: "/admin/tenants", Method: http.MethodGet, Handler: h.ListTenants},
		{Path: "/admin/tenants", Method: http.MethodPost, Handler: h.CreateTenant},
		{Path: "/admin/tenants/{slug}", Method: http.MethodGet, Handler: h.GetTenant},
		{Path: "/admin/tenants/{slug}", Method: http.MethodPut, Handler: h.UpdateTenant},
		{Path: "/admin/tenants/{slug}", Method: http.MethodDelete, Handler: h.DeleteTenant},
		{Path: "/admin/tenants/{slug}/invitations", Method: http.MethodPost, Handler: middlewares.WithStrictRateLimit(h.InviteTenantAdmin)},
		{Path: "/admin/tenants/{slug}/invitations/{id}", Method: http.MethodDelete, Handler: h.DeleteInvitation},
		{Path: "/admin/tenants/{slug}/admins/{userId}", Method: http.MethodDelete, Handler: h.RemoveTenantAdmin},
		{Path: "/admin/superadmins", Method: http.MethodGet, Handler: h.ListSuperadmins},
		{Path: "/admin/superadmins", Method: http.MethodPost, Handler: h.AddSuperadmin},
		{Path: "/admin/superadmins/{userId}", Method: http.MethodDelete, Handler: h.RemoveSuperadmin},
		{Path: "/admin/tag-categories", Method: http.MethodPost, Handler: h.CreateTagCategory},
		{Path: "/admin/tag-categories/{id}", Method: http.MethodPut, Handler: h.UpdateTagCategory},
		{Path: "/admin/tag-categories/{id}", Method: http.MethodDelete, Handler: h.DeleteTagCategory},
		{Path: "/admin/tags", Method: http.MethodPost, Handler: h.CreateTag},
		{Path: "/admin/tags/{id}", Method: http.MethodPut, Handler: h.UpdateTag},
		{Path: "/admin/tags/{id}", Method: http.MethodDelete, Handler: h.DeleteTag},
		{Path: "/admin/branding", Method: http.MethodPut, Handler: h.UpdateBranding},
		// the analytics assistant: its language model, and questions about every council
		{Path: "/admin/ai-settings", Method: http.MethodGet, Handler: h.GetAISettings},
		{Path: "/admin/ai-settings", Method: http.MethodPut, Handler: h.SaveAISettings},
		{Path: "/admin/analytics/ask", Method: http.MethodPost, Handler: h.AskAnalytics},
		{Path: "/admin/audit", Method: http.MethodGet, Handler: h.GetAudit},
		{Path: "/admin/booking-policy", Method: http.MethodGet, Handler: h.GetBookingPolicy},
		{Path: "/admin/booking-policy", Method: http.MethodPut, Handler: h.SaveBookingPolicy},
	}
	for _, route := range routes {
		// every change is logged once it succeeded (audit), inside the authentication
		route.Handler = middlewares.AuthMiddleware(audit.Route(r.db, string(route.Method), route.Path, route.Handler))
		route.AdditionalMethods = []router.HTTPMethod{router.OPTIONS}
		r.router.NewRoute(route)
	}
	// the tag catalogue is public: the filters of every page use it
	r.router.NewRoute(router.Route{
		Path:              "/tags",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
		Handler:           middlewares.VersionMiddleware(h.TagCatalogue),
	})
	// so is the installation's name and logo: every page shows them
	r.router.NewRoute(router.Route{
		Path:              "/branding",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
		Handler:           middlewares.VersionMiddleware(h.Branding),
	})
}
