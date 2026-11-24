package agent

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"librequenda/auth"
	"librequenda/mailer"

	"github.com/golang-jwt/jwt/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// connect: a client on the server, in memory (no database: only what refuses before reaching it).
func connect(t *testing.T) *mcp.ClientSession {
	t.Helper()
	var ctx = context.Background()
	var serverSide, clientSide = mcp.NewInMemoryTransports()
	if _, err := newServer(newAgent(nil)).Connect(ctx, serverSide, nil); err != nil {
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
	var want = []string{"analytics_query", "analytics_views", "book_turn", "cancel_booking", "facility_details", "join_public_booking", "list_councils", "list_events", "my_bookings", "phone_send_code", "phone_verify_code", "search_facilities", "sign_in", "sign_out"}
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
		{"analytics_query", map[string]any{"sql": "SELECT 1"}, "not signed in"},
		{"sign_in", map[string]any{}, "session"}, // in memory there is no Mcp-Session-Id to keep a sign-in for
		{"phone_send_code", map[string]any{}, "only for the phone line"},
		{"phone_verify_code", map[string]any{"code": "123456"}, "only for the phone line"},
	}
	for _, c := range cases {
		refused(t, client, c.tool, c.args, c.want)
	}
}

// refused: the tool answers an error saying want.
func refused(t *testing.T, client *mcp.ClientSession, tool string, args map[string]any, want string) {
	t.Helper()
	var result, err = client.CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", tool, err)
	}
	if text := resultText(result); !result.IsError || !strings.Contains(text, want) {
		t.Errorf("%s %v: error %v %q, want one saying %q", tool, args, result.IsError, text, want)
	}
}

func resultText(result *mcp.CallToolResult) string {
	if len(result.Content) > 0 {
		if content, ok := result.Content[0].(*mcp.TextContent); ok {
			return content.Text
		}
	}
	return ""
}

// overHTTP: a client on the server through Streamable HTTP, as agents and the voice
// service reach it, sending these headers with every request.
func overHTTP(t *testing.T, a *agent, headers map[string]string) *mcp.ClientSession {
	t.Helper()
	var server = newServer(a)
	var site = httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{JSONResponse: true}))
	t.Cleanup(site.Close)
	var transport = &mcp.StreamableClientTransport{Endpoint: site.URL, HTTPClient: &http.Client{Transport: withHeaders(headers)}, DisableStandaloneSSE: true}
	var client, err = mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

type withHeaders map[string]string

func (h withHeaders) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	for name, value := range h {
		req.Header.Set(name, value)
	}
	return http.DefaultTransport.RoundTrip(req)
}

// testAgent: tokens are "token-<subject>", and phone codes are kept instead of emailed.
func testAgent(db *sql.DB, sent *[]mailer.Message) *agent {
	var a = newAgent(db)
	a.voiceKey = "voice-secret"
	a.verify = func(token string) (*jwt.RegisteredClaims, error) {
		var subject, ok = strings.CutPrefix(token, "token-")
		if !ok {
			return nil, auth.ErrInvalidToken
		}
		return &jwt.RegisteredClaims{Subject: subject}, nil
	}
	a.send = func(_ context.Context, message mailer.Message) error {
		*sent = append(*sent, message)
		return nil
	}
	return a
}

// Who a request acts for, from its headers, before anything reaches the database.
func TestHeaders(t *testing.T) {
	var sent []mailer.Message
	var a = testAgent(nil, &sent)
	var phone = map[string]string{"X-Voice-Key": "voice-secret", "X-Caller-Phone": "+34600000000"}
	var cases = []struct {
		headers map[string]string
		tool    string
		args    map[string]any
		want    string
	}{
		{map[string]string{"Authorization": "Bearer forged"}, "my_bookings", nil, "access token"},
		{map[string]string{"Authorization": "Bearer forged"}, "sign_in", nil, "access token"},
		{map[string]string{"X-Voice-Key": "wrong", "X-Caller-Phone": "+34600000000"}, "phone_send_code", nil, "only for the phone line"},
		{map[string]string{"X-Voice-Key": "voice-secret", "X-Caller-Phone": "not a number"}, "phone_send_code", nil, "only for the phone line"},
		{map[string]string{"X-Voice-Key": "wrong", "X-Caller-Phone": "+34600000000"}, "my_bookings", nil, "call sign_in"},
		{phone, "phone_verify_code", map[string]any{"code": "123456"}, "no code is waiting"},
		{phone, "sign_in", nil, "phone_send_code"},
		{phone, "join_public_booking", map[string]any{"booking_id": "0192f5a0-1c2d-7e3f-8a9b-0c1d2e3f4a5b"}, "phone_send_code"},
		{phone, "book_turn", map[string]any{"facility_id": "0192f5a0-1c2d-7e3f-8a9b-0c1d2e3f4a5b", "starts_at": "2026-11-07T09:00:00+01:00", "accept_rules": true}, "holder_name"},
		{phone, "book_turn", map[string]any{"facility_id": "0192f5a0-1c2d-7e3f-8a9b-0c1d2e3f4a5b", "starts_at": "2026-11-07T09:00:00+01:00", "accept_rules": true, "holder_name": "Ana", "public": true}, "public booking needs an account"},
	}
	// a web call without signing in, when the installation lets it book with a name and a number
	var guest = map[string]string{"X-Voice-Key": "voice-secret", "X-Web-Guest": "1"}
	var turn = map[string]any{"facility_id": "0192f5a0-1c2d-7e3f-8a9b-0c1d2e3f4a5b", "starts_at": "2026-11-07T09:00:00+01:00", "accept_rules": true, "holder_name": "Ana"}
	cases = append(cases, []struct {
		headers map[string]string
		tool    string
		args    map[string]any
		want    string
	}{
		{guest, "book_turn", turn, "ask the person for a phone number"},
		{guest, "my_bookings", nil, "cannot be listed or cancelled"},
		{guest, "cancel_booking", map[string]any{"booking_id": "0192f5a0-1c2d-7e3f-8a9b-0c1d2e3f4a5b"}, "cannot be listed or cancelled"},
		{map[string]string{"X-Voice-Key": "wrong", "X-Web-Guest": "1"}, "book_turn", turn, "call sign_in"},
		{map[string]string{"X-Web-Guest": "1"}, "book_turn", turn, "call sign_in"},
	}...)
	for _, c := range cases {
		refused(t, overHTTP(t, a, c.headers), c.tool, c.args, c.want)
	}
	// without a voice line, its headers are nobody's
	a.voiceKey = ""
	refused(t, overHTTP(t, a, map[string]string{"X-Voice-Key": "", "X-Caller-Phone": "+34600000000"}), "phone_send_code", nil, "only for the phone line")
}

func TestPhoneCodes(t *testing.T) {
	var codes = &phoneCodes{by: map[string]*phoneCode{}}
	var now = time.Now()
	if _, ok := codes.allow("+34600000000", now); !ok {
		t.Fatal("the first code was refused")
	}
	if _, ok := codes.allow("+34600000000", now.Add(30*time.Second)); ok {
		t.Error("a second code within a minute")
	}
	if _, ok := codes.allow("+34600000001", now.Add(30*time.Second)); !ok {
		t.Error("another number waits for this one")
	}
	for i := 1; i < codesPerHour; i++ {
		if _, ok := codes.allow("+34600000000", now.Add(time.Duration(i)*2*time.Minute)); !ok {
			t.Fatalf("code %d refused", i+1)
		}
	}
	if _, ok := codes.allow("+34600000000", now.Add(20*time.Minute)); ok {
		t.Error("more than codesPerHour in an hour")
	}
	if _, ok := codes.allow("+34600000000", now.Add(61*time.Minute)); !ok {
		t.Error("an hour later it is still refused")
	}
	for email, want := range map[string]string{"maria@gmail.com": "m***@gmail.com", "Ánxela@example.gal": "Á***@example.gal", "nope": "***"} {
		if got := emailHint(email); got != want {
			t.Errorf("emailHint(%q) = %q, want %q", email, got, want)
		}
	}
	if code, err := newCode(); err != nil || len(code) != codeDigits {
		t.Errorf("newCode() = %q, %v", code, err)
	}
}
