# Preguntarlles aos datos (asistente de analítica)

As persoas administradoras dun concello fan preguntas sobre os datos do seu concello coas súas propias palabras:
«que instalación tivo máis cancelacións
o mes pasado?», «que horas se reservan máis os sábados?». A analítica é sempre por concello:
unha persoa da superadministración, que configura o modelo, pregunta dentro dun concello, nunca sobre todos
eles á vez. Mentres está activado, unha burbulla «Fala cos datos» na esquina inferior dereita dos
**Indicadores** dun concello abre a conversa sobre os datos dese concello (`/datos`). Un modelo de linguaxe escribe SQL,
LibreQuenda execútao en modo só lectura e o modelo responde, en Markdown e, cando se pide unha gráfica ou axuda, con gráficas (barras, barras apiladas ou horizontais, liñas, sectores, mapas de calor: o modelo escolle o tipo e o SQL dos seus datos, e a gráfica amosa o que devolveu ese SQL, nunca números que escribise o modelo). O SQL non se lle amosa á persoa (cada gráfica
ten a súa táboa); a API rexistra cada consulta, por persoa.

Os axentes fan o mesmo a través do servidor MCP (`analytics_views`, `analytics_query`), para
os concellos que xestiona a persoa que iniciou sesión: consulta [mcp.md](mcp.md).

## Configuración

1. `ANALYTICS_DB_PASSWORD` en `.env` (xérao `install.sh`): o contrasinal do
   usuario de PostgreSQL de só lectura co que se executan as consultas.
2. En **Plataforma → Axustes (⚙) → Asistente de datos (IA)**, unha persoa da superadministración configura o
   modelo de linguaxe e actívao:
   - **Compatible con OpenAI**: OpenAI, Mistral, Groq, OpenRouter… ou un modelo no teu propio
     servidor (Ollama, vLLM, LM Studio), p. ex. `http://ollama:11434/v1`.
   - **Anthropic**: `https://api.anthropic.com`.

   A clave da API gárdase cifrada con `SECRETS_KEY` e non se volve amosar.

Os resultados das consultas que le o modelo están agregados e non levan nomes, correos, teléfonos
nin texto libre, pero cun provedor externo si saen da instalación. Un modelo no
teu propio servidor mantenos dentro dela.

## Que o fai seguro

As garantías son de PostgreSQL, non unha revisión do SQL que escribe o modelo:

- **Un usuario de só lectura** (`librequenda_analytics`, `db/init/08_analytics_role.sh`) no que
  todas as transaccións son de só lectura, cun límite de tempo, e que só pode facer `SELECT` sobre as
  **vistas de analítica** (`db/init/07_analytics_views.sql`): ningún permiso sobre ningunha táboa.
- **Unha vista por táboa, sen** datos persoais (nomes, correos, teléfonos, DNI, enderezos;
  só o ano de nacemento), segredos nin tokens (QR, NFC, calendario, claves dos lectores, SMTP), nin
  o texto libre que escribe a xente (descricións, motivos, notas, normas, títulos, contidos
  dos avisos), que podería levar instrucións para o modelo. As persoas son identificadores, para contar.
- **Cada concello ve só as súas filas**: a API indica os concellos con `SET LOCAL`, as
  vistas filtran por eles e o usuario de só lectura non os pode cambiar (retíraselle
  `set_config`). Unha persoa administradora dun concello nunca ve os datos doutro concello,
  diga o que diga o SQL. Nas vistas non hai nada que non pertenza a ningún concello.
- **Unha única sentenza**: o SQL execútase envolto nunha subconsulta (non pode ir despois unha segunda sentenza),
  con 500 filas como máximo e un límite de 8 segundos, nunha conexión que se pecha despois.
- **Só sobre os datos**: unha pregunta sobre calquera outra cousa (código, coñecemento xeral, redacción…)
  rexéitase cunha mensaxe fixa: o modelo ten unha ferramenta `off_topic` que remata a conversa,
  e a propia API rexeita unha resposta con código, ou unha longa que non respalde ningunha consulta.
- **Rexistrado**: cada pregunta queda no rexistro de actividade, e cada consulta que executou
  o asistente, no rexistro da API.

Como moito, un modelo enganado polo que le podería executar outras consultas de só lectura sobre as mesmas
vistas dos mesmos concellos: a persoa que pregunta podería velas de todos modos.
