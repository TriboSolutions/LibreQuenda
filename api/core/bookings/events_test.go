package bookings

import (
	"database/sql"
	"librequenda/model"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEventInputValidate(t *testing.T) {
	var now = time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	var madrid, _ = time.LoadLocation("Europe/Madrid")
	var ok = func() EventInput {
		return EventInput{FacilityId: uuid.New(), Start: "2026-11-06T10:00", End: "2026-11-06T14:00", Title: "  Torneo   de fútbol 5v5 "}
	}
	var zero, tooMany, ten = int32(0), int32(maxEventCapacity + 1), int32(10)
	var cases = []struct {
		name   string
		change func(*EventInput)
		want   string
	}{
		{"ok", func(*EventInput) {}, ""},
		{"own capacity", func(d *EventInput) { d.Capacity = &ten }, ""},
		{"outside the turns: any minute", func(d *EventInput) { d.Start = "2026-11-06T07:17" }, ""},
		{"no facility", func(d *EventInput) { d.FacilityId = uuid.Nil }, "errors.invalid_request_body"},
		{"not a local time", func(d *EventInput) { d.Start = "2026-11-06T10:00:00Z" }, "errors.invalid_request_body"},
		{"no title", func(d *EventInput) { d.Title = "   " }, "errors.event_title"},
		{"long title", func(d *EventInput) { d.Title = strings.Repeat("á", maxEventTitle+1) }, "errors.event_title"},
		{"long description", func(d *EventInput) { d.Description = strings.Repeat("a", maxEventDescription+1) }, "errors.event_description_too_long"},
		{"ends at the start", func(d *EventInput) { d.End = d.Start }, "errors.event_ends_before_start"},
		{"a week is the most", func(d *EventInput) { d.End = "2026-11-13T10:01" }, "errors.event_too_long"},
		{"in the past", func(d *EventInput) { d.Start = "2026-11-04T09:00" }, "errors.event_in_past"},
		{"capacity zero", func(d *EventInput) { d.Capacity = &zero }, "errors.event_capacity"},
		{"capacity too high", func(d *EventInput) { d.Capacity = &tooMany }, "errors.event_capacity"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var in = ok()
			c.change(&in)
			if got := in.validate(now, madrid); got != c.want {
				t.Errorf("validate() = %q, want %q", got, c.want)
			}
		})
	}
	var in = ok()
	in.validate(now, madrid)
	if in.Title != "Torneo de fútbol 5v5" {
		t.Errorf("title = %q, want it trimmed", in.Title)
	}
	if want := time.Date(2026, 11, 6, 9, 0, 0, 0, time.UTC); !in.StartsAt.Equal(want) {
		t.Errorf("starts at %v, want 10:00 in Madrid (%v)", in.StartsAt, want)
	}
}

func TestPlaces(t *testing.T) {
	var ten = sql.NullInt32{Int32: 10, Valid: true}
	var one = sql.NullInt32{Int32: 1, Valid: true}
	var cases = []struct {
		name     string
		capacity sql.NullInt32
		kind     model.BookingKind
		want     int
	}{
		{"booking: the owner takes one", ten, model.BookingKindPublic, 9},
		{"booking for one", one, model.BookingKindPrivate, 0},
		{"event: no owner", ten, model.BookingKindEvent, 10},
		{"event for one", one, model.BookingKindEvent, 1},
		{"no capacity", sql.NullInt32{}, model.BookingKindEvent, 0},
	}
	for _, c := range cases {
		if got := Places(c.capacity, c.kind); got != c.want {
			t.Errorf("%s: Places() = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestEventCapacity(t *testing.T) {
	if eventCapacity(sql.NullInt32{}, sql.NullInt32{}) == nil {
		t.Error("an event without its own capacity at a facility without one must be refused")
	}
	if eventCapacity(sql.NullInt32{Int32: 50, Valid: true}, sql.NullInt32{}) != nil {
		t.Error("its own capacity is enough")
	}
	if eventCapacity(sql.NullInt32{}, sql.NullInt32{Int32: 10, Valid: true}) != nil {
		t.Error("the facility's capacity is enough")
	}
}

func TestStaffBookingOfAnEvent(t *testing.T) {
	var now = time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	var row = model.ListFacilityBookingsForStaffRow{
		ID: uuid.New(), StartsAt: now.Add(time.Hour), EndsAt: now.Add(5 * time.Hour),
		Channel: model.BookingChannelStaff, Kind: model.BookingKindEvent, Title: "Fiesta del baloncesto",
	}
	var dto = mapStaffBooking(row, now, true)
	if !dto.IsEvent || dto.Name != "Fiesta del baloncesto" {
		t.Errorf("event = %+v, want its title as the name", dto)
	}
	if dto.CheckinCode != "" || dto.CheckinState != "" {
		t.Errorf("nobody checks in to an event: code %q, state %q", dto.CheckinCode, dto.CheckinState)
	}
	if !dto.CanCancel {
		t.Error("staff can cancel an event before it starts")
	}
}
