# Next: a personal home (the feed)

*Proposal, not built yet.* Today `/` is the same for everyone: every facility as a list beside a map, with filters. That is the right page for a visitor, but a resident who books the same court every Tuesday should not have to search for it every week. The idea:

- **Not signed in:** `/` stays as it is (list, map and filters).
- **Signed in:** `/` becomes a feed made from that person's own use. The search (list and map) moves to its own address, for example `/buscar`, one tap away from the feed and from the top bar.

**What the feed would show, most urgent first** (each card with its action right there):

1. **Now and next:** today's or the next booking, with the entry QR, directions, "invite" when there are spots left, and cancel while it still can be.
2. **Things waiting for them:** invitations to join a booking, and closures or cancellations that touch their bookings (what the bell already knows).
3. **Book your usual again:** the facility, weekday and hour they booked most in the last weeks ("Pádel, Tuesdays at 19:00"), when it is free this week, as one-tap booking. It says why it shows it.
4. **Their facilities today:** the facilities they book most (or mark as favourite), with today's free turns.
5. **Free soon near them:** facilities with free turns in the next hours that share a sport tag with what they usually book, in their council or near their location if they allow it.
6. **Quiet times:** the hours their usual facility tends to be quiet. The logic existed as the facility page's "Suggestions" (removed; see `api/core/tenants/suggestions.go` in git history) and can be reused here.
7. **New for them:** facilities or sports recently added in their council.

**Someone with no history yet** gets a short start instead: choose their council and the sports they play (from the tag catalogue). That is saved as their preferences and fills cards 4 and 5 until they have bookings.

**How it could be built:**

- `GET /me/feed` answers a list of typed cards (`next_booking`, `invitation`, `usual_turn`, `favourite_today`, `free_soon`, `quiet_hours`, `new_facility`), each with its data and the reason it is there. The web draws one component per card type. New card types can be added without changing the others.
- Plain rules ordered by urgency, not machine learning: easy to explain to a resident or a council ("because you book padel on Tuesdays"), and easy to test.
- New data: the person's preferences (council, sport tags) and optional favourite facilities, for example `user_preference` and `user_favourite_facility` tables. Everything else already exists (bookings, invitations, notifications, tags, opening hours).

**Privacy:** the feed only uses what LibreQuenda already keeps for the booking itself (the person's own bookings and the preferences they choose). It does not track pages viewed or clicks, does not share anything between residents or with the council, and the location is only used in the browser when they allow it, never stored. `/perfil` would say what the feed uses and let them reset their preferences.

**Open questions:** which cards councils care most about for the GOBTEC demo, whether favourites are explicit (a star) or inferred from bookings, and whether the feed should also arrive by email (a weekly "your usual turn is free").
