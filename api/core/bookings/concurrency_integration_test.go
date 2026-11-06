package bookings

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Many people confirm the same turn at the same instant: exactly one gets it, every
// other one is told it is taken (409 slot_booked), and one booking is stored.
// Run against a freshly created, disposable PostgreSQL 18 database with
// 00_extensions.sql and 01_tables.sql loaded:
// BOOKINGS_TEST_DATABASE_URL=postgres://… go test ./core/bookings -run SameTurnAtOnce
func TestSameTurnAtOnceGoesToOnePerson(t *testing.T) {
	var dsn = os.Getenv("BOOKINGS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires a disposable PostgreSQL fixture database")
	}
	var conn, err = sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetMaxOpenConns(60) // every attempt on a connection of its own, as separate API requests
	var ctx = context.Background()
	var tenant = uuid.New()
	var facility = uuid.New()
	var statements = []string{
		fmt.Sprintf(`INSERT INTO tenant (id, slug, name) VALUES ('%s', 'lalin', 'Concello de Lalín')`, tenant),
		fmt.Sprintf(`INSERT INTO facility (id, tenant_id, slug, name, slot_minutes) VALUES ('%s', '%s', 'pista', 'Pista', 60)`, facility, tenant),
	}
	for day := 1; day <= 7; day++ {
		statements = append(statements, fmt.Sprintf(`INSERT INTO opening_hours (facility_id, weekday, opens_at, closes_at) VALUES ('%s', %d, '08:00', '22:00')`, facility, day))
	}
	const people = 100
	var users = make([]uuid.UUID, people)
	for i := range users {
		users[i] = uuid.New()
		statements = append(statements, fmt.Sprintf(`INSERT INTO app_user (id, idp_subject, email, email_verified, full_name, birth_date) VALUES ('%s', 'cc-%d', 'p%d@example.com', TRUE, 'Persoa %d', '1990-01-01')`, users[i], i, i, i))
	}
	for _, statement := range statements {
		if _, err = conn.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	var madrid, _ = time.LoadLocation("Europe/Madrid")
	var tomorrow = time.Now().In(madrid).AddDate(0, 0, 1)
	var turn = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 10, 0, 0, 0, madrid)

	var service = &bookingsService{db: conn}
	var start = make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	var booked, taken int
	var other []string
	for _, user := range users {
		wg.Add(1)
		go func(user uuid.UUID) {
			defer wg.Done()
			<-start // all at once
			var _, problem = service.CreateBooking(ctx, user, CreateBookingDto{FacilityId: facility, StartsAt: turn, AcceptRules: true})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case problem == nil:
				booked++
			case problem.Code == http.StatusConflict && problem.Message == "errors.slot_booked":
				taken++
			default:
				other = append(other, fmt.Sprintf("%d %s", problem.Code, problem.Message))
			}
		}(user)
	}
	close(start)
	wg.Wait()
	var stored int
	_ = conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM booking WHERE facility_id = $1 AND status = 'confirmed'`, facility).Scan(&stored)
	if booked != 1 || taken != people-1 || len(other) > 0 || stored != 1 {
		t.Fatalf("booked %d, told it was taken %d, other answers %v, stored %d", booked, taken, other, stored)
	}
}
