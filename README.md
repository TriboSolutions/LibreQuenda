# LibreQuenda

Free-software platform for managing and booking **municipal sports facilities**, built for Galician councils (*concellos*).

Residents look up facilities, book, change or cancel their bookings, see their history and get notifications. Councils manage their facilities, opening hours and usage rules, and follow basic indicators. One deployment can serve one council or many.

LibreQuenda is Tribo Solutions' proposal for the [I Certame GOBTEC de Software Libre](https://www.agasol.gal/i-certame-gobtec/) (AGASOL, 2026).

> **Status:** in development. Residents browse councils and facilities, book, cancel, invite others to fill a booking, and get notifications and emails. Council admins manage facilities, opening hours, closures, usage rules, images and entrance QR readers, and follow live indicators they can export as CSV. Superadministrators manage councils and their admins. Still to come: changing bookings, data export. See the [architecture plan](docs/specs/architecture.md) and [todo.md](todo.md).

## Stack

| Part | Technology |
|---|---|
| Core API | Go, [Penguin](https://github.com/angelbarreiros/Penguin) (HTTP router, scheduler, logger), [sqlc](https://sqlc.dev) |
| Database | PostgreSQL |
| Image storage | [Garage](https://garagehq.deuxfleurs.fr) (self-hosted, S3-compatible) |
| Maps | [Leaflet](https://leafletjs.com) + [OpenStreetMap](https://www.openstreetmap.org) |
| Identity | [Zitadel](https://zitadel.com) (OIDC) |
| Web | React, Vite, pnpm |
| Proxy | [Caddy](https://caddyserver.com) (single entry point, automatic HTTPS) |
| Deployment | Docker Compose |

## Repository layout

```
.agents/   AI agent instructions for development
.github/   CI workflows
app/       Mobile app (optional, later)
deploy/    Self-hosting kit (production)
proxy/     Caddy: serves the web app, proxies the API and Zitadel
storage/   Garage: S3-compatible storage for uploaded images
zitadel/   One-off setup: creates the LibreQuenda project + OIDC app, writes config.json
docs/      Architecture specs, brand assets, generated OpenAPI spec
api/       Go core API
db/        PostgreSQL image: init schema, seed, sqlc queries
web/       React web app
```

## Installation

Requirements: a server with Docker (Compose v2), two DNS names pointing to it (the web app and the sign-in, e.g. `reservas.concello.gal` and `auth.reservas.concello.gal`), ports 80 and 443 open, and an SMTP relay for email.

```sh
cp .env.example .env
# fill in every value: domains, SMTP, the first admin, and the secrets (each line says how to generate it)
docker compose up -d --build --wait
```

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

**Council administration:** a council's administrators (and superadministrators) open `/<council>/admin` to create facilities with photo, description, address, map pin, booking rules, weekly opening hours and closures. Each facility has its own page at `/<council>/<facility>`.

**Residents** never see Zitadel's own screens beyond signing in: name, email and password are changed from `/perfil`. Zitadel is branded as LibreQuenda by `zitadel/setup.sh`.

**Email:** LibreQuenda sends booking confirmations, a reminder 15 minutes before with the entry QR, cancellations (with the reason when the council closes a facility) and administrator invitations. Zitadel sends verification codes and password resets through the same relay.

**Images** are stored in **Garage** (self-hosted, S3-compatible; service `storage`). The database only keeps references such as `facilities/<id>.webp`; the API turns them into `/api/media/…` URLs when it answers. **Maps** use OpenStreetMap (Leaflet); "Como chegar" is a plain link to Google Maps directions. Address search goes through the API to OpenStreetMap Nominatim (or `GEOCODER_URL`).

**Sign in with Google (optional):** create an OAuth client ID (type *Web application*) in Google Cloud with the authorized redirect URI `https://<ZITADEL_DOMAIN>/idps/callback`, then set `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` in `.env` and run `docker compose up -d`. Leave them empty to remove the Google button again.

**TLS behind another proxy:** if a load balancer in front already terminates TLS, set `CADDY_AUTO_HTTPS=false` and keep `PUBLIC_SCHEME=https`; Caddy then serves plain HTTP to it.

**Images from CI:** every push to `main` and every version tag builds the images and publishes them to `ghcr.io/<owner>/librequenda-<image>` (`.github/workflows/ci.yml`).

## Documentation

- [Architecture plan](docs/specs/architecture.md)
- [Contest bases (v3, Galician)](https://www.agasol.gal/wp-content/uploads/2026/10/Bases_I_Certame_GOBTEC_2026_GL_v3.pdf), published by AGASOL
- API spec: generated into [`docs/openapi/`](docs/openapi/)

## License

Copyright (C) 2026 Tribo Solutions

This program is free software: you can redistribute it and/or modify it under the terms of the GNU Affero General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version. See [LICENSE](LICENSE).
