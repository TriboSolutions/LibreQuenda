# LibreQuenda features

What the app does today, what is being built, and ideas for later. For the plan and what is left before the contest, see [todo.md](todo.md); for how it works inside, [docs/specs/architecture.md](docs/specs/architecture.md).

Status: **built** · **in progress** (open work) · **idea** (not started).

## Residents

| Feature | Status |
|---|---|
| Home page with every facility of every council: list beside a map, search, filters by council, sport and kind of place (tag catalogue), "near me" with a distance | built |
| Council pages and facility pages: photos, description, address and directions, contact (the facility's, else the council's), opening hours, closures, usage rules, minimum age, the booking window that applies | built |
| Day view of free, booked, closed and past turns | built |
| Book a free turn, accepting the usage rules; change it to another time of the same facility; cancel it until the deadline | built |
| Invite people to fill a booking's capacity by link, with an account or as a guest (name and age); leave a booking; the owner removes people | built |
| Public bookings: make a booking public, and join others' public bookings from "For you" | in progress |
| My bookings: upcoming and history, countdown to each turn ("starts in 2 h", "you can go in now"), attendance of past ones | built |
| "For you" on the home page: your usual turn when it falls today and is free, the facilities you book most with turns left today, similar ones you have not tried; can be turned off | built |
| Notifications in the app (bell, refreshed every 30 s) and emails: confirmation, reminder 15 minutes before, change, cancellation by you or by the council (with the reason and the closure dates) | built |
| Emails explain how to check in at that facility (only its ways), with directions and the facility's and council's contacts | built |
| Calendar feed (.ics) to subscribe from Google, Outlook or Apple, with one-click buttons | built |
| Profile: photo (copied to the sign-in account), birth date, phone; preferences apart (email language, reminders, "For you") | built |
| Crop the profile photo in a round frame | in progress |
| Account in Zitadel: sign up and in with a password or Google, change name, email and password from the profile | built |
| Verified email required to book (keeps bot accounts from taking turns) | in progress |
| Your data as a file (JSON and CSV) and deleting your account (bookings anonymised, sign-in account deleted) | built |
| Four languages (Galician, Spanish, English, Catalan), light and dark theme | built |

## Checking in at the door

| Feature | Status |
|---|---|
| QR reader at the door (`/lector`, any phone or tablet with a camera, paired with a device key): scans the booking's QR or the personal flash QR | built |
| Turn a facility's readers off without removing them (broken, switched off) | built |
| Printed QR on the door, read with the phone's camera from the app; the phone's location must be within the facility's distance (checked before the camera opens, never stored) | built |
| Fixed entry code per facility, printable (also in braille, to scale for swell paper or an embosser), typed in the app with the location checked | built |
| Staff confirmation from the bookings calendar, undoable for 5 minutes; contact people per facility (who to ask on site), falling back to the facility's and the council's contact | built |
| "Check in" dialog in My bookings with one tab per way the facility takes | built |
| Flash booking: book the free turn under way or about to start, with any way to check in (personal QR at a reader, entry code, door QR, staff); can be turned off per facility | built |
| NFC tags at the door | idea (shown as a preview in the Entrada tab) |

## Councils

| Feature | Status |
|---|---|
| Facilities: create, edit, deactivate (optionally cancelling upcoming bookings with a notice), photo, map pin (address search), tags, capacity, minimum age, usage rules | built |
| Weekly opening hours and closures (cancelling the bookings inside, with an apology and the reason) | built |
| Ways to check in per facility (Entrada tab) and printable QR and code sheets | built |
| Booking window per council, inherited by its facilities or overridden in each one's Reserva tab: how far ahead (hours or days), how close to the start from the app, cancel and change deadline, upcoming bookings, bookings per day, turns in a row, optional no-show penalty | built |
| Weekly limit per person and turns that open at a fixed hour (e.g. Sunday 20:00) | in progress |
| Staff booking by phone or at the desk, for an account or just a name and phone; cancel any booking; print the entry QR | built |
| Indicators: bookings, occupancy, cancellations, no-shows (counting every way to check in), busiest days and hours, comparison with another period, per facility; CSV and Excel export | built |
| No-shows remember when check-in was off (history of the facility's ways to check in) | in progress |
| Council settings page: details, administrators (invite, remove), booking window, own email relay (SMTP) with a test email, full data export (CSV + Frictionless datapackage + images, optionally with residents' personal data) | built |

## Platform

| Feature | Status |
|---|---|
| Superadministrators: create, edit, deactivate and delete councils, invite their administrators, manage superadministrators | built |
| Tag catalogue in categories, shared by every council | built |
| The installation's own name and logo (self-hosting) | built |
| Self-hosting with Docker Compose: PostgreSQL with row-level security per council, Zitadel for accounts, Garage for files, Caddy in front | built |
| Accessibility (WCAG 2.1 AA and the AAA criteria that apply), with automated checks in CI and an accessibility statement page | built |
| CI: Go and web checks, vulnerability and image scanning (govulncheck, pnpm audit, Trivy), images published to ghcr.io | built |
| Short-lived access tokens, second factor required for administrators, least-privilege Zitadel bootstrap | in progress |
| Rate limit on sign-up and sign-in at the proxy (no captcha: Zitadel's login has none) | in progress |
| Demo deployment, production kit (`deploy/`), OpenAPI spec | idea (see todo.md) |

## Ideas for later

Grouped by who gains most; none of these is started.

**Residents**

- **Waiting list:** join the queue of a full turn and get a notice (and a few minutes to take it) when someone cancels.
- **Find players:** say you are looking for people for a sport and level, and see others who are, building on public bookings.
- **Web push notifications** (no app needed) and **SMS** for people without email, through the council's provider.
- **Installable app that works offline:** the manifest exists; a service worker would keep My bookings and the entry QR available without coverage at the door.
- **Report a problem** with a facility (broken net, no light) from the booking, landing in the council's maintenance list.
- **Weather notice** for outdoor facilities (MeteoGalicia's open data), suggesting to change or cancel before the deadline.

**Councils**

- **Recurring bookings for clubs and schools:** a season's slots booked in one go by staff, with their own limits, and priority windows for registered entities.
- **Fees and payment:** a price per turn or a pass, paid through the public administrations' payment gateway (Pasarela de Pagamentos), with exemptions.
- **Limits across facilities:** e.g. at most two upcoming padel bookings across all the council's courts.
- **Kiosk screen at the entrance:** today's turns and who is in, on the reader's tablet.
- **Embeddable widget** of a facility's free turns for the council's own website.
- **Open data:** publish anonymised occupancy as open data (DCAT-AP, datos.gob.es / abertos.xunta.gal) on a schedule.

**Platform**

- **Cl@ve / Chave365** sign-in for residents (to confirm with AMTEGA/AGASOL).
- **Native mobile app** (`app/`), if the installable web app is not enough.
- **Audit log** of administrators' actions (who closed what, who cancelled whose booking).
