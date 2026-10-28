package calendar

import (
	"database/sql"
	"librequenda/middlewares"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router"
)

type calendarRoutes struct {
	db     *sql.DB
	router *router.Router
}

var routesOnce *sync.Once = new(sync.Once)
var calendarRoutesInstance *calendarRoutes

func CalendarRoutes(db *sql.DB, router *router.Router) *calendarRoutes {
	routesOnce.Do(func() {
		calendarRoutesInstance = &calendarRoutes{
			db:     db,
			router: router,
		}
	})
	return calendarRoutesInstance
}

func (r *calendarRoutes) RegisterRoutes() {
	var h = CalendarHandler(r.db)

	// /perfil: turn the feed on (or give it a new address), see it, turn it off
	r.router.NewRoute(router.Route{
		Path:              "/me/calendar",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
		Handler:           middlewares.AuthMiddleware(h.GetFeed),
	})
	r.router.NewRoute(router.Route{
		Path:    "/me/calendar",
		Method:  http.MethodPost,
		Handler: middlewares.AuthMiddleware(h.ResetFeed),
	})
	r.router.NewRoute(router.Route{
		Path:    "/me/calendar",
		Method:  http.MethodDelete,
		Handler: middlewares.AuthMiddleware(h.TurnOffFeed),
	})

	// Public: calendar apps fetch /calendar/<token>.ics with no sign-in
	r.router.NewRoute(router.Route{
		Path:              "/calendar/{file}",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD},
		Handler:           middlewares.VersionMiddleware(h.Ics),
	})
}
