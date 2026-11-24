#!/bin/sh
# ============================================================
#  End-to-end check of the sign-in with a public identity provider, against a
#  Keycloak standing in for Cl@ve or Chave365 (development only, needs Docker):
#  - SAML creates the LibreQuenda account, with the name, surname and email from
#    the attributes and the NIF kept in Zitadel's metadata
#  - OIDC, as the same person, signs in to that same account (linked by email)
#  The stack is built from this checkout and runs as its own compose project with
#  no published port. Usage: zitadel/test-idp/run.sh [--keep] (leave it running)
# ============================================================
set -eu
cd "$(dirname "$0")/../.."

PROJECT=${PROJECT:-lq-idp-test}
PLAYWRIGHT=1.65.1
ENV_FILE=$(mktemp)
KEEP=$([ "${1:-}" = --keep ] && echo true || echo false)

secret() { tr -dc A-Za-z0-9 </dev/urandom | head -c "${1:-32}"; }
cat >"$ENV_FILE" <<EOF
LIBREQUENDA_VERSION=idp-test
POSTGRES_DB=librequenda
POSTGRES_USER=librequenda
POSTGRES_PASSWORD=$(secret)
APP_DB_PASSWORD=$(secret)
ANALYTICS_DB_PASSWORD=$(secret)
PUBLIC_SCHEME=http
PUBLIC_PORT=80
WEB_DOMAIN=localhost
ZITADEL_DOMAIN=auth.localhost
ZITADEL_EXTERNALSECURE=false
CADDY_AUTO_HTTPS=false
ZITADEL_VERSION=v4.20.0
ZITADEL_MASTERKEY=$(secret)
LOGIN_SESSION_COOKIE_SECRET=$(secret)
ZITADEL_DB_PASSWORD=$(secret)
ZITADEL_ADMIN_EMAIL=admin@example.gal
ZITADEL_ADMIN_FIRSTNAME=Admin
ZITADEL_ADMIN_LASTNAME=Proba
ZITADEL_ADMIN_PASSWORD=Proba-$(secret 12)-1
STORAGE_ACCESS_KEY=GK$(openssl rand -hex 12)
STORAGE_SECRET_KEY=$(openssl rand -hex 32)
GARAGE_RPC_SECRET=$(openssl rand -hex 32)
GARAGE_ADMIN_TOKEN=$(secret)
SECRETS_KEY=$(openssl rand -hex 32)
SMTP_FROM_ADDRESS=noreply@example.gal
REQUIRE_ADMIN_MFA=false
SAML_IDP_NAME=Cl@ve
SAML_IDP_METADATA_URL=http://keycloak:8080/realms/clave-test/protocol/saml/descriptor
SAML_IDP_EMAIL_VERIFIED=true
ZITADEL_HTTPCLIENT_DENYLIST=localhost,0.0.0.0/8,100.64.0.0/10,127.0.0.0/8,169.254.0.0/16,198.18.0.0/15,::/128,::1/128,fe80::/10
OIDC_IDP_NAME=Chave365 OIDC
OIDC_IDP_ISSUER=http://keycloak:8080/realms/clave-test
OIDC_IDP_CLIENT_ID=librequenda-oidc
OIDC_IDP_CLIENT_SECRET=test-oidc-secret
EOF

compose() {
	docker compose -p "$PROJECT" --project-directory . --env-file "$ENV_FILE" \
		-f docker-compose.yaml -f zitadel/test-idp/compose.yaml "$@"
}
finish() {
	status=$?
	if [ "$KEEP" = true ]; then
		echo "left running: docker compose -p $PROJECT down -v (env file $ENV_FILE)"
	else
		compose down -v --remove-orphans >/dev/null 2>&1 || true
		rm -f "$ENV_FILE"
	fi
	[ "$status" -eq 0 ] && echo "PASS" || echo "FAIL"
}
trap finish EXIT
fail() {
	echo "FAIL: $*" >&2
	exit 1
}

compose up -d --build --wait --quiet-pull
compose logs zitadel-setup-idp | grep -q "SAML attributes mapped by the API" ||
	fail "zitadel-setup-idp did not set up the attribute mapping: $(compose logs zitadel-setup-idp)"

# Keycloak learns the service provider (Zitadel's SAML identity provider) once it exists
SP=$(compose logs zitadel-setup | grep -o 'http://auth.localhost/idps/[0-9]*/saml/metadata' | tail -1)
[ -n "$SP" ] || fail "no SAML service provider in zitadel-setup's log"
compose exec -T keycloak sh -c '/opt/keycloak/bin/kcadm.sh config credentials --server http://localhost:8080 --realm master --user admin --password admin >/dev/null &&
	/opt/keycloak/bin/kcadm.sh create clients -r clave-test -f -' >/dev/null <<EOF
{
  "clientId": "$SP",
  "protocol": "saml",
  "redirectUris": ["http://auth.localhost/*"],
  "attributes": {
    "saml_assertion_consumer_url_post": "${SP%/metadata}/acs",
    "saml.client.signature": "false",
    "saml.server.signature": "true",
    "saml_force_name_id_format": "true",
    "saml_name_id_format": "persistent"
  },
  "protocolMappers": [
    {"name": "given", "protocol": "saml", "protocolMapper": "saml-user-property-mapper",
     "config": {"user.attribute": "firstName", "attribute.name": "http://eidas.europa.eu/attributes/naturalperson/CurrentGivenName", "attribute.nameformat": "URI Reference"}},
    {"name": "family", "protocol": "saml", "protocolMapper": "saml-user-property-mapper",
     "config": {"user.attribute": "lastName", "attribute.name": "http://eidas.europa.eu/attributes/naturalperson/CurrentFamilyName", "attribute.nameformat": "URI Reference"}},
    {"name": "email", "protocol": "saml", "protocolMapper": "saml-user-property-mapper",
     "config": {"user.attribute": "email", "attribute.name": "urn:oid:0.9.2342.19200300.100.1.3", "attribute.nameformat": "URI Reference"}},
    {"name": "nif", "protocol": "saml", "protocolMapper": "saml-user-property-mapper",
     "config": {"user.attribute": "username", "attribute.name": "http://eidas.europa.eu/attributes/naturalperson/PersonIdentifier", "attribute.nameformat": "URI Reference"}}
  ]
}
EOF

signin() {
	docker run --rm --network "container:$(compose ps -q proxy)" -v "$PWD/zitadel/test-idp:/test:ro" \
		"mcr.microsoft.com/playwright:v$PLAYWRIGHT-noble" sh -c \
		"mkdir -p /tmp/t && cp /test/login.mjs /tmp/t && cd /tmp/t && npm i -s --no-save playwright-core@$PLAYWRIGHT >/dev/null 2>&1 && node login.mjs '$1'"
}

SAML=$(signin Cl@ve) || fail "SAML sign-in: $SAML"
echo "SAML: $SAML"
echo "$SAML" | jq -e '.status == 200 and .me.email == "uxia.castro@example.gal" and .me.full_name == "Uxía Castro Pazos" and .me.email_verified' >/dev/null ||
	fail "the account does not carry the SAML attributes"
SUB=$(echo "$SAML" | jq -r .sub)

NIF=$(compose exec -T api sh -c "curl -fsS -X POST -H 'Host: auth.localhost' -H \"Authorization: Bearer \$(cat /secrets/zitadel-users.pat)\" \
	-H 'Content-Type: application/json' -d '{}' http://zitadel-api:8080/management/v1/users/$SUB/metadata/_search" |
	jq -r '.result[]? | select(.key == "nif") | .value' | base64 -d)
echo "NIF in Zitadel's metadata: $NIF"
[ "$NIF" = 12345678Z ] || fail "the NIF was not kept"

OIDC=$(signin "Chave365 OIDC") || fail "OIDC sign-in: $OIDC"
echo "OIDC: $OIDC"
[ "$(echo "$OIDC" | jq -r .sub)" = "$SUB" ] || fail "OIDC signed in to another account, not the one SAML created"

ACCOUNTS=$(compose exec -T db psql -U librequenda -d librequenda -tAc "SELECT count(*) FROM app_user WHERE email = 'uxia.castro@example.gal'")
[ "$ACCOUNTS" = 1 ] || fail "$ACCOUNTS LibreQuenda accounts for the person"
