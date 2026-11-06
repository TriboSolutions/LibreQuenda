# Asking the data (analytics assistant)

Council administrators ask questions about their council's data in their own words:
«which facility had the most cancellations
last month?», «which hours are booked most on Saturdays?». Analytics are always by council:
a superadministrator, who sets the model, asks inside one council, never about all of
them at once. While it is switched on, a «Fala cos datos» bubble in the bottom right corner of a
council's **Indicators** opens the conversation on that council's data (`/datos`). A language model writes SQL,
LibreQuenda runs it read-only and the model answers, in Markdown and, when a chart is asked for or helps, with charts (bars, stacked or horizontal bars, lines, pies, heatmaps: the model picks the kind and the SQL of its data, and the chart shows what that SQL returned, never numbers the model wrote). The SQL is not shown to the person (each chart
has its table); the API logs every query, per person.

Agents do the same through the MCP server (`analytics_views`, `analytics_query`), for
the councils the signed-in person runs: see [mcp.md](mcp.md).

## Setting it up

1. `ANALYTICS_DB_PASSWORD` in `.env` (`install.sh` generates it): the password of the
   read-only PostgreSQL user the queries run as.
2. In **Platform → Settings (⚙) → Data assistant (AI)**, a superadministrator sets the
   language model and switches it on:
   - **OpenAI-compatible**: OpenAI, Mistral, Groq, OpenRouter… or a model on your own
     server (Ollama, vLLM, LM Studio), e.g. `http://ollama:11434/v1`.
   - **Anthropic**: `https://api.anthropic.com`.

   The API key is stored encrypted with `SECRETS_KEY` and never shown again.

The query results the model reads are aggregated and carry no names, emails, phones
or free text, but with an outside provider they do leave the installation. A model on
your own server keeps them in it.

## What keeps it safe

The guarantees are PostgreSQL's, not a check of the SQL the model writes:

- **A read-only user** (`librequenda_analytics`, `db/init/08_analytics_role.sh`) whose
  every transaction is read-only, with a time limit, and who can only `SELECT` from the
  **analytics views** (`db/init/07_analytics_views.sql`): no right on any table.
- **One view per table, without** personal data (names, emails, phones, DNI, addresses;
  only the year of birth), secrets or tokens (QR, NFC, calendar, reader keys, SMTP), or
  the free text people write (descriptions, reasons, notes, rules, titles, notification
  contents), which could carry instructions to the model. People are ids, to count.
- **Each council sees only its rows**: the API names the councils with `SET LOCAL`, the
  views filter by them, and the read-only user cannot change them (`set_config` is
  revoked from it). A council's administrator never sees another council's data,
  whatever the SQL says. Nothing that belongs to no council is in the views.
- **One statement**: the SQL runs wrapped as a subquery (no second statement can follow),
  with at most 500 rows and an 8-second limit, on a connection that is closed afterwards.
- **Recorded**: every question is in the activity log, and every query the assistant
  ran in the API's log.

At most, a model misled by what it reads could run other read-only queries on the same
views of the same councils: the person asking could see those anyway.
