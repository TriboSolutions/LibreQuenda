package shared

import (
	"os"
	"strings"
)

var DB_USER = os.Getenv("DB_USER") // librequenda_app, never the owner
var DB_PASSWORD = os.Getenv("DB_PASSWORD")

// ANALYTICS_DB_PASSWORD: the read-only analytics user's (librequenda_analytics); "" turns the analytics queries off.
var ANALYTICS_DB_PASSWORD = os.Getenv("ANALYTICS_DB_PASSWORD")
var POSTGRES_DB = os.Getenv("POSTGRES_DB")
var POSTGRES_PORT = os.Getenv("POSTGRES_PORT")
var VERSION = os.Getenv("VERSION")

// Identity (Zitadel)
var OIDC_ISSUER = os.Getenv("OIDC_ISSUER")                       // public issuer, e.g. http://auth.localhost
var ZITADEL_DOMAIN = os.Getenv("ZITADEL_DOMAIN")                 // Host header Zitadel resolves the instance from
var ZITADEL_INTERNAL_URL = os.Getenv("ZITADEL_INTERNAL_URL")     // e.g. http://zitadel-api:8080
var RUNTIME_CONFIG_PATH = os.Getenv("RUNTIME_CONFIG_PATH")       // config.json written by zitadel-setup
var ZITADEL_USERS_PAT_FILE = os.Getenv("ZITADEL_USERS_PAT_FILE") // token of the user-manager service account

// Administrators must sign in with a second factor ("true"; off unless set, so a
// local stack works with password-only accounts). Checked through token
// introspection, with the API application's credentials written by zitadel-setup.
var REQUIRE_ADMIN_MFA = os.Getenv("REQUIRE_ADMIN_MFA") == "true"
var ZITADEL_INTROSPECTION_FILE = envOr("ZITADEL_INTROSPECTION_FILE", "/secrets/zitadel-introspection.json")

// The first superadministrator: granted on this person's login with a verified email.
var SUPERADMIN_EMAIL = strings.ToLower(strings.TrimSpace(os.Getenv("SUPERADMIN_EMAIL")))

// Image storage (Garage, S3-compatible)
var STORAGE_ENDPOINT = os.Getenv("STORAGE_ENDPOINT") // host:port, e.g. storage:3900
var STORAGE_REGION = os.Getenv("STORAGE_REGION")
var STORAGE_BUCKET = os.Getenv("STORAGE_BUCKET")
var STORAGE_ACCESS_KEY = os.Getenv("STORAGE_ACCESS_KEY")
var STORAGE_SECRET_KEY = os.Getenv("STORAGE_SECRET_KEY")
var STORAGE_USE_SSL = os.Getenv("STORAGE_USE_SSL") == "true"

// Support: who people write to or call when they do not understand something in the app.
// Each installation is its own (self-hosted), so they come from the environment; "" = not shown.
var SUPPORT_EMAIL = strings.TrimSpace(os.Getenv("SUPPORT_EMAIL"))
var SUPPORT_PHONE = strings.TrimSpace(os.Getenv("SUPPORT_PHONE"))

// Address search for facility map pins (OpenStreetMap Nominatim, or a self-hosted one)
var GEOCODER_URL = strings.TrimRight(envOr("GEOCODER_URL", "https://nominatim.openstreetmap.org"), "/")

func envOr(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

// Proxies (IPs or CIDRs) whose X-Forwarded-For is trusted for the client IP:
// the Caddy proxy on the Docker network by default.
var TRUSTED_PROXIES = trustedProxies(os.Getenv("TRUSTED_PROXIES"))

func trustedProxies(value string) []string {
	if value == "" {
		return []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.1/32", "::1/128"}
	}
	var proxies []string
	for _, p := range strings.Split(value, ",") {
		if p = strings.TrimSpace(p); p != "" {
			proxies = append(proxies, p)
		}
	}
	return proxies
}

// Outgoing email: the relay for councils without one of their own (tenant_smtp).
// Empty SMTP_HOST = only councils with their own relay get emails.
// STARTTLS is used when the server offers it; port 465 (implicit TLS) is not supported.
var SMTP_HOST = os.Getenv("SMTP_HOST")
var SMTP_PORT = envOr("SMTP_PORT", "587")
var SMTP_USERNAME = os.Getenv("SMTP_USERNAME")
var SMTP_PASSWORD = os.Getenv("SMTP_PASSWORD")
var SMTP_FROM = os.Getenv("SMTP_FROM") // e.g. LibreQuenda <noreply@concello.gal>

// Key that encrypts stored credentials (a council's SMTP password): 32 bytes in
// hex (openssl rand -hex 32). Losing it means typing those passwords again.
var SECRETS_KEY = os.Getenv("SECRETS_KEY")

// Address of the web app, for links in emails, e.g. https://reservas.concello.gal
var PUBLIC_URL = strings.TrimRight(os.Getenv("PUBLIC_URL"), "/")
