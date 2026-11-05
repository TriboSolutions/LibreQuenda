# LibreQuenda

Free-software platform for managing and booking **municipal sports facilities**, built for Galician councils (*concellos*).

Residents look up facilities, book, change or cancel their bookings, invite others or open a booking to people nearby, check in at the door (QR, printed QR, entry code or staff), report problems and get notifications and emails. Councils manage their facilities, opening hours, closures, usage rules and images, answer residents' reports, and follow live indicators they can export. Superadministrators manage the councils and their administrators, and an activity log records every administrative change. One installation can serve one council or many, on the web and in an Android app that each installation can build with its own name and logo.

LibreQuenda is Tribo Solutions' proposal for the [I Certame GOBTEC de Software Libre](https://www.agasol.gal/i-certame-gobtec/) (AGASOL, 2026).

## Requirements

- A Linux server with [Docker](https://docs.docker.com/engine/install/) (Compose v2) and `curl`, with 2 GB of RAM (the whole stack uses about 400 MB idle) and some disk for the database, images and backups.
- Two domain names pointing (DNS A/AAAA) to it: the web app and the sign-in, e.g. `reservas.concello.gal` and `auth.reservas.concello.gal`.
- Ports 80 and 443 open (HTTPS certificates come from Let's Encrypt), or another proxy in front that handles HTTPS.
- An SMTP account to send email (confirmations, reminders, sign-in codes).
- For your own Android app: nothing else, it is built in Docker too, but the build needs about 4 GB of RAM (on the server or any other machine).

## Install

On the server, in the directory that will hold the installation:

```sh
mkdir librequenda && cd librequenda
curl -fsSLO https://raw.githubusercontent.com/TriboSolutions/LibreQuenda/main/install.sh
bash install.sh
```

The installer asks for the domains, the first administrator and the SMTP account, generates every secret, writes `.env`, and starts LibreQuenda from the prebuilt images. At the end it can also build **your own Android app** (your name, logo and package, already pointing to your server) as an APK to install directly and an AAB for Google Play.

Then open `https://<your domain>/admin`, sign in as the administrator, and create your council.

**Keep a copy of `.env` (and `android/` if you built the app) somewhere safe.** `ZITADEL_MASTERKEY` cannot be recovered, backups cannot be restored without it, and app updates must be signed with the same key.

Run it again at any time to change the configuration (Enter keeps each current value), or:

```sh
docker compose pull && docker compose up -d --wait   # update LibreQuenda
bash install.sh app                                  # build your app again (e.g. after an update)
```

Everything else (configuration options, backups and restoring them, sign-in settings, installing by hand) is in [docs/self-hosting.md](docs/self-hosting.md). For development, see [docs/development.md](docs/development.md).

## License

Copyright (C) 2026 Tribo Solutions

This program is free software: you can redistribute it and/or modify it under the terms of the GNU Affero General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version. See [LICENSE](LICENSE).
