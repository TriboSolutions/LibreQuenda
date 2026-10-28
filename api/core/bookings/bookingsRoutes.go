package bookings

import (
	"database/sql"
	"librequenda/middlewares"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router"
)

type bookingsRoutes struct {
	db     *sql.DB
	router *router.Router
}

var routesOnce *sync.Once = new(sync.Once)
var bookingsRoutesInstance *bookingsRoutes

func BookingsRoutes(db *sql.DB, router *router.Router) *bookingsRoutes {
	routesOnce.Do(func() {
		bookingsRoutesInstance = &bookingsRoutes{
			db:     db,
			router: router,
		}
	})
	return bookingsRoutesInstance
}

func (r *bookingsRoutes) RegisterRoutes() {
	var h = BookingsHandler(r.db)
	r.router.NewRoute(router.Route{
		Path:              "/me/bookings",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
		Handler:           middlewares.AuthMiddleware(h.GetMyBookings),
	})
	r.router.NewRoute(router.Route{
		Path:    "/me/bookings",
		Method:  http.MethodPost,
		Handler: middlewares.AuthMiddleware(h.CreateBooking),
	})
	r.router.NewRoute(router.Route{
		Path:              "/me/bookings/{id}",
		Method:            http.MethodDelete,
		AdditionalMethods: []router.HTTPMethod{router.OPTIONS},
		Handler:           middlewares.AuthMiddleware(h.CancelBooking),
	})
	r.router.NewRoute(router.Route{
		Path:    "/me/bookings/{id}",
		Method:  http.MethodPatch,
		Handler: middlewares.AuthMiddleware(h.ChangeBooking),
	})
}
