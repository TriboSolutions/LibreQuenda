package bookings

import (
	"database/sql"
	"librequenda/model"
	"librequenda/schedule"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestStaffBookingInputValidate(t *testing.T) {
	var base = StaffBookingInput{FacilityId: uuid.New(), StartsAt: time.Now(), AcceptRules: true}
	var with = func(change func(*StaffBookingInput)) StaffBookingInput {
		var input = base
		change(&input)
		return input
	}
	var cases = []struct {
		name  string
		input StaffBookingInput
		want  string
	}{
		{"by account", with(func(i *StaffBookingInput) { i.Email = " ana@example.org " }), ""},
		{"by name", with(func(i *StaffBookingInput) { i.Name = "  Carmen   Souto "; i.Phone = "600000000" }), ""},
		{"neither", base, "errors.staff_booking_holder"},
		{"both", with(func(i *StaffBookingInput) { i.Email = "ana@example.org"; i.Name = "Ana" }), "errors.staff_booking_holder"},
		{"bad email", with(func(i *StaffBookingInput) { i.Email = "ana" }), "errors.invalid_email"},
		{"long name", with(func(i *StaffBookingInput) { i.Name = strings.Repeat("a", 81) }), "errors.staff_booking_holder_too_long"},
		{"rules not told", with(func(i *StaffBookingInput) { i.Name = "Ana"; i.AcceptRules = false }), "errors.rules_not_accepted"},
		{"no turn", with(func(i *StaffBookingInput) { i.Name = "Ana"; i.StartsAt = time.Time{} }), "errors.invalid_request_body"},
	}
	for _, c := range cases {
		var input = c.input
		if got := input.validate(); got != c.want {
			t.Errorf("%s: validate = %q, want %q", c.name, got, c.want)
		}
	}
	var named = with(func(i *StaffBookingInput) { i.Name = "  Carmen   Souto " })
	named.validate()
	if named.Name != "Carmen Souto" {
		t.Errorf("name not tidied: %q", named.Name)
	}
}

func TestMapStaffBooking(t *testing.T) {
	var now = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	var walkIn = model.ListFacilityBookingsForStaffRow{
		ID: uuid.New(), StartsAt: now.Add(time.Hour), EndsAt: now.Add(2 * time.Hour),
		Channel: model.BookingChannelStaff, CheckinToken: uuid.New(),
		HolderName: sql.NullString{String: "Carmen", Valid: true}, HolderPhone: sql.NullString{String: "600", Valid: true},
	}
	var got = mapStaffBooking(walkIn, now, false)
	if got.HasAccount || got.Name != "Carmen" || got.Phone != "600" || !got.BookedByStaff || !got.CanCancel || got.CheckinCode == "" {
		t.Errorf("walk-in booking mapped as %+v", got)
	}

	var online = model.ListFacilityBookingsForStaffRow{
		ID: uuid.New(), StartsAt: now.Add(-time.Hour), EndsAt: now.Add(time.Hour),
		Channel: model.BookingChannelOnline, CheckinToken: uuid.New(),
		UserID: uuid.NullUUID{UUID: uuid.New(), Valid: true}, Email: sql.NullString{String: "ana@example.org", Valid: true},
	}
	got = mapStaffBooking(online, now, false)
	if !got.HasAccount || got.Name != "ana@example.org" || got.CanCancel || got.CheckinCode != "" {
		t.Errorf("online booking mapped as %+v", got)
	}
	if got.CheckinState != "" {
		t.Errorf("staff entry off, but checkin_state = %q", got.CheckinState)
	}
	if got = mapStaffBooking(online, now, true); got.CheckinState != StaffCheckinOpen {
		t.Errorf("staff entry on, turn under way: checkin_state = %q, want open", got.CheckinState)
	}
}

func TestStaffCheckinState(t *testing.T) {
	var start = time.Date(2026, 11, 3, 17, 0, 0, 0, time.UTC)
	var end = start.Add(time.Hour)
	var cases = []struct {
		name string
		now  time.Time
		want string
	}{
		{"an hour before", start.Add(-time.Hour), StaffCheckinTooEarly},
		{"just before the early window", start.Add(-16 * time.Minute), StaffCheckinTooEarly},
		{"early window opens", start.Add(-15 * time.Minute), StaffCheckinOpen},
		{"under way", start.Add(30 * time.Minute), StaffCheckinOpen},
		{"last minute", end.Add(-time.Minute), StaffCheckinOpen},
		{"ended", end, StaffCheckinEnded},
	}
	for _, c := range cases {
		if got := staffCheckinState(start, end, c.now); got != c.want {
			t.Errorf("%s: staffCheckinState = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestMapStaffBookingCheckedIn(t *testing.T) {
	var now = time.Date(2026, 11, 3, 17, 10, 0, 0, time.UTC)
	var row = model.ListFacilityBookingsForStaffRow{
		ID: uuid.New(), StartsAt: now.Add(-10 * time.Minute), EndsAt: now.Add(50 * time.Minute),
		Channel: model.BookingChannelOnline, CheckinToken: uuid.New(),
		UserID:          uuid.NullUUID{UUID: uuid.New(), Valid: true},
		CheckedInAt:     sql.NullTime{Time: now.Add(-2 * time.Minute), Valid: true},
		CheckedInMethod: model.NullCheckinMethod{CheckinMethod: model.CheckinMethodStaff, Valid: true},
		CheckedInByName: "Xoán Conserxe",
	}
	var got = mapStaffBooking(row, now, true)
	if got.CheckedInMethod != "staff" || got.CheckedInBy != "Xoán Conserxe" || got.CheckinState != "" {
		t.Errorf("staff entry mapped as %+v", got)
	}
	if got.UndoUntil == nil || !got.UndoUntil.Equal(now.Add(3*time.Minute)) {
		t.Errorf("undo_until = %v, want 3 minutes from now", got.UndoUntil)
	}

	row.CheckedInAt.Time = now.Add(-6 * time.Minute) // past the undo window
	if got = mapStaffBooking(row, now, true); got.UndoUntil != nil {
		t.Errorf("undo still offered after the window: %v", got.UndoUntil)
	}

	row.CheckedInMethod.CheckinMethod = model.CheckinMethodReader // a reader's entry: nobody to name, nothing to undo
	row.CheckedInAt.Time = now.Add(-time.Minute)
	row.CheckedInByName = ""
	if got = mapStaffBooking(row, now, true); got.CheckedInMethod != "reader" || got.CheckedInBy != "" || got.UndoUntil != nil {
		t.Errorf("reader entry mapped as %+v", got)
	}
}

func TestStaffSlots(t *testing.T) {
	var at = func(hour int) time.Time { return time.Date(2026, 11, 3, hour, 0, 0, 0, time.UTC) }
	var slots = []schedule.Slot{
		{Start: at(9), End: at(10), Status: schedule.SlotFree},
		{Start: at(10), End: at(11), Status: schedule.SlotBooked},
		{Start: at(11), End: at(12), Status: schedule.SlotBooked},
	}
	// one booking from a longer, older turn length covers two slots
	var bookings = []StaffBookingDto{{Id: "b1", StartsAt: at(10), EndsAt: at(12)}}
	var got = staffSlots(slots, bookings)
	if got[0].BookingId != "" || got[1].BookingId != "b1" || got[2].BookingId != "b1" {
		t.Errorf("staffSlots = %+v", got)
	}
}
