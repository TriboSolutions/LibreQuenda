package bookings

import (
	"database/sql"
	"librequenda/checkin"
	"librequenda/model"
	"librequenda/storage"
	"sort"
	"time"

	"github.com/google/uuid"
)

type CreateBookingDto struct {
	FacilityId  uuid.UUID `json:"facility_id"`
	StartsAt    time.Time `json:"starts_at"`
	AcceptRules bool      `json:"accept_rules"` // the resident ticked "I have read the rules"; stored as rules_accepted_at
}

func (d *CreateBookingDto) Validate() []string {
	var errs []string
	if d.FacilityId == uuid.Nil {
		errs = append(errs, "facility_id is required")
	}
	if d.StartsAt.IsZero() {
		errs = append(errs, "starts_at is required")
	}
	return errs
}

// ChangeBookingDto: the new turn, on the same facility.
type ChangeBookingDto struct {
	StartsAt time.Time `json:"starts_at"`
}

func (d *ChangeBookingDto) Validate() []string {
	if d.StartsAt.IsZero() {
		return []string{"starts_at is required"}
	}
	return nil
}

type BookingDto struct {
	Id           string     `json:"id"`
	StartsAt     time.Time  `json:"starts_at"`
	EndsAt       time.Time  `json:"ends_at"`
	Status       string     `json:"status"`
	FacilityId   string     `json:"facility_id"`
	FacilityName string     `json:"facility_name"`
	FacilitySlug string     `json:"facility_slug"`
	TenantSlug   string     `json:"tenant_slug"`
	TenantName   string     `json:"tenant_name"`
	CanCancel    bool       `json:"can_cancel"`
	CanChange    bool       `json:"can_change"` // owner only: another time, on the same terms as cancelling
	CancelUntil  time.Time  `json:"cancel_until"`
	CheckinCode  string     `json:"checkin_code"` // QR content, upcoming confirmed bookings only
	CheckedInAt  *time.Time `json:"checked_in_at"`
	// owner only: the ways in the facility offers ("reader", "printed_qr", "entry_code", "staff")
	CheckinWays []string `json:"checkin_ways"`
	// owner only, with staff confirmation on: who to ask on site
	StaffContacts []checkin.StaffContact `json:"staff_contacts,omitempty"`
	Attendance    string                 `json:"attendance"` // past confirmed bookings only: "attended", "no_show" or "unknown"; "" otherwise
	// invites: the owner shares a link and others fill the facility's capacity
	Role             string      `json:"role"`              // "owner" or "participant" (joined through an invite)
	OwnerName        string      `json:"owner_name"`        // participant only
	Capacity         int         `json:"capacity"`          // people including the owner; 0 when the facility takes no invites
	ParticipantCount int         `json:"participant_count"` // people who joined, not counting the owner
	CanLeave         bool        `json:"can_leave"`         // participant only, until the start
	People           []PersonDto `json:"people"`            // who is coming: the owner first, then by joining order
	// what the booking details show; empty in the create and change responses
	Facility FacilityDetailsDto `json:"facility"`
}

// FacilityDetailsDto: where to go and what to know before going, for the booking details and the invite page.
type FacilityDetailsDto struct {
	ImageUrl          string   `json:"image_url"`
	Sport             string   `json:"sport"`
	Address           string   `json:"address"`
	Latitude          *float64 `json:"latitude"`
	Longitude         *float64 `json:"longitude"`
	AccessibilityInfo string   `json:"accessibility_info"`
	SlotMinutes       int32    `json:"slot_minutes"`
	MinCancelHours    int32    `json:"min_cancel_hours"`
	MinAge            *int32   `json:"min_age"` // age control: null = none
	UsageRules        string   `json:"usage_rules"`
}

// sport: facility_sport(f.id), its sport tags' names (the facility row has no sport of its own)
func FacilityDetails(f model.Facility, sport string) FacilityDetailsDto {
	var details = FacilityDetailsDto{
		ImageUrl:          storage.URL(f.ImageRef.String),
		Sport:             sport,
		Address:           f.Address.String,
		AccessibilityInfo: f.AccessibilityInfo.String,
		SlotMinutes:       f.SlotMinutes,
		MinCancelHours:    f.MinCancelHours,
		UsageRules:        f.UsageRulesText.String,
	}
	if f.Latitude.Valid && f.Longitude.Valid {
		details.Latitude = &f.Latitude.Float64
		details.Longitude = &f.Longitude.Float64
	}
	if f.MinAge.Valid {
		details.MinAge = &f.MinAge.Int32
	}
	return details
}

// PersonDto is what players see of each other: a name, never an email or an age.
type PersonDto struct {
	Name    string `json:"name"`
	IsOwner bool   `json:"is_owner"`
	IsGuest bool   `json:"is_guest"`
	IsMe    bool   `json:"is_me"`
}

// people lists the owner and who joined; a booking nobody joined lists no one.
func people(owner PersonDto, joined []model.ListPeopleByBookingsRow, me uuid.UUID) []PersonDto {
	if len(joined) == 0 {
		return []PersonDto{}
	}
	var list = make([]PersonDto, 0, len(joined)+1)
	list = append(list, owner)
	for _, row := range joined {
		list = append(list, PersonDto{Name: row.PublicName, IsGuest: !row.UserID.Valid, IsMe: row.UserID.Valid && row.UserID.UUID == me})
	}
	return list
}

type MyBookingsResponse struct {
	Upcoming []BookingDto `json:"upcoming"` // soonest first
	History  []BookingDto `json:"history"`  // most recent first
}

const historyLimit = 100

const (
	RoleOwner       = "owner"
	RoleParticipant = "participant"
)

// Attendance of a past booking, from the QR readers at the door.
const (
	AttendanceAttended = "attended"
	AttendanceNoShow   = "no_show"
	AttendanceUnknown  = "unknown" // no reader when the turn began: nobody could check in
)

// attendance: "" until a confirmed booking has ended; then whether anyone checked in,
// which only says something if a reader was there to scan the QR.
func attendance(status model.BookingStatus, endsAt time.Time, checkedIn, readerAtStart bool, now time.Time) string {
	switch {
	case status != model.BookingStatusConfirmed || endsAt.After(now):
		return ""
	case checkedIn:
		return AttendanceAttended
	case readerAtStart:
		return AttendanceNoShow
	default:
		return AttendanceUnknown
	}
}

// Spots is the facility capacity and how many joined, per booking the user owns.
type Spots struct {
	Capacity     sql.NullInt32
	Participants int64
}

func invitableCapacity(capacity sql.NullInt32) int {
	if !capacity.Valid || capacity.Int32 <= 1 {
		return 0
	}
	return int(capacity.Int32)
}

// MapMyBookings merges the bookings the user made with the ones they joined.
func MapMyBookings(rows []model.ListBookingsByUserRow, joined []model.ListParticipationsByUserRow, spots map[uuid.UUID]Spots, joinedBy map[uuid.UUID][]model.ListPeopleByBookingsRow, me uuid.UUID, now time.Time) *MyBookingsResponse {
	var response = &MyBookingsResponse{Upcoming: []BookingDto{}, History: []BookingDto{}}
	var all = make([]BookingDto, 0, len(rows)+len(joined))
	for _, row := range rows {
		var booking = mapBooking(row, now)
		booking.Capacity = invitableCapacity(spots[row.ID].Capacity)
		booking.ParticipantCount = int(spots[row.ID].Participants)
		booking.People = people(PersonDto{IsOwner: true, IsMe: true}, joinedBy[row.ID], me)
		all = append(all, booking)
	}
	for _, row := range joined {
		var booking = mapParticipation(row, now)
		booking.People = people(PersonDto{Name: row.OwnerName, IsOwner: true}, joinedBy[row.ID], me)
		all = append(all, booking)
	}
	// most recent first; upcoming is then read backwards, soonest first
	sort.SliceStable(all, func(i, j int) bool { return all[i].StartsAt.After(all[j].StartsAt) })

	for i := len(all) - 1; i >= 0; i-- {
		if all[i].Status == string(model.BookingStatusConfirmed) && all[i].EndsAt.After(now) {
			response.Upcoming = append(response.Upcoming, all[i])
		}
	}
	for _, booking := range all {
		if (booking.Status != string(model.BookingStatusConfirmed) || !booking.EndsAt.After(now)) && len(response.History) < historyLimit {
			response.History = append(response.History, booking)
		}
	}
	return response
}

// mapParticipation: someone else's booking; the QR and cancelling stay with its owner.
func mapParticipation(row model.ListParticipationsByUserRow, now time.Time) BookingDto {
	var checkedInAt *time.Time
	if row.CheckedInAt.Valid {
		checkedInAt = &row.CheckedInAt.Time
	}
	return BookingDto{
		Id:               row.ID.String(),
		StartsAt:         row.StartsAt,
		EndsAt:           row.EndsAt,
		Status:           string(row.Status),
		FacilityId:       row.FacilityID.String(),
		FacilityName:     row.FacilityName,
		FacilitySlug:     row.FacilitySlug,
		TenantSlug:       row.TenantSlug,
		TenantName:       row.TenantName,
		CheckedInAt:      checkedInAt,
		Attendance:       attendance(row.Status, row.EndsAt, row.CheckedInAt.Valid, row.ReaderAtStart, now),
		Role:             RoleParticipant,
		OwnerName:        row.OwnerName,
		Capacity:         invitableCapacity(row.Capacity),
		ParticipantCount: int(row.ParticipantCount),
		CanLeave:         row.Status == model.BookingStatusConfirmed && now.Before(row.StartsAt),
		Facility:         FacilityDetails(row.Facility, row.FacilitySport),
	}
}

func mapBooking(row model.ListBookingsByUserRow, now time.Time) BookingDto {
	var cancelUntil = row.StartsAt.Add(-time.Duration(row.MinCancelHours) * time.Hour)
	var canCancel = row.Status == model.BookingStatusConfirmed && now.Before(cancelUntil)
	var checkinCode string
	if row.Status == model.BookingStatusConfirmed && row.EndsAt.After(now) {
		checkinCode = checkin.Payload(row.CheckinToken)
	}
	var checkedInAt *time.Time
	if row.CheckedInAt.Valid {
		checkedInAt = &row.CheckedInAt.Time
	}
	return BookingDto{
		Id:            row.ID.String(),
		StartsAt:      row.StartsAt,
		EndsAt:        row.EndsAt,
		Status:        string(row.Status),
		FacilityId:    row.FacilityID.String(),
		FacilityName:  row.FacilityName,
		FacilitySlug:  row.FacilitySlug,
		TenantSlug:    row.TenantSlug,
		TenantName:    row.TenantName,
		CanCancel:     canCancel,
		CanChange:     canCancel,
		CancelUntil:   cancelUntil,
		CheckinCode:   checkinCode,
		CheckedInAt:   checkedInAt,
		Attendance:    attendance(row.Status, row.EndsAt, row.CheckedInAt.Valid, row.ReaderAtStart, now),
		Role:          RoleOwner,
		CheckinWays:   checkinWays(row),
		StaffContacts: staffContacts(row),
		Facility:      FacilityDetails(row.Facility, row.FacilitySport),
	}
}

// checkinWays: how the owner can check in at the facility, in the order the dialog shows them.
func checkinWays(row model.ListBookingsByUserRow) []string {
	var ways = []string{}
	for _, way := range []struct {
		on     bool
		method model.CheckinMethod
	}{
		{row.EntryCode, model.CheckinMethodEntryCode},
		{row.PrintedQr, model.CheckinMethodPrintedQr},
		{row.HasReader, model.CheckinMethodReader},
		{row.StaffConfirm, model.CheckinMethodStaff},
	} {
		if way.on {
			ways = append(ways, string(way.method))
		}
	}
	return ways
}

// staffContacts: who to ask on site, when the facility has staff confirmation; without
// staff contacts, the facility's own contact, else the council's.
func staffContacts(row model.ListBookingsByUserRow) []checkin.StaffContact {
	if !row.StaffConfirm {
		return nil
	}
	return checkin.StaffOrFallback(row.StaffContacts, row.FacilityName, row.Facility.ContactPhone.String, row.Facility.ContactEmail.String,
		row.TenantName, row.TenantPhone.String, row.TenantEmail.String)
}
