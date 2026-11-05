#!/bin/sh
# ============================================================
#  Zitadel bootstrap for LibreQuenda (idempotent, runs on every `up`)
#  - creates the "LibreQuenda" project and its "web" OIDC app (PKCE, no secret)
#  - keeps the app's redirect URIs in sync with the environment
#  - enables "Sign in with Google" when GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET are set
#  - signs up and in with a password (passkeys only with ZITADEL_PASSKEYS=true)
#  - short access tokens (ZITADEL_ACCESS_TOKEN_MINUTES, 10 by default), renewed by the web
#  - brands the login and console as LibreQuenda (logo, colours, no Zitadel links)
#  - creates the "librequenda-api" service account (user and settings manager of
#    the residents' organisation) and writes its token to /api-secrets, for /perfil
#    (name, email, password) and to copy the installation's name and logo there
#  - with ZITADEL_SETUP_REVOKE_TOKEN=true, revokes its own admin token at the end;
#    without a working token, an installation already set up is left as it is
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

CONFIGURED=/runtime/config.json # written by an earlier successful run

for i in $(seq 1 60); do
	[ -s "$PAT_FILE" ] && break
	[ -s "$CONFIGURED" ] && [ "$i" -ge 5 ] && break # set up before, token gone: see below
	log "waiting for $PAT_FILE ($i)"
	sleep 2
done
if [ ! -s "$PAT_FILE" ]; then
	if [ -s "$CONFIGURED" ]; then
		log "no admin token ($PAT_FILE): keeping the existing configuration (see README, 'Zitadel admin token')"
		exit 0
	fi
	log "no admin token at $PAT_FILE: Zitadel writes it on its very first start"
	exit 1
fi
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

# A revoked or expired admin token (ZITADEL_SETUP_REVOKE_TOKEN, or its expiry)
# leaves an installation that is already set up as it is, instead of failing
# the start of everything that waits for this setup.
if ! call GET /auth/v1/users/me >/dev/null 2>&1; then
	if [ -s "$CONFIGURED" ]; then
		log "the admin token no longer works: keeping the existing configuration"
		exit 0
	fi
	log "the admin token does not work, and nothing is set up yet"
	exit 1
fi

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

# --- OIDC app for the Android app (app/) ---
# A native app: PKCE, no secret, back to the app through its own URI scheme
# (the applicationId). Same project, so its tokens carry the audience the API checks.
ANDROID_APP_NAME=android
ANDROID_REDIRECT="gal.tribo.librequenda:/oauth2redirect"
ANDROID_OIDC_CONFIG=$(jq -n \
	--arg redirect "$ANDROID_REDIRECT" \
	--argjson dev "$([ "$PUBLIC_SCHEME" = http ] && echo true || echo false)" \
	'{
		redirectUris: [$redirect],
		postLogoutRedirectUris: [$redirect],
		responseTypes: ["OIDC_RESPONSE_TYPE_CODE"],
		grantTypes: ["OIDC_GRANT_TYPE_AUTHORIZATION_CODE", "OIDC_GRANT_TYPE_REFRESH_TOKEN"],
		appType: "OIDC_APP_TYPE_NATIVE",
		authMethodType: "OIDC_AUTH_METHOD_TYPE_NONE",
		accessTokenType: "OIDC_TOKEN_TYPE_JWT",
		idTokenUserinfoAssertion: true,
		devMode: $dev
	}')

ANDROID_APP=$(call POST "/management/v1/projects/$PROJECT_ID/apps/_search" \
	"{\"queries\":[{\"nameQuery\":{\"name\":\"$ANDROID_APP_NAME\",\"method\":\"TEXT_QUERY_METHOD_EQUALS\"}}]}" \
	| jq -c '.result[0] // empty')

if [ -z "$ANDROID_APP" ]; then
	ANDROID_CLIENT_ID=$(call POST "/management/v1/projects/$PROJECT_ID/apps/oidc" \
		"$(echo "$ANDROID_OIDC_CONFIG" | jq --arg name "$ANDROID_APP_NAME" '. + {name: $name}')" | jq -r '.clientId')
	log "created OIDC app $ANDROID_APP_NAME ($ANDROID_CLIENT_ID)"
else
	ANDROID_CLIENT_ID=$(echo "$ANDROID_APP" | jq -r '.oidcConfig.clientId')
	call PUT "/management/v1/projects/$PROJECT_ID/apps/$(echo "$ANDROID_APP" | jq -r '.id')/oidc_config" "$ANDROID_OIDC_CONFIG" >/dev/null 2>&1 \
		|| true # "no changes" is reported as an error
	log "OIDC app $ANDROID_APP_NAME exists ($ANDROID_CLIENT_ID), config synced"
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

# --- token lifetimes ---
# Short access tokens, so a blocked account or a removed role stops working
# within minutes; the web renews them silently with its refresh token
# (offline_access), so residents stay signed in while they use it.
ACCESS_MINUTES="${ZITADEL_ACCESS_TOKEN_MINUTES:-10}"
REFRESH_IDLE_DAYS="${ZITADEL_REFRESH_IDLE_DAYS:-30}"
REFRESH_DAYS="${ZITADEL_REFRESH_DAYS:-90}"
call PUT /admin/v1/settings/oidc "$(jq -n \
	--arg access "$((ACCESS_MINUTES * 60))s" \
	--arg idle "$((REFRESH_IDLE_DAYS * 86400))s" \
	--arg absolute "$((REFRESH_DAYS * 86400))s" \
	'{accessTokenLifetime: $access, idTokenLifetime: $access, refreshTokenIdleExpiration: $idle, refreshTokenExpiration: $absolute}')" \
	>/dev/null 2>&1 || true # "no changes" is reported as an error
log "tokens: access ${ACCESS_MINUTES} min, refresh ${REFRESH_IDLE_DAYS} days idle, ${REFRESH_DAYS} days at most"

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
# Only ORG_USER_MANAGER and ORG_SETTINGS_MANAGER on the organisation residents
# register in, never IAM owner. The settings role lets the API give that organisation
# the installation's logo and name (its own label policy and email texts,
# api/zitadel/branding.go); the instance's LibreQuenda branding above stays.
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
API_ROLES='["ORG_USER_MANAGER","ORG_SETTINGS_MANAGER"]'
call POST /management/v1/orgs/me/members "{\"userId\":\"$API_USER_ID\",\"roles\":$API_ROLES}" >/dev/null 2>&1 ||
	call PUT "/management/v1/orgs/me/members/$API_USER_ID" "{\"roles\":$API_ROLES}" >/dev/null 2>&1 ||
	true # already a member with these roles ("no changes" is reported as an error)
if [ ! -s "$API_PAT_FILE" ]; then
	call POST "/management/v1/users/$API_USER_ID/pats" '{"expirationDate":"2099-01-01T00:00:00Z"}' | jq -r '.token' > "$API_PAT_FILE.tmp"
	mv "$API_PAT_FILE.tmp" "$API_PAT_FILE"
	log "wrote the token of $API_USER"
fi

# --- token introspection for the API: is an administrator's sign-in two-factor? ---
# Zitadel's JWT access tokens carry no "amr"; introspection does. The API checks
# it on admin routes (REQUIRE_ADMIN_MFA) with this API application's credentials.
INTROSPECTION_APP=api-introspection
INTROSPECTION_FILE=/api-secrets/zitadel-introspection.json
INTROSPECTION=$(call POST "/management/v1/projects/$PROJECT_ID/apps/_search" \
	"{\"queries\":[{\"nameQuery\":{\"name\":\"$INTROSPECTION_APP\",\"method\":\"TEXT_QUERY_METHOD_EQUALS\"}}]}" \
	| jq -c '.result[0] // empty')
if [ -z "$INTROSPECTION" ]; then
	call POST "/management/v1/projects/$PROJECT_ID/apps/api" \
		"{\"name\":\"$INTROSPECTION_APP\",\"authMethodType\":\"API_AUTH_METHOD_TYPE_BASIC\"}" \
		| jq '{clientId, clientSecret}' > "$INTROSPECTION_FILE.tmp"
	mv "$INTROSPECTION_FILE.tmp" "$INTROSPECTION_FILE"
	log "created API app $INTROSPECTION_APP for token introspection"
elif [ ! -s "$INTROSPECTION_FILE" ]; then
	# its secret is only shown once: a new one when the file is gone
	INTROSPECTION_ID=$(echo "$INTROSPECTION" | jq -r '.id')
	call POST "/management/v1/projects/$PROJECT_ID/apps/$INTROSPECTION_ID/api_config/_generate_client_secret" '{}' \
		| jq --arg id "$(echo "$INTROSPECTION" | jq -r '.apiConfig.clientId')" '{clientId: $id, clientSecret}' > "$INTROSPECTION_FILE.tmp"
	mv "$INTROSPECTION_FILE.tmp" "$INTROSPECTION_FILE"
	log "new secret for API app $INTROSPECTION_APP"
fi

# --- runtime config for the SPA ---
jq -n \
	--arg authority "$ZITADEL_URL" \
	--arg clientId "$CLIENT_ID" \
	--arg projectId "$PROJECT_ID" \
	--arg androidClientId "$ANDROID_CLIENT_ID" \
	'{
		authority: $authority,
		clientId: $clientId,
		androidClientId: $androidClientId,
		projectId: $projectId,
		scope: ("openid profile email offline_access urn:zitadel:iam:org:project:id:" + $projectId + ":aud urn:zitadel:iam:org:project:id:zitadel:aud")
	}' > /runtime/config.json.tmp
mv /runtime/config.json.tmp /runtime/config.json
log "wrote /runtime/config.json"

# --- optionally, no admin token left behind ---
# Everything above is in place, so the instance-owner token can go: later
# starts keep this configuration (a new token is needed to change it again).
if [ "${ZITADEL_SETUP_REVOKE_TOKEN:-false}" = "true" ]; then
	SETUP_USER_ID=$(call GET /auth/v1/users/me | jq -r '.user.id')
	for token in $(call POST "/management/v1/users/$SETUP_USER_ID/pats/_search" '{}' | jq -r '.result[]?.id'); do
		call DELETE "/management/v1/users/$SETUP_USER_ID/pats/$token" >/dev/null
	done
	log "revoked the admin token (ZITADEL_SETUP_REVOKE_TOKEN)"
fi
