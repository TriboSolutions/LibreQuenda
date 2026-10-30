package counciladmin

import (
	"database/sql"
	"librequenda/model"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMapIssueKeepsTheEmailOnlyWhenShared(t *testing.T) {
	var row = model.ListCouncilIssuesRow{
		ID: uuid.New(), FacilityID: uuid.New(), FacilityName: "Pavillón", Category: model.IssueCategoryLighting,
		Description: "No light on court 2", Status: model.IssueStatusNew, CreatedAt: time.Now(),
		ReporterFirstName: "Lucía", ReporterEmail: sql.NullString{String: "lucia@example.com", Valid: true},
	}
	if got := mapIssue(row); got.ReporterEmail != "" || got.ReporterName != "Lucía" {
		t.Errorf("not shared: the first name only, got %+v", got)
	}
	row.ShareContact = true
	if got := mapIssue(row); got.ReporterEmail != "lucia@example.com" {
		t.Errorf("shared: the email shows, got %q", got.ReporterEmail)
	}
	row.PhotoRef = sql.NullString{String: "issues/a.png", Valid: true}
	if got := mapIssue(row); got.PhotoUrl == "" {
		t.Error("the photo's address is given")
	}
}
