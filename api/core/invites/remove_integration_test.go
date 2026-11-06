package invites

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Run against a freshly created, disposable PostgreSQL 18 database with
// 00_extensions.sql and 01_tables.sql loaded:
// INVITES_TEST_DATABASE_URL=postgres://… go test ./core/invites -run RemovedParticipant
func TestRemovedParticipantIsToldAndKeptOut(t *testing.T) {
	var dsn = os.Getenv("INVITES_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires a disposable PostgreSQL fixture database")
	}
	var conn, err = sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var ctx = context.Background()
	var tenant = uuid.MustParse("00000000-0000-7000-8000-000000000001")
	var facility = uuid.MustParse("00000000-0000-7000-8000-0000000000f1")
	var booking = uuid.MustParse("00000000-0000-7000-8000-0000000000b0")
	var ana = uuid.MustParse("00000000-0000-7000-8000-0000000000a1")
	var brais = uuid.MustParse("00000000-0000-7000-8000-0000000000a2")
	for _, statement := range []string{
		`INSERT INTO tenant (id, slug, name) VALUES ('` + tenant.String() + `', 'lalin', 'Concello de Lalín')`,
		`INSERT INTO facility (id, tenant_id, slug, name, capacity) VALUES ('` + facility.String() + `', '` + tenant.String() + `', 'pista', 'Pista', 4)`,
		`INSERT INTO app_user (id, idp_subject, email, email_verified, full_name, birth_date) VALUES
			('` + ana.String() + `', 'rp-ana', 'ana@example.com', TRUE, 'Ana Pérez', '1990-01-01'),
			('` + brais.String() + `', 'rp-brais', 'brais@example.com', TRUE, 'Brais Souto', '1990-01-01')`,
		`INSERT INTO booking (id, facility_id, user_id, starts_at, ends_at, kind) VALUES ('` + booking.String() + `', '` + facility.String() + `', '` + ana.String() + `', NOW() + INTERVAL '1 day', NOW() + INTERVAL '1 day 1 hour', 'public')`,
	} {
		if _, err = conn.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	var service = &invitesService{db: conn}
	if problem := service.JoinPublic(ctx, brais, booking); problem != nil {
		t.Fatalf("join: %+v", problem)
	}
	var participant uuid.UUID
	if err = conn.QueryRowContext(ctx, `SELECT id FROM booking_participant WHERE user_id = $1`, brais).Scan(&participant); err != nil {
		t.Fatal(err)
	}
	// only the owner takes someone out
	if problem := service.RemoveParticipant(ctx, brais, booking, participant); problem == nil || problem.Code != http.StatusNotFound {
		t.Fatalf("removed by someone else: %+v", problem)
	}
	if problem := service.RemoveParticipant(ctx, ana, booking, participant); problem != nil {
		t.Fatalf("remove: %+v", problem)
	}
	var told int
	_ = conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification WHERE user_id = $1 AND type = 'participant_removed' AND payload->>'name' = 'Ana' AND payload->>'facility_name' = 'Pista'`, brais).Scan(&told)
	if told != 1 {
		t.Fatalf("notices to who was taken out: %d", told)
	}
	if problem := service.JoinPublic(ctx, brais, booking); problem == nil || problem.Message != "errors.invite_removed" {
		t.Fatalf("joining again: %+v", problem)
	}
}
