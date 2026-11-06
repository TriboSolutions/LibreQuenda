# Reservar desde un axente de IA (MCP)

Cada instalación serve un servidor [MCP](https://modelcontextprotocol.io) en
`https://<your domain>/api/mcp` (Streamable HTTP). Con el, un axente de IA (Claude,
ChatGPT, Cursor ou calquera outro cliente MCP) atopa instalacións e quendas libres, e reserva,
cancela ou se une a partidas públicas en nome da persoa á que está a axudar, con todas as normas que aplica
a web: ventá de reserva, límites por persoa e do concello, instalacións só para veciños,
idade mínima, reservas que se solapan.

## Engadilo a un axente

- **Claude** (web e escritorio): *Settings → Connectors → Add custom connector*, co URL de arriba.
- **Claude Code**: `claude mcp add --transport http librequenda https://<your domain>/api/mcp`
- **Outros**: engade un servidor MCP remoto (Streamable HTTP) con ese URL. Non fan falla clave da API nin
  axustes de OAuth.

## Iniciar sesión

Consultar instalacións, quendas e eventos non require conta. Reservar si: o axente chama a
`sign_in`, que responde cunha ligazón á propia páxina de inicio de sesión da instalación e un código
curto. A persoa abre a ligazón en calquera dispositivo, comproba o código, inicia sesión como sempre e
aproba «LibreQuenda MCP». O axente nunca ve un contrasinal nin un token: quedan na
API, en memoria, só para esa sesión do axente, e desaparecen cando remata a sesión, despois de dúas
horas sen actividade, con `sign_out` ou cando se reinicia a API.

É o fluxo de dispositivo de OAuth ([RFC 8628](https://www.rfc-editor.org/rfc/rfc8628)) contra o
Zitadel da instalación, coa aplicación «LibreQuenda MCP» que crea `zitadel-setup`.

## Ferramentas

| Ferramenta | Conta | Que fai |
|---|---|---|
| `list_councils` | | Os concellos da instalación |
| `search_facilities` | | Instalacións coas súas quendas libres e partidas públicas nun día; por palabras, concello, deporte, hora ou distancia |
| `facility_details` | | Unha instalación nun día: quendas, normas de uso, prazo de cancelación, aforo, idade, só para veciños |
| `list_events` | | Os próximos eventos dos concellos |
| `sign_in` / `sign_out` | | Inicia ou pecha a sesión da persoa nesta sesión do axente |
| `my_bookings` | si | As súas próximas reservas |
| `book_turn` | si | Reserva unha quenda libre, unha vez que a persoa aceptou as normas de uso da instalación |
| `cancel_booking` | si | Cancela unha das súas reservas, antes do seu prazo |
| `join_public_booking` | si | Únese a unha reserva pública ou a un evento con prazas libres |
| `analytics_views` | admin | As vistas de analítica de só lectura e as súas columnas |
| `analytics_query` | admin | Un único `SELECT` de só lectura sobre elas, para os concellos que xestiona a persoa (consulta [analytics.md](analytics.md)) |

As reservas feitas a través dun axente son coma calquera outra: correo de confirmación, recordatorios, e
aparecen na web e na app.
