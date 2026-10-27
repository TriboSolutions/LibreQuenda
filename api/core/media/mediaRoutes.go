package media

import (
	"database/sql"
	"librequenda/middlewares"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router"
)

type mediaRoutes struct {
	db     *sql.DB
	router *router.Router
}

var routesOnce *sync.Once = new(sync.Once)
var mediaRoutesInstance *mediaRoutes

func MediaRoutes(db *sql.DB, router *router.Router) *mediaRoutes {
	routesOnce.Do(func() {
		mediaRoutesInstance = &mediaRoutes{
			db:     db,
			router: router,
		}
	})
	return mediaRoutesInstance
}

func (r *mediaRoutes) RegisterRoutes() {
	var h = MediaHandler(r.db)
	r.router.NewRoute(router.Route{
		Path:              "/media",
		Method:            http.MethodPost,
		AdditionalMethods: []router.HTTPMethod{router.OPTIONS},
		Handler:           middlewares.AuthMiddleware(h.Upload),
	})
	// Public: images on council and facility pages
	r.router.NewRoute(router.Route{
		Path:              "/media/{ref...}",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD},
		Handler:           h.Serve,
	})
}
