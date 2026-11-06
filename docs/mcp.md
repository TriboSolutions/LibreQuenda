# Booking from an AI agent (MCP)

Every installation serves an [MCP](https://modelcontextprotocol.io) server at
`https://<your domain>/api/mcp` (Streamable HTTP). With it, an AI agent (Claude,
ChatGPT, Cursor, or any other MCP client) finds facilities and free turns, and books,
cancels or joins public games for the person it is helping, with every rule the web
applies: booking window, per-person and council limits, residents-only facilities,
minimum age, overlapping bookings.

## Adding it to an agent

- **Claude** (web and desktop): *Settings → Connectors → Add custom connector*, with the URL above.
- **Claude Code**: `claude mcp add --transport http librequenda https://<your domain>/api/mcp`
- **Others**: add a remote MCP server (Streamable HTTP) with that URL. No API key and no OAuth
  settings are needed.

## Signing in

Looking up facilities, turns and events needs no account. Booking does: the agent calls
`sign_in`, which answers with a link to the installation's own sign-in page and a short
code. The person opens the link on any device, checks the code, signs in as usual and
approves «LibreQuenda MCP». The agent never sees a password or a token: they stay in
the API, in memory, for that agent session only, and go when the session ends, after two
idle hours, with `sign_out`, or when the API restarts.

This is OAuth's device flow ([RFC 8628](https://www.rfc-editor.org/rfc/rfc8628)) against the
installation's Zitadel, with the «LibreQuenda MCP» application that `zitadel-setup` creates.

## Tools

| Tool | Account | What it does |
|---|---|---|
| `list_councils` | | The councils on the installation |
| `search_facilities` | | Facilities with their free turns and public games on a day; by words, council, sport, time or distance |
| `facility_details` | | One facility on one day: turns, usage rules, cancellation deadline, capacity, age, residents only |
| `list_events` | | The councils' upcoming events |
| `sign_in` / `sign_out` | | Sign the person in or out of this agent session |
| `my_bookings` | yes | Their upcoming bookings |
| `book_turn` | yes | Book a free turn, once they accepted the facility's usage rules |
| `cancel_booking` | yes | Cancel one of their bookings, before its deadline |
| `join_public_booking` | yes | Join a public booking or an event with spots left |

Bookings made through an agent are like any other: confirmation email, reminders, and
they show on the web and in the app.
