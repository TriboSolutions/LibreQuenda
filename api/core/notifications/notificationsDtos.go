package notifications

import (
	"encoding/json"
	"librequenda/model"
	"time"
)

type NotificationDto struct {
	Id           string     `json:"id"`
	Type         string     `json:"type"` // booking_confirmed, booking_reminder, booking_cancelled, booking_cancelled_by_council…
	FacilityName string     `json:"facility_name"`
	StartsAt     *time.Time `json:"starts_at"`
	EndsAt       *time.Time `json:"ends_at"`
	Reason       string     `json:"reason"`      // council cancellations
	ClosedFrom   *time.Time `json:"closed_from"` // council cancellations caused by a closure
	ClosedUntil  *time.Time `json:"closed_until"`
	TenantName   string     `json:"tenant_name"` // council_admin_granted
	TenantSlug   string     `json:"tenant_slug"`
	CreatedAt    time.Time  `json:"created_at"`
	Read         bool       `json:"read"`
}

type NotificationsResponse struct {
	UnreadCount   int64             `json:"unread_count"`
	Notifications []NotificationDto `json:"notifications"` // most recent first
}

// payload is what bookings.NotificationPayload and CancelBookingsInRange store.
type payload struct {
	FacilityName string     `json:"facility_name"`
	StartsAt     *time.Time `json:"starts_at"`
	EndsAt       *time.Time `json:"ends_at"`
	Reason       string     `json:"reason"`
	ClosedFrom   *time.Time `json:"closed_from"`
	ClosedUntil  *time.Time `json:"closed_until"`
	TenantName   string     `json:"tenant_name"`
	TenantSlug   string     `json:"tenant_slug"`
}

func MapNotifications(rows []model.ListNotificationsByUserRow, unread int64) *NotificationsResponse {
	var response = &NotificationsResponse{UnreadCount: unread, Notifications: make([]NotificationDto, 0, len(rows))}
	for _, row := range rows {
		var data payload
		_ = json.Unmarshal(row.Payload, &data) // an unreadable payload still shows its type
		response.Notifications = append(response.Notifications, NotificationDto{
			Id:           row.ID.String(),
			Type:         string(row.Type),
			FacilityName: data.FacilityName,
			StartsAt:     data.StartsAt,
			EndsAt:       data.EndsAt,
			Reason:       data.Reason,
			ClosedFrom:   data.ClosedFrom,
			ClosedUntil:  data.ClosedUntil,
			TenantName:   data.TenantName,
			TenantSlug:   data.TenantSlug,
			CreatedAt:    row.CreatedAt,
			Read:         row.ReadAt.Valid,
		})
	}
	return response
}
