package bookings

import (
	"database/sql"
	"librequenda/model"
)

// Places: how many people can join a booking. A resident's booking counts its
// owner in the capacity, so a facility for one person takes nobody else; an
// event has no owner, so all of its capacity is for those who join. 0 without
// a capacity.
func Places(capacity sql.NullInt32, kind model.BookingKind) int {
	if !capacity.Valid {
		return 0
	}
	if kind == model.BookingKindEvent {
		return max(int(capacity.Int32), 0)
	}
	return max(int(capacity.Int32)-1, 0)
}

// kindOf: a resident's own booking is public or private, never an event.
func kindOf(public bool) string {
	if public {
		return string(model.BookingKindPublic)
	}
	return string(model.BookingKindPrivate)
}
