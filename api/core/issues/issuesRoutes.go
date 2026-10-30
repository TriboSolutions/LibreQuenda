package issues

import (
	"database/sql"
	"librequenda/middlewares"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router"
)

type issuesRoutes struct {
	db     *sql.DB
	router *router.Router
}

var routesOnce *sync.Once = new(sync.Once)
var issuesRoutesInstance *issuesRoutes

func IssuesRoutes(db *sql.DB, router *router.Router) *issuesRoutes {
	routesOnce.Do(func() {
		issuesRoutesInstance = &issuesRoutes{db: db, router: router}
	})
	return issuesRoutesInstance
}

// A signed-in resident reports a problem at a facility; strict: reports reach
// people who have to act on them, so a burst of them is refused.
func (r *issuesRoutes) RegisterRoutes() {
	var h = IssuesHandler(r.db)
	r.router.NewRoute(router.Route{
		Path:              "/me/issues",
		Method:            http.MethodPost,
		AdditionalMethods: []router.HTTPMethod{router.OPTIONS},
		Handler:           middlewares.AuthMiddleware(middlewares.WithStrictRateLimit(h.Report)),
	})
}
