#!/usr/bin/env bash
# ============================================================
#  LibreQuenda installer
#
#    install.sh         sets up (or reconfigures) a server: asks what the .env needs,
#                       generates the secrets, starts the stack, and offers your own
#                       Android app for it
#    install.sh app     only builds your own Android app (again, e.g. for an update)
#
#  Needs Docker with Compose v2 and curl. Run it in the directory that holds (or
#  will hold) the installation, or set LIBREQUENDA_DIR.
#  Without a checkout: curl -fsSL https://raw.githubusercontent.com/TriboSolutions/LibreQuenda/main/install.sh | bash
#
#  LIBREQUENDA_REF     the branch or tag the files and the app come from (main)
#  LIBREQUENDA_SOURCE  a local checkout to use instead of downloading it
# ============================================================
set -euo pipefail
trap 'printf "\e[31mx stopped at line %s: %s\e[0m\n" "$LINENO" "$BASH_COMMAND" >&2' ERR

REPO=TriboSolutions/LibreQuenda
REF=${LIBREQUENDA_REF:-main}
SOURCE=${LIBREQUENDA_SOURCE:-}

bold=$'\e[1m' dim=$'\e[2m' red=$'\e[31m' green=$'\e[32m' yellow=$'\e[33m' off=$'\e[0m'
[ -t 1 ] || { bold='' dim='' red='' green='' yellow='' off=''; }

say() { printf '%s\n' "$*"; }
title() { printf '\n%s== %s ==%s\n' "$bold" "$*" "$off"; }
note() { printf '%s%s%s\n' "$dim" "$*" "$off"; }
warn() { printf '%s! %s%s\n' "$yellow" "$*" "$off"; }
die() { printf '%sx %s%s\n' "$red" "$*" "$off" >&2; exit 1; }
ok() { printf '%s✓ %s%s\n' "$green" "$*" "$off"; }

# questions come from the terminal, also when the script itself is piped into bash
exec 3</dev/tty || die "run it in a terminal: it asks questions"

# ---------- the .env, as a map ----------
declare -A ENV=()

env_load() { # KEY=VALUE lines; one layer of quotes removed
	local line key value
	while IFS= read -r line || [ -n "$line" ]; do
		[[ $line =~ ^([A-Z0-9_]+)=(.*)$ ]] || continue
		key=${BASH_REMATCH[1]} value=${BASH_REMATCH[2]}
		if [[ $value =~ ^\'(.*)\'$ || $value =~ ^\"(.*)\"$ ]]; then value=${BASH_REMATCH[1]}; fi
		ENV[$key]=$value
	done <"$1"
}

env_quote() { # unquoted when plain; single quotes (no interpolation in Compose) otherwise
	if [[ $1 =~ ^[A-Za-z0-9._@:/+=,-]*$ ]]; then printf '%s' "$1"; else printf "'%s'" "$1"; fi
}

env_write() { # .env.example's layout and comments, with our values; keys it lacks go last
	local example=$1 out=$2 line key
	declare -A seen=()
	{
		while IFS= read -r line || [ -n "$line" ]; do
			if [[ $line =~ ^([A-Z0-9_]+)= ]] && [ -n "${ENV[${BASH_REMATCH[1]}]+set}" ]; then
				key=${BASH_REMATCH[1]}; seen[$key]=1
				printf '%s=%s\n' "$key" "$(env_quote "${ENV[$key]}")"
			else
				printf '%s\n' "$line"
			fi
		done <"$example"
		for key in "${!ENV[@]}"; do
			[ -n "${seen[$key]+set}" ] || printf '%s=%s\n' "$key" "$(env_quote "${ENV[$key]}")"
		done
	} >"$out.tmp"
	chmod 600 "$out.tmp"
	mv "$out.tmp" "$out"
}

# ---------- questions ----------
# ask KEY "question" [default] [validator]: the current value (or the default) on Enter
ask() {
	local key=$1 q=$2 def=${3:-} check=${4:-} cur answer
	cur=${ENV[$key]:-$def}
	while true; do
		printf '%s%s%s' "$bold" "$q" "$off"
		[ -n "$cur" ] && printf ' %s[%s]%s' "$dim" "$cur" "$off"
		printf ': '
		IFS= read -r answer <&3 || die "no answer"
		answer=${answer:-$cur}
		if [[ $answer == *"'"* ]]; then warn "no single quotes ('), please"; continue; fi
		if [ -n "$check" ] && ! "$check" "$answer"; then continue; fi
		ENV[$key]=$answer
		return
	done
}

# ask_secret KEY "question" generator: hidden; Enter keeps the current one or generates one
ask_secret() {
	local key=$1 q=$2 gen=$3 check=${4:-} answer hint
	if [ -n "${ENV[$key]:-}" ]; then hint="Enter keeps the current one"; else hint="Enter generates one"; fi
	while true; do
		printf '%s%s%s %s(%s)%s: ' "$bold" "$q" "$off" "$dim" "$hint" "$off"
		IFS= read -rs answer <&3 || die "no answer"; printf '\n'
		if [ -z "$answer" ]; then
			[ -n "${ENV[$key]:-}" ] && return
			ENV[$key]=$($gen); GENERATED+=("$key"); return
		fi
		if [[ $answer == *"'"* ]]; then warn "no single quotes ('), please"; continue; fi
		if [ -n "$check" ] && ! "$check" "$answer"; then continue; fi
		ENV[$key]=$answer; return
	done
}

yesno() { # yesno "question" y|n
	local answer def=$2
	while true; do
		printf '%s%s%s %s[%s]%s: ' "$bold" "$1" "$off" "$dim" "$([ "$def" = y ] && echo Y/n || echo y/N)" "$off"
		IFS= read -r answer <&3 || die "no answer"
		case ${answer:-$def} in [yY]*|[sS]*) return 0 ;; [nN]*) return 1 ;; esac
	done
}

# ---------- validators ----------
v_domain() { [[ $1 =~ ^([a-z0-9]([a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,}$ ]] || { warn "a domain like reservas.concello.gal"; return 1; }; }
v_email() { [[ $1 =~ ^[^@[:space:]]+@[^@[:space:]]+\.[^@[:space:]]+$ ]] || { warn "an email address"; return 1; }; }
v_email_opt() { [ -z "$1" ] || v_email "$1"; }
v_nonempty() { [ -n "$1" ] || { warn "required"; return 1; }; }
v_port() { [[ $1 =~ ^[0-9]+$ ]] && [ "$1" -ge 1 ] && [ "$1" -le 65535 ] || { warn "a port number"; return 1; }; }
v_password() { # Zitadel's default policy: 8+, upper, lower, number, symbol
	[ ${#1} -ge 8 ] && [[ $1 =~ [A-Z] && $1 =~ [a-z] && $1 =~ [0-9] && $1 =~ [^A-Za-z0-9] ]] \
		|| { warn "at least 8 characters, with upper and lower case, a number and a symbol"; return 1; }
}
v_app_id() {
	[[ $1 =~ ^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$ ]] || { warn "lowercase words separated by dots, e.g. gal.lalin.reservas"; return 1; }
	[ "$1" != gal.tribo.librequenda ] || { warn "that one is LibreQuenda's own app: choose yours"; return 1; }
}
v_colour() { [[ $1 =~ ^#[0-9A-Fa-f]{6}$ ]] || { warn "a colour like #FFFFFF"; return 1; }; }
abs_path() { # a path typed by the person: ~ and relative to where the script was started
	local p=$1
	[[ $p == "~/"* ]] && p="$HOME/${p#\~/}"
	[[ $p == /* ]] || p="$START_PWD/$p"
	printf '%s' "$p"
}
v_logo() {
	[ -z "$1" ] && return 0
	[ -f "$(abs_path "$1")" ] || { warn "no such file"; return 1; }
	[[ ${1,,} =~ \.(png|jpe?g|webp)$ ]] || { warn "a PNG, JPG or WebP image"; return 1; }
}

# ---------- secrets ----------
rand_alnum() { # no "tr </dev/urandom | head": tr dies of SIGPIPE, and pipefail stops everything
	local n=${1:-32} out=""
	while [ ${#out} -lt "$n" ]; do out+=$(head -c 64 /dev/urandom | LC_ALL=C tr -dc A-Za-z0-9); done
	printf '%s' "${out:0:n}"
}
rand_hex() { od -An -tx1 -N"$1" /dev/urandom | tr -d ' \n'; }
rand_b64() { head -c 32 /dev/urandom | base64 | tr -d '\n'; }
gen_alnum32() { rand_alnum 32; }
gen_hex32() { rand_hex 32; }
gen_storage_key() { printf 'GK%s' "$(rand_hex 12)"; }
gen_password() { printf '%s-Aa1' "$(rand_alnum 20)"; }
GENERATED=()

secret() { # secret KEY generator: generated once, then kept for good
	[ -n "${ENV[$1]:-}" ] || ENV[$1]=$($2)
}

# ---------- files from the repository ----------
fetch() { # fetch <path in the repo> <destination>
	if [ -n "$SOURCE" ]; then cp "$SOURCE/$1" "$2"
	else curl -fsSL "https://raw.githubusercontent.com/$REPO/$REF/$1" -o "$2" || die "could not download $1 ($REF)"; fi
}

source_tree() { # the whole repository (for the app), in $DIR/.source
	if [ -n "$SOURCE" ]; then SRC=$(cd "$SOURCE" && pwd); return; fi
	SRC="$DIR/.source"
	say "Downloading LibreQuenda's source ($REF)…"
	rm -rf "$SRC"; mkdir -p "$SRC"
	curl -fsSL "https://codeload.github.com/$REPO/tar.gz/$REF" | tar -xz -C "$SRC" --strip-components=1 \
		|| die "could not download the source ($REF)"
}

need() { command -v "$1" >/dev/null 2>&1 || die "$1 is needed: $2"; }

check_docker() {
	need docker "https://docs.docker.com/engine/install/"
	docker compose version >/dev/null 2>&1 || die "Docker Compose v2 is needed (docker compose …)"
	docker info >/dev/null 2>&1 || die "Docker is not running, or this user cannot use it (sudo, or the docker group)"
}

# ============================================================
#  Server
# ============================================================
server_questions() {
	title "Addresses"
	note "Two names pointing (DNS A/AAAA) to this server: the web app, and the sign-in."
	ask WEB_DOMAIN "Web app domain (e.g. reservas.concello.gal)" "" v_domain
	local auth_default="auth.${ENV[WEB_DOMAIN]}"
	ask ZITADEL_DOMAIN "Sign-in domain" "$auth_default" v_domain
	for d in "${ENV[WEB_DOMAIN]}" "${ENV[ZITADEL_DOMAIN]}"; do
		getent hosts "$d" >/dev/null 2>&1 || warn "$d does not resolve yet: point it to this server before people use it"
	done
	if yesno "Is there another proxy or load balancer in front that already handles HTTPS?" \
		"$([ "${ENV[CADDY_AUTO_HTTPS]:-true}" = false ] && echo y || echo n)"; then
		ENV[CADDY_AUTO_HTTPS]=false
		note "Caddy serves plain HTTP to it; the public addresses stay https://."
	else
		ENV[CADDY_AUTO_HTTPS]=true
		note "Caddy gets the certificates from Let's Encrypt: ports 80 and 443 must be open."
	fi

	title "First administrator"
	note "Signs in to create the councils (at /admin) and manages the sign-in service."
	ask ZITADEL_ADMIN_EMAIL "Email" "" v_email
	ask ZITADEL_ADMIN_FIRSTNAME "First name" "" v_nonempty
	ask ZITADEL_ADMIN_LASTNAME "Last name" "" v_nonempty
	ask_secret ZITADEL_ADMIN_PASSWORD "Password" gen_password v_password

	title "Email (SMTP)"
	note "Booking confirmations, reminders, sign-in codes and password resets go out through it."
	ask SMTP_HOST "SMTP server (e.g. smtp.concello.gal)" "" v_nonempty
	ask SMTP_PORT "Port (587 STARTTLS, 465 TLS)" "587" v_port
	ask SMTP_USERNAME "Username"
	ask_secret SMTP_PASSWORD "Password" true
	ask SMTP_FROM_ADDRESS "Sender address" "noreply@${ENV[WEB_DOMAIN]}" v_email
	local sender_name=${ENV[SMTP_FROM]:-}
	[[ $sender_name =~ ^(.*[^ ])\ *\<.*\>$ ]] && sender_name=${BASH_REMATCH[1]} || sender_name=LibreQuenda
	ENV[_SENDER]=$sender_name
	ask _SENDER "Sender name" "LibreQuenda" v_nonempty
	ENV[SMTP_FROM]="${ENV[_SENDER]} <${ENV[SMTP_FROM_ADDRESS]}>"
	unset 'ENV[_SENDER]'
	if [ "${ENV[SMTP_PORT]}" = 465 ]; then ENV[ZITADEL_SMTP_TLS]=true; else ENV[ZITADEL_SMTP_TLS]=false; fi

	title "Support (optional)"
	note "Shown in the app and the web for people who need help. Enter leaves it out."
	ask SUPPORT_EMAIL "Support email" "" v_email_opt
	ask SUPPORT_PHONE "Support phone"

	# everything else: generated once and never changed (the masterkey cannot change)
	secret POSTGRES_PASSWORD gen_alnum32
	secret APP_DB_PASSWORD gen_alnum32
	secret ANALYTICS_DB_PASSWORD gen_alnum32
	secret ZITADEL_MASTERKEY gen_alnum32
	secret LOGIN_SESSION_COOKIE_SECRET rand_b64
	secret ZITADEL_DB_PASSWORD gen_alnum32
	secret STORAGE_ACCESS_KEY gen_storage_key
	secret STORAGE_SECRET_KEY gen_hex32
	secret GARAGE_RPC_SECRET gen_hex32
	secret GARAGE_ADMIN_TOKEN rand_b64
	secret SECRETS_KEY gen_hex32
}

add_app_id() { # adds $1 to ANDROID_APP_IDS
	local ids=${ENV[ANDROID_APP_IDS]:-}
	[[ ",$ids," == *",$1,"* ]] && return
	ENV[ANDROID_APP_IDS]=${ids:+$ids,}$1
}

start_stack() {
	title "Starting LibreQuenda"
	(cd "$DIR" && docker compose pull --quiet && docker compose up -d --wait) || {
		warn "something did not start: docker compose ps, and docker compose logs <service>, in $DIR"
		exit 1
	}
	ok "running at https://${ENV[WEB_DOMAIN]}"
}

install_server() {
	check_docker
	need curl "to download LibreQuenda's files"
	mkdir -p "$DIR"

	if [ -f "$DIR/.env" ]; then
		env_load "$DIR/.env"
		say "Found an installation in $DIR: Enter keeps each current value."
	else
		say "Installing LibreQuenda in $DIR"
	fi
	fetch docker-compose.yaml "$DIR/docker-compose.yaml"
	fetch .env.example "$DIR/.env.example"
	fetch install.sh "$DIR/install.sh.new" && chmod +x "$DIR/install.sh.new" && mv "$DIR/install.sh.new" "$DIR/install.sh"
	local example="$DIR/.env.example"
	# the example's defaults, under what is already set
	declare -A current=(); local k
	for k in "${!ENV[@]}"; do current[$k]=${ENV[$k]}; done
	env_load "$example"
	for k in "${!current[@]}"; do ENV[$k]=${current[$k]}; done
	# the example's placeholder addresses are not answers
	for k in WEB_DOMAIN ZITADEL_DOMAIN SMTP_FROM SMTP_FROM_ADDRESS; do
		[[ ${ENV[$k]:-} == *example.gal* ]] && ENV[$k]=""
	done

	server_questions

	local want_app=n
	title "Your own Android app"
	note "LibreQuenda's app can be built with your name, logo and package, already pointing"
	note "to this server, as an APK to install directly and an AAB for Google Play."
	if yesno "Build your own app now?" n; then
		want_app=y
		app_questions
		add_app_id "${APP[APP_ID]}"
	fi

	[ -f "$DIR/.env" ] && cp "$DIR/.env" "$DIR/.env.backup-$(date +%Y%m%d-%H%M%S)"
	env_write "$example" "$DIR/.env"
	ok "wrote $DIR/.env"

	start_stack
	if [ "$want_app" = y ]; then build_app; check_registered; fi

	title "Done"
	say "Web app:        https://${ENV[WEB_DOMAIN]}"
	say "Administration: https://${ENV[WEB_DOMAIN]}/admin  (sign in as ${ENV[ZITADEL_ADMIN_EMAIL]})"
	say "Sign-in admin:  https://${ENV[ZITADEL_DOMAIN]}/ui/console"
	if [[ " ${GENERATED[*]} " == *" ZITADEL_ADMIN_PASSWORD "* ]]; then
		say ""; say "${bold}Administrator password: ${ENV[ZITADEL_ADMIN_PASSWORD]}${off}"
	fi
	say ""
	warn "Keep a copy of $DIR/.env somewhere safe: ZITADEL_MASTERKEY cannot be recovered,"
	warn "and the backups cannot be restored without it."
	note "Update later: cd $DIR && docker compose pull && docker compose up -d --wait"
}

# ============================================================
#  Android app
# ============================================================
declare -A APP=()

app_questions() {
	APPDIR="$DIR/android"
	[ -f "$APPDIR/app.conf" ] && while IFS='=' read -r k v; do [ -n "$k" ] && APP[$k]=$v; done <"$APPDIR/app.conf"

	# the server it talks to: this installation's, or asked
	if [ -n "${ENV[WEB_DOMAIN]:-}" ]; then
		APP[WEB_DOMAIN]=${ENV[WEB_DOMAIN]}
	else
		local saved=${APP[WEB_DOMAIN]:-}
		ENV[_WEB]=$saved
		ask _WEB "Your LibreQuenda's web domain (e.g. reservas.concello.gal)" "" v_domain
		APP[WEB_DOMAIN]=${ENV[_WEB]}; unset 'ENV[_WEB]'
	fi

	local reversed; reversed=$(awk -F. '{for (i = NF; i > 0; i--) printf "%s%s", $i, (i > 1 ? "." : "")}' <<<"${APP[WEB_DOMAIN]}" \
		| tr '-' '_' | sed -E 's/(^|\.)([0-9])/\1n\2/g')
	ENV[_NAME]=${APP[APP_NAME]:-}; ask _NAME "App name (under the icon)" "Reservas" v_nonempty
	ENV[_ID]=${APP[APP_ID]:-}
	note "The package identifies the app on phones and on Google Play: it can never change once"
	note "published. Sign-in returns to the app through it, so the server lists it (ANDROID_APP_IDS)."
	ask _ID "Package (applicationId)" "$reversed" v_app_id
	ENV[_LOGO]=${APP[LOGO]:-}
	note "A square logo, at least 512×512 (PNG with transparency is best). Enter: LibreQuenda's icon."
	ask _LOGO "Logo file" "" v_logo
	ENV[_BG]=${APP[ICON_BACKGROUND]:-}; ask _BG "Icon background colour" "#FFFFFF" v_colour
	APP[APP_NAME]=${ENV[_NAME]} APP[APP_ID]=${ENV[_ID]} APP[ICON_BACKGROUND]=${ENV[_BG]}
	APP[LOGO]=${ENV[_LOGO]:+$(abs_path "${ENV[_LOGO]}")}
	unset 'ENV[_NAME]' 'ENV[_ID]' 'ENV[_LOGO]' 'ENV[_BG]'
}

app_save() {
	mkdir -p "$APPDIR"
	if [ -n "${APP[LOGO]}" ]; then # a copy next to the key, for the next build
		local ext=${APP[LOGO]##*.} copy
		copy="$APPDIR/logo.${ext,,}"
		[ "$(realpath "${APP[LOGO]}")" = "$(realpath -m "$copy")" ] || cp "${APP[LOGO]}" "$copy"
		APP[LOGO]=$copy
	fi
	{
		for k in WEB_DOMAIN APP_NAME APP_ID LOGO ICON_BACKGROUND; do printf '%s=%s\n' "$k" "${APP[$k]}"; done
	} >"$APPDIR/app.conf"
}

keystore() { # the signing key: made once, needed for every update of the app
	local ks="$APPDIR/release.jks" pw="$APPDIR/keystore.env"
	if [ -f "$ks" ] && [ -f "$pw" ]; then return; fi
	[ -f "$ks" ] && die "$ks has no $pw next to it (ANDROID_KEYSTORE_PASSWORD, ANDROID_KEY_ALIAS, ANDROID_KEY_PASSWORD)"
	local pass; pass=$(rand_alnum 32)
	say "Creating the app's signing key…"
	docker run --rm -v "$APPDIR:/keys" --entrypoint keytool "$BUILDER" -genkeypair -v \
		-keystore /keys/release.jks -storetype PKCS12 -storepass "$pass" -keypass "$pass" \
		-alias release -keyalg RSA -keysize 4096 -validity 10000 \
		-dname "CN=${APP[APP_NAME]//,/ }, O=${APP[WEB_DOMAIN]}" >/dev/null 2>&1 || die "could not create the signing key"
	docker run --rm -v "$APPDIR:/keys" --entrypoint chown "$BUILDER" "$(id -u):$(id -g)" /keys/release.jks
	printf 'ANDROID_KEYSTORE_PASSWORD=%s\nANDROID_KEY_ALIAS=release\nANDROID_KEY_PASSWORD=%s\n' "$pass" "$pass" >"$pw"
	chmod 600 "$ks" "$pw"
	NEW_KEY=y
}

build_app() {
	title "Building ${APP[APP_NAME]} (${APP[APP_ID]})"
	app_save
	source_tree
	BUILDER="librequenda-app-builder:${REF//\//-}"
	say "Preparing the build environment (the first time takes a few minutes)…"
	docker build -q -t "$BUILDER" "$SRC/app/builder" >/dev/null || die "could not prepare the build environment"
	NEW_KEY=n
	keystore

	local logo_mount=()
	[ -n "${APP[LOGO]}" ] && logo_mount=(-v "${APP[LOGO]}:/in/logo:ro")
	local version=${ENV[LIBREQUENDA_VERSION]:-latest}
	[ "$version" = latest ] && version="$REF-$(date +%Y%m%d)"
	say "Building (5 to 10 minutes the first time)…"
	docker run --rm \
		-v "$SRC:/src:ro" -v "$APPDIR/release.jks:/in/release.jks:ro" "${logo_mount[@]}" -v "$APPDIR:/out" \
		-v librequenda-gradle:/root/.gradle \
		--env-file "$APPDIR/keystore.env" \
		-e APP_ID="${APP[APP_ID]}" -e APP_NAME="${APP[APP_NAME]}" -e ICON_BACKGROUND="${APP[ICON_BACKGROUND]}" \
		-e API_URL="https://${APP[WEB_DOMAIN]}/api" \
		-e VERSION_NAME="$version" -e VERSION_CODE="$(($(date +%s) / 60))" \
		-e OUT_OWNER="$(id -u):$(id -g)" \
		"$BUILDER" || die "the build failed (the output above says why)"
	ok "built $APPDIR/${APP[APP_ID]}.apk and $APPDIR/${APP[APP_ID]}.aab"

	say ""
	say "  ${bold}${APP[APP_ID]}.apk${off}  install it directly: send it to phones, or publish it on your website"
	say "  ${bold}${APP[APP_ID]}.aab${off}  upload it to Google Play (Play Console → your app → a new release)"
	if [ "$NEW_KEY" = y ]; then
		say ""
		warn "Back up $APPDIR/release.jks and $APPDIR/keystore.env now, somewhere else."
		warn "Every update of the app must be signed with this key: without it, phones and"
		warn "Google Play refuse updates and the app has to be published again under a new package."
	fi
	note "Update the app later (after updating LibreQuenda): cd $DIR && ./install.sh app"
}

# the server must let sign-in return to the app: ANDROID_APP_IDS, then the setup runs again
register_app() {
	local id=${APP[APP_ID]}
	if [ ! -f "$DIR/.env" ] || [ ! -f "$DIR/docker-compose.yaml" ]; then
		say ""
		warn "The LibreQuenda server is not in $DIR. For sign-in to work in the app, add its package"
		warn "to the server's .env and restart it there:"
		say "    ANDROID_APP_IDS=$id        ${dim}(comma-separated if there are several)${off}"
		say "    docker compose up -d --wait"
		return
	fi
	[[ ",${ENV[ANDROID_APP_IDS]:-}," == *",$id,"* ]] && return
	add_app_id "$id"
	cp "$DIR/.env" "$DIR/.env.backup-$(date +%Y%m%d-%H%M%S)"
	local example="$DIR/.env.example"; [ -f "$example" ] || fetch .env.example "$example"
	env_write "$example" "$DIR/.env"
	ok "added $id to ANDROID_APP_IDS in $DIR/.env"
	start_stack
}

check_registered() { # the server says which apps may sign in (config.json)
	local id=${APP[APP_ID]} json
	json=$(curl -fsS --max-time 10 "https://${APP[WEB_DOMAIN]}/config.json" 2>/dev/null) || {
		warn "could not reach https://${APP[WEB_DOMAIN]}/config.json to check that sign-in accepts the app"; return; }
	if grep -q "\"$id\"" <<<"$json"; then
		ok "the server accepts sign-in from $id"
	else
		warn "the server does not list $id yet, so signing in to the app will fail."
		if [ "${ENV[ZITADEL_SETUP_REVOKE_TOKEN]:-false}" = true ]; then
			warn "ZITADEL_SETUP_REVOKE_TOKEN=true: the setup can no longer change Zitadel. Create a new token"
			warn "for librequenda-setup (docs/self-hosting.md, 'Zitadel admin token'), then: docker compose up -d --wait"
		else
			warn "Check: docker compose logs zitadel-setup"
		fi
	fi
}

install_app() {
	check_docker
	need curl "to download LibreQuenda's source"
	mkdir -p "$DIR"
	[ -f "$DIR/.env" ] && env_load "$DIR/.env"
	title "Your own Android app"
	app_questions
	register_app
	build_app
	check_registered
}

# ============================================================
START_PWD=$PWD
DIR=$(realpath -m "${LIBREQUENDA_DIR:-$PWD}")
case "${1:-}" in
	"") install_server ;;
	app) install_app ;;
	-h|--help) sed -n '2,17p' "$0" | sed 's/^#  \{0,1\}//'; exit 0 ;;
	*) die "unknown command: $1 (install.sh, or install.sh app)" ;;
esac
