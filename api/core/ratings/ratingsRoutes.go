package ratings

import (
	"database/sql"
	"librequenda/middlewares"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router"
)

type ratingsRoutes struct {
	db     *sql.DB
	router *router.Router
}

var routesOnce *sync.Once = new(sync.Once)
var ratingsRoutesInstance *ratingsRoutes

func RatingsRoutes(db *sql.DB, router *router.Router) *ratingsRoutes {
	routesOnce.Do(func() {
		ratingsRoutesInstance = &ratingsRoutes{db: db, router: router}
	})
	return ratingsRoutesInstance
}

// After a booking: the facility's stars and, if it was public, the others'.
func (r *ratingsRoutes) RegisterRoutes() {
	var h = RatingsHandler(r.db)
	r.router.NewRoute(router.Route{
		Path:              "/me/bookings/{id}/rating",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
		Handler:           middlewares.AuthMiddleware(h.GetRating),
	})
	r.router.NewRoute(router.Route{
		Path:    "/me/bookings/{id}/rating",
		Method:  http.MethodPut,
		Handler: middlewares.AuthMiddleware(h.Rate),
	})
}
