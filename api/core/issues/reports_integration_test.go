package issues

import (
	"context"
	"database/sql"
	"fmt"
	"librequenda/auth"
	"librequenda/core/issue"
	"librequenda/model"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Run against a freshly created, disposable PostgreSQL 18 database with
// 00_extensions.sql, 01_tables.sql and testdata/reports.sql loaded.
func TestReportLimitAndAdminNotifications(t *testing.T) {
	var dsn = os.Getenv("ISSUE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires a disposable PostgreSQL fixture database")
	}
	var conn, err = sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var service = &issuesService{db: conn}
	var ctx = context.Background()
	var input = issue.Report{
		FacilityID: "30000000-0000-0000-0000-000000000001",
		BookingID:  "40000000-0000-0000-0000-000000000001",
		Category:   "equipment", Description: "The reader is broken", Urgent: true,
	}
	// A council admin who is not a participant cannot report against the booking.
	if _, problem := service.Report(ctx, auth.Session{Subject: "test-3"}, input); problem == nil || problem.Code != http.StatusNotFound {
		t.Fatalf("unrelated user: %+v", problem)
	}
	var future = input
	future.BookingID = "40000000-0000-0000-0000-000000000002"
	if _, problem := service.Report(ctx, auth.Session{Subject: "test-1"}, future); problem == nil || problem.Message != "errors.issue_active_booking_required" {
		t.Fatalf("future booking: %+v", problem)
	}
	// Owner and participant race for the same five spots. Neither has checked in.
	var wg sync.WaitGroup
	var results = make(chan int, 12)
	for n := range 12 {
		wg.Go(func() {
			var _, problem = service.Report(ctx, auth.Session{Subject: fmt.Sprintf("test-%d", n%2+1)}, input)
			if problem == nil {
				results <- http.StatusOK
				return
			}
			if problem.Message != "errors.too_many_booking_issues" {
				t.Errorf("unexpected rejection: %+v", problem)
			}
			results <- problem.Code
		})
	}
	wg.Wait()
	close(results)
	var accepted int
	for code := range results {
		if code == http.StatusOK {
			accepted++
		}
	}
	if accepted != 5 {
		t.Fatalf("accepted %d reports, want exactly 5", accepted)
	}
	var q = model.New(conn)
	// Exactly one notice for each report to each council admin, none to others.
	for n := 1; n <= 5; n++ {
		var id = uuid.MustParse(fmt.Sprintf("20000000-0000-0000-0000-00000000000%d", n))
		var rows, listErr = q.ListNotificationsByUser(ctx, model.ListNotificationsByUserParams{UserID: id, MaxRows: 50})
		if listErr != nil {
			t.Fatal(listErr)
		}
		var want int
		if n == 3 || n == 4 {
			want = 5
		}
		if len(rows) != want {
			t.Errorf("user %d: %d notices, want %d", n, len(rows), want)
		}
	}
	var council = uuid.MustParse("10000000-0000-0000-0000-000000000001")
	var reports, listErr = q.ListCouncilIssues(ctx, model.ListCouncilIssuesParams{TenantID: council})
	if listErr != nil {
		t.Fatal(listErr)
	}
	for _, report := range reports {
		if err = q.UpdateFacilityIssue(ctx, model.UpdateFacilityIssueParams{ID: report.ID, Status: model.IssueStatusSolved}); err != nil {
			t.Fatal(err)
		}
	}
	if _, problem := service.Report(ctx, auth.Session{Subject: "test-1"}, input); problem == nil || problem.Message != "errors.too_many_booking_issues" {
		t.Fatalf("resolved reports must still count: %+v", problem)
	}
	var remaining, countErr = q.CountNewCouncilIssues(ctx, council)
	if countErr != nil {
		t.Fatal(countErr)
	}
	if remaining != 0 {
		t.Fatalf("resolved urgent reports still counted: %d", remaining)
	}
	var emails, emailErr = q.ListPendingIssueEmails(ctx, model.ListPendingIssueEmailsParams{EnvRelay: true, MaxRows: 50})
	if emailErr != nil {
		t.Fatal(emailErr)
	}
	if len(emails) != 10 {
		t.Fatalf("%d queued emails, want 10", len(emails))
	}
}
