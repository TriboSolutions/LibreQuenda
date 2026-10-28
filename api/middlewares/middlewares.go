package middlewares

import (
	"librequenda/shared"
	"net/http"

	"github.com/angelbarreiros/Penguin/router/cors"
	"github.com/angelbarreiros/Penguin/router/helpers"
	"github.com/angelbarreiros/Penguin/router/middlewares"
)

var corsConfig = cors.NewCORSConfig(
	cors.WithAllowedOrigins([]string{cors.AllowAllOrigin}),
	cors.WithAllowedHeaders([]string{"Authorization", "Content-Type", "X-App-Version"}),
	cors.WithExposedHeaders([]string{"X-App-Version"}),
	cors.WithMaxAge(3600),
)

// VersionMiddleware is the base of every API route (AuthMiddleware wraps it):
// CORS, the default rate limit and the app version check.
func VersionMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return middlewares.WithCors(corsConfig, WithRateLimit(withVersion(handler)))
}

// Rate limits are per client IP and per route. The IP comes from X-Forwarded-For
// only when the request comes from a trusted proxy (Caddy), so users behind it
// are not throttled together.
func rateLimit(handler http.HandlerFunc, burst int32, perSecond float64) http.HandlerFunc {
	return middlewares.WithRateLimiting(handler,
		middlewares.RateLimitOptStartingLimit(burst),
		middlewares.RateLimitOptLimitPerSecond(perSecond),
		middlewares.RateLimitOptTrustedProxies(shared.TRUSTED_PROXIES),
		middlewares.RateLimitOptTrustedProxyHeader("X-Forwarded-For"),
	)
}

// WithRateLimit: the default for every route, roomy enough for a council's
// office sharing one IP.
func WithRateLimit(handler http.HandlerFunc) http.HandlerFunc {
	return rateLimit(handler, 60, 10)
}

// WithStrictRateLimit: 5 at once, then one every 12 seconds. For what sends
// email, checks codes or passwords, reaches other servers, or lets anonymous
// people write.
func WithStrictRateLimit(handler http.HandlerFunc) http.HandlerFunc {
	return rateLimit(handler, 5, 1.0/12)
}

func withVersion(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var appVersion = shared.VERSION
		if appVersion != "" {
			w.Header().Set("X-App-Version", appVersion)
		}

		if r.Method != http.MethodOptions {
			var clientVersion = r.Header.Get("X-App-Version")
			if appVersion != "" && clientVersion != "" && clientVersion != appVersion {
				helpers.SendI18NErrorResponse(w, r, http.StatusConflict, "errors.version_mismatch")
				return
			}
		}

		handler(w, r)
	}
}
