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
	var got = mapStaffBooking(walkIn, now)
	if got.HasAccount || got.Name != "Carmen" || got.Phone != "600" || !got.BookedByStaff || !got.CanCancel || got.CheckinCode == "" {
		t.Errorf("walk-in booking mapped as %+v", got)
	}

	var online = model.ListFacilityBookingsForStaffRow{
		ID: uuid.New(), StartsAt: now.Add(-time.Hour), EndsAt: now.Add(time.Hour),
		Channel: model.BookingChannelOnline, CheckinToken: uuid.New(),
		UserID: uuid.NullUUID{UUID: uuid.New(), Valid: true}, Email: sql.NullString{String: "ana@example.org", Valid: true},
	}
	got = mapStaffBooking(online, now)
	if !got.HasAccount || got.Name != "ana@example.org" || got.CanCancel || got.CheckinCode != "" {
		t.Errorf("online booking mapped as %+v", got)
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
