package counciladmin

import (
	"context"
	"database/sql"
	"librequenda/auth"
	"librequenda/core/me"
	"librequenda/model"
	"net/http"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Run against a freshly created, disposable PostgreSQL 18 database with
// 00_extensions.sql and 01_tables.sql loaded:
// RESIDENT_TEST_DATABASE_URL=postgres://… go test ./core/counciladmin -run ResidentRequest
func TestResidentRequestFlow(t *testing.T) {
	var dsn = os.Getenv("RESIDENT_TEST_DATABASE_URL")
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
	var admin = uuid.MustParse("00000000-0000-7000-8000-0000000000a1")
	var ana = uuid.MustParse("00000000-0000-7000-8000-0000000000b1")
	for _, statement := range []string{
		`INSERT INTO tenant (id, slug, name, contact_phone) VALUES ('` + tenant.String() + `', 'lalin', 'Concello de Lalín', '986 000 000')`,
		`INSERT INTO app_user (id, idp_subject, email, email_verified, full_name) VALUES ('` + admin.String() + `', 'rr-admin', 'admin@example.com', TRUE, 'Admin'), ('` + ana.String() + `', 'rr-ana', 'ana@example.com', TRUE, 'Ana')`,
		`INSERT INTO tenant_membership (tenant_id, user_id, role) VALUES ('` + tenant.String() + `', '` + admin.String() + `', 'admin')`,
	} {
		if _, err = conn.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	var people = me.GetMeService(conn)
	var staff = &counciladminService{db: conn}
	var session = auth.Session{Subject: "rr-ana"}
	var input = me.ResidentRequestInput{Dni: "12345678z", FullName: "Ana Pérez", PostalCode: "36500", Street: "Rúa Nova 3"}

	if _, problem := people.RequestResidency(ctx, session, "lalin", me.ResidentRequestInput{Dni: "12345678A", FullName: "Ana", PostalCode: "36500", Street: "Rúa"}); problem == nil || problem.Message != "errors.invalid_dni" {
		t.Fatalf("bad DNI: %+v", problem)
	}
	if _, problem := people.RequestResidency(ctx, session, "lalin", input); problem != nil {
		t.Fatalf("request: %+v", problem)
	}
	if _, problem := people.RequestResidency(ctx, session, "lalin", input); problem == nil || problem.Code != http.StatusConflict {
		t.Fatalf("second pending request: %+v", problem)
	}
	var count, _ = staff.CountResidentRequests(ctx, tenant)
	var list, problem = staff.ListResidentRequests(ctx, tenant, 1)
	if problem != nil || count.Pending != 1 || len(list.Requests) != 1 || list.Requests[0].Dni != "12345678Z" {
		t.Fatalf("admin list: %+v %+v %+v", count, list, problem)
	}
	var notices int
	_ = conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification WHERE user_id = $1 AND type = 'resident_requested' AND payload->>'name' = 'Ana Pérez'`, admin).Scan(&notices)
	if notices != 1 {
		t.Fatalf("admin notices: %d", notices)
	}

	// withdrawn, the week's request is still used
	if problem = people.WithdrawResidency(ctx, session, "lalin"); problem != nil {
		t.Fatalf("withdraw: %+v", problem)
	}
	if _, problem = people.RequestResidency(ctx, session, "lalin", input); problem == nil || problem.Code != http.StatusTooManyRequests {
		t.Fatalf("second request in a week: %+v", problem)
	}

	// a week later: verified, and the request (with the DNI) is gone
	if _, err = conn.ExecContext(ctx, `UPDATE app_user SET resident_requested_at = NOW() - INTERVAL '8 days' WHERE id = $1`, ana); err != nil {
		t.Fatal(err)
	}
	if _, problem = people.RequestResidency(ctx, session, "lalin", input); problem != nil {
		t.Fatalf("request a week later: %+v", problem)
	}
	list, _ = staff.ListResidentRequests(ctx, tenant, 1)
	var id = uuid.MustParse(list.Requests[0].Id)
	if _, problem = staff.DecideResidentRequest(ctx, tenant, admin, id, true); problem != nil {
		t.Fatalf("verify: %+v", problem)
	}
	var left, verified int
	_ = conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM resident_request`).Scan(&left)
	_ = conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM tenant_resident WHERE user_id = $1`, ana).Scan(&verified)
	if left != 0 || verified != 1 {
		t.Fatalf("after verifying: %d requests, %d verified", left, verified)
	}
	if _, problem = staff.DecideResidentRequest(ctx, tenant, admin, id, false); problem == nil || problem.Code != http.StatusNotFound {
		t.Fatalf("deciding twice: %+v", problem)
	}
	if _, problem = people.RequestResidency(ctx, session, "lalin", input); problem == nil || problem.Message != "errors.resident_already_verified_self" {
		t.Fatalf("request once verified: %+v", problem)
	}
	// and the verification can be taken back
	if problem = staff.RemoveResident(ctx, tenant, ana); problem != nil {
		t.Fatalf("unverify: %+v", problem)
	}
	_ = conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification WHERE user_id = $1 AND type = 'resident_verified'`, ana).Scan(&notices)
	if notices != 1 {
		t.Fatalf("verified notices: %d", notices)
	}

	// administrators of the council, and superadministrators, are verified by their role
	var queries = model.New(conn)
	if ok, _ := queries.IsTenantResident(ctx, model.IsTenantResidentParams{TenantID: tenant, UserID: admin}); !ok {
		t.Fatal("the council's administrator is not taken as a resident")
	}
	if _, problem = people.RequestResidency(ctx, auth.Session{Subject: "rr-admin"}, "lalin", input); problem == nil || problem.Message != "errors.resident_already_verified_self" {
		t.Fatalf("administrator's request: %+v", problem)
	}
	if ok, _ := queries.IsTenantResident(ctx, model.IsTenantResidentParams{TenantID: tenant, UserID: ana}); ok {
		t.Fatal("unverified Ana taken as a resident")
	}
	if _, err = conn.ExecContext(ctx, `INSERT INTO platform_admin (user_id) VALUES ($1)`, ana); err != nil {
		t.Fatal(err)
	}
	var councils, _ = queries.ListResidentCouncilsByUser(ctx, ana)
	if ok, _ := queries.IsTenantResident(ctx, model.IsTenantResidentParams{TenantID: tenant, UserID: ana}); !ok || len(councils) != 1 || councils[0].Slug != "lalin" {
		t.Fatalf("superadministrator not taken as a resident: %+v", councils)
	}
}
