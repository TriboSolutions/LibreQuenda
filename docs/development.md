# Development

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
backup/    Daily PostgreSQL backups into Garage, and their restore
zitadel/   One-off setup: creates the LibreQuenda project + OIDC app, writes config.json
docs/      Architecture specs, brand assets, generated OpenAPI spec
api/       Go core API
db/        PostgreSQL image: init schema, seed, sqlc queries
web/       React web app
```


## Documentation

- [Features](../features.md) and [todo](../todo.md)
- [Running a server](self-hosting.md), [check-in at the entrance](check-in.md), [activity log](audit-log.md), [proposal: a personal home](specs/feed.md)

- [Contest bases (v3, Galician)](https://www.agasol.gal/wp-content/uploads/2026/10/Bases_I_Certame_GOBTEC_2026_GL_v3.pdf), published by AGASOL
- API spec: generated into [`docs/openapi/`](openapi/)
