package agent

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// connect: a client on the server, in memory (no database: only what refuses before reaching it).
func connect(t *testing.T) *mcp.ClientSession {
	t.Helper()
	var ctx = context.Background()
	var serverSide, clientSide = mcp.NewInMemoryTransports()
	if _, err := newServer(nil).Connect(ctx, serverSide, nil); err != nil {
		t.Fatal(err)
	}
	var client, err = mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(ctx, clientSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func TestTools(t *testing.T) {
	var client = connect(t)
	var list, err = client.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range list.Tools {
		names = append(names, tool.Name)
		if tool.Description == "" || tool.InputSchema == nil {
			t.Errorf("%s: no description or input schema", tool.Name)
		}
	}
	slices.Sort(names)
	var want = []string{"book_turn", "cancel_booking", "facility_details", "join_public_booking", "list_councils", "list_events", "my_bookings", "search_facilities", "sign_in", "sign_out"}
	if !slices.Equal(names, want) {
		t.Fatalf("tools = %v, want %v", names, want)
	}
}

// what is refused before any booking reaches the database
func TestRefusals(t *testing.T) {
	var client = connect(t)
	var cases = []struct {
		tool string
		args map[string]any
		want string
	}{
		{"book_turn", map[string]any{"facility_id": "0192f5a0-1c2d-7e3f-8a9b-0c1d2e3f4a5b", "starts_at": "2026-11-07T09:00:00+01:00", "accept_rules": false}, "usage_rules"},
		{"book_turn", map[string]any{"facility_id": "nope", "starts_at": "tomorrow", "accept_rules": true}, "RFC 3339"},
		{"book_turn", map[string]any{"facility_id": "0192f5a0-1c2d-7e3f-8a9b-0c1d2e3f4a5b", "starts_at": "2026-11-07T09:00:00+01:00", "accept_rules": true}, "not signed in"},
		{"my_bookings", map[string]any{}, "not signed in"},
		{"cancel_booking", map[string]any{"booking_id": "nope"}, "not a booking"},
		{"join_public_booking", map[string]any{"booking_id": "0192f5a0-1c2d-7e3f-8a9b-0c1d2e3f4a5b"}, "not signed in"},
		{"search_facilities", map[string]any{"from": "9am"}, "not valid"},
		{"sign_in", map[string]any{}, "session"}, // in memory there is no Mcp-Session-Id to keep a sign-in for
	}
	for _, c := range cases {
		var result, err = client.CallTool(context.Background(), &mcp.CallToolParams{Name: c.tool, Arguments: c.args})
		if err != nil {
			t.Fatalf("%s: %v", c.tool, err)
		}
		var text string
		if len(result.Content) > 0 {
			if content, ok := result.Content[0].(*mcp.TextContent); ok {
				text = content.Text
			}
		}
		if !result.IsError || !strings.Contains(text, c.want) {
			t.Errorf("%s %v: error %v %q, want one saying %q", c.tool, c.args, result.IsError, text, c.want)
		}
	}
}
