package me

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"
)

func TestWriteExportCSV(t *testing.T) {
	var start = time.Date(2026, 11, 3, 18, 0, 0, 0, time.UTC)
	var cancelled = start.Add(-48 * time.Hour)
	var data = &ExportResponse{
		GeneratedAt: start,
		Account:     ExportAccount{Id: "u1", Email: "uxia@example.org", FullName: "Uxía Pérez, Lago", BirthDate: "1990-05-17", CreatedAt: start, UpdatedAt: start},
		Roles:       []TenantRoleDto{{TenantName: "Lalín", Role: "admin"}},
		Bookings: []ExportBooking{
			{Id: "b1", Council: "Lalín", Facility: "Pista 1", StartsAt: start, EndsAt: start.Add(time.Hour), Status: "cancelled", CancelledAt: &cancelled, CancelledBy: "user", CreatedAt: start, ParticipantCount: 3},
		},
		Notifications: []ExportNotification{{Type: "booking_cancelled", Payload: []byte(`{"facility_name":"Pista 1"}`), CreatedAt: start}},
	}
	var out bytes.Buffer
	if err := writeExportCSV(&out, data); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "\uFEFF") {
		t.Error("the file starts with a BOM, for Excel")
	}

	var reader = csv.NewReader(strings.NewReader(strings.TrimPrefix(out.String(), "\uFEFF")))
	reader.FieldsPerRecord = -1
	var rows, err = reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	var find = func(first string) []string {
		for _, row := range rows {
			if row[0] == first {
				return row
			}
		}
		t.Fatalf("no row starts with %q", first)
		return nil
	}
	if row := find("full_name"); row[1] != "Uxía Pérez, Lago" {
		t.Errorf("full_name = %q", row[1])
	}
	if row := find("b1"); row[7] != "2026-11-01T18:00:00Z" || row[8] != "user" || row[9] != "" || row[11] != "3" {
		t.Errorf("booking row = %q", row)
	}
	if row := find("Lalín"); row[1] != "admin" {
		t.Errorf("role row = %q", row)
	}
	if row := find("booking_cancelled"); row[4] != `{"facility_name":"Pista 1"}` {
		t.Errorf("notification row = %q", row)
	}
	for _, title := range []string{"account", "roles", "bookings", "participations", "notifications"} {
		find(title)
	}
}
