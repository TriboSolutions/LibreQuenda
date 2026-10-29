package calendar

import (
	"bytes"
	"database/sql"
	"librequenda/model"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

func TestWriteICS(t *testing.T) {
	var start = time.Date(2026, 11, 3, 18, 0, 0, 0, time.FixedZone("CET", 3600))
	var rows = []model.ListCalendarBookingsRow{
		{
			ID: uuid.MustParse("0192f5a0-0000-7000-8000-000000000001"), StartsAt: start, EndsAt: start.Add(time.Hour),
			Status: model.BookingStatusConfirmed, UpdatedAt: start.Add(-time.Hour),
			FacilityName: "Pista de pádel 1", Address: sql.NullString{String: "Rúa do Parque, 3; Lalín", Valid: true},
			Latitude: sql.NullFloat64{Float64: 42.66, Valid: true}, Longitude: sql.NullFloat64{Float64: -8.11, Valid: true},
			TenantName: "Concello de Lalín",
		},
		{
			ID: uuid.MustParse("0192f5a0-0000-7000-8000-000000000002"), StartsAt: start, EndsAt: start.Add(time.Hour),
			Status: model.BookingStatusCancelled, UpdatedAt: start, FacilityName: strings.Repeat("Piscina climatizada ", 6), TenantName: "Lalín",
		},
	}
	var out bytes.Buffer
	if err := writeICS(&out, rows, "Reservas, Lalín", "https://reservas.example.org/reservas", start); err != nil {
		t.Fatal(err)
	}
	var ics = out.String()

	for _, want := range []string{
		"BEGIN:VCALENDAR\r\nVERSION:2.0\r\n",
		`X-WR-CALNAME:Reservas\, Lalín` + "\r\n", // the installation's name (branding), escaped
		"UID:0192f5a0-0000-7000-8000-000000000001@librequenda\r\n",
		"DTSTART:20261103T170000Z\r\n", // in UTC
		"DTEND:20261103T180000Z\r\n",
		`LOCATION:Pista de pádel 1\, Rúa do Parque\, 3\; Lalín` + "\r\n",
		"GEO:42.660000;-8.110000\r\n",
		"STATUS:CONFIRMED\r\n",
		"STATUS:CANCELLED\r\n",
		"URL:https://reservas.example.org/reservas\r\n",
		"END:VCALENDAR\r\n",
	} {
		if !strings.Contains(ics, want) {
			t.Errorf("missing %q in\n%s", want, ics)
		}
	}
	if strings.Contains(strings.ReplaceAll(ics, "\r\n", ""), "\n") {
		t.Error("every line ends in CRLF")
	}
	for _, line := range strings.Split(strings.TrimSuffix(ics, "\r\n"), "\r\n") {
		if len(line) > 75 {
			t.Errorf("line longer than 75 octets: %q", line)
		}
	}
	if !strings.Contains(ics, "\r\n ") {
		t.Error("the long SUMMARY is folded onto a continuation line")
	}
}

func TestFoldKeepsUTF8(t *testing.T) {
	var long = "SUMMARY:" + strings.Repeat("á", 80)
	var folded = fold(long)
	if got := strings.ReplaceAll(strings.TrimSuffix(folded, "\r\n"), "\r\n ", ""); got != long {
		t.Errorf("unfolding gives back the line, got %q", got)
	}
	for _, line := range strings.Split(strings.TrimSuffix(folded, "\r\n"), "\r\n") {
		if len(line) > 75 || !utf8.ValidString(line) {
			t.Errorf("bad folded line %q", line)
		}
	}
}

func utf8Valid(s string) bool { return strings.ToValidUTF8(s, "�") == s }
