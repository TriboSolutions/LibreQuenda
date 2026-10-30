#!/bin/sh
# ============================================================
#  Zitadel bootstrap for LibreQuenda (idempotent, runs on every `up`)
#  - creates the "LibreQuenda" project and its "web" OIDC app (PKCE, no secret)
#  - keeps the app's redirect URIs in sync with the environment
#  - enables "Sign in with Google" when GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET are set
#  - signs up and in with a password (passkeys only with ZITADEL_PASSKEYS=true)
#  - brands the login and console as LibreQuenda (logo, colours, no Zitadel links)
#  - creates the "librequenda-api" service account (user manager only) and writes
#    its token to /api-secrets, for /perfil (name, email, password)
#  - writes /runtime/config.json, served by Caddy to the SPA; its scope also asks
#    for Zitadel's own audience, so the API can copy the profile photo to the
#    person's account with their token (api/zitadel/avatar.go)
# ============================================================
set -eu
# a failed API call inside a pipe must stop the script, never read as "not found"
set -o pipefail

PAT_FILE=/zitadel/bootstrap/admin.pat
API=http://zitadel-api:8080
PROJECT_NAME=LibreQuenda
APP_NAME=web

PUBLIC_ORIGIN="${PUBLIC_SCHEME}://${WEB_DOMAIN}"
ZITADEL_URL="${PUBLIC_SCHEME}://${ZITADEL_DOMAIN}"

log() { echo "[zitadel-setup] $*"; }

for i in $(seq 1 60); do
	[ -s "$PAT_FILE" ] && break
	log "waiting for $PAT_FILE ($i)"
	sleep 2
done
PAT=$(cat "$PAT_FILE")

# Zitadel resolves the instance from the Host header
call() {
	method=$1 path=$2 body=${3:-}
	curl -fsS -X "$method" "$API$path" \
		-H "Host: ${ZITADEL_DOMAIN}" \
		-H "Authorization: Bearer $PAT" \
		-H "Content-Type: application/json" \
		${body:+-d "$body"}
}

# --- project ---
PROJECT_ID=$(call POST /management/v1/projects/_search \
	"{\"queries\":[{\"nameQuery\":{\"name\":\"$PROJECT_NAME\",\"method\":\"TEXT_QUERY_METHOD_EQUALS\"}}]}" \
	| jq -r '.result[0].id // empty')

if [ -z "$PROJECT_ID" ]; then
	PROJECT_ID=$(call POST /management/v1/projects "{\"name\":\"$PROJECT_NAME\"}" | jq -r '.id')
	log "created project $PROJECT_NAME ($PROJECT_ID)"
else
	log "project $PROJECT_NAME exists ($PROJECT_ID)"
fi

# --- OIDC app ---
# Redirects: the proxied web app and the Vite dev server
OIDC_CONFIG=$(jq -n \
	--arg origin "$PUBLIC_ORIGIN" \
	--argjson dev "$([ "$PUBLIC_SCHEME" = http ] && echo true || echo false)" \
	'{
		redirectUris: ([$origin + "/callback"] + (if $dev then ["http://localhost:5173/callback"] else [] end)),
		postLogoutRedirectUris: ([$origin + "/"] + (if $dev then ["http://localhost:5173/"] else [] end)),
		responseTypes: ["OIDC_RESPONSE_TYPE_CODE"],
		grantTypes: ["OIDC_GRANT_TYPE_AUTHORIZATION_CODE", "OIDC_GRANT_TYPE_REFRESH_TOKEN"],
		appType: "OIDC_APP_TYPE_USER_AGENT",
		authMethodType: "OIDC_AUTH_METHOD_TYPE_NONE",
		accessTokenType: "OIDC_TOKEN_TYPE_JWT",
		idTokenUserinfoAssertion: true,
		devMode: $dev
	}')

APP=$(call POST "/management/v1/projects/$PROJECT_ID/apps/_search" \
	"{\"queries\":[{\"nameQuery\":{\"name\":\"$APP_NAME\",\"method\":\"TEXT_QUERY_METHOD_EQUALS\"}}]}" \
	| jq -c '.result[0] // empty')

if [ -z "$APP" ]; then
	CLIENT_ID=$(call POST "/management/v1/projects/$PROJECT_ID/apps/oidc" \
		"$(echo "$OIDC_CONFIG" | jq --arg name "$APP_NAME" '. + {name: $name}')" | jq -r '.clientId')
	log "created OIDC app $APP_NAME ($CLIENT_ID)"
else
	APP_ID=$(echo "$APP" | jq -r '.id')
	CLIENT_ID=$(echo "$APP" | jq -r '.oidcConfig.clientId')
	call PUT "/management/v1/projects/$PROJECT_ID/apps/$APP_ID/oidc_config" "$OIDC_CONFIG" >/dev/null 2>&1 \
		|| true # "no changes" is reported as an error
	log "OIDC app $APP_NAME exists ($CLIENT_ID), config synced"
fi

# --- Google login (optional) ---
# Enabled when GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET are set, removed from the login otherwise.
GOOGLE_IDP_NAME=Google

GOOGLE_IDP_ID=$(call POST /admin/v1/idps/templates/_search \
	"{\"queries\":[{\"idpNameQuery\":{\"name\":\"$GOOGLE_IDP_NAME\",\"method\":\"TEXT_QUERY_METHOD_EQUALS\"}}]}" \
	| jq -r '.result[0].id // empty')

if [ -n "${GOOGLE_CLIENT_ID:-}" ] && [ -n "${GOOGLE_CLIENT_SECRET:-}" ]; then
	GOOGLE_CONFIG=$(jq -n \
		--arg name "$GOOGLE_IDP_NAME" \
		--arg clientId "$GOOGLE_CLIENT_ID" \
		--arg clientSecret "$GOOGLE_CLIENT_SECRET" \
		'{
			name: $name,
			clientId: $clientId,
			clientSecret: $clientSecret,
			scopes: ["openid", "profile", "email"],
			providerOptions: {
				isCreationAllowed: true,
				isAutoCreation: true,
				isAutoUpdate: true,
				isLinkingAllowed: true
			}
		}')

	if [ -z "$GOOGLE_IDP_ID" ]; then
		GOOGLE_IDP_ID=$(call POST /admin/v1/idps/google "$GOOGLE_CONFIG" | jq -r '.id')
		log "created Google identity provider ($GOOGLE_IDP_ID)"
	else
		call PUT "/admin/v1/idps/google/$GOOGLE_IDP_ID" "$GOOGLE_CONFIG" >/dev/null 2>&1 \
			|| true # "no changes" is reported as an error
		log "Google identity provider exists ($GOOGLE_IDP_ID), config synced"
	fi

	if ! call POST /admin/v1/policies/login/idps/_search '{}' | jq -e --arg id "$GOOGLE_IDP_ID" '.result[]? | select(.idpId == $id)' >/dev/null; then
		call POST /admin/v1/policies/login/idps "{\"idpId\":\"$GOOGLE_IDP_ID\"}" >/dev/null
		log "Google login enabled"
	fi
elif [ -n "$GOOGLE_IDP_ID" ]; then
	call DELETE "/admin/v1/policies/login/idps/$GOOGLE_IDP_ID" >/dev/null 2>&1 || true
	log "GOOGLE_CLIENT_ID not set: Google login disabled"
fi

# --- sign-up and sign-in with a password ---
# Without passkeys the login goes straight to a password, with no "passkey or
# password" step; ZITADEL_PASSKEYS=true offers passkeys again. The rest of the
# instance's login policy is kept as it is.
if [ "${ZITADEL_PASSKEYS:-false}" = "true" ]; then PASSWORDLESS=PASSWORDLESS_TYPE_ALLOWED; else PASSWORDLESS=PASSWORDLESS_TYPE_NOT_ALLOWED; fi
LOGIN_POLICY=$(call GET /admin/v1/policies/login | jq -c --arg pw "$PASSWORDLESS" '.policy | del(.details, .isDefault, .idps) | .passwordlessType = $pw')
call PUT /admin/v1/policies/login "$LOGIN_POLICY" >/dev/null 2>&1 || true # "no changes" is reported as an error
log "login: passkeys ${ZITADEL_PASSKEYS:-false}"

# --- branding: residents see LibreQuenda, not Zitadel ---
# Colours from web/src/globals.css (light and dark); no "powered by" watermark.
call PUT /admin/v1/policies/label '{
	"primaryColor": "#0a6fb0", "warnColor": "#d93f3f", "backgroundColor": "#f6f9fb", "fontColor": "#111827",
	"primaryColorDark": "#5fb0e6", "warnColorDark": "#e5605a", "backgroundColorDark": "#0e0f10", "fontColorDark": "#f3f4f6",
	"disableWatermark": true, "hideLoginNameSuffix": true, "themeMode": "THEME_MODE_AUTO"
}' >/dev/null 2>&1 || true # "no changes" is reported as an error
for asset in logo:librequenda-logo.png logo/dark:librequenda-logo-blanco.png icon:librequenda-simbolo.png icon/dark:librequenda-simbolo-blanco.png; do
	curl -fsS -X POST "$API/assets/v1/instance/policy/label/${asset%%:*}" \
		-H "Host: ${ZITADEL_DOMAIN}" \
		-H "Authorization: Bearer $PAT" \
		-F "file=@/brand/${asset#*:};type=image/png" >/dev/null
done
call POST /admin/v1/policies/label/_activate '{}' >/dev/null
# No documentation, help or support links pointing at zitadel.com in the console
call PUT /admin/v1/policies/privacy '{"tosLink":"","privacyLink":"","helpLink":"","supportEmail":"","docsLink":"","customLink":"","customLinkText":""}' \
	>/dev/null 2>&1 || true
log "branding applied"

# --- service account for the API: residents manage their account from /perfil ---
# Only ORG_USER_MANAGER on the organisation residents register in, never IAM owner.
API_USER=librequenda-api
API_PAT_FILE=/api-secrets/zitadel-users.pat
API_USER_ID=$(call POST /management/v1/users/_search \
	"{\"queries\":[{\"userNameQuery\":{\"userName\":\"$API_USER\",\"method\":\"TEXT_QUERY_METHOD_EQUALS\"}}]}" \
	| jq -r '.result[0].id // empty')
if [ -z "$API_USER_ID" ]; then
	API_USER_ID=$(call POST /management/v1/users/machine \
		"{\"userName\":\"$API_USER\",\"name\":\"LibreQuenda API\",\"accessTokenType\":\"ACCESS_TOKEN_TYPE_BEARER\"}" | jq -r '.userId')
	log "created service account $API_USER ($API_USER_ID)"
fi
call POST /management/v1/orgs/me/members "{\"userId\":\"$API_USER_ID\",\"roles\":[\"ORG_USER_MANAGER\"]}" \
	>/dev/null 2>&1 || true # already a member
if [ ! -s "$API_PAT_FILE" ]; then
	call POST "/management/v1/users/$API_USER_ID/pats" '{"expirationDate":"2099-01-01T00:00:00Z"}' | jq -r '.token' > "$API_PAT_FILE.tmp"
	mv "$API_PAT_FILE.tmp" "$API_PAT_FILE"
	log "wrote the token of $API_USER"
fi

# --- runtime config for the SPA ---
jq -n \
	--arg authority "$ZITADEL_URL" \
	--arg clientId "$CLIENT_ID" \
	--arg projectId "$PROJECT_ID" \
	'{
		authority: $authority,
		clientId: $clientId,
		projectId: $projectId,
		scope: ("openid profile email offline_access urn:zitadel:iam:org:project:id:" + $projectId + ":aud urn:zitadel:iam:org:project:id:zitadel:aud")
	}' > /runtime/config.json.tmp
mv /runtime/config.json.tmp /runtime/config.json
log "wrote /runtime/config.json"
