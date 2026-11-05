package middlewares

import (
	"librequenda/auth"
	"net/http"
	"strings"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

// AuthMiddleware requires a valid Zitadel access token (Authorization: Bearer <jwt>).
func AuthMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return VersionMiddleware(withAccessToken(handler))
}

func withAccessToken(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var accessToken, ok = strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || accessToken == "" {
			helpers.SendI18NErrorResponse(w, r, http.StatusUnauthorized, "errors.unauthorized")
			return
		}

		var claims, err = auth.VerifyAccessToken(accessToken)
		if err != nil {
			helpers.SendI18NErrorResponse(w, r, http.StatusUnauthorized, "errors.unauthorized")
			return
		}

		var ctx = auth.WithSession(r.Context(), auth.Session{
			Subject:     claims.Subject,
			AccessToken: accessToken,
		})
		handler(w, r.WithContext(ctx))
	}
}

// SignedIn: whether the request carries a valid access token, on a route anyone can call.
func SignedIn(r *http.Request) bool {
	var accessToken, ok = strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || accessToken == "" {
		return false
	}
	var _, err = auth.VerifyAccessToken(accessToken)
	return err == nil
}
