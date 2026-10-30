package bookings

import (
	"database/sql"
	"librequenda/model"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMapMyBookingsMergesJoined(t *testing.T) {
	var now = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	var at = func(hours int) time.Time { return now.Add(time.Duration(hours) * time.Hour) }
	var owned = []model.ListBookingsByUserRow{ // most recent first, as the query returns them
		{ID: uuid.New(), StartsAt: at(48), EndsAt: at(49), Status: model.BookingStatusConfirmed, MinCancelHours: 2},
		{ID: uuid.New(), StartsAt: at(-48), EndsAt: at(-47), Status: model.BookingStatusConfirmed},
	}
	var joined = []model.ListParticipationsByUserRow{
		{ID: uuid.New(), StartsAt: at(24), EndsAt: at(25), Status: model.BookingStatusConfirmed, OwnerName: "Xoán", Capacity: sql.NullInt32{Int32: 4, Valid: true}, ParticipantCount: 2},
		{ID: uuid.New(), StartsAt: at(-24), EndsAt: at(-23), Status: model.BookingStatusCancelled},
	}
	var spots = map[uuid.UUID]Spots{
		owned[0].ID: {Capacity: sql.NullInt32{Int32: 10, Valid: true}, Participants: 3},
		owned[1].ID: {Capacity: sql.NullInt32{Int32: 1, Valid: true}},
	}

	var me = uuid.New()
	var joinedBy = map[uuid.UUID][]model.ListPeopleByBookingsRow{
		owned[0].ID:  {{BookingID: owned[0].ID, PublicName: "Brais"}, {BookingID: owned[0].ID, UserID: uuid.NullUUID{UUID: uuid.New(), Valid: true}, PublicName: "Antía"}},
		joined[0].ID: {{BookingID: joined[0].ID, UserID: uuid.NullUUID{UUID: me, Valid: true}, PublicName: "Uxía"}},
	}

	var got = MapMyBookings(owned, joined, spots, joinedBy, me, now)

	var ids = func(list []BookingDto) []string {
		var out []string
		for _, b := range list {
			out = append(out, b.Id)
		}
		return out
	}
	var wantUpcoming = []string{joined[0].ID.String(), owned[0].ID.String()} // soonest first
	var wantHistory = []string{joined[1].ID.String(), owned[1].ID.String()}  // most recent first
	if g := ids(got.Upcoming); len(g) != 2 || g[0] != wantUpcoming[0] || g[1] != wantUpcoming[1] {
		t.Fatalf("upcoming = %v, want %v", g, wantUpcoming)
	}
	if g := ids(got.History); len(g) != 2 || g[0] != wantHistory[0] || g[1] != wantHistory[1] {
		t.Fatalf("history = %v, want %v", g, wantHistory)
	}

	var participation, mine = got.Upcoming[0], got.Upcoming[1]
	if participation.Role != RoleParticipant || !participation.CanLeave || participation.CanCancel || participation.CheckinCode != "" {
		t.Errorf("participation = %+v: want a participant who can leave, without cancel or QR", participation)
	}
	if participation.Capacity != 4 || participation.ParticipantCount != 2 || participation.OwnerName != "Xoán" {
		t.Errorf("participation spots = %d/%d owner %q", participation.ParticipantCount, participation.Capacity, participation.OwnerName)
	}
	if mine.Role != RoleOwner || mine.Capacity != 10 || mine.ParticipantCount != 3 || mine.CheckinCode == "" {
		t.Errorf("own booking = %+v", mine)
	}
	if got.History[1].Capacity != 0 {
		t.Errorf("capacity 1 takes no invites, got capacity %d", got.History[1].Capacity)
	}

	if p := mine.People; len(p) != 3 || !p[0].IsOwner || !p[0].IsMe || !p[1].IsGuest || p[1].Name != "Brais" || p[2].IsGuest {
		t.Errorf("people on my booking = %+v", p)
	}
	if p := participation.People; len(p) != 2 || p[0].Name != "Xoán" || p[0].IsMe || !p[1].IsMe {
		t.Errorf("people on the joined booking = %+v", p)
	}
	if p := got.History[1].People; len(p) != 0 {
		t.Errorf("a booking nobody joined lists no one, got %+v", p)
	}
}

func TestAttendance(t *testing.T) {
	var now = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	var past, future = now.Add(-time.Hour), now.Add(time.Hour)
	var cases = []struct {
		name           string
		status         model.BookingStatus
		endsAt         time.Time
		checkedIn      bool
		checkinAtStart bool
		want           string
	}{
		{"upcoming", model.BookingStatusConfirmed, future, false, true, ""},
		{"in progress, already inside", model.BookingStatusConfirmed, future, true, true, ""},
		{"cancelled", model.BookingStatusCancelled, past, false, true, ""},
		{"checked in", model.BookingStatusConfirmed, past, true, true, AttendanceAttended},
		{"not scanned at a reader", model.BookingStatusConfirmed, past, false, true, AttendanceNoShow},
		{"no reader then", model.BookingStatusConfirmed, past, false, false, AttendanceUnknown},
	}
	for _, c := range cases {
		if got := attendance(c.status, c.endsAt, c.checkedIn, c.checkinAtStart, now); got != c.want {
			t.Errorf("%s: attendance = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestCheckinWays(t *testing.T) {
	var none = checkinWays(model.ListBookingsByUserRow{})
	if none == nil || len(none) != 0 {
		t.Fatalf("no way in must be an empty list, not null: %#v", none)
	}
	var all = checkinWays(model.ListBookingsByUserRow{HasReader: true, PrintedQr: true, EntryCode: true, StaffConfirm: true})
	var want = []string{"entry_code", "printed_qr", "reader", "staff"}
	if strings.Join(all, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", all, want)
	}
}
