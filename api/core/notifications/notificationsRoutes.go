package notifications

import (
	"database/sql"
	"librequenda/middlewares"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router"
)

type notificationsRoutes struct {
	db     *sql.DB
	router *router.Router
}

var routesOnce *sync.Once = new(sync.Once)
var notificationsRoutesInstance *notificationsRoutes

func NotificationsRoutes(db *sql.DB, router *router.Router) *notificationsRoutes {
	routesOnce.Do(func() {
		notificationsRoutesInstance = &notificationsRoutes{
			db:     db,
			router: router,
		}
	})
	return notificationsRoutesInstance
}

// The web polls GET /me/notifications; no push channel is needed for this.
func (r *notificationsRoutes) RegisterRoutes() {
	var h = NotificationsHandler(r.db)
	r.router.NewRoute(router.Route{
		Path:              "/me/notifications",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
		Handler:           middlewares.AuthMiddleware(h.GetNotifications),
	})
	r.router.NewRoute(router.Route{
		Path:              "/me/notifications/read",
		Method:            http.MethodPost,
		AdditionalMethods: []router.HTTPMethod{router.OPTIONS},
		Handler:           middlewares.AuthMiddleware(h.MarkAllRead),
	})
}
