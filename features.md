# LibreQuenda features

What the app does today, what is being built, and ideas for later. For the plan and what is left before the contest, see [todo.md](todo.md).

Status: **built** · **in progress** (open work) · **idea** (not started).

## Residents

| Feature | Status |
|---|---|
| Home page with every facility of every council: list beside a map, search, filters by council, sport, kind of place (tag catalogue) and kind of booking (public games, events), "near me" with a distance | built |
| Council pages and facility pages: photos, description, address and directions, contact (the facility's, else the council's), opening hours, closures, usage rules, minimum age, the booking window that applies, the residents' average rating | built |
| Day view of free, booked, closed and past turns | built |
| Weather forecast at outdoor facilities (MeteoGalicia's open data, by the council's municipality): the day on the facility page, the turn's period in the booking dialog, with a warning in words for rain, storm, snow or hail, strong wind, heat or frost | built |
| Book a free turn, accepting the usage rules; change it to another time of the same facility; cancel it until the deadline | built |
| Invite people to fill a booking's capacity by link, with an account or as a guest (name and age); leave a booking; the owner removes people | built |
| Public bookings: make a booking public, and join others' public bookings from "For you" | built |
| Council events: join the council's events (a title, its own capacity and time) from the facility page, "For you" or the top of the app's home, and hear when one is moved or cancelled | built |
| My bookings: upcoming and history, countdown to each turn ("starts in 2 h", "you can go in now"), attendance of past ones | built |
| "For you" on the home page: your usual turn when it falls today and is free, the facilities you book most with turns left today, similar ones you have not tried, public games and events to join; can be turned off | built |
| Ratings: when a booking ends, rate the facility from 1 to 5 stars and, after a public booking, the others with an account; anonymous (only the average and the count are shown), final once given; averages on facilities and on each person's profile | built |
| Notifications in the app (bell, refreshed every 30 s) and emails: confirmation, reminder 15 minutes before, change, cancellation by you or by the council (with the reason and the closure dates) | built |
| Emails explain how to check in at that facility (only its ways), with directions and the facility's and council's contacts | built |
| Calendar feed (.ics) to subscribe from Google, Outlook or Apple, with one-click buttons | built |
| Profile: photo framed in a circle (copied to the sign-in account), birth date, phone; preferences apart (email language, reminders, "For you") | built |
| Account in Zitadel: sign up and in with a password or Google (passkeys optional), change name, email and password from the profile | built |
| Verified email required to book, join a turn or book at the door (keeps bot accounts from taking turns) | built |
| Your data as a file (JSON and CSV) and deleting your account (bookings anonymised, sign-in account deleted) | built |
| Residents-only facilities marked «Só para veciños» on their page, their cards and their events; someone not verified there asks to be: phone the council, or send a request (DNI or NIE, name, postal code, street), one a week at most, and withdraw it while pending; told by notice and email when the council decides | built |
| Report a problem at a facility (broken net, no light, dirty changing rooms…) from a booking or the facility's page, with a category, a photo and whether it is urgent; told by notice and email when the council fixes or rejects it | built |
| Terms of use and privacy policy pages; support contact in the footer | built |
| Four languages (Galician, Spanish, English, Catalan), light and dark theme | built |

## Android app

| Feature | Status |
|---|---|
| Residents' app (Kotlin, Jetpack Compose) on the same API and Zitadel sign-in as the web (native OIDC app): home with events, "For you" and today's public games; signed out, the facilities list to look through | built |
| Facility search in a list or on a map («Ver no mapa»), facility page and booking | built |
| Bookings and history: change a booking's turn, report a problem, rate it, check in every way (reader QR, entry code, door QR, NFC tag, staff) | built |
| Flash booking from the app, also with an NFC tag | built |
| Notices without push (background check), local reminders, Android notifications to rate a booking once it ends | built |
| Profile, preferences, language, support and legal texts; texts generated from the web's locales | built |
| Residents-only facilities marked «Só para veciños» (facility page, cards, events and public games), and asking to be verified from a bottom sheet: phone the council or send the request, see it pending and withdraw it (administration stays on the web) | built |

## Checking in at the door

| Feature | Status |
|---|---|
| QR reader at the door (`/lector`, any phone or tablet with a camera, paired with a device key): scans the booking's QR or the personal flash QR | built |
| Turn a facility's readers off without removing them (broken, switched off) | built |
| Printed QR on the door, read with the phone's camera from the app; the phone's location must be within the facility's distance (checked before the camera opens, never stored) | built |
| Fixed entry code per facility, printable (also in braille, to scale for swell paper or an embosser), typed in the app with the location checked | built |
| NFC tags at the door: the tag's address and «new address» in the admin, read by the Android app or opened on the web's entrance page | built |
| Staff confirmation from the bookings calendar, undoable for 5 minutes; contact people per facility (who to ask on site), falling back to the facility's and the council's contact | built |
| "Check in" dialog in My bookings with one tab per way the facility takes | built |
| Flash booking: book the free turn under way or about to start, with any way to check in (personal QR at a reader, entry code, door QR, NFC, staff); can be turned off per facility | built |

## Councils

| Feature | Status |
|---|---|
| Facilities: create, edit, deactivate (optionally cancelling upcoming bookings with a notice), photo, map pin (address search), tags, capacity, minimum age, usage rules | built |
| Residents-only facilities: administrators verify people as the council's residents (Veciños tab, by email), and take the verification back (with a confirmation); the council's administrators, and superadministrators everywhere, count as verified without a request; only they book and join there (bookings, public games, events, invite links), guests cannot | built |
| Requests to be verified: they reach the Veciños tab (badge with the pending ones), the administrators' bell and their email («X quere ser veciño»); staff check the DNI and address against their own census, then verify or discard. Either way the request is deleted at once, with its DNI and address, and one nobody decided on is deleted after 15 days | built |
| Weekly opening hours and closures (cancelling the bookings inside, with an apology and the reason) | built |
| Ways to check in per facility (Entrada tab) and printable QR and code sheets | built |
| Booking window per council, inherited by its facilities or overridden in each one's Reserva tab: how far ahead (hours or days) or next week opening at a fixed hour (e.g. Sunday 20:00), how close to the start from the app, cancel and change deadline, upcoming bookings, bookings per day and per week, turns in a row, optional no-show penalty | built |
| A council-wide cap on each person's upcoming bookings across all its facilities; one council at a time per account (a platform setting, on by default); nobody can hold two bookings at the same time | built |
| Events (Eventos tab): a kind of booking only administrators create, with a title, an optional capacity, any start and end and no per-person limits; it lists the bookings or closures in the way, which staff cancel first; moving or cancelling one tells who joined | built |
| Staff booking by phone or at the desk, for an account or just a name and phone; cancel any booking; print the entry QR | built |
| Indicators: bookings, occupancy, cancellations, no-shows (counting every way to check in), busiest days and hours, ratings (average, spread from 1 to 5, over time), comparison with another period, per facility; CSV and Excel export | built |
| No-shows remember when check-in was off (history of the facility's ways to check in) | built |
| Widget of free turns to embed in the council's own website (`/<council>/widget`, all or one facility, light or dark, any of the four languages), with the code to paste and a preview in the council settings | built |
| Maintenance list (Incidencias tab): residents' reports with a badge of new ones, filters by facility and status, status and a note for the reporter, the reporter's email only if they shared it | built |
| Council settings page: details, administrators (invite, remove), booking window, own email relay (SMTP) with a test email, full data export (CSV + Frictionless datapackage + images, optionally with residents' personal data) | built |

## Platform

| Feature | Status |
|---|---|
| Superadministrators: create, edit, deactivate and delete councils, invite their administrators, manage superadministrators | built |
| Tag catalogue in categories, shared by every council | built |
| The installation's own name and logo (self-hosting), also in the emails and on Zitadel's sign-in pages and emails | built |
| Self-hosting with Docker Compose: PostgreSQL with row-level security per council, Zitadel for accounts, Garage for files, Caddy in front | built |
| Private storage: one bucket per council, images only through presigned links | built |
| Accessibility (WCAG 2.1 AA and the AAA criteria that apply), with automated checks in CI and an accessibility statement page | built |
| CI: Go, web and Android checks (path-filtered on pull requests), vulnerability and image scanning (govulncheck, pnpm audit, Trivy), images published to ghcr.io, signed release APK on version tags | built |
| Short-lived access tokens (10 min, renewed silently), second factor required for administrators, least-privilege Zitadel bootstrap | built |
| Rate limit on sign-up and sign-in at the proxy (no captcha: Zitadel's login has none) | built |
| Activity log of every administrative change, for superadministrators (all) and each council's administrators (their own), append-only with a retention period, CSV download | built |
| API: sorting, filtering and pagination in the backend, with validated query params | built |
| Demo deployment, production kit (`deploy/`), OpenAPI spec | idea (see todo.md) |

## Ideas for later

Grouped by who gains most; none of these is started.

**Residents**

- **Waiting list:** join the queue of a full turn and get a notice (and a few minutes to take it) when someone cancels.
- **Find players:** say you are looking for people for a sport and level, and see others who are, building on public bookings.
- **Push notifications** (web push, and real push in the Android app instead of the background check) and **SMS** for people without email, through the council's provider.
- **Installable web app that works offline:** the manifest exists; a service worker would keep My bookings and the entry QR available without coverage at the door.

**Councils**

- **Recurring bookings for clubs and schools:** a season's slots booked in one go by staff, with their own limits, and priority windows for registered entities.
- **Fees and payment:** a price per turn or a pass, paid through the public administrations' payment gateway (Pasarela de Pagamentos), with exemptions.
- **Limits per sport across facilities:** e.g. at most two upcoming padel bookings across all the council's courts (today the council-wide cap counts every facility).
- **Kiosk screen at the entrance:** today's turns and who is in, on the reader's tablet.
- **Open data:** publish anonymised occupancy as open data (DCAT-AP, datos.gob.es / abertos.xunta.gal) on a schedule.

**Platform**

- **Cl@ve / Chave365** sign-in for residents (to confirm with AMTEGA/AGASOL).
