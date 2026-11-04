# LibreQuenda — Architecture Plan

> Status: draft · 2026-10-27
> Source of requirements: [Bases do I Certame GOBTEC 2026 (v3)](../certame/Bases_I_Certame_GOBTEC_2026_GL_v3.pdf)

## 1. Purpose

LibreQuenda is a free-software platform for **managing and booking municipal sports facilities** for Galician councils (*concellos*). It replaces manual booking by phone, email or in person with an online service that residents use to book and councils use to manage their facilities.

Scope is limited to what the contest bases ask for (§1.1, §1.2). Anything else is listed in [§11 Out of scope](#11-out-of-scope).

## 2. Requirements traceability

### 2.1 Minimum features (bases §1.1)

| Block | Requirement (bases) | Covered by |
|---|---|---|
| Cidadanía | Consulta de instalacións | Public facility catalogue per council, with availability |
| Cidadanía | Reserva online | Booking of free slots; council staff book for residents who phone or come to the desk (§6.1) |
| Cidadanía | Modificación e cancelación | Change or cancel own bookings within the facility's rules |
| Cidadanía | Historial | List of the user's past and upcoming bookings |
| Cidadanía | Avisos | Email + in-app notifications (§7) |
| Concello | Alta e mantemento de instalacións | Council admin: create/edit/deactivate facilities |
| Concello | Horarios e normas de uso | Opening hours, closures and usage rules per facility |
| Concello | Indicadores básicos | Indicators dashboard (§8) |
| Reutilización | Replicable, escalable, configurable, multi-municipio | Multi-tenant API (§4), self-hosting kit (§10) |

### 2.2 Technical constraints (bases §1.2)

| Constraint | How it is met |
|---|---|
| Software libre (OSI/FSF) | AGPL-3.0-or-later, as recommended by bases §8 |
| Web, responsive, modular | React SPA, mobile-first; separate API, web and identity services |
| Interoperabilidade: autenticación pública e estándares abertos | OIDC via Zitadel; federation with public identity systems (§5.4); REST + OpenAPI 3.1; iCalendar, CSV exports |
| Seguridade e cumprimento: RXPD, xestión segura de usuarios | Credentials only in the identity provider, roles in the API, data minimisation, data export and erasure (§9) |
| Accesibilidade: WCAG 2.1 AA (RD 1112/2018, EN 301549) | Accessibility built into the web UI and checked in CI (§9.3) |

### 2.3 Optional extras (bases §1.3)

Not committed. Candidates, sorted by cost:
- **Personal calendars:** `.ics` export or a subscription feed for the user's bookings. Cheap.
- **QR access:** QR code on each booking, checked by council staff. Cheap.
- **Usage analytics:** extends the indicators in §8.
- **Mobile app:** `app/` uses the same API and the same login (§5.3).
- **Time-slot recommendation:** not planned.

## 3. System overview

```
                   ┌─────────────────────────────┐
  Residents ──────►│  Web (React SPA)            │
  Council admins ─►│  web/                       │
                   └───────┬───────────┬─────────┘
                           │ OIDC      │ HTTPS + Bearer JWT
                           ▼           ▼
            ┌──────────────────┐  ┌──────────────────────────┐
            │ Zitadel          │  │ Core API (Go + Penguin)  │
            │ identity only    │  │ api/                     │──► SMTP (council / EU provider)
            │ (login, MFA,     │  │ tenants, facilities,     │
            │  federation)     │  │ bookings, roles, avisos  │
            └────────┬─────────┘  └───────────┬──────────────┘
                     │                        │
                     ▼                        ▼
            ┌─────────────────────────────────────────────┐   ┌──────────────────────┐
            │ PostgreSQL                                  │   │ Garage (S3)          │
            │  db "zitadel"  (role zitadel)               │   │ council logos and    │
            │  db "librequenda" (role librequenda)        │   │ facility images      │
            └─────────────────────────────────────────────┘   └──────────────────────┘

  Mobile app (app/, optional) ── same OIDC login, same API
```

| Component | Technology | Licence |
|---|---|---|
| Core API | Go, Penguin (router, scheduler, logger), sqlc | AGPL-3.0 (ours); sqlc MIT; Penguin Apache-2.0 |
| Web | React | AGPL-3.0 (ours) |
| Identity provider | Zitadel | AGPL-3.0 |
| Database | PostgreSQL | PostgreSQL Licence |
| Image storage | Garage (S3-compatible, single node) | AGPL-3.0 |
| Maps | Leaflet + OpenStreetMap tiles; address search with Nominatim (see §9.2) | BSD-2-Clause; ODbL data |
| Email | Any SMTP server | n/a |

## 4. Multi-tenancy (multi-municipio)

- A **tenant** is a council. All business data (facilities, rules, bookings, indicators) belongs to exactly one tenant.
- Tenancy lives **entirely in the Core API and its database**. Zitadel has no knowledge of councils.
- **A person has one account for every council.** A resident can book in Lalín and Silleda with the same login. Residents need no per-council registration.
- Every query is filtered by `tenant_id`. PostgreSQL row-level security may be added as a second line of defence.
- Each tenant has its own settings: name, slug, logo, contact email and phone, timezone (default `Europe/Madrid`), active flag. An SMTP sender per tenant is planned (§7).
- One deployment can serve one council (self-hosted) or many councils (for example, hosted by a Deputación).

## 5. Identity and access

### 5.1 Responsibilities

| | Zitadel | Core API |
|---|---|---|
| Who the user is (login, password, MFA, federation) | ✅ | ❌ |
| User profile mirror (`sub`, name, email) | source | copy, refreshed on login |
| What the user can do (roles per council) | ❌ | ✅ |

Zitadel runs with a **single organisation** for all users. Per-council organisations are deliberately not used, because they would give one person several accounts.

### 5.2 Login flow

1. The web app redirects to Zitadel (OIDC Authorization Code + PKCE).
2. The user logs in (password, or a federated identity provider when enabled).
3. The web app receives an access token (JWT) and calls the API with `Authorization: Bearer <JWT>`.
4. The API checks the JWT **locally** against Zitadel's public keys: signature (RS256), issuer, audience (the LibreQuenda project id), expiry. The keys are downloaded **on API startup** (`auth.InitializeAuth`, retried until Zitadel answers) and refreshed in the background every hour and whenever a token has an unknown key id (key rotation). No call to Zitadel per request.
5. The access token only carries `sub` (no name or email). `GET /api/me` asks Zitadel's userinfo endpoint once with the same token, then creates or updates `app_user` (`UpsertUserBySubject`) and returns it with the user's council roles.

Inside Docker the public issuer (`http://auth.localhost`) is not reachable, so the API calls `http://zitadel-api:8080` with `Host: <ZITADEL_DOMAIN>`. The expected audience is read from `config.json`.

Known limitation: an ID token from the same login also passes the signature/issuer/audience checks (both carry `client_id`). It is issued for the same user, so it grants nothing extra, and `/api/me` still fails for it at the userinfo step.

The API never receives passwords and never reads Zitadel's database.

### 5.3 Mobile

The mobile app uses the same OIDC flow through the system browser (RFC 8252). It uses the same accounts and tokens, and needs no API changes. The app must not hardcode a server URL, so that self-hosted deployments can use it.

### 5.4 Public authentication (federation)

- Zitadel can federate external SAML 2.0 / OIDC identity providers.
- **Cl@ve** (SAML 2.0, eIDAS profile) and **Chave365** (Xunta) can only be enabled by a public administration that has joined them. LibreQuenda is designed to accept them **through configuration, without code changes**.
- **Not yet validated:** compatibility of Cl@ve's eIDAS SAML profile with Zitadel's generic SAML support, and the integration method for Chave365. Both are pilot-phase tasks with a participating council.
- Digital certificates (DNIe, FNMT) are reached through Cl@ve, not natively.

### 5.5 Roles (authorisation)

Stored in the Core API:

| Role | Stored in | Scope | Can |
|---|---|---|---|
| *(none: any logged-in user)* | | all councils | browse, book, cancel own bookings, see own history |
| `admin` | `tenant_membership` | one council | manage that council's facilities, opening hours, usage rules and images, and follow its indicators |
| superadministrator | `platform_admin` | deployment | create, edit and delete councils; invite and remove council admins; add and remove other superadministrators; everything a council admin can do, in any council |

How roles are granted (web UI at `/admin`, API under `/api/admin/…`):

- **First superadministrator:** `SUPERADMIN_EMAIL` (defaults to `ZITADEL_ADMIN_EMAIL`). It is granted the first time that person signs in.
- **Council admins:** a superadministrator invites an email address (`tenant_invitation`). The role is granted the first time someone signs in with that email. The person may not have an account yet when invited. No invitation email is sent yet: the superadministrator tells the person.
- **Roles only follow a verified email.** Zitadel's `email_verified` claim is mirrored in `app_user`. Without it, anyone could register with someone else's address and receive their role.

Admin endpoints require a second factor (`REQUIRE_ADMIN_MFA`). Zitadel's JWT access token carries no `amr` claim, so the API introspects an administrator's token (`amr`: `pwd` alone, or with `otp`, `u2f`, `mfa`…) and caches the answer until the token expires.

## 6. Domain model

```
tenant ─┬─< facility ─┬─< opening_hours
        │             ├─< closure
        │             └─< booking >── app_user
        ├─< tenant_membership >── app_user
        └─< tenant_invitation
app_user ─┬─< notification
          └── platform_admin
```

| Entity | Key fields | Notes |
|---|---|---|
| `tenant` | id, slug, name, timezone, contact_email, contact_phone, logo_ref, is_active | A council |
| `facility` | id, tenant_id, slug, name, description, sport, address, latitude/longitude, capacity, accessibility_info, image_ref, is_active | The bookable unit (a court, pool lane, room…). Page at `/<council>/<facility>` |
| `facility` usage rules | slot_minutes, max_advance_days, min_cancel_hours, max_active_bookings_per_user, usage_rules_text | "Normas de uso". Columns on `facility`. |
| `opening_hours` | facility_id, weekday, opens_at, closes_at | Weekly schedule ("horarios") |
| `closure` | facility_id, starts_at, ends_at, reason | Holidays, maintenance |
| `booking` | id, facility_id, user_id, starts_at, ends_at, status (`confirmed`, `cancelled`), cancelled_by (`user`, `council`), rules_accepted_at, channel (`online`, `staff`), booked_by, holder_name, holder_phone, created_at, cancelled_at | No overlaps: PostgreSQL exclusion constraint on (facility_id, tstzrange) for confirmed bookings. `user_id` becomes NULL on account erasure, so indicators keep the booking. `holder_*`: someone without an account, booked by staff (§6.1) |
| `app_user` | id, idp_subject (Zitadel `sub`), email, full_name, email_verified | Mirror of identity, nothing more |
| `platform_admin` | user_id | Superadministrators |
| `tenant_membership` | user_id, tenant_id, role | Authorisation |
| `tenant_invitation` | tenant_id, email, role, invited_by, accepted_at | A role waiting for its person (§5.5) |
| `notification` | id, user_id, booking_id, type, payload, created_at, read_at, emailed_at | "Avisos". `emailed_at` NULL = pending in the email queue |

Booking rules are enforced in the API inside a transaction: within opening hours, not during a closure, within the advance window, below the user's active-booking limit, and no overlap (guaranteed by the database). The resident must accept the facility's usage rules to book (`rules_accepted_at`).

### 6.1 Booking by phone or at the desk

Many residents, often elderly people, phone or come to the desk instead of using the app. Council admins have a **Reservas** tab (`/<council>/admin?vista=reservas`): one facility's day, turn by turn, with who holds each booking and how to reach them.

- **Book a free turn for someone,** either for their account (by email: they get the usual confirmation, QR and reminders, and see it in the app) or by name and an optional phone, without an account. The turn must be free on the same schedule as in the app. Someone with an account cannot be booked into two places at once. Staff confirm they told the person the usage rules (`rules_accepted_at`). The per-person limit and the minimum age are left to staff, who are talking to the person.
- **Entry QR:** for a booking by name, staff print the QR (or the person photographs it), so the entrance reader works for them too.
- **Cancel any booking before it starts,** e.g. when the resident phones to say they cannot come. It counts as cancelled by the council. People with an account get a notice and email, with the reason if staff gave one.
- **RXPD:** the name and phone of someone without an account are removed 30 days after the booking ends (hourly job). The booking stays, anonymous, for the indicators.

Images (council logos, facility photos, photos of reported problems, profile photos) are stored in Garage, in **private buckets**: one per council (`council-<id>`) for its logo, its facilities' images and the photos of problems reported there, and the installation's (`STORAGE_BUCKET`) for profile photos and the installation logo. Nothing is served without a signature. The database only keeps a storage reference, never a URL: `<council id>/<folder>/<file>` for a council's files, `<folder>/<file>` for the installation's. The API turns a reference into a **presigned link** (S3 SigV4, valid 7 days, the same link reused for 3 days so browsers keep the image cached) only in the responses that show it; Caddy passes `GET`/`HEAD` requests carrying a signature under `/storage/*` to Garage, which checks it. A form can only save a reference to its own council's files (and a profile its own folder), so nobody gets another council's file signed by sending its reference. Uploads (`POST /media`) say whose file they are: `council` (slug, for its administrators) for logos and facility images, `facility` (id, for anyone signed in) for a problem's photo. A council's bucket is created with its first file and deleted with the council. Uploads are limited to JPEG, PNG and WebP up to 5 MB, with the type detected from the file's bytes (no SVG, which can carry scripts).

## 7. Notifications (avisos)

| Event | Recipient |
|---|---|
| Booking confirmed / changed / cancelled | User |
| Reminder before booking (e.g. 24 h) | User |
| Booking cancelled by the council (closure, maintenance) | Affected users |

Channels: **email** (SMTP, configurable per tenant) and an **in-app** notification list. Sending happens in a background worker inside the API process, using a PostgreSQL-backed queue (`notification.emailed_at`), so no extra broker is needed.

Current state:
- **Built:** notifications are written to the `notification` table. A scheduled job (Penguin scheduler, every 15 minutes) creates a `booking_reminder` for each confirmed booking starting within 24 hours.
- **Not built yet:** the email worker (SMTP), the in-app list (`GET /me/notifications`), and per-tenant SMTP settings.
- **Open:** how the web app learns about new notifications: polling, Server-Sent Events or WebSockets (§13).

## 8. Indicators (indicadores básicos)

Computed live with SQL over bookings, in one read-only snapshot per request. No separate analytics store. The council dashboard (`/<council>/admin?vista=indicadores`) shows them for the last 7, 30, 90 or 365 days, for all facilities or one, and refreshes on its own every 30 s to 15 min (or only by hand).

- **Bookings** per facility and per day, week or month (chosen from the length of the period), confirmed and cancelled
- **Occupancy:** booked hours ÷ bookable hours. Bookable hours are the facility's slots in the period with closures taken out, cut by the same `schedule` package that builds availability
- **Cancellation rate**, split into cancelled by the resident and by the council (`booking.cancelled_by`)
- **Busiest days and hours:** confirmed bookings by weekday and starting hour, as a heatmap
- **Active people:** distinct accounts with a confirmed booking, as owner or invited (guests without an account are not counted)
- **No-shows:** past confirmed bookings never checked in, only at facilities with a QR reader

A booking belongs to the period its start falls in, in the council's time zone. Every chart has a table alternative for screen readers.

**Comparison and facility detail.** Any period (the last 7, 30, 90 or 365 days, or a range of your own up to a year) is compared by default with the same number of days just before it, or with the same dates a year earlier (`compare=previous|year|none`). The API returns the same figures for both periods (`previous`), computed in the same snapshot; the web shows growth in % for counts and averages and in percentage points for rates (cancellations, no-shows and occupancy), green or red by whether the change is good news. Choosing a facility, from the filter or the per-facility table, turns the dashboard into that facility's detail: the same boards for it alone, plus occupancy by weekday and how people book (people per booking, owner plus invited; and the median notice between booking and playing), each against the compared period.

CSV export (`…/indicators/export?kind=summary|daily|hours|usage`). **Every indicators export is anonymous**: no row is a booking or a person, so nothing in it can be matched back to someone, not even by the council against its own database.

| `kind` | One row per | Contents |
|---|---|---|
| `summary` | facility | the dashboard figures for the period |
| `daily` | facility × day (empty days too) | bookings, confirmed, cancelled by resident / council, booked and open hours, occupancy, check-ins, invited people |
| `hours` | facility × weekday × starting hour | confirmed bookings added up over the whole period (never a single date and hour, which would be one booking) |
| `usage` | band of bookings per person (1, 2–4, 5–9, 10+) | how many people booked that often, as owner or invited |

**Council data export** (`GET /tenants/{slug}/admin/export`, data sovereignty: "portabilidade íntegra dos datos"). The council takes everything it keeps in LibreQuenda to another system as one ZIP. It has one CSV per table (`council`, `facilities`, `opening_hours`, `closures`, `checkin_devices`, `bookings`, `booking_participants`; RFC 4180, UTF-8), the council logo and facility images under `media/`, and a `datapackage.json` ([Frictionless Data Package](https://specs.frictionlessdata.io/)) giving every column's type and meaning plus the primary and foreign keys between the tables, so any Frictionless tool can validate and load it. Times are RFC 3339 in UTC; opening hours are local to the council's `timezone`. Unlike the indicators, it has a row per booking. It always leaves out secrets (SMTP, device keys, QR and invite tokens). By default a person is only their account id, the same across tables, and guests appear only as guests. With `?personal=1` (a checkbox in the web, off by default) it also brings `people.csv`, with the name, email, phone, birth date and language of everyone who booked or joined a booking at the council, plus the guests' names and ages, so the council can move upcoming bookings with their owners. The council is the controller of that data. Each personal export is logged with the administrator's account id (GDPR art. 5.2). It is read in one snapshot and rate-limited like other heavy routes; the web offers it in the council sheet.

Counts of 1 or 2 come out as `<3`, and the figures worked out from them (hours, rates, occupancy) are left empty, so a quiet slot does not point at whoever uses it. This is primary suppression only: combining several totals can still narrow a hidden count down, which is accepted for figures this coarse. The dashboard is internal and shows exact numbers.

Column names are fixed English identifiers. `excel=1` gives `;` separators and decimal commas for Spanish-locale Excel; both carry a UTF-8 byte order mark.

## 9. Security, privacy and accessibility

### 9.1 Security
- Passwords and MFA only in Zitadel. The API holds no credentials.
- Separate PostgreSQL roles per database. The API cannot read identity data, and Zitadel cannot read bookings.
- Zitadel console not publicly exposed. Reverse proxy with TLS and rate limiting.
- Dependency and container image scanning in CI. Prompt updates of Zitadel and PostgreSQL.

#### Production hardening

Development defaults that must change before a real deployment:

- **Public runtime config is fine.** `/config.json` only holds public OIDC client data (authority, client id, project id, scope), which is visible in the login URL and in tokens anyway. Security comes from the registered redirect URIs and PKCE. Never add secrets to it.
- **Admin token (`admin.pat`), done.** The `librequenda-setup` service account must be an instance owner: the setup changes instance policies (login, OIDC token lifetimes, branding, privacy, identity providers), which no narrower role can. Its token lives alone in the `zitadel-admin` volume, mounted read-only by `zitadel-setup` and by nothing else. `ZITADEL_SETUP_REVOKE_TOKEN=true` revokes it once the setup succeeds (later starts keep the configuration); `ZITADEL_ADMIN_PAT_EXPIRATION` sets its expiry on the first start.
- **Separate bootstrap volumes, done.** `zitadel-login` mounts only `zitadel-login-client` (its `login-client.pat`); the API only `api-secrets` (its user-manager token and the introspection credentials).
- **Already handled:** with `PUBLIC_SCHEME=https`, `zitadel-setup` turns off the OIDC app's `devMode` (no plain-HTTP redirects) and drops the `localhost:5173` redirect URIs.

### 9.2 Data protection (RXPD / GDPR)
- Data minimisation: name, email and bookings only. For residents booked by staff without an account, a name and an optional phone, removed 30 days after the booking (§6.1).
- **Export:** a user can download their data (JSON/CSV).
- **Erasure:** account deletion removes `app_user` data in the API (bookings anonymised for indicators) and deletes the identity in Zitadel via its API.
- Personal data stays on the deployment's infrastructure. The SMTP provider is chosen by the council.
- **Third parties contacted today (maps):**
  - **Map tiles:** the browser loads them directly from `tile.openstreetmap.org`, so OpenStreetMap sees the visitor's IP address. Not configurable yet: a self-hosted or configurable tile server is pending.
  - **Address search** (council admins placing a facility on the map): the API calls Nominatim, so the visitor's IP never reaches it. Defaults to the public `nominatim.openstreetmap.org` (max one request per second); `GEOCODER_URL` points it at a council's own Nominatim.
  - **Directions:** a link that opens Google Maps. Nothing is sent until the user follows it.

### 9.3 Accessibility
- Target: **WCAG 2.1 AA** (RD 1112/2018, EN 301549).
- Semantic HTML, keyboard navigation, visible focus, sufficient contrast, accessible forms and error messages, screen-reader-friendly calendar and slot picker.
- Automated checks (axe) in CI plus manual review. An accessibility statement is published per deployment.

## 10. API and deployment

### 10.1 API
- REST over HTTPS, JSON. HTTP routing with **Penguin**. The OpenAPI 3.1 spec is written after the code and published in [`docs/openapi/`](../openapi/). Penguin does not generate OpenAPI, so the method is still to be decided.
- Health endpoints: `GET /health/api`, `GET /health/db`.
- Routes today (all under `/api`; "planned" ones do not exist yet):

| Group | Routes | Auth |
|---|---|---|
| Public | `GET /tenants` (council list: name, logo, public contact; no usage figures), `GET /tenants/{slug}` (one council), `GET /tenants/{slug}/day?date=` (every active facility with its slots for one day), `GET /tenants/{slug}/facilities/{facility}?date=` (details, slots, weekly schedule), images through presigned `/storage/…` links | none |
| Resident | `GET /me`, `GET\|POST /me/bookings`, `DELETE /me/bookings/{id}` (cancel) | logged in |
| Council admin | `GET /tenants/{slug}/admin/facilities/{id}/bookings?date=` (staff day view), `POST /tenants/{slug}/admin/bookings` (book for a resident), `POST …/admin/bookings/{id}/cancel`, `GET\|POST /tenants/{slug}/admin/facilities`, `GET\|PUT\|DELETE …/facilities/{id}`, `PUT …/facilities/{id}/schedule`, `GET /tenants/{slug}/admin/indicators?days=&facility=`, `GET …/indicators/export?kind=&excel=` (CSV), `GET …/admin/export` (ZIP, full data), `GET /geocode`, `POST /media` (upload) | council admin or superadmin |
| Superadmin | `GET\|POST /admin/tenants`, `GET\|PUT\|DELETE /admin/tenants/{slug}`, `POST /admin/tenants/{slug}/invitations`, `DELETE …/invitations/{id}`, `DELETE …/admins/{userId}`, `GET\|POST /admin/superadmins`, `DELETE /admin/superadmins/{userId}` | superadmin |
| Planned | `PATCH /me/bookings/{id}`, `GET /me/notifications`, `GET /me/export`, `…/closures` | |

### 10.2 Self-hosting kit

`docker compose` at the repository root (`docker-compose.yaml`): a real SMTP relay is required, PostgreSQL is internal only, and every secret must be set in `.env`.

| Service | Purpose |
|---|---|
| `proxy` | **Caddy, the only public entry point.** Serves the React build, proxies `/api/*` to the API, and serves Zitadel on its own domain. Automatic HTTPS when `CADDY_AUTO_HTTPS=true`; plain HTTP otherwise (development, or TLS terminated by a proxy in front). Config: `proxy/Caddyfile` (no container labels, no Docker socket). |
| `db` | PostgreSQL 18: databases `librequenda` and `zitadel`, each with its own role |
| `api` | Core API (not published on the host, reachable only through the proxy) |
| `zitadel-api`, `zitadel-login` | Identity provider (v4.20.0) and its Login v2 UI |
| `storage` | Garage, S3-compatible store for uploaded images. Internal only: private buckets (one per council); the API uploads to it and hands out presigned links, which the proxy passes to it under `/storage/…`. Secrets (`STORAGE_*`, `GARAGE_*`) must be set in `.env` |
| `zitadel-setup` | One-off, idempotent: creates the `LibreQuenda` project and its `web` OIDC app (PKCE, no secret) and writes `config.json` (authority, client id, scopes), which the proxy serves to the SPA at `/config.json` |

```
${WEB_DOMAIN}          /        -> React SPA (static files in the proxy image)
                       /api/*   -> api:8000
${ZITADEL_DOMAIN}      /ui/v2/login* -> zitadel-login:3000
                       /*       -> zitadel-api:8080 (h2c)
```

Configured via `.env` (see `.env.example`). The mobile app is not part of the compose stack.

### 10.3 Database (`db/`)
- `db/` builds the PostgreSQL image (official `postgres:18-alpine` + init scripts).
- `db/init/`: schema (`00_extensions`, `01_tables`, `02_indexes`, `03A_triggers`), run on first start. Demo data (`09_initial_data.sql`) is never loaded by this stack.
- `db/queries/`: sqlc queries. `api/sqlc.yaml` reads them plus `db/init/01_tables.sql` and generates `api/model/`.
- Schema changes: edited directly in `db/init/` while there are no production deployments. A migration tool will be added before the first real deployment.

## 11. Out of scope

Not requested by the bases, so not part of the core:
- Payments and fees
- Clubs and season-long allocations
- Resident (empadronado) priority
- Waitlists, recurring bookings, no-show tracking
- AI components (the product uses none)

## 12. Repository layout

```
LibreQuenda/
├── .agents/        AI agent instructions for development
├── .github/        CI workflows
├── app/            Mobile app (optional, later)
├── deploy/         Production self-hosting kit
├── proxy/          Caddy image: Caddyfile + web build
├── docs/
│   ├── brand/      Logo, app icon and favicon (SVG sources)
│   ├── certame/    Contest bases
│   ├── openapi/    Generated OpenAPI spec
│   └── specs/      This and other design documents
├── api/            Go core API
├── db/             PostgreSQL image: init schema, seed, sqlc queries
├── web/            React web app
├── LICENSE         AGPL-3.0-or-later
└── README.md
```

## 13. Open points

- [x] Penguin licence: Apache-2.0 since v0.2.0.
- [ ] Decide how to produce the OpenAPI spec without Huma (hand-written after the code, or annotations).
- [x] Zitadel licence: AGPL-3.0 (confirmed). Login v2 runs as a separate container (`zitadel-login`).
- [x] Production hardening of the Zitadel bootstrap (see §9.1).
- [x] Go API: verify the Zitadel JWT and upsert `app_user` (`GET /api/me`).
- [x] Short access-token lifetime (10 min by default) in Zitadel, so blocked users lose access quickly.
- [x] MFA for admins: the JWT access token has no `amr`; token introspection has it (`REQUIRE_ADMIN_MFA`).
- [ ] Cl@ve / Chave365 federation: confirm the integration path (ask AMTEGA / AGASOL).
- [ ] Decide on the §1.3 extras (QR, `.ics`).
- [ ] Mobile app technology.
- [x] UI languages: Galician (fallback), Spanish, English and Catalan. The bases don't require any.
- [x] Council and admin setup: superadmin web UI with invitations by email address (§5.5), instead of a CLI or seed.
- [ ] Notification delivery to the web app: polling, Server-Sent Events or WebSockets (§7).
- [ ] Map tiles: configurable or self-hosted tile server, so visitors' IPs don't reach OpenStreetMap (§9.2).
