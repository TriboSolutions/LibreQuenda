// Package notice: what a cancellation notice stores, so the app, the web and the
// email all say what was cancelled, where, when and why.
package notice

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"librequenda/model"

	"github.com/google/uuid"
)

// Causes of a cancellation, in the notice's payload ("cause"), beyond the ones
// its type and fields already say (a closure: closed_from; staff: by_staff).
const (
	CauseOwner               = "owner"                 // booking_cancelled, to who joined: its owner cancelled it (name: their first name)
	CauseAccountDeleted      = "account_deleted"       // booking_cancelled, to who joined: its owner deleted their account
	CauseResidentRevoked     = "resident_revoked"      // booking_cancelled_by_council: the council took back their verification
	CauseResidentRevokedLeft = "resident_revoked_left" // …and they left a game they had joined there
)

// Cancelled: the booking as the notice tells it. Title: an event's name, "" otherwise.
type Cancelled struct {
	FacilityID                           uuid.UUID
	FacilityName, TenantSlug, TenantName string
	Title                                string
	StartsAt, EndsAt                     time.Time
}

// Payload: where and when, the council and the event's name, plus extra (cause, name, by_staff…).
func Payload(c Cancelled, extra map[string]any) json.RawMessage {
	var fields = map[string]any{
		"facility_id":   c.FacilityID,
		"facility_name": c.FacilityName,
		"tenant_slug":   c.TenantSlug,
		"tenant_name":   c.TenantName,
		"starts_at":     c.StartsAt,
		"ends_at":       c.EndsAt,
		"title":         c.Title,
	}
	for key, value := range extra {
		fields[key] = value
	}
	var payload, _ = json.Marshal(fields)
	return payload
}

// Tell: a notice (bell, app and email) for one person about one booking.
func Tell(ctx context.Context, queries *model.Queries, userId uuid.UUID, bookingId uuid.UUID, kind model.NotificationType, payload json.RawMessage) error {
	return queries.CreateNotification(ctx, model.CreateNotificationParams{
		UserID:    userId,
		BookingID: uuid.NullUUID{UUID: bookingId, Valid: true},
		Type:      kind,
		Payload:   payload,
	})
}

// TellParticipants: the same notice for everyone with an account who joined the booking.
func TellParticipants(ctx context.Context, queries *model.Queries, bookingId uuid.UUID, kind model.NotificationType, payload json.RawMessage) error {
	var users, err = queries.ListParticipantAccounts(ctx, bookingId)
	if err != nil {
		return err
	}
	for _, user := range users {
		if err = Tell(ctx, queries, user.UUID, bookingId, kind, payload); err != nil {
			return err
		}
	}
	return nil
}

// FirstName: how the others are named in a notice, never with surnames.
func FirstName(fullName string) string {
	if fields := strings.Fields(fullName); len(fields) > 0 {
		return fields[0]
	}
	return ""
}
