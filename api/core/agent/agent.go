// Package agent: LibreQuenda's MCP server (Model Context Protocol), at /api/mcp, so
// an AI agent (Claude, ChatGPT, Cursor…) can look up facilities and turns, and book,
// cancel or join for the person, who signs in on the installation's own page.
package agent

import (
	"database/sql"
	"net/http"
	"time"

	"librequenda/shared"

	"github.com/angelbarreiros/Penguin/router"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const instructions = `LibreQuenda books municipal sports facilities (Galician councils). Times are the councils' local time (Europe/Madrid); turns have fixed starts.

To book: search_facilities (or list_councils first) finds facilities with their free turns and public games for a day; facility_details gives one facility's turns, its usage rules and conditions. Before booking, show the person the usage rules and the cancellation deadline and get their agreement. Booking, cancelling, joining and my_bookings need the person signed in: call sign_in, show them the link and the code it returns, wait until they say they approved, then call sign_in again (or the tool you need). Nobody's password or tokens ever pass through you.

Council administrators can also analyse their data: analytics_views lists the read-only views, analytics_query runs one SELECT on them (their councils only).

Every rule of the web applies (booking window, per-person limits, residents-only facilities, minimum age, overlapping bookings); when one refuses, tell the person why in their own words.`

func newServer(db *sql.DB) *mcp.Server {
	var server = mcp.NewServer(&mcp.Implementation{Name: "librequenda", Title: "LibreQuenda", Version: shared.VERSION}, &mcp.ServerOptions{Instructions: instructions})
	var a = &agent{db: db, grants: &grants{by: map[string]*grant{}}}
	var readOnly = &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: new(false)}
	mcp.AddTool(server, &mcp.Tool{Name: "list_councils", Description: "The councils on this installation, with their slugs and contact.", Annotations: readOnly}, a.councils)
	mcp.AddTool(server, &mcp.Tool{Name: "search_facilities", Description: "Facilities with their free turns and public games to join for one day, narrowed by words, council, sport, time or distance.", Annotations: readOnly}, a.search)
	mcp.AddTool(server, &mcp.Tool{Name: "facility_details", Description: "One facility on one day: its free turns, public games, usage rules, cancellation deadline, capacity, minimum age and whether it is for residents only.", Annotations: readOnly}, a.details)
	mcp.AddTool(server, &mcp.Tool{Name: "list_events", Description: "The councils' upcoming events anyone can sign up for (join them with join_public_booking).", Annotations: readOnly}, a.events)
	mcp.AddTool(server, &mcp.Tool{Name: "sign_in", Description: "Signs the person in, for booking: returns a link and a code for them to approve on the installation's sign-in page; call it again once they did. Says who is signed in when they already are.", Annotations: &mcp.ToolAnnotations{OpenWorldHint: new(false)}}, a.signIn)
	mcp.AddTool(server, &mcp.Tool{Name: "sign_out", Description: "Signs the person out of this agent session.", Annotations: &mcp.ToolAnnotations{IdempotentHint: true, OpenWorldHint: new(false)}}, a.signOut)
	mcp.AddTool(server, &mcp.Tool{Name: "my_bookings", Description: "The signed-in person's upcoming bookings, their own and the ones they joined.", Annotations: readOnly}, a.myBookings)
	mcp.AddTool(server, &mcp.Tool{Name: "book_turn", Description: "Books a free turn for the signed-in person, after they accepted the facility's usage rules. They get the confirmation by email and see it on the web and the app.", Annotations: &mcp.ToolAnnotations{OpenWorldHint: new(false)}}, a.book)
	mcp.AddTool(server, &mcp.Tool{Name: "cancel_booking", Description: "Cancels one of the signed-in person's own bookings, while its cancellation deadline allows. Whoever joined it is told.", Annotations: &mcp.ToolAnnotations{DestructiveHint: new(true), OpenWorldHint: new(false)}}, a.cancel)
	mcp.AddTool(server, &mcp.Tool{Name: "analytics_views", Description: "For council administrators (signed in): the read-only analytics views (bookings, occupancy, cancellations, check-ins, ratings, issues… without personal data or free text) and their columns, to write analytics_query SQL.", Annotations: readOnly}, a.analyticsViews)
	mcp.AddTool(server, &mcp.Tool{Name: "analytics_query", Description: "For council administrators (signed in): runs one read-only SELECT on the analytics views, limited to the councils they run, and returns its columns and rows (at most 500). Aggregate: counts, rates and averages by facility, day, hour…", Annotations: readOnly}, a.analyticsQuery)
	mcp.AddTool(server, &mcp.Tool{Name: "join_public_booking", Description: "Joins someone's public booking or a council event that has spots left, for the signed-in person.", Annotations: &mcp.ToolAnnotations{OpenWorldHint: new(false)}}, a.join)
	return server
}

type agentRoutes struct {
	db     *sql.DB
	router *router.Router
}

func AgentRoutes(db *sql.DB, r *router.Router) *agentRoutes {
	return &agentRoutes{db: db, router: r}
}

// RegisterRoutes: /mcp (Streamable HTTP). Public: looking up needs no account, and
// booking signs in through the tools, never with a token sent by the client.
func (r *agentRoutes) RegisterRoutes() {
	var server = newServer(r.db)
	var handler = mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{
		JSONResponse:        true,          // plain JSON answers: nothing for a proxy to buffer
		SessionTimeout:      2 * time.Hour, // an idle session goes (and with it, its sign-in)
		MaxRequestBodyBytes: shared.MaxJSONBody,
	})
	for _, method := range []string{http.MethodPost, http.MethodGet, http.MethodDelete} {
		r.router.NewRoute(router.Route{Path: "/mcp", Method: router.HTTPMethod(method), Handler: handler.ServeHTTP})
	}
}
