package entrance

import (
	"database/sql"
	"librequenda/middlewares"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router"
)

type entranceRoutes struct {
	db     *sql.DB
	router *router.Router
}

var routesOnce *sync.Once = new(sync.Once)
var entranceRoutesInstance *entranceRoutes

func EntranceRoutes(db *sql.DB, router *router.Router) *entranceRoutes {
	routesOnce.Do(func() {
		entranceRoutesInstance = &entranceRoutes{
			db:     db,
			router: router,
		}
	})
	return entranceRoutesInstance
}

// The resident's own phone at the door: the printed QR opens /<council>/<facility>/entrada.
func (r *entranceRoutes) RegisterRoutes() {
	var h = EntranceHandler(r.db)
	r.router.NewRoute(router.Route{
		Path:              "/tenants/{slug}/facilities/{facility}/entrance",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.OPTIONS},
		Handler:           middlewares.AuthMiddleware(h.Get),
	})
	// strict: the entry code is guessable only by trying many
	r.router.NewRoute(router.Route{
		Path:              "/tenants/{slug}/facilities/{facility}/entrance",
		Method:            http.MethodPost,
		AdditionalMethods: []router.HTTPMethod{router.OPTIONS},
		Handler:           middlewares.AuthMiddleware(middlewares.WithStrictRateLimit(h.CheckIn)),
	})
	// flash booking with the entry code: which facility the phone is at
	r.router.NewRoute(router.Route{
		Path:              "/entrance/nearby",
		Method:            http.MethodPost,
		AdditionalMethods: []router.HTTPMethod{router.OPTIONS},
		Handler:           middlewares.AuthMiddleware(h.Nearby),
	})
}
