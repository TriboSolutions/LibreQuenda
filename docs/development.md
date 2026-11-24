# Desenvolvemento

## Tecnoloxías

| Parte | Tecnoloxía |
|---|---|
| API principal | Go, [Penguin](https://github.com/angelbarreiros/Penguin) (enrutador HTTP, planificador, rexistro), [sqlc](https://sqlc.dev) |
| Base de datos | PostgreSQL |
| Almacenamento de imaxes | [Garage](https://garagehq.deuxfleurs.fr) (autoaloxado, compatible con S3) |
| Mapas | [Leaflet](https://leafletjs.com) + [OpenStreetMap](https://www.openstreetmap.org) |
| Identidade | [Zitadel](https://zitadel.com) (OIDC) |
| Web | React, Vite, pnpm |
| Liña de voz | Python, [Pipecat](https://github.com/pipecat-ai/pipecat), modelos do [Proxecto Nós](https://nos.gal/gl/proxecto-nos), uv |
| Proxy | [Caddy](https://caddyserver.com) (punto de entrada único, HTTPS automático) |
| Despregamento | Docker Compose |


## Estrutura do repositorio

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
voice/     Voice line: Pipecat, Galician speech models, the MCP tools
```


## Documentación

- [Funcións](../features.md) e [pendentes](../todo.md)
- [Manter un servidor](self-hosting.md), [rexistrar a entrada na instalación](check-in.md), [rexistro de actividade](audit-log.md), [proposta: un inicio persoal](specs/feed.md)

- [Bases do certame (v3, en galego)](https://www.agasol.gal/wp-content/uploads/2026/10/Bases_I_Certame_GOBTEC_2026_GL_v3.pdf), publicadas por AGASOL
- Especificación da API: [`docs/openapi/openapi.yaml`](openapi/openapi.yaml) (OpenAPI 3.1); axentes: [servidor MCP](mcp.md), [liña de voz](voice.md); [preguntarlles aos datos](analytics.md)
