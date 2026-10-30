# TODO

Deadline: **18 Nov 2026 23:59** (certame@agasol.gal). Final pitch: 26 Nov 2026.
Scope: bases §1.1 (minimum) + §1.2 (technical). See [architecture plan](docs/specs/architecture.md).

## Residents (bases §1.1 — Cidadanía)

- [x] Facility catalogue and availability per council
- [x] Book a free slot
- [x] Cancel own booking
- [x] Flash booking, like Enjoy: the personal QR (account menu → «Reserva flash») books a free turn at the reader, from now until the turn under way ends or the one about to start
- [x] Change own booking (`PATCH /me/bookings/{id}`): another day or time of the same facility; the owner can remove people who joined
- [x] Booking history (past and upcoming)
- [x] Booking window: the council sets a default every facility inherits, and a facility can keep its own: how far ahead (hours or days), how close to the start from the app, cancel/change deadline, upcoming bookings, bookings per person and day, turns in a row, and an optional no-show penalty. Staff booking for someone skip the per-person limits
- [x] In-app notification list (`GET /me/notifications`)
- [x] Decide if in-app notifications need real-time push (WebSockets, SSE) or if polling is enough: polling every 30 s
- [x] Email notifications: booking confirmed / reminder (15 min before) / cancelled; owner only
- [x] Notify affected users when the council cancels bookings (closure, or deactivation when the council chooses to cancel)

## Councils (bases §1.1 — Concello)

- [x] Create / edit / deactivate facilities
- [x] Opening hours (schedule)
- [x] Closures (cancel the bookings inside, apology notice and email with the reason)
- [x] Usage rules editable per facility
- [x] Indicators dashboard: bookings per facility, occupancy, cancellation rate, busiest days/hours, active users
- [x] Indicators CSV export
- [x] Full council data export (`…/admin/export`): ZIP of CSVs + Frictionless `datapackage.json` + images, for data portability; `?personal=1` adds residents' personal data
- [x] Council members management (`…/admin/members`)
- [x] SMTP settings per council (falls back to the `SMTP_*` relay; with neither, the council's emails wait)

## Privacy (RXPD / GDPR)

- [ ] Data export for the user (`GET /me/export`, JSON/CSV)
- [ ] Account erasure: anonymise bookings, delete the identity in Zitadel

## Accessibility (WCAG 2.1 AA, plus the AAA criteria that apply)

- [x] Keyboard navigation and focus review (slot picker, dialogs)
- [x] axe checks in CI (`pnpm test:a11y`, web/e2e)
- [x] Accessibility statement page (`/accesibilidade`, RD 1112/2018 model)

## Interoperability and docs

- [ ] Decide how to produce the OpenAPI spec, then write it into `docs/openapi/`
- [ ] Translations complete in gl / es / en / ca (i18n-doctor)

## Deployment and security

- [x] CI in `.github/workflows/ci.yml`: Go (gofmt, sqlc diff, vet, tests), web (lint, types, build), the five images (published to ghcr.io on main and tags)
- [ ] Dependency and image scanning in CI (govulncheck, pnpm audit, Trivy)
- [ ] `deploy/` production self-hosting kit
- [ ] Zitadel bootstrap hardening (least-privilege `admin.pat`, split bootstrap volumes) — architecture §9.1
- [ ] Short access-token lifetime (5–15 min)
- [ ] Verify the `amr` claim for admin MFA
- [ ] Cl@ve / Chave365: confirm the integration path with AMTEGA / AGASOL

## Contest submission

- [ ] Project report (memoria) for the bases
- [ ] Demo deployment with seed data (the seed has three real councils: Lalín, Lugo and Ourense)
- [ ] Pitch deck for 26 Nov

## Optional (bases §1.3, only if cheap)

- [x] `.ics` calendar feed of the user's bookings (subscribe from /perfil)
- [ ] QR code on each booking
- [ ] Mobile app (`app/`)
