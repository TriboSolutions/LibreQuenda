package bookings

import (
	"librequenda/checkin"
	"librequenda/model"
	"librequenda/schedule"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// StaffBookingInput: council staff book a free turn for someone who phoned or
// came to the desk, either for their account (Email) or by name (Name, Phone).
type StaffBookingInput struct {
	FacilityId  uuid.UUID `json:"facility_id"`
	StartsAt    time.Time `json:"starts_at"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	Phone       string    `json:"phone"`
	AcceptRules bool      `json:"accept_rules"` // staff told the person the facility's usage rules
}

// validate trims the input and returns an error key, or "".
func (d *StaffBookingInput) validate() string {
	d.Email = strings.TrimSpace(d.Email)
	d.Name = strings.Join(strings.Fields(d.Name), " ")
	d.Phone = strings.TrimSpace(d.Phone)
	switch {
	case d.FacilityId == uuid.Nil || d.StartsAt.IsZero():
		return "errors.invalid_request_body"
	case !d.AcceptRules:
		return "errors.rules_not_accepted"
	case (d.Email == "") == (d.Name == ""):
		return "errors.staff_booking_holder" // an account or a name, exactly one
	case d.Email != "":
		if _, err := mail.ParseAddress(d.Email); err != nil {
			return "errors.invalid_email"
		}
	case utf8.RuneCountInString(d.Name) > 80 || utf8.RuneCountInString(d.Phone) > 30:
		return "errors.staff_booking_holder_too_long"
	}
	return ""
}

// StaffCancelInput: why staff cancel, told to the resident if they have an account.
type StaffCancelInput struct {
	Reason string `json:"reason"`
}

// StaffDayDto: one facility's turns on one day, and who holds each booking.
type StaffDayDto struct {
	Date        string            `json:"date"` // YYYY-MM-DD in the council's time zone
	Timezone    string            `json:"timezone"`
	FacilityId  string            `json:"facility_id"`
	MinAge      *int32            `json:"min_age"` // a reminder for staff: nobody checks the age of someone without an account
	SlotMinutes int32             `json:"slot_minutes"`
	Slots       []StaffSlotDto    `json:"slots"`
	Bookings    []StaffBookingDto `json:"bookings"` // confirmed, by start
}

type StaffSlotDto struct {
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	Status    string    `json:"status"`     // free, booked, closed, past, too_far
	BookingId string    `json:"booking_id"` // the booking that takes the turn, "" when none
}

// StaffBookingDto: what staff see of a booking. Contact details are there so the
// council can call about a closure or a cancellation.
type StaffBookingDto struct {
	Id               string     `json:"id"`
	StartsAt         time.Time  `json:"starts_at"`
	EndsAt           time.Time  `json:"ends_at"`
	BookedByStaff    bool       `json:"booked_by_staff"`
	HasAccount       bool       `json:"has_account"`
	Name             string     `json:"name"` // "" once a deleted account or a forgotten name
	Email            string     `json:"email"`
	Phone            string     `json:"phone"`
	CheckedInAt      *time.Time `json:"checked_in_at"`
	ParticipantCount int64      `json:"participant_count"`
	CanCancel        bool       `json:"can_cancel"`   // until it starts
	CheckinCode      string     `json:"checkin_code"` // booked by staff only: to print the entry QR for the person
}

func mapStaffBooking(row model.ListFacilityBookingsForStaffRow, now time.Time) StaffBookingDto {
	var dto = StaffBookingDto{
		Id:               row.ID.String(),
		StartsAt:         row.StartsAt,
		EndsAt:           row.EndsAt,
		BookedByStaff:    row.Channel == model.BookingChannelStaff,
		HasAccount:       row.UserID.Valid,
		Name:             row.FullName.String,
		Email:            row.Email.String,
		Phone:            row.Phone.String,
		ParticipantCount: row.ParticipantCount,
		CanCancel:        row.StartsAt.After(now),
	}
	if !row.UserID.Valid {
		dto.Name = row.HolderName.String
		dto.Phone = row.HolderPhone.String
	} else if dto.Name == "" {
		dto.Name = dto.Email
	}
	if row.CheckedInAt.Valid {
		dto.CheckedInAt = &row.CheckedInAt.Time
	}
	if dto.BookedByStaff && row.EndsAt.After(now) {
		dto.CheckinCode = checkin.Payload(row.CheckinToken)
	}
	return dto
}

// staffSlots joins each turn with the booking that overlaps it.
func staffSlots(slots []schedule.Slot, bookings []StaffBookingDto) []StaffSlotDto {
	var out = make([]StaffSlotDto, 0, len(slots))
	for _, slot := range slots {
		var dto = StaffSlotDto{StartsAt: slot.Start, EndsAt: slot.End, Status: string(slot.Status)}
		for _, b := range bookings {
			if b.StartsAt.Before(slot.End) && b.EndsAt.After(slot.Start) {
				dto.BookingId = b.Id
				break
			}
		}
		out = append(out, dto)
	}
	return out
}
