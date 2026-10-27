package me

import (
	"database/sql"
	"librequenda/middlewares"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router"
)

type meRoutes struct {
	db     *sql.DB
	router *router.Router
}

var routesOnce *sync.Once = new(sync.Once)
var meRoutesInstance *meRoutes

func MeRoutes(db *sql.DB, router *router.Router) *meRoutes {
	routesOnce.Do(func() {
		meRoutesInstance = &meRoutes{
			db:     db,
			router: router,
		}
	})
	return meRoutesInstance
}

func (r *meRoutes) RegisterRoutes() {
	var h = MeHandler(r.db)
	r.router.NewRoute(router.Route{
		Path:              "/me",
		Method:            http.MethodGet,
		AdditionalMethods: []router.HTTPMethod{router.HEAD, router.OPTIONS},
		Handler:           middlewares.AuthMiddleware(h.GetMe),
	})
	r.router.NewRoute(router.Route{
		Path:              "/me/profile",
		Method:            http.MethodPut,
		AdditionalMethods: []router.HTTPMethod{router.OPTIONS},
		Handler:           middlewares.AuthMiddleware(h.UpdateProfile),
	})

	// the account in Zitadel, edited from /perfil (never Zitadel's console)
	var account = []router.Route{
		{Path: "/me/account", Method: http.MethodGet, Handler: h.GetAccount},
		{Path: "/me/account/name", Method: http.MethodPut, Handler: h.SetName},
		// password guesses and code guesses are rate-limited per client IP
		{Path: "/me/account/password", Method: http.MethodPut, Handler: middlewares.WithRetries(h.SetPassword)},
		{Path: "/me/account/email", Method: http.MethodPut, Handler: h.SetEmail},
		{Path: "/me/account/email/resend", Method: http.MethodPost, Handler: middlewares.WithRetries(h.ResendEmailCode)},
		{Path: "/me/account/email/verify", Method: http.MethodPost, Handler: middlewares.WithRetries(h.VerifyEmail)},
	}
	for _, route := range account {
		route.Handler = middlewares.AuthMiddleware(route.Handler)
		route.AdditionalMethods = []router.HTTPMethod{router.OPTIONS}
		r.router.NewRoute(route)
	}
}
