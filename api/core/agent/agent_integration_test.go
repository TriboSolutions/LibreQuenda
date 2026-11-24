package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"regexp"
	"testing"
	"time"

	"librequenda/mailer"

	"github.com/angelbarreiros/Penguin/router/helpers"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The web's voice calls (a bearer token) and the phone line (a code by email, or a
// name and the caller's number), end to end over Streamable HTTP.
// Run against a freshly created, disposable PostgreSQL 18 database with
// 00_extensions.sql and 01_tables.sql loaded:
// AGENT_TEST_DATABASE_URL=postgres://… go test ./core/agent -run VoiceLine
func TestVoiceLine(t *testing.T) {
	var dsn = os.Getenv("AGENT_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires a disposable PostgreSQL fixture database")
	}
	helpers.GetI18nInstance(helpers.SetDefaultLocale("gl"), helpers.SetDirectory("../../i18n"))
	var conn, err = sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var ctx = context.Background()
	var council, pista, ana = uuid.New(), uuid.New(), uuid.New()
	var anaSubject = "voice-" + ana.String()
	var number = func() string { return fmt.Sprintf("6%08d", rand.IntN(100_000_000)) } // each run its own: the database is kept
	var anaNumber = number()
	var statements = []string{
		fmt.Sprintf(`INSERT INTO tenant (id, slug, name) VALUES ('%s', 'voz-%s', 'Concello de Lalín')`, council, council.String()[:8]),
		fmt.Sprintf(`INSERT INTO facility (id, tenant_id, slug, name, slot_minutes, max_bookings_per_day) VALUES ('%s', '%s', 'pista', 'Pista', 60, 1)`, pista, council),
		fmt.Sprintf(`INSERT INTO app_user (id, idp_subject, email, email_verified, full_name, birth_date, phone, preferred_language)
			VALUES ('%s', '%s', 'ana@example.com', TRUE, 'Ana Pérez', '1990-01-01', '%s %s %s', 'es')`, ana, anaSubject, anaNumber[:3], anaNumber[3:6], anaNumber[6:]),
	}
	for day := 1; day <= 7; day++ {
		statements = append(statements, fmt.Sprintf(`INSERT INTO opening_hours (facility_id, weekday, opens_at, closes_at) VALUES ('%s', %d, '08:00', '22:00')`, pista, day))
	}
	for _, statement := range statements {
		if _, err = conn.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	var madrid, _ = time.LoadLocation("Europe/Madrid")
	var at = func(days, hour int) string {
		var day = time.Now().In(madrid).AddDate(0, 0, days)
		return time.Date(day.Year(), day.Month(), day.Day(), hour, 0, 0, 0, madrid).Format(time.RFC3339)
	}
	var sent []mailer.Message
	var a = testAgent(conn, &sent)

	t.Run("bearer", func(t *testing.T) {
		var web = overHTTP(t, a, map[string]string{"Authorization": "Bearer token-" + anaSubject})
		var in SignInOut
		call(t, web, "sign_in", nil, &in)
		if in.Status != "signed_in" || in.Email != "ana@example.com" {
			t.Errorf("sign_in with a bearer token: %+v", in)
		}
		var theirs MyBookingsOut
		call(t, web, "my_bookings", nil, &theirs)
		refused(t, overHTTP(t, a, nil), "my_bookings", nil, "not signed in") // without it, nobody
	})

	var caller = map[string]string{"X-Voice-Key": "voice-secret", "X-Caller-Phone": "+34 " + anaNumber}
	t.Run("code by email", func(t *testing.T) {
		var phone = overHTTP(t, a, caller)
		var out SendCodeOut
		call(t, phone, "phone_send_code", nil, &out)
		if out.Status != "sent" || !out.Found || out.EmailHint != "a***@example.com" || len(sent) != 1 || sent[0].To != "ana@example.com" {
			t.Fatalf("phone_send_code: %+v, sent %d", out, len(sent))
		}
		var code = regexp.MustCompile(`\b\d{6}\b`).FindString(sent[0].Text)
		if sent[0].Subject != "Tu código para LibreQuenda: "+code {
			t.Errorf("in Ana's language, with the code: %q", sent[0].Subject)
		}
		refused(t, phone, "phone_send_code", nil, "moments ago")
		refused(t, phone, "phone_verify_code", map[string]any{"code": "000000"}, "tries left")
		var in SignInOut
		call(t, phone, "phone_verify_code", map[string]any{"code": code}, &in)
		if in.Status != "signed_in" || in.Name != "Ana Pérez" {
			t.Errorf("phone_verify_code: %+v", in)
		}
		call(t, phone, "sign_in", nil, &in)
		if in.Status != "signed_in" {
			t.Errorf("sign_in after the code: %+v", in)
		}
		refused(t, phone, "phone_verify_code", map[string]any{"code": code}, "no code is waiting") // used once
		refused(t, phone, "analytics_views", nil, "not open to a sign-in by phone")
		var made Booking
		call(t, phone, "book_turn", map[string]any{"facility_id": pista.String(), "starts_at": at(1, 10), "accept_rules": true}, &made)
		var channel string
		var user uuid.NullUUID
		if err := conn.QueryRowContext(ctx, `SELECT channel, user_id FROM booking WHERE id = $1`, made.BookingId).Scan(&channel, &user); err != nil || channel != "phone" || user.UUID != ana || made.Confirmation != "email" {
			t.Errorf("booked by phone for Ana: %s %v %+v (%v)", channel, user, made, err)
		}
		// a new call (another MCP session) starts signed out
		refused(t, overHTTP(t, a, caller), "join_public_booking", map[string]any{"booking_id": made.BookingId}, "phone_send_code")
		var nobody SendCodeOut
		call(t, overHTTP(t, a, map[string]string{"X-Voice-Key": "voice-secret", "X-Caller-Phone": number()}), "phone_send_code", nil, &nobody)
		if nobody.Found || nobody.Status != "no_account" {
			t.Errorf("a number nobody has: %+v", nobody)
		}
	})

	t.Run("name and number", func(t *testing.T) {
		var phone = overHTTP(t, a, map[string]string{"X-Voice-Key": "voice-secret", "X-Caller-Phone": number()})
		var made Booking
		call(t, phone, "book_turn", map[string]any{"facility_id": pista.String(), "starts_at": at(1, 12), "accept_rules": true, "holder_name": " Xan  Souto "}, &made)
		if made.Confirmation != "read_out" || made.HolderName != "Xan Souto" || made.Facility != "Pista" || made.Council != "Concello de Lalín" || !made.CanCancel {
			t.Errorf("booked for a name: %+v", made)
		}
		refused(t, phone, "book_turn", map[string]any{"facility_id": pista.String(), "starts_at": at(1, 14), "accept_rules": true, "holder_name": "Xan"}, "per day")
		var theirs MyBookingsOut
		call(t, phone, "my_bookings", nil, &theirs)
		if len(theirs.Upcoming) != 1 || theirs.Upcoming[0].BookingId != made.BookingId || theirs.Upcoming[0].HolderName != "Xan Souto" {
			t.Fatalf("their bookings: %+v", theirs)
		}
		refused(t, overHTTP(t, a, caller), "cancel_booking", map[string]any{"booking_id": made.BookingId}, "could not find that booking") // not Ana's
		var done Done
		call(t, phone, "cancel_booking", map[string]any{"booking_id": made.BookingId}, &done)
		call(t, phone, "my_bookings", nil, &theirs)
		if !done.Done || len(theirs.Upcoming) != 0 {
			t.Errorf("after cancelling: %v, %+v", done.Done, theirs)
		}
	})
}

// call: the tool's structured answer into out, failing on an error.
func call(t *testing.T, client *mcp.ClientSession, tool string, args map[string]any, out any) {
	t.Helper()
	var result, err = client.CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", tool, err)
	}
	if result.IsError {
		t.Fatalf("%s %v: %s", tool, args, resultText(result))
	}
	var raw, _ = json.Marshal(result.StructuredContent)
	if err = json.Unmarshal(raw, out); err != nil {
		t.Fatalf("%s: %v", tool, err)
	}
}
