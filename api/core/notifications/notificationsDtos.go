package notifications

import (
	"encoding/json"
	"librequenda/model"
	"librequenda/shared"
	"time"

	"github.com/google/uuid"
)

type NotificationDto struct {
	Name         string     `json:"name"`
	Id           string     `json:"id"`
	BookingId    string     `json:"booking_id"` // "" for notices about no booking
	PollId       string     `json:"poll_id"`    // poll_opened: the poll to vote in
	IsPublic     bool       `json:"is_public"`  // rating_requested: the others can be rated too
	Type         string     `json:"type"`       // booking_confirmed, booking_reminder, booking_cancelled, booking_cancelled_by_council…
	FacilityName string     `json:"facility_name"`
	FacilitySlug string     `json:"facility_slug"` // issue_solved, issue_rejected
	StartsAt     *time.Time `json:"starts_at"`
	EndsAt       *time.Time `json:"ends_at"`
	Reason       string     `json:"reason"`      // council cancellations; the council's note on an issue
	ClosedFrom   *time.Time `json:"closed_from"` // council cancellations caused by a closure
	ClosedUntil  *time.Time `json:"closed_until"`
	ByStaff      bool       `json:"by_staff"`    // council cancellation of one booking by staff
	Cause        string     `json:"cause"`       // why it was cancelled, beyond the type (notice.Cause*); "" otherwise
	Title        string     `json:"title"`       // the event's name, when the booking was one
	TenantName   string     `json:"tenant_name"` // council_admin_granted
	TenantSlug   string     `json:"tenant_slug"`
	CreatedAt    time.Time  `json:"created_at"`
	Read         bool       `json:"read"`
}

type NotificationsResponse struct {
	UnreadCount   int64             `json:"unread_count"`
	Notifications []NotificationDto `json:"notifications"` // most recent first
	Pagination    shared.PageDto    `json:"pagination"`
}

// payload is what bookings.NotificationPayload, CancelBookingsInRange,
// bookings.StaffCancel and counciladmin.ChangeIssue store.
type payload struct {
	Name         string     `json:"name"`
	FacilityName string     `json:"facility_name"`
	FacilitySlug string     `json:"facility_slug"`
	StartsAt     *time.Time `json:"starts_at"`
	EndsAt       *time.Time `json:"ends_at"`
	Reason       string     `json:"reason"`
	ClosedFrom   *time.Time `json:"closed_from"`
	ClosedUntil  *time.Time `json:"closed_until"`
	ByStaff      bool       `json:"by_staff"`
	TenantName   string     `json:"tenant_name"`
	TenantSlug   string     `json:"tenant_slug"`
	IsPublic     bool       `json:"is_public"`
	Cause        string     `json:"cause"`
	Title        string     `json:"title"`
}

func MapNotifications(rows []model.ListNotificationsByUserRow, unread int64) *NotificationsResponse {
	var response = &NotificationsResponse{UnreadCount: unread, Notifications: make([]NotificationDto, 0, len(rows))}
	for _, row := range rows {
		var data payload
		_ = json.Unmarshal(row.Payload, &data) // an unreadable payload still shows its type
		response.Notifications = append(response.Notifications, NotificationDto{
			Id:           row.ID.String(),
			BookingId:    bookingId(row.BookingID),
			PollId:       bookingId(row.PollID),
			IsPublic:     data.IsPublic,
			Name:         data.Name,
			Type:         string(row.Type),
			FacilityName: data.FacilityName,
			FacilitySlug: data.FacilitySlug,
			StartsAt:     data.StartsAt,
			EndsAt:       data.EndsAt,
			Reason:       data.Reason,
			ClosedFrom:   data.ClosedFrom,
			ClosedUntil:  data.ClosedUntil,
			ByStaff:      data.ByStaff,
			Cause:        data.Cause,
			Title:        data.Title,
			TenantName:   data.TenantName,
			TenantSlug:   data.TenantSlug,
			CreatedAt:    row.CreatedAt,
			Read:         row.ReadAt.Valid,
		})
	}
	return response
}

func bookingId(id uuid.NullUUID) string {
	if !id.Valid {
		return ""
	}
	return id.UUID.String()
}
