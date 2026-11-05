# Running a LibreQuenda server

How an installation works and how to operate it. To install, see the [README](../README.md) (`install.sh` asks for everything below that is required).

## Without the installer

Only two files are needed on the server: `docker-compose.yaml` and `.env`. The images are prebuilt by CI and pulled from `ghcr.io/tribosolutions/librequenda-*` (`api`, `db`, `proxy` with the web app inside, `storage`, `zitadel-setup`, `backup`); nothing is built there.

```sh
mkdir librequenda && cd librequenda
curl -fLO https://raw.githubusercontent.com/TriboSolutions/LibreQuenda/main/docker-compose.yaml
curl -fL -o .env https://raw.githubusercontent.com/TriboSolutions/LibreQuenda/main/.env.example
# fill in every value: domains, SMTP, the first admin, and the secrets (each line says how to generate it)
docker compose up -d --wait
```

`LIBREQUENDA_VERSION` in `.env` picks the images' tag: `latest` follows `main`; a release tag (e.g. `1.2.3`) stays on that release. To update: `docker compose pull && docker compose up -d --wait`. To run your own changes instead, build the images from this repository with the same names (e.g. `docker build -t ghcr.io/tribosolutions/librequenda-api:latest api`).

## How it runs

The stack refuses to start while a required value is missing. Everything goes through one Caddy proxy (`proxy/`), which obtains and renews the TLS certificates:

| What | URL |
|---|---|
| Web app | `https://<WEB_DOMAIN>` |
| Core API | `https://<WEB_DOMAIN>/api/…` (e.g. `/api/health/db`) |
| Sign-in (Zitadel) | `https://<ZITADEL_DOMAIN>` |
| OIDC discovery | `https://<ZITADEL_DOMAIN>/.well-known/openid-configuration` |

PostgreSQL is not published on the host: use `docker compose exec db psql -U librequenda`.

The first Zitadel admin (`ZITADEL_ADMIN_*`), Zitadel's email settings and the `ZITADEL_MASTERKEY` are only applied on the very first start; the masterkey can never be changed afterwards, so back it up. Data lives in Docker volumes (`postgres_data`, `garage_data`, …).

**LibreQuenda starts empty.** The superadministrator (`SUPERADMIN_EMAIL`, by default the Zitadel admin) signs in and opens `/admin` to create councils, invite each council's administrator by email address and manage other superadministrators. The invited person gets an email at once; the role is granted when they sign in with that email verified (sign-in asks for the emailed code), with a notice in the bell.

**Your own name and logo:** in `/admin`, the "Personalizar a aplicación" tab sets the app's name and logo for the whole installation: the top bar, tab titles and browser icon, the QR reader screens, the calendar feed and the emails use them. Leave them empty to go back to LibreQuenda's. The footer keeps naming LibreQuenda as the free software it runs on (AGPL-3.0). The sign-in screens are Zitadel's and keep the branding from `zitadel/setup.sh`.

**Council administration:** a council's administrators (and superadministrators) open `/<council>/admin` to create facilities with photo, description, address, map pin, booking rules, weekly opening hours and closures. Each facility has its own page at `/<council>/<facility>`.

**Residents** never see Zitadel's own screens beyond signing in: name, email and password are changed from `/perfil`. Zitadel is branded as LibreQuenda by `zitadel/setup.sh`.

**Email:** LibreQuenda sends booking confirmations, a reminder 15 minutes before with the entry QR, cancellations (with the reason when the council closes a facility) and administrator invitations. Zitadel sends verification codes and password resets through the same relay.

**Images** are stored in **Garage** (self-hosted, S3-compatible; service `storage`). Buckets are private, one per council; the database only keeps references such as `<council id>/facilities/<id>.webp`, and the API turns them into presigned `/storage/…` links (valid a week) when it answers. **Maps** use OpenStreetMap (Leaflet); "Como chegar" is a plain link to Google Maps directions. Address search goes through the API to OpenStreetMap Nominatim (or `GEOCODER_URL`).

**Sign in with Google (optional):** create an OAuth client ID (type *Web application*) in Google Cloud with the authorized redirect URI `https://<ZITADEL_DOMAIN>/idps/callback`, then set `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` in `.env` and run `docker compose up -d`. Leave them empty to remove the Google button again.

**Passwords, not passkeys:** sign-up and sign-in go straight to a password. Set `ZITADEL_PASSKEYS=true` in `.env` and run `docker compose up -d` to also offer passkeys.

**Facility tags:** descriptive metadata for classification and filtering (including finding similar facilities). Tags must never enable features or change booking, access or other operational rules; those belong to explicit facility settings.

**Weather:** each facility has a “Show weather forecast” switch in its editing form (off for new facilities, independent of tags). When enabled, it shows MeteoGalicia's forecast for their council's municipality, set as its INE code in the council's details (e.g. 36024 for Lalín). The API fetches it (keyless, `servizos.meteogalicia.gal/mgrss/predicion/jsonPredConcellos.action`), caches it for an hour and shows nothing if it is unreachable; `WEATHER_URL` points it elsewhere. Data © MeteoGalicia, credited where it is shown.

**Short sessions:** access tokens last 10 minutes (`ZITADEL_ACCESS_TOKEN_MINUTES`), so a blocked account or a removed role stops working that soon. The web renews them silently with its refresh token, so people stay signed in while they use it; after 30 idle days (`ZITADEL_REFRESH_IDLE_DAYS`), or 90 days in all (`ZITADEL_REFRESH_DAYS`), they sign in again. Run `docker compose up -d` after changing them.

**Second factor for administrators:** council administrators and superadministrators only get into the administration after signing in with a second factor (an authenticator app, a security key or a passkey), set up in their sign-in account. Zitadel's access tokens don't say how the person signed in, so the API asks Zitadel (token introspection, with an API application `zitadel-setup` creates) once per token. On by default (`REQUIRE_ADMIN_MFA=true` in `docker-compose.yaml`); the local development stack leaves it off. Residents are not affected. Before turning it on in an existing installation, ask the administrators to set up a second factor first.

**Sign-up and sign-in limits:** Zitadel's login has no captcha, so the proxy limits its form posts per IP address: 5 account registrations per hour (`SIGNUP_LIMIT_PER_HOUR`) and 30 sign-in, password, code or email-verification attempts per 10 minutes (`SIGNIN_LIMIT_PER_10_MIN`). Over a limit the person gets "429 Too Many Requests" and can try again a little later. Behind another proxy, the limits count the visitor's address only if that proxy is on a private network (`trusted_proxies`).

**Zitadel admin token:** on its very first start Zitadel creates the `librequenda-setup` service account and writes its token (`admin.pat`) to the `zitadel-admin` volume, which only `zitadel-setup` mounts (read-only); the login app's own token goes to `zitadel-login-client`, and the API only ever sees `api-secrets`. The account is an instance owner, the least role that can set the instance's login, token and branding policies. To leave no such token behind, set `ZITADEL_SETUP_REVOKE_TOKEN=true`: once the setup succeeds it revokes the token, and later starts keep the configuration as it is (to change it again, create a new token for `librequenda-setup` in Zitadel's console and put it in that volume). `ZITADEL_ADMIN_PAT_EXPIRATION` sets the token's expiry on the very first start.

*Upgrading an installation that still has the single `zitadel-bootstrap` volume:* copy the two tokens to the new volumes once, then start as usual (replace `librequenda` with your compose project name):

```sh
docker run --rm -v librequenda_zitadel-bootstrap:/old:ro -v librequenda_zitadel-admin:/admin -v librequenda_zitadel-login-client:/login \
  alpine sh -c 'cp /old/admin.pat /admin/ && cp /old/login-client.pat /login/'
docker compose up -d
docker volume rm librequenda_zitadel-bootstrap   # once everything works
```

A local `docker-compose.dev.yaml` keeps working as it is; to mirror production, mount `zitadel-admin` at `/zitadel/bootstrap` in `zitadel-setup` and `zitadel-login-client` at `/zitadel/login-client` in `zitadel-login` (with `ZITADEL_SERVICE_USER_TOKEN_FILE` and the `ZITADEL_FIRSTINSTANCE_*PATPATH` paths as in `docker-compose.yaml`).

**TLS behind another proxy:** if a load balancer in front already terminates TLS, set `CADDY_AUTO_HTTPS=false` and keep `PUBLIC_SCHEME=https`; Caddy then serves plain HTTP to it.

**Backups:** the `backup` service dumps PostgreSQL every day at 03:30 (`BACKUP_AT`, Europe/Madrid): LibreQuenda's database and Zitadel's (the accounts the bookings belong to), each with `pg_dump` and read back before it is kept, plus the roles. They go into the `backups` bucket of this server's Garage (`BACKUP_BUCKET`), one folder per backup, and are removed after 14 days (`BACKUP_KEEP_DAYS`). The service turns unhealthy in `docker compose ps` after a backup fails, until the next one works; `docker compose logs backup` says why. A backup on the same server does not survive losing its disk: point `BACKUP_S3_URL`, `BACKUP_S3_REGION`, `BACKUP_S3_ACCESS_KEY` and `BACKUP_S3_SECRET_KEY` to an S3-compatible storage elsewhere (another Garage, a provider) to keep them off it. Uploaded images stay in Garage and are not in these backups. Zitadel's data is encrypted with `ZITADEL_MASTERKEY`: keep a copy of `.env` apart, since a backup cannot be restored without it.

```sh
docker compose exec backup backup                 # one now
docker compose stop api zitadel-api zitadel-login # nothing may use the databases while restoring
docker compose exec backup restore                # lists the backups
docker compose exec backup restore 20261105T023000Z
docker compose start api zitadel-api zitadel-login
```

**Images from CI:** every push to `main` and every version tag builds the images and publishes them to `ghcr.io/<owner>/librequenda-<image>` (`.github/workflows/ci.yml`).

## Your own Android app

`./install.sh app` builds LibreQuenda's Android app with your name, logo and package (applicationId), pointing to your server, in Docker (`app/builder/`): an APK to install directly and an AAB for Google Play. It keeps its answers, the logo and the signing key in `android/` next to `.env`.

- **The signing key** (`android/release.jks` and `android/keystore.env`) signs every update: phones and Google Play refuse an update signed with another key. Back it up with `.env`.
- **Sign-in** returns to the app through its package (`<applicationId>:/oauth2redirect`), so Zitadel must know it: the installer adds it to `ANDROID_APP_IDS` in `.env` and restarts the stack, and `zitadel-setup` registers it. The server lists the accepted ones in `/config.json` (`androidAppIds`). Built on another machine, add it to the server's `.env` by hand and run `docker compose up -d --wait`. With `ZITADEL_SETUP_REVOKE_TOKEN=true` the setup can no longer change Zitadel: give it a new token first (see "Zitadel admin token").
- **Updates:** after updating LibreQuenda, run `./install.sh app` again: same package and key, a higher version code.
