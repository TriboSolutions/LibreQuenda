package invites

import (
	"database/sql"
	"librequenda/middlewares"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router"
)

type invitesRoutes struct {
	db     *sql.DB
	router *router.Router
}

var routesOnce *sync.Once = new(sync.Once)
var invitesRoutesInstance *invitesRoutes

func InvitesRoutes(db *sql.DB, router *router.Router) *invitesRoutes {
	routesOnce.Do(func() {
		invitesRoutesInstance = &invitesRoutes{
			db:     db,
			router: router,
		}
	})
	return invitesRoutesInstance
}

func (r *invitesRoutes) RegisterRoutes() {
	var h = InvitesHandler(r.db)

	// owner: share the link, see and remove who joined
	r.router.NewRoute(router.Route{
		Path:              "/me/bookings/{id}/invite",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
		Handler:           middlewares.AuthMiddleware(h.GetInvite),
	})
	r.router.NewRoute(router.Route{
		Path:    "/me/bookings/{id}/invite",
		Method:  http.MethodPost,
		Handler: middlewares.AuthMiddleware(h.ShareInvite),
	})
	r.router.NewRoute(router.Route{
		Path:              "/me/bookings/{id}/participants/{participantId}",
		Method:            http.MethodDelete,
		AdditionalMethods: []router.HTTPMethod{router.OPTIONS},
		Handler:           middlewares.AuthMiddleware(h.RemoveParticipant),
	})

	// participant with an account
	r.router.NewRoute(router.Route{
		Path:              "/me/bookings/{id}/participation",
		Method:            http.MethodDelete,
		AdditionalMethods: []router.HTTPMethod{router.OPTIONS},
		Handler:           middlewares.AuthMiddleware(h.Leave),
	})
	r.router.NewRoute(router.Route{
		Path:              "/me/invites/{token}/join",
		Method:            http.MethodPost,
		AdditionalMethods: []router.HTTPMethod{router.OPTIONS},
		Handler:           middlewares.AuthMiddleware(h.JoinAsUser),
	})

	// Public: anyone with the link sees the booking and can join as a guest.
	// Guest joins are rate-limited per client IP.
	r.router.NewRoute(router.Route{
		Path:              "/invites/{token}",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
		Handler:           middlewares.VersionMiddleware(h.GetPublicInvite),
	})
	r.router.NewRoute(router.Route{
		Path:              "/invites/{token}/join",
		Method:            http.MethodPost,
		AdditionalMethods: []router.HTTPMethod{router.OPTIONS},
		Handler:           middlewares.VersionMiddleware(middlewares.WithRetries(h.JoinAsGuest)),
	})
}
