# LibreQuenda

Free-software platform for managing and booking **municipal sports facilities**, built for Galician councils (*concellos*).

Residents look up facilities, book, change or cancel their bookings, see their history and get notifications. Councils manage their facilities, opening hours and usage rules, and follow basic indicators. One deployment can serve one council or many.

LibreQuenda is Tribo Solutions' proposal for the [I Certame GOBTEC de Software Libre](https://www.agasol.gal/i-certame-gobtec/) (AGASOL, 2026).

> **Status:** in development. Residents browse councils and facilities, book, change and cancel, invite others to fill a booking, check in at the door in several ways (QR reader, printed QR, entry code, staff), get notifications and emails, and see recommendations from their own bookings. Council admins manage facilities, opening hours, closures, usage rules, images, ways to check in and booking windows, and follow live indicators they can export as CSV, along with the council's full data. Superadministrators manage councils, their admins and the app's name and logo. See [features.md](features.md), the [architecture plan](docs/specs/architecture.md) and [todo.md](todo.md).

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

**Your own name and logo:** in `/admin`, the "Personalizar a aplicación" tab sets the app's name and logo for the whole installation: the top bar, tab titles and browser icon, the QR reader screens, the calendar feed and the emails use them. Leave them empty to go back to LibreQuenda's. The footer keeps naming LibreQuenda as the free software it runs on (AGPL-3.0). The sign-in screens are Zitadel's and keep the branding from `zitadel/setup.sh`.

**Council administration:** a council's administrators (and superadministrators) open `/<council>/admin` to create facilities with photo, description, address, map pin, booking rules, weekly opening hours and closures. Each facility has its own page at `/<council>/<facility>`.

**Residents** never see Zitadel's own screens beyond signing in: name, email and password are changed from `/perfil`. Zitadel is branded as LibreQuenda by `zitadel/setup.sh`.

**Email:** LibreQuenda sends booking confirmations, a reminder 15 minutes before with the entry QR, cancellations (with the reason when the council closes a facility) and administrator invitations. Zitadel sends verification codes and password resets through the same relay.

**Images** are stored in **Garage** (self-hosted, S3-compatible; service `storage`). The database only keeps references such as `facilities/<id>.webp`; the API turns them into `/api/media/…` URLs when it answers. **Maps** use OpenStreetMap (Leaflet); "Como chegar" is a plain link to Google Maps directions. Address search goes through the API to OpenStreetMap Nominatim (or `GEOCODER_URL`).

**Sign in with Google (optional):** create an OAuth client ID (type *Web application*) in Google Cloud with the authorized redirect URI `https://<ZITADEL_DOMAIN>/idps/callback`, then set `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` in `.env` and run `docker compose up -d`. Leave them empty to remove the Google button again.

**Passwords, not passkeys:** sign-up and sign-in go straight to a password. Set `ZITADEL_PASSKEYS=true` in `.env` and run `docker compose up -d` to also offer passkeys.

**Short sessions:** access tokens last 10 minutes (`ZITADEL_ACCESS_TOKEN_MINUTES`), so a blocked account or a removed role stops working that soon. The web renews them silently with its refresh token, so people stay signed in while they use it; after 30 idle days (`ZITADEL_REFRESH_IDLE_DAYS`), or 90 days in all (`ZITADEL_REFRESH_DAYS`), they sign in again. Run `docker compose up -d` after changing them.

**Second factor for administrators:** council administrators and superadministrators only get into the administration after signing in with a second factor (an authenticator app, a security key or a passkey), set up in their sign-in account. Zitadel's access tokens don't say how the person signed in, so the API asks Zitadel (token introspection, with an API application `zitadel-setup` creates) once per token. On by default (`REQUIRE_ADMIN_MFA=true` in `docker-compose.yaml`); the local development stack leaves it off. Residents are not affected. Before turning it on in an existing installation, ask the administrators to set up a second factor first.

**TLS behind another proxy:** if a load balancer in front already terminates TLS, set `CADDY_AUTO_HTTPS=false` and keep `PUBLIC_SCHEME=https`; Caddy then serves plain HTTP to it.

**Images from CI:** every push to `main` and every version tag builds the images and publishes them to `ghcr.io/<owner>/librequenda-<image>` (`.github/workflows/ci.yml`).

## Check-in at the entrance

Councils need to know who actually turned up: it frees no-show turns, feeds the attendance indicators and keeps the facility for the people who booked it.

**What LibreQuenda does today.** The facility has a reader at the door (`/lector`): any phone, tablet or small computer with a camera, paired to the facility with a device key from `/<council>/admin`. The resident shows one of two QR codes:

- **Booking QR**: each booking has its own (in the app, in the confirmation and reminder emails, and printable by staff for someone without an account). The reader lets the person in from 15 minutes before the start until the end.
- **Flash booking QR**: the personal QR in the account menu. It changes every minute and works once, so a screenshot is useless. If the turn under way (or the one about to start) is free, scanning it books it on the spot, within the usual limits and rules.

Staff can also book from the desk ("Reserva manual"), print the entry QR for the person, and cancel.

The resident can also check in from their own phone, reading the facility instead of being read. Both ways are switched on per facility in the facility editor's **Entrada** tab, and both need the facility's map pin:

- **Printed QR at the door**: the tab prints a sheet with the facility's QR. It opens `/<council>/<facility>/entrada` with a random token, so the address alone does not work. "New QR" changes the token and the sheets already printed stop working.
- **Entry code**: the tab shows the facility's fixed 6-digit code and prints it for the door. It comes from an HMAC of a per-facility secret, so "New code" gives a new one and the printed copies stop working. The resident types it on the same `/entrada` page; their booking still has to be open for entry (from 15 minutes before it starts until it ends), and the endpoint has the strict rate limit, so guessing codes is slow.

Either way the phone's location is required: within the facility's distance (100 m unless the council changes it, between 25 and 1000 m) of its pin, with an accuracy of 150 m or better. The browser asks the resident; the API checks the location and drops it, it is never stored. A location can be faked by someone determined, so these two are the cheap options, not the strict ones. Then the check-in is the reader's own: the person's booking for this turn goes in. Without one, the page offers **Book and check in**, a flash booking of the free turn under the usual rules. The council can turn flash booking off for the facility in the same tab, which also turns it off for the personal QR at the readers.

**Staff confirmation** is the third way the tab switches on, and needs no pin. In the "Reserva manual" calendar, staff open a booking and press **Mark as in**, from 15 minutes before the start until the turn ends, as at a reader. It works for bookings by name too, and the booking shows who marked it. A staff entry can be undone for 5 minutes, in case they picked the wrong booking; entries from a reader or a phone cannot.

Every entry records how it happened (`reader`, `printed_qr`, `entry_code` or `staff`), and the council's data export includes it.

**Other ways it could work (not built yet).** The options below let a council choose by budget and by how much it cares about cheating. They would all end in the same check-in on the booking, so the indicators stay the same.

| Way | At the door | How the resident checks in | Cost | Hard to cheat? |
| --- | --- | --- | --- | --- |
| Booking or flash QR (built) | A device with a camera running `/lector` | Shows the QR from their phone or a printout | A cheap phone or tablet | Yes: tokens are per booking or change every minute |
| Printed QR (built) | A sheet of paper with the facility's QR | Scans it with their phone; the app checks in their booking for that facility and turn | Nearly free | Low: a photo of the sheet works anywhere, so the phone's location is always checked |
| Entry code (built) | A sheet at the door or the desk with the facility's fixed code | Types the code on their phone | Nearly free | Low on its own: the code does not change until the council makes a new one; the phone's location is always checked |
| NFC tag | A sticker on the door | Taps it with their phone, which opens the check-in page | A few euros per door | Low with a plain tag (it can be copied). High with a tag that signs every tap (e.g. NTAG 424 DNA "SUN" messages), which the API checks |
| Staff confirmation (built) | Nobody, or the desk | Staff mark them as in from the booking on the "Reserva manual" calendar | Staff time | As good as the staff |

What the ones not built yet would need:

- **NFC tag:** a plain tag can hold the `/entrada` address of the printed QR today. A signed one needs the tag's key, stored per door, and a check of the tap counter so a recorded tap cannot be replayed.

## Next: a personal home (the feed)

*Proposal, not built yet.* Today `/` is the same for everyone: every facility as a list beside a map, with filters. That is the right page for a visitor, but a resident who books the same court every Tuesday should not have to search for it every week. The idea:

- **Not signed in:** `/` stays as it is (list, map and filters).
- **Signed in:** `/` becomes a feed made from that person's own use. The search (list and map) moves to its own address, for example `/buscar`, one tap away from the feed and from the top bar.

**What the feed would show, most urgent first** (each card with its action right there):

1. **Now and next:** today's or the next booking, with the entry QR, directions, "invite" when there are spots left, and cancel while it still can be.
2. **Things waiting for them:** invitations to join a booking, and closures or cancellations that touch their bookings (what the bell already knows).
3. **Book your usual again:** the facility, weekday and hour they booked most in the last weeks ("Pádel, Tuesdays at 19:00"), when it is free this week, as one-tap booking. It says why it shows it.
4. **Their facilities today:** the facilities they book most (or mark as favourite), with today's free turns.
5. **Free soon near them:** facilities with free turns in the next hours that share a sport tag with what they usually book, in their council or near their location if they allow it.
6. **Quiet times:** the hours their usual facility tends to be quiet. The logic existed as the facility page's "Suggestions" (removed; see `api/core/tenants/suggestions.go` in git history) and can be reused here.
7. **New for them:** facilities or sports recently added in their council.

**Someone with no history yet** gets a short start instead: choose their council and the sports they play (from the tag catalogue). That is saved as their preferences and fills cards 4 and 5 until they have bookings.

**How it could be built:**

- `GET /me/feed` answers a list of typed cards (`next_booking`, `invitation`, `usual_turn`, `favourite_today`, `free_soon`, `quiet_hours`, `new_facility`), each with its data and the reason it is there. The web draws one component per card type. New card types can be added without changing the others.
- Plain rules ordered by urgency, not machine learning: easy to explain to a resident or a council ("because you book padel on Tuesdays"), and easy to test.
- New data: the person's preferences (council, sport tags) and optional favourite facilities, for example `user_preference` and `user_favourite_facility` tables. Everything else already exists (bookings, invitations, notifications, tags, opening hours).

**Privacy:** the feed only uses what LibreQuenda already keeps for the booking itself (the person's own bookings and the preferences they choose). It does not track pages viewed or clicks, does not share anything between residents or with the council, and the location is only used in the browser when they allow it, never stored. `/perfil` would say what the feed uses and let them reset their preferences.

**Open questions:** which cards councils care most about for the GOBTEC demo, whether favourites are explicit (a star) or inferred from bookings, and whether the feed should also arrive by email (a weekly "your usual turn is free").

## Documentation

- [Architecture plan](docs/specs/architecture.md)
- [Contest bases (v3, Galician)](https://www.agasol.gal/wp-content/uploads/2026/10/Bases_I_Certame_GOBTEC_2026_GL_v3.pdf), published by AGASOL
- API spec: generated into [`docs/openapi/`](docs/openapi/)

## License

Copyright (C) 2026 Tribo Solutions

This program is free software: you can redistribute it and/or modify it under the terms of the GNU Affero General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version. See [LICENSE](LICENSE).
