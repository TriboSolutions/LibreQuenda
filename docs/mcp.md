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

Un cliente que xa ten o token de acceso da persoa (o asistente de voz da web, que fala con quen ten a
sesión iniciada) pode envialo en cada petición, `Authorization: Bearer <token>`, como a web cando chama á
API: se é válido, esa petición actúa como esa persoa e `sign_in` responde `signed_in`. Sen a cabeceira
todo segue igual; cun token caducado ou falso, as ferramentas que precisan conta din que non vale.

## Liña de teléfono

O servizo de voz atende chamadas de teléfono e usa este mesmo servidor. A API confía nel só cando envía
`X-Voice-Key: <VOICE_API_KEY>` (consulta [self-hosting.md](self-hosting.md)) e o número de quen chama,
`X-Caller-Phone` (E.164, p. ex. `+34600123456`; sen prefixo, España). Esa sesión MCP é entón a de quen chama:

- **Con conta:** `phone_send_code` busca a conta que ten ese número no perfil (se hai unha soa, co correo
  verificado) e envíalle por correo, no seu idioma, un código de 6 cifras que dura 10 minutos; responde se
  a atopou e unha pista do enderezo (`m***@gmail.com`). Quen chama dille o código ao asistente e
  `phone_verify_code` inicia a sesión desa conta ata que remata a chamada, en memoria como `sign_in`. Como
  moito un código por minuto e cinco por hora a cada número, e cinco intentos por código. A analítica non se
  abre a unha sesión por teléfono.
- **Sen conta:** `book_turn` con `holder_name` reserva ao nome de quen chama e ao seu número, coas mesmas
  normas ca unha conta (ventá de reserva, límites por día, semana, reservas próximas e do concello, un
  concello á vez, reservas que se solapan), contadas por número. As instalacións só para veciños ou con
  idade mínima piden entrar coa conta, porque por teléfono non se pode comprobar. `my_bookings` e
  `cancel_booking` traballan sobre as reservas dese número. Non hai correo ao que mandar a confirmación: o
  asistente le os datos (`confirmation: read_out`).

Estas reservas levan a canle `phone`, que o persoal ve no calendario do concello e que contan os
indicadores, as exportacións e as vistas de analítica. O nome e o número esquécense 30 días despois da
reserva, como os das reservas do persoal.

## Ferramentas

| Ferramenta | Conta | Que fai |
|---|---|---|
| `list_councils` | | Os concellos da instalación |
| `search_facilities` | | Instalacións coas súas quendas libres e partidas públicas nun día; por palabras, concello, deporte, hora ou distancia |
| `facility_details` | | Unha instalación nun día: quendas, normas de uso, prazo de cancelación, aforo, idade, só para veciños |
| `list_events` | | Os próximos eventos dos concellos |
| `sign_in` / `sign_out` | | Inicia ou pecha a sesión da persoa nesta sesión do axente |
| `my_bookings` | si | As súas próximas reservas (por teléfono sen conta, as do número de quen chama) |
| `book_turn` | si | Reserva unha quenda libre, unha vez que a persoa aceptou as normas de uso da instalación (por teléfono sen conta, co seu nome) |
| `cancel_booking` | si | Cancela unha das súas reservas, antes do seu prazo |
| `join_public_booking` | si | Únese a unha reserva pública ou a un evento con prazas libres |
| `phone_send_code` | | Só na liña de teléfono: envía por correo un código á conta que ten o número de quen chama |
| `phone_verify_code` | | Só na liña de teléfono: inicia a sesión desa conta co código |
| `analytics_views` | admin | As vistas de analítica de só lectura e as súas columnas |
| `analytics_query` | admin | Un único `SELECT` de só lectura sobre elas, para os concellos que xestiona a persoa (consulta [analytics.md](analytics.md)) |

As reservas feitas a través dun axente son coma calquera outra: correo de confirmación, recordatorios, e
aparecen na web e na app.
