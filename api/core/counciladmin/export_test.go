package counciladmin

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"librequenda/model"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCouncilExportZip(t *testing.T) {
	var facility = uuid.New()
	var owner = uuid.New()
	var data = councilExport{
		generatedAt: time.Date(2026, 10, 28, 10, 0, 0, 0, time.UTC),
		tenant:      model.Tenant{ID: uuid.New(), Slug: "lalin", Name: "Concello de Lalín", Timezone: "Europe/Madrid"},
		facilities: []model.ExportFacilitiesRow{{
			Facility: model.Facility{ID: facility, Slug: "padel-1", Name: "Pádel 1, cuberta", SlotMinutes: 60, UsageRulesText: sql.NullString{String: "Calzado \"deportivo\"\nsempre", Valid: true}},
			Sport:    "Pádel",
			Tags:     []string{"cuberta", "padel"},
		}},
		hours: []model.OpeningHour{{ID: uuid.New(), FacilityID: facility, Weekday: 1, OpensAt: time.Date(0, 1, 1, 9, 30, 0, 0, time.UTC), ClosesAt: time.Date(0, 1, 1, 21, 0, 0, 0, time.UTC)}},
		bookings: []model.ExportBookingsRow{
			{ID: uuid.New(), FacilityID: facility, UserID: uuid.NullUUID{UUID: owner, Valid: true}, Status: model.BookingStatusConfirmed},
			{ID: uuid.New(), FacilityID: facility, Status: model.BookingStatusCancelled, CancelledBy: model.NullBookingCanceller{BookingCanceller: model.BookingCancellerCouncil, Valid: true}},
		},
		participants: []model.ExportBookingParticipantsRow{{ID: uuid.New(), IsGuest: true}},
		issues:       []model.FacilityIssue{{ID: uuid.New(), Category: model.IssueCategoryLighting, Description: "No light", Status: model.IssueStatusNew}},
	}
	var buffer bytes.Buffer
	if err := data.WriteZip(context.Background(), &buffer); err != nil {
		t.Fatal(err)
	}
	var archive, err = zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var files = map[string]*zip.File{}
	for _, f := range archive.File {
		files[f.Name] = f
	}

	var descriptor struct {
		Resources []struct {
			Path   string
			Schema struct{ Fields []struct{ Name string } }
		}
	}
	var reader, _ = files["datapackage.json"].Open()
	if err = json.NewDecoder(reader).Decode(&descriptor); err != nil {
		t.Fatal(err)
	}
	if len(descriptor.Resources) != 8 { // with the maintenance list (facility_issues)
		t.Fatalf("%d tables, want 8", len(descriptor.Resources))
	}
	var rowCounts = map[string]int{}
	for _, resource := range descriptor.Resources {
		var file = files[resource.Path]
		if file == nil {
			t.Fatalf("%s is described but not in the ZIP", resource.Path)
		}
		var r, _ = file.Open()
		var rows, csvErr = csv.NewReader(r).ReadAll() // also fails on rows of the wrong width
		if csvErr != nil {
			t.Fatalf("%s: %v", resource.Path, csvErr)
		}
		for i, field := range resource.Schema.Fields {
			if rows[0][i] != field.Name {
				t.Errorf("%s column %d is %q, the descriptor says %q", resource.Path, i, rows[0][i], field.Name)
			}
		}
		rowCounts[resource.Path] = len(rows) - 1
	}
	if rowCounts["council.csv"] != 1 || rowCounts["bookings.csv"] != 2 || rowCounts["opening_hours.csv"] != 1 || rowCounts["closures.csv"] != 0 {
		t.Errorf("row counts %v", rowCounts)
	}

	var r, _ = files["opening_hours.csv"].Open()
	var rows, _ = csv.NewReader(r).ReadAll()
	if rows[1][3] != "09:30" || rows[1][4] != "21:00" {
		t.Errorf("opening hours %v, want 09:30 - 21:00", rows[1])
	}
	r, _ = files["bookings.csv"].Open()
	rows, _ = csv.NewReader(r).ReadAll()
	if rows[1][2] != owner.String() || rows[2][2] != "" || rows[2][8] != "council" {
		t.Errorf("bookings %v", rows[1:])
	}
}

func TestMediaPath(t *testing.T) {
	if got := mediaPath("facilities/01a1.webp"); got != "media/facilities/01a1.webp" {
		t.Errorf("mediaPath = %q", got)
	}
}

func TestCouncilExportPersonal(t *testing.T) {
	var owner = uuid.New()
	var base = councilExport{
		tenant:       model.Tenant{ID: uuid.New(), Slug: "lalin"},
		bookings:     []model.ExportBookingsRow{{ID: uuid.New(), UserID: uuid.NullUUID{UUID: owner, Valid: true}, Status: model.BookingStatusConfirmed}},
		participants: []model.ExportBookingParticipantsRow{{ID: uuid.New(), IsGuest: true, GuestName: sql.NullString{String: "Uxía", Valid: true}, GuestAge: sql.NullInt16{Int16: 12, Valid: true}}},
		people:       []model.ExportPeopleRow{{ID: owner, FullName: sql.NullString{String: "Brais Otero", Valid: true}, Email: "brais@example.org", BirthDate: sql.NullTime{Time: time.Date(1990, 5, 2, 0, 0, 0, 0, time.UTC), Valid: true}}},
	}
	for _, personal := range []bool{false, true} {
		var data = base
		data.personal = personal
		var buffer bytes.Buffer
		if err := data.WriteZip(context.Background(), &buffer); err != nil {
			t.Fatal(err)
		}
		var archive, _ = zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
		var people, participants []byte
		for _, f := range archive.File {
			var r, _ = f.Open()
			var content bytes.Buffer
			content.ReadFrom(r)
			switch f.Name {
			case "people.csv":
				people = content.Bytes()
			case "booking_participants.csv":
				participants = content.Bytes()
			}
		}
		if !personal {
			if people != nil || bytes.Contains(participants, []byte("Uxía")) {
				t.Errorf("without personal=1 the export holds personal data:\n%s", participants)
			}
			continue
		}
		if !bytes.Contains(people, []byte("brais@example.org")) || !bytes.Contains(people, []byte(",1990-05-02,")) {
			t.Errorf("people.csv:\n%s", people)
		}
		if !bytes.Contains(participants, []byte(",Uxía,12")) {
			t.Errorf("booking_participants.csv:\n%s", participants)
		}
	}
}
