package platform

import (
	"database/sql"
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
	}
	for _, route := range routes {
		route.Handler = middlewares.AuthMiddleware(route.Handler)
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
}
