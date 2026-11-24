package bookings

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// A phone caller without an account books for their name and number with the
// rules of an account, counted by the number, and cancels by it.
// Run against a freshly created, disposable PostgreSQL 18 database with
// 00_extensions.sql and 01_tables.sql loaded:
// BOOKINGS_TEST_DATABASE_URL=postgres://… go test ./core/bookings -run PhoneCaller
func TestPhoneCallerBooksWithTheRulesOfAnAccount(t *testing.T) {
	var dsn = os.Getenv("BOOKINGS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires a disposable PostgreSQL fixture database")
	}
	var conn, err = sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var ctx = context.Background()
	var council, otherCouncil = uuid.New(), uuid.New()
	var pista, tenis, veciños, maiores, alleo = uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	var ana = uuid.New()
	var statements = []string{
		fmt.Sprintf(`INSERT INTO tenant (id, slug, name) VALUES ('%s', 'tel-%s', 'Concello de Lalín'), ('%s', 'tel-%s', 'Concello de Silleda')`, council, council.String()[:8], otherCouncil, otherCouncil.String()[:8]),
		fmt.Sprintf(`INSERT INTO tenant_booking_rules (tenant_id, max_active_bookings_council) VALUES ('%s', 2)`, council),
		fmt.Sprintf(`INSERT INTO facility (id, tenant_id, slug, name, slot_minutes, max_bookings_per_day) VALUES
			('%s', '%s', 'pista', 'Pista', 60, 1), ('%s', '%s', 'tenis', 'Tenis', 60, 0)`, pista, council, tenis, council),
		fmt.Sprintf(`INSERT INTO facility (id, tenant_id, slug, name, slot_minutes, residents_only) VALUES ('%s', '%s', 'vecinos', 'Só veciños', 60, TRUE)`, veciños, council),
		fmt.Sprintf(`INSERT INTO facility (id, tenant_id, slug, name, slot_minutes, min_age) VALUES ('%s', '%s', 'maiores', 'Maiores', 60, 18)`, maiores, council),
		fmt.Sprintf(`INSERT INTO facility (id, tenant_id, slug, name, slot_minutes) VALUES ('%s', '%s', 'alleo', 'Pista allea', 60)`, alleo, otherCouncil),
		fmt.Sprintf(`INSERT INTO app_user (id, idp_subject, email, email_verified, full_name, birth_date) VALUES ('%s', 'tel-%s', 'ana@example.com', TRUE, 'Ana', '1990-01-01')`, ana, ana),
	}
	for _, facility := range []uuid.UUID{pista, tenis, veciños, maiores, alleo} {
		for day := 1; day <= 7; day++ {
			statements = append(statements, fmt.Sprintf(`INSERT INTO opening_hours (facility_id, weekday, opens_at, closes_at) VALUES ('%s', %d, '08:00', '22:00')`, facility, day))
		}
	}
	for _, statement := range statements {
		if _, err = conn.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	var madrid, _ = time.LoadLocation("Europe/Madrid")
	var at = func(days, hour int) time.Time {
		var day = time.Now().In(madrid).AddDate(0, 0, days)
		return time.Date(day.Year(), day.Month(), day.Day(), hour, 0, 0, 0, madrid)
	}
	var service = &bookingsService{db: conn}
	var maria = Booker{Name: "María", Phone: fmt.Sprintf("+346%08d", rand.IntN(100_000_000))} // each run its own: the database is kept
	var xan = Booker{Name: "Xan", Phone: fmt.Sprintf("+347%08d", rand.IntN(100_000_000))}
	var book = func(who Booker, facility uuid.UUID, start time.Time) (string, string) {
		t.Helper()
		var made, problem = service.PhoneBooking(ctx, who, CreateBookingDto{FacilityId: facility, StartsAt: start, AcceptRules: true})
		if problem != nil {
			return "", problem.Message
		}
		return made.Id, ""
	}

	var first, refused = book(maria, pista, at(1, 10))
	if refused != "" {
		t.Fatalf("first booking refused: %s", refused)
	}
	var channel, name, phone string
	if err = conn.QueryRowContext(ctx, `SELECT channel, holder_name, holder_phone FROM booking WHERE id = $1`, first).Scan(&channel, &name, &phone); err != nil {
		t.Fatal(err)
	}
	if channel != "phone" || name != "María" || phone != maria.Phone {
		t.Errorf("stored as %s, %s, %s", channel, name, phone)
	}
	for _, c := range []struct {
		who      Booker
		facility uuid.UUID
		start    time.Time
		want     string
	}{
		{maria, pista, at(1, 12), "errors.daily_limit_reached"},   // the facility's limit, by her number
		{maria, tenis, at(1, 10), "errors.booking_overlaps"},      // in one place at a time
		{maria, tenis, at(1, 12), ""},                             // her second one at the council
		{maria, tenis, at(2, 12), "errors.council_limit_reached"}, // the council's cap of 2
		{maria, alleo, at(3, 12), "errors.other_council_booking"}, // one council at a time
		{xan, pista, at(1, 12), ""},                               // another number counts apart
		{xan, veciños, at(1, 15), "errors.phone_residents_only"},  // residents cannot be checked by phone
		{xan, maiores, at(1, 15), "errors.phone_age_restricted"},  // nor the age
		{Booker{Phone: "+34600000003"}, tenis, at(1, 18), "errors.phone_holder_required"},
	} {
		if _, refused = book(c.who, c.facility, c.start); refused != c.want {
			t.Errorf("%s at %s: refused %q, want %q", c.who.Name, c.start.Format(time.DateTime), refused, c.want)
		}
	}

	var theirs, problem = service.PhoneBookings(ctx, maria.Phone)
	if problem != nil || len(theirs) != 2 || theirs[0].Id != first || theirs[0].HolderName != "María" || !theirs[0].CanCancel {
		t.Fatalf("her bookings %+v %v", theirs, problem)
	}
	if problem = service.CancelPhoneBooking(ctx, xan.Phone, uuid.MustParse(first)); problem == nil || problem.Message != "errors.booking_not_found" {
		t.Errorf("cancelled from another number: %+v", problem)
	}
	if problem = service.CancelPhoneBooking(ctx, maria.Phone, uuid.MustParse(first)); problem != nil {
		t.Fatalf("cancel: %+v", problem)
	}
	var status, by string
	if err = conn.QueryRowContext(ctx, `SELECT status, cancelled_by FROM booking WHERE id = $1`, first).Scan(&status, &by); err != nil || status != "cancelled" || by != "user" {
		t.Errorf("after cancelling: %s by %s (%v)", status, by, err)
	}
	if theirs, _ = service.PhoneBookings(ctx, maria.Phone); len(theirs) != 1 {
		t.Errorf("%d bookings left, want 1", len(theirs))
	}

	// signed in on the phone: an account's booking, made by phone
	var made, accountProblem = service.PhoneBooking(ctx, Account(ana), CreateBookingDto{FacilityId: tenis, StartsAt: at(1, 16), AcceptRules: true})
	if accountProblem != nil {
		t.Fatalf("account by phone: %+v", accountProblem)
	}
	var user uuid.NullUUID
	var holder sql.NullString
	if err = conn.QueryRowContext(ctx, `SELECT channel, user_id, holder_name FROM booking WHERE id = $1`, made.Id).Scan(&channel, &user, &holder); err != nil {
		t.Fatal(err)
	}
	if channel != "phone" || user.UUID != ana || holder.Valid {
		t.Errorf("account by phone stored as %s, %v, %v", channel, user, holder)
	}
}
