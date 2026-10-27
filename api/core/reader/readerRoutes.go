package reader

import (
	"database/sql"
	"librequenda/middlewares"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router"
)

type readerRoutes struct {
	db     *sql.DB
	router *router.Router
}

var routesOnce *sync.Once = new(sync.Once)
var readerRoutesInstance *readerRoutes

func ReaderRoutes(db *sql.DB, router *router.Router) *readerRoutes {
	routesOnce.Do(func() {
		readerRoutesInstance = &readerRoutes{
			db:     db,
			router: router,
		}
	})
	return readerRoutesInstance
}

// Entrance readers sign in with their device key (Authorization: Bearer lqd_…), not a user token.
func (r *readerRoutes) RegisterRoutes() {
	var h = ReaderHandler(r.db)
	r.router.NewRoute(router.Route{
		Path:              "/reader",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
		Handler:           middlewares.VersionMiddleware(h.GetReader),
	})
	r.router.NewRoute(router.Route{
		Path:              "/reader/checkin",
		Method:            http.MethodPost,
		AdditionalMethods: []router.HTTPMethod{router.OPTIONS},
		Handler:           middlewares.VersionMiddleware(h.CheckIn),
	})
}
