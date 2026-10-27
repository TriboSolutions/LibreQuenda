#!/bin/sh
# ============================================================
#  CADDY_AUTO_HTTPS=true  -> Caddy obtains and renews TLS certificates
#                            (Let's Encrypt for real domains, its local CA for *.localhost)
#  CADDY_AUTO_HTTPS=false -> plain HTTP on port 80: local development, or TLS
#                            terminated by another proxy / load balancer in front
# ============================================================
set -eu

: "${WEB_DOMAIN:?WEB_DOMAIN is not set}"
: "${ZITADEL_DOMAIN:?ZITADEL_DOMAIN is not set}"

case "${CADDY_AUTO_HTTPS:-false}" in
	true) SITE_SCHEME=https ;;
	false) SITE_SCHEME=http ;;
	*)
		echo "[proxy] CADDY_AUTO_HTTPS must be true or false, got '${CADDY_AUTO_HTTPS}'" >&2
		exit 1
		;;
esac

export WEB_ADDRESS="${SITE_SCHEME}://${WEB_DOMAIN}"
export ZITADEL_ADDRESS="${SITE_SCHEME}://${ZITADEL_DOMAIN}"
echo "[proxy] automatic HTTPS: ${CADDY_AUTO_HTTPS:-false} -> ${WEB_ADDRESS}, ${ZITADEL_ADDRESS}"

exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
