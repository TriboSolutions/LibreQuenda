package counciladmin

import (
	"database/sql"
	"librequenda/model"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMapIssueKeepsContactDetailsOnlyWhenShared(t *testing.T) {
	var row = model.ListCouncilIssuesRow{
		ID: uuid.New(), FacilityID: uuid.New(), FacilityName: "Pavillón", Category: model.IssueCategoryLighting,
		Description: "No light on court 2", Status: model.IssueStatusNew, CreatedAt: time.Now(),
		ReporterPhone:     sql.NullString{String: "+34 600 123 456", Valid: true},
		ReporterFirstName: "Lucía", ReporterEmail: sql.NullString{String: "lucia@example.com", Valid: true},
	}
	if got := mapIssue(row); got.ReporterPhone != "" || got.ReporterEmail != "" || got.ReporterName != "Lucía" {
		t.Errorf("not shared: the first name only, got %+v", got)
	}
	row.ShareContact = true
	if got := mapIssue(row); got.ReporterEmail != "lucia@example.com" || got.ReporterPhone != "+34 600 123 456" {
		t.Errorf("shared: the email shows, got %q", got.ReporterEmail)
	}
	row.ReporterPhone = sql.NullString{}
	if got := mapIssue(row); got.ReporterPhone != "" {
		t.Error("missing phone should stay empty")
	}
	row.FacilityImageRef = sql.NullString{String: "0192f5a0-1c2d-7e3f-8a9b-0c1d2e3f4a5b/facilities/court.jpg", Valid: true}
	if got := mapIssue(row); got.FacilityImageURL == "" {
		t.Error("facility photo should be mapped")
	}
	row.PhotoRef = sql.NullString{String: "0192f5a0-1c2d-7e3f-8a9b-0c1d2e3f4a5b/issues/a.png", Valid: true}
	if got := mapIssue(row); got.PhotoUrl == "" {
		t.Error("the photo's address is given")
	}
}
