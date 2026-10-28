package invites

import (
	"database/sql"
	"librequenda/core/bookings"
	"librequenda/model"
	"strings"
	"time"
	"unicode/utf8"
)

// --- owner -----------------------------------------------------------------

type ParticipantDto struct {
	Id       string    `json:"id"`
	Name     string    `json:"name"`
	IsGuest  bool      `json:"is_guest"`
	Age      *int      `json:"age"` // guests only: people with an account keep their date of birth private
	JoinedAt time.Time `json:"joined_at"`
}

type InviteDto struct {
	Token        string           `json:"token"` // "" until the owner shares the booking
	Capacity     int              `json:"capacity"`
	SpotsLeft    int              `json:"spots_left"`
	State        string           `json:"state"`
	Participants []ParticipantDto `json:"participants"`
}

// --- public invite page ----------------------------------------------------

type PublicInviteDto struct {
	FacilityName string    `json:"facility_name"`
	FacilitySlug string    `json:"facility_slug"`
	TenantSlug   string    `json:"tenant_slug"`
	TenantName   string    `json:"tenant_name"`
	Timezone     string    `json:"timezone"`
	OwnerName    string    `json:"owner_name"` // first name only
	StartsAt     time.Time `json:"starts_at"`
	EndsAt       time.Time `json:"ends_at"`
	Capacity     int       `json:"capacity"`
	SpotsLeft    int       `json:"spots_left"`
	State        string    `json:"state"`
	// image, address, rules and minimum age: people see them before joining
	Facility bookings.FacilityDetailsDto `json:"facility"`
}

type GuestJoinDto struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

const (
	maxGuestNameLength = 80
	maxGuestAge        = 120
)

// Validate trims the name and answers the i18n key of the first problem, or "".
func (d *GuestJoinDto) Validate() string {
	d.Name = strings.Join(strings.Fields(d.Name), " ")
	if d.Name == "" || utf8.RuneCountInString(d.Name) > maxGuestNameLength {
		return "errors.invalid_guest_name"
	}
	if d.Age < 1 || d.Age > maxGuestAge {
		return "errors.invalid_guest_age"
	}
	return ""
}

// --- rules -----------------------------------------------------------------

const (
	StateOpen   = "open"
	StateFull   = "full"
	StateClosed = "closed" // cancelled, started, or the facility no longer takes invites
)

// Capacity counts the owner, so a facility for one person takes no invites.
func invitable(capacity sql.NullInt32) bool {
	return capacity.Valid && capacity.Int32 > 1
}

func spotsLeft(capacity sql.NullInt32, participants int64) int {
	if !invitable(capacity) {
		return 0
	}
	return max(int(capacity.Int32)-1-int(participants), 0)
}

func inviteState(status model.BookingStatus, startsAt time.Time, capacity sql.NullInt32, participants int64, now time.Time) string {
	if status != model.BookingStatusConfirmed || !now.Before(startsAt) || !invitable(capacity) {
		return StateClosed
	}
	if spotsLeft(capacity, participants) == 0 {
		return StateFull
	}
	return StateOpen
}

func firstName(fullName string) string {
	var name, _, _ = strings.Cut(strings.TrimSpace(fullName), " ")
	return name
}

func mapParticipants(rows []model.ListParticipantsByBookingRow) []ParticipantDto {
	var participants = make([]ParticipantDto, 0, len(rows))
	for _, row := range rows {
		var participant = ParticipantDto{Id: row.ID.String(), Name: row.Name, IsGuest: !row.UserID.Valid, JoinedAt: row.JoinedAt}
		if row.GuestAge.Valid {
			var age = int(row.GuestAge.Int16)
			participant.Age = &age
		}
		participants = append(participants, participant)
	}
	return participants
}
