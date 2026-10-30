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

- [x] Data export for the user (`GET /me/export`, JSON/CSV)
- [x] Account erasure: anonymise bookings, delete the identity in Zitadel

## Accessibility (WCAG 2.1 AA, plus the AAA criteria that apply)

- [x] Keyboard navigation and focus review (slot picker, dialogs)
- [x] axe checks in CI (`pnpm test:a11y`, web/e2e)
- [x] Accessibility statement page (`/accesibilidade`, RD 1112/2018 model)

## Interoperability and docs

- [ ] Decide how to produce the OpenAPI spec, then write it into `docs/openapi/`
- [x] Translations complete in gl / es / en / ca: same keys in all four, web and API (checked 30 Oct)
- [ ] Native review of the es / en / ca wording (written alongside the Galician)

## Deployment and security

- [x] CI in `.github/workflows/ci.yml`: Go (gofmt, sqlc diff, vet, tests), web (lint, types, build), the five images (published to ghcr.io on main and tags)
- [x] Dependency and image scanning in CI (govulncheck, pnpm audit, Trivy)
- [ ] `deploy/` production self-hosting kit
- [ ] Zitadel bootstrap hardening (least-privilege `admin.pat`, split bootstrap volumes) — architecture §9.1
- [ ] Short access-token lifetime (5–15 min)
- [ ] Verify the `amr` claim for admin MFA
- [ ] Cl@ve / Chave365: confirm the integration path with AMTEGA / AGASOL

## Activity log

- [x] Audit log of administrative changes (superadmins: all; council admins: their council), append-only, `AUDIT_RETENTION_DAYS` retention, pseudonymised on account erasure, CSV download
- [ ] Add a line to `audit/actions.go` for each new administrative route (until then they are logged as "other")

## Contest submission

- [ ] Project report (memoria) for the bases
- [ ] Demo deployment with seed data (the seed has three real councils: Lalín, Lugo and Ourense)
- [ ] Pitch deck for 26 Nov

## Check-in and booking (built 28–30 Oct, follow-ups)

- [x] Ways to check in per facility: QR reader (can be switched off without removing it), printed QR at the door and fixed entry code (both need the phone within the facility's distance), staff confirmation with contact people (falls back to the facility's, then the council's contact)
- [x] «Check in» from My bookings (countdown, one tab per way, camera for the door QR) and flash booking with any way
- [x] Emails explain only the facility's ways to check in, with directions and contacts
- [x] No-show indicators count every way to check in, not only QR readers
- [x] Booking window per council, inherited by its facilities or overridden in each one's «Reserva» tab; optional no-show penalty
- [x] «For you» recommendations on the home page (can be turned off in the profile's Preferences)
- [x] Profile photo copied to the person's Zitadel account; Zitadel signs up and in with a password (passkeys with `ZITADEL_PASSKEYS=true`)
- [ ] NFC tag: still a preview in the Entrada tab (needs a public check-in address per tag and, for signed tags, NTAG 424 DNA verification)
- [x] Indicators: a history of when each facility could take check-ins (`facility_checkin_period`), so turns while the QR reader was off, or every way was off, are not no-shows (#31)
- [x] Profile photo: checked end to end on 30 Oct, the app's photo reaches the Zitadel account. Zitadel's own sign-in pages (login v2, v4.20) always show initials, never the photo; its console shows it
- [x] Profile photo: framed in a circle (move and zoom) before it is uploaded, and reframed later (#27)
- [ ] Profile photo: removing a photo that only exists in Zitadel (none in the app) does not remove it there
- [ ] Try the camera and location flows on a real phone over HTTPS (door QR, entry code, reader)
- [x] Booking window: bookings per person and week, and next week's turns opening all at once at a fixed weekday and time instead of the rolling window
- [ ] Booking window, later if wanted: limits across all the council's facilities
- [ ] «For you» as the full signed-in feed of the README proposal (search on its own page, invitations, quiet hours, new facilities)

## Optional (bases §1.3, only if cheap)

- [x] `.ics` calendar feed of the user's bookings (subscribe from /perfil)
- [x] QR code on each booking (in «Check in» and in the confirmation and reminder emails, where the facility has a reader)
- [ ] Mobile app (`app/`)
