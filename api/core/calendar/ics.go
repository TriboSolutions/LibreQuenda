package calendar

import (
	"fmt"
	"io"
	"librequenda/model"
	"strings"
	"time"
	"unicode/utf8"
)

// The feed is iCalendar (RFC 5545), the open format every calendar app reads.
// Times go out in UTC, so no VTIMEZONE is needed.
const icsTime = "20060102T150405Z"

// writeICS writes the bookings as a VCALENDAR. link is where the event points
// to (the bookings page); now stamps the file.
func writeICS(w io.Writer, rows []model.ListCalendarBookingsRow, link string, now time.Time) error {
	var b strings.Builder
	var line = func(name, value string) { b.WriteString(fold(name + ":" + value)) }

	line("BEGIN", "VCALENDAR")
	line("VERSION", "2.0")
	line("PRODID", "-//LibreQuenda//Reservas//GL")
	line("CALSCALE", "GREGORIAN")
	line("METHOD", "PUBLISH")
	line("X-WR-CALNAME", "LibreQuenda")
	// how often apps should fetch it again (Apple, Outlook)
	line("REFRESH-INTERVAL;VALUE=DURATION", "PT1H")
	line("X-PUBLISHED-TTL", "PT1H")
	for _, row := range rows {
		line("BEGIN", "VEVENT")
		line("UID", row.ID.String()+"@librequenda")
		line("DTSTAMP", now.UTC().Format(icsTime))
		line("LAST-MODIFIED", row.UpdatedAt.UTC().Format(icsTime))
		line("DTSTART", row.StartsAt.UTC().Format(icsTime))
		line("DTEND", row.EndsAt.UTC().Format(icsTime))
		line("SUMMARY", text(row.FacilityName))
		var location = row.FacilityName
		if row.Address.Valid && row.Address.String != "" {
			location += ", " + row.Address.String
		}
		line("LOCATION", text(location))
		if row.Latitude.Valid && row.Longitude.Valid {
			line("GEO", fmt.Sprintf("%.6f;%.6f", row.Latitude.Float64, row.Longitude.Float64))
		}
		line("DESCRIPTION", text(row.TenantName))
		if link != "" {
			line("URL", link)
		}
		if row.Status == model.BookingStatusCancelled {
			line("STATUS", "CANCELLED")
		} else {
			line("STATUS", "CONFIRMED")
		}
		line("TRANSP", "OPAQUE")
		line("END", "VEVENT")
	}
	line("END", "VCALENDAR")

	var _, err = io.WriteString(w, b.String())
	return err
}

// text escapes a TEXT value (RFC 5545 §3.3.11).
var textEscaper = strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\r\n", `\n`, "\n", `\n`, "\r", `\n`)

func text(value string) string { return textEscaper.Replace(value) }

// fold ends the content line with CRLF, splitting it every 75 octets without
// cutting a UTF-8 character (RFC 5545 §3.1).
func fold(contentLine string) string {
	var b strings.Builder
	var limit = 75
	for len(contentLine) > limit {
		var cut = limit
		for cut > 0 && !utf8.RuneStart(contentLine[cut]) {
			cut--
		}
		b.WriteString(contentLine[:cut])
		b.WriteString("\r\n ")
		contentLine = contentLine[cut:]
		limit = 74 // the leading space counts
	}
	b.WriteString(contentLine)
	b.WriteString("\r\n")
	return b.String()
}
