package me

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"io"
	"librequenda/core/rating"
	"librequenda/model"
	"librequenda/storage"
	"strconv"
	"time"
)

// ExportResponse: everything LibreQuenda keeps about the user (GET /me/export).
// Other people's details are left out: bookings only say how many joined.
type ExportResponse struct {
	GeneratedAt  time.Time       `json:"generated_at"`
	Account      ExportAccount   `json:"account"`
	IsSuperadmin bool            `json:"is_superadmin"`
	Roles        []TenantRoleDto `json:"roles"`
	ResidentOf   []ResidentOfDto `json:"resident_of"` // councils that verified the user as their resident
	// pending requests to be verified, with the identity data the user gave
	ResidentRequests []ExportResidentRequest `json:"resident_requests"`
	Bookings         []ExportBooking         `json:"bookings"`
	Participations   []ExportParticipation   `json:"participations"` // bookings of someone else the user joined
	Notifications    []ExportNotification    `json:"notifications"`
	Issues           []ExportIssue           `json:"issues"` // problems the user reported at facilities
	Ratings          ExportRatings           `json:"ratings"`
	Votes            []ExportVote            `json:"votes"` // answers the user gave in the councils' polls
}

type ExportVote struct {
	Council   string    `json:"council"`
	Question  string    `json:"question"`
	Answer    string    `json:"answer"`
	VotedAt   time.Time `json:"voted_at"`
	ChangedAt time.Time `json:"changed_at"`
}

func mapExportVotes(rows []model.ListVotesByUserRow) []ExportVote {
	var votes = make([]ExportVote, 0, len(rows))
	for _, v := range rows {
		votes = append(votes, ExportVote{Council: v.TenantName, Question: v.Question, Answer: v.Label, VotedAt: v.CreatedAt, ChangedAt: v.UpdatedAt})
	}
	return votes
}

// ExportResidentRequest: a request to be verified, until the council decides
// (or 15 days pass); then it is deleted.
type ExportResidentRequest struct {
	Council     string    `json:"council"`
	Dni         string    `json:"dni"`
	FullName    string    `json:"full_name"`
	PostalCode  string    `json:"postal_code"`
	Street      string    `json:"street"`
	RequestedAt time.Time `json:"requested_at"`
}

func mapExportResidentRequests(rows []model.ListResidentRequestsByUserRow) []ExportResidentRequest {
	var out = make([]ExportResidentRequest, 0, len(rows))
	for _, r := range rows {
		out = append(out, ExportResidentRequest{Council: r.Name, Dni: r.Dni, FullName: r.FullName, PostalCode: r.PostalCode, Street: r.Street, RequestedAt: r.CreatedAt})
	}
	return out
}

// ExportRatings: the stars the user gave, and the average others gave them.
type ExportRatings struct {
	Facilities []ExportFacilityRating `json:"facilities"`
	People     []ExportPersonRating   `json:"people"`   // who was rated is left out, like other people's details
	Received   *rating.Summary        `json:"received"` // null until someone rated them
}

type ExportFacilityRating struct {
	Facility  string    `json:"facility"`
	BookingId string    `json:"booking_id"`
	Stars     int16     `json:"stars"`
	RatedAt   time.Time `json:"rated_at"`
}

type ExportPersonRating struct {
	BookingId string    `json:"booking_id"`
	Stars     int16     `json:"stars"`
	RatedAt   time.Time `json:"rated_at"`
}

func mapExportRatings(facilities []model.ExportFacilityRatingsByUserRow, people []model.ExportPeopleRatingsByUserRow, received model.GetUserRatingSummaryRow) ExportRatings {
	var ratings = ExportRatings{
		Facilities: make([]ExportFacilityRating, 0, len(facilities)),
		People:     make([]ExportPersonRating, 0, len(people)),
		Received:   rating.Of(received.Average, received.Count),
	}
	for _, r := range facilities {
		ratings.Facilities = append(ratings.Facilities, ExportFacilityRating{Facility: r.FacilityName, BookingId: r.BookingID.String(), Stars: r.Stars, RatedAt: r.UpdatedAt})
	}
	for _, r := range people {
		ratings.People = append(ratings.People, ExportPersonRating{BookingId: r.BookingID.String(), Stars: r.Stars, RatedAt: r.UpdatedAt})
	}
	return ratings
}

type ExportAccount struct {
	Id                  string    `json:"id"`
	Email               string    `json:"email"`
	EmailVerified       bool      `json:"email_verified"`
	FullName            string    `json:"full_name"`
	BirthDate           string    `json:"birth_date"` // YYYY-MM-DD, "" if never filled in
	Phone               string    `json:"phone"`
	AvatarUrl           string    `json:"avatar_url"`
	PreferredLanguage   string    `json:"preferred_language"`
	EmailReminders      bool      `json:"email_reminders"`
	EmailNotifications  bool      `json:"email_notifications"`
	ShowRecommendations bool      `json:"show_recommendations"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type ExportBooking struct {
	PublicDescription string     `json:"public_description"`
	Id                string     `json:"id"`
	Council           string     `json:"council"`
	Facility          string     `json:"facility"`
	StartsAt          time.Time  `json:"starts_at"`
	EndsAt            time.Time  `json:"ends_at"`
	Status            string     `json:"status"`
	RulesAcceptedAt   *time.Time `json:"rules_accepted_at"`
	CancelledAt       *time.Time `json:"cancelled_at"`
	CancelledBy       string     `json:"cancelled_by"` // user or council, "" if not cancelled
	CheckedInAt       *time.Time `json:"checked_in_at"`
	CreatedAt         time.Time  `json:"created_at"`
	ParticipantCount  int64      `json:"participant_count"` // people who joined through the invite link
}

type ExportParticipation struct {
	BookingId string    `json:"booking_id"`
	Council   string    `json:"council"`
	Facility  string    `json:"facility"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	Status    string    `json:"status"`
	JoinedAt  time.Time `json:"joined_at"`
}

type ExportIssue struct {
	Id           string     `json:"id"`
	Facility     string     `json:"facility"`
	Category     string     `json:"category"`
	Description  string     `json:"description"`
	Urgent       bool       `json:"urgent"`
	ShareContact bool       `json:"share_contact"` // the council's staff could see the email and profile phone
	Status       string     `json:"status"`
	StaffNote    string     `json:"staff_note"`
	CreatedAt    time.Time  `json:"created_at"`
	ResolvedAt   *time.Time `json:"resolved_at"`
}

func mapExportIssues(rows []model.ListIssuesByReporterRow) []ExportIssue {
	var issues = make([]ExportIssue, 0, len(rows))
	for _, i := range rows {
		issues = append(issues, ExportIssue{
			Id: i.ID.String(), Facility: i.FacilityName, Category: string(i.Category), Description: i.Description,
			Urgent: i.Urgent, ShareContact: i.ShareContact, Status: string(i.Status), StaffNote: i.StaffNote.String,
			CreatedAt: i.CreatedAt, ResolvedAt: optionalTime(i.ResolvedAt),
		})
	}
	return issues
}

type ExportNotification struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
	ReadAt    *time.Time      `json:"read_at"`
	EmailedAt *time.Time      `json:"emailed_at"`
}

func optionalTime(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func mapExportAccount(user model.ExportUserRow) ExportAccount {
	var account = ExportAccount{
		Id:                  user.ID.String(),
		Email:               user.Email,
		EmailVerified:       user.EmailVerified,
		FullName:            user.FullName.String,
		Phone:               user.Phone.String,
		PreferredLanguage:   user.PreferredLanguage.String,
		EmailReminders:      user.EmailReminders,
		EmailNotifications:  user.EmailNotifications,
		ShowRecommendations: user.ShowRecommendations,
		CreatedAt:           user.CreatedAt,
		UpdatedAt:           user.UpdatedAt,
	}
	if user.BirthDate.Valid {
		account.BirthDate = user.BirthDate.Time.Format("2006-01-02")
	}
	if user.AvatarRef.Valid {
		account.AvatarUrl = storage.URL(user.AvatarRef.String)
	}
	return account
}

func mapExportBookings(rows []model.ExportBookingsByUserRow) []ExportBooking {
	var bookings = make([]ExportBooking, 0, len(rows))
	for _, b := range rows {
		bookings = append(bookings, ExportBooking{
			Id:                b.ID.String(),
			Council:           b.TenantName,
			Facility:          b.FacilityName,
			StartsAt:          b.StartsAt,
			EndsAt:            b.EndsAt,
			Status:            string(b.Status),
			RulesAcceptedAt:   optionalTime(b.RulesAcceptedAt),
			CancelledAt:       optionalTime(b.CancelledAt),
			CancelledBy:       string(b.CancelledBy.BookingCanceller),
			CheckedInAt:       optionalTime(b.CheckedInAt),
			CreatedAt:         b.CreatedAt,
			ParticipantCount:  b.ParticipantCount,
			PublicDescription: b.PublicDescription,
		})
	}
	return bookings
}

func mapExportParticipations(rows []model.ExportParticipationsByUserRow) []ExportParticipation {
	var participations = make([]ExportParticipation, 0, len(rows))
	for _, p := range rows {
		participations = append(participations, ExportParticipation{
			BookingId: p.ID.String(),
			Council:   p.TenantName,
			Facility:  p.FacilityName,
			StartsAt:  p.StartsAt,
			EndsAt:    p.EndsAt,
			Status:    string(p.Status),
			JoinedAt:  p.JoinedAt,
		})
	}
	return participations
}

func mapExportNotifications(rows []model.ExportNotificationsByUserRow) []ExportNotification {
	var notifications = make([]ExportNotification, 0, len(rows))
	for _, n := range rows {
		notifications = append(notifications, ExportNotification{
			Type:      string(n.Type),
			Payload:   n.Payload,
			CreatedAt: n.CreatedAt,
			ReadAt:    optionalTime(n.ReadAt),
			EmailedAt: optionalTime(n.EmailedAt),
		})
	}
	return notifications
}

// --- CSV --------------------------------------------------------------------

func csvTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func csvOptionalTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return csvTime(*t)
}

// writeExportCSV writes the export as one file of several tables, each under
// its own title and header row, with a blank line between them. The BOM lets
// Excel read it as UTF-8.
func writeExportCSV(w io.Writer, data *ExportResponse) error {
	if _, err := io.WriteString(w, "\uFEFF"); err != nil {
		return err
	}
	var out = csv.NewWriter(w)
	var a = data.Account
	out.Write([]string{"account"})
	out.Write([]string{"field", "value"})
	for _, row := range [][]string{
		{"id", a.Id},
		{"email", a.Email},
		{"email_verified", strconv.FormatBool(a.EmailVerified)},
		{"full_name", a.FullName},
		{"birth_date", a.BirthDate},
		{"phone", a.Phone},
		{"avatar_url", a.AvatarUrl},
		{"preferred_language", a.PreferredLanguage},
		{"email_reminders", strconv.FormatBool(a.EmailReminders)},
		{"email_notifications", strconv.FormatBool(a.EmailNotifications)},
		{"show_recommendations", strconv.FormatBool(a.ShowRecommendations)},
		{"is_superadmin", strconv.FormatBool(data.IsSuperadmin)},
		{"created_at", csvTime(a.CreatedAt)},
		{"updated_at", csvTime(a.UpdatedAt)},
		{"generated_at", csvTime(data.GeneratedAt)},
	} {
		out.Write(row)
	}

	out.Write(nil)
	out.Write([]string{"roles"})
	out.Write([]string{"council", "role"})
	for _, r := range data.Roles {
		out.Write([]string{r.TenantName, r.Role})
	}

	out.Write(nil)
	out.Write([]string{"bookings"})
	out.Write([]string{"id", "council", "facility", "starts_at", "ends_at", "status", "rules_accepted_at", "cancelled_at", "cancelled_by", "checked_in_at", "created_at", "participant_count", "public_description"})
	for _, b := range data.Bookings {
		out.Write([]string{b.Id, b.Council, b.Facility, csvTime(b.StartsAt), csvTime(b.EndsAt), b.Status, csvOptionalTime(b.RulesAcceptedAt), csvOptionalTime(b.CancelledAt), b.CancelledBy, csvOptionalTime(b.CheckedInAt), csvTime(b.CreatedAt), strconv.FormatInt(b.ParticipantCount, 10), b.PublicDescription})
	}

	out.Write(nil)
	out.Write([]string{"participations"})
	out.Write([]string{"booking_id", "council", "facility", "starts_at", "ends_at", "status", "joined_at"})
	for _, p := range data.Participations {
		out.Write([]string{p.BookingId, p.Council, p.Facility, csvTime(p.StartsAt), csvTime(p.EndsAt), p.Status, csvTime(p.JoinedAt)})
	}

	out.Write(nil)
	out.Write([]string{"notifications"})
	out.Write([]string{"type", "created_at", "read_at", "emailed_at", "payload"})
	for _, n := range data.Notifications {
		out.Write([]string{n.Type, csvTime(n.CreatedAt), csvOptionalTime(n.ReadAt), csvOptionalTime(n.EmailedAt), string(n.Payload)})
	}

	out.Write(nil)
	out.Write([]string{"resident_requests"})
	out.Write([]string{"council", "dni", "full_name", "postal_code", "street", "requested_at"})
	for _, q := range data.ResidentRequests {
		out.Write([]string{q.Council, q.Dni, q.FullName, q.PostalCode, q.Street, csvTime(q.RequestedAt)})
	}

	out.Write(nil)
	out.Write([]string{"issues"})
	out.Write([]string{"id", "facility", "category", "description", "urgent", "share_contact", "status", "staff_note", "created_at", "resolved_at"})
	for _, i := range data.Issues {
		out.Write([]string{i.Id, i.Facility, i.Category, i.Description, strconv.FormatBool(i.Urgent), strconv.FormatBool(i.ShareContact), i.Status, i.StaffNote, csvTime(i.CreatedAt), csvOptionalTime(i.ResolvedAt)})
	}

	out.Write(nil)
	out.Write([]string{"votes"})
	out.Write([]string{"council", "question", "answer", "voted_at", "changed_at"})
	for _, v := range data.Votes {
		out.Write([]string{v.Council, v.Question, v.Answer, csvTime(v.VotedAt), csvTime(v.ChangedAt)})
	}
	out.Flush()
	return out.Error()
}
