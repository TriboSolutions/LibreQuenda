package counciladmin

import (
	"github.com/google/uuid"
	"librequenda/audit"
	"librequenda/core/bookings"
	"librequenda/core/invites"
	"librequenda/model"
	"net/http"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

// The "Reservas" tab: staff see a facility's day and book or cancel for residents
// who phone or come to the desk.

func (h *counciladminHandler) GetStaffDay(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var id, idOk = facilityId(w, r)
	if !idOk {
		return
	}
	var response, err = bookings.GetBookingsService(h.db).StaffDay(r.Context(), tenantId, id, r.URL.Query().Get("date"))
	send(w, r, response, err)
}

func (h *counciladminHandler) CreateStaffBooking(w http.ResponseWriter, r *http.Request) {
	var tenantId, staffId, ok = h.councilAdminUser(w, r)
	if !ok {
		return
	}
	var input bookings.StaffBookingInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = bookings.GetBookingsService(h.db).StaffBook(r.Context(), tenantId, staffId, input)
	if err == nil {
		// who it is for stays out of the log: the booking holds it, and forgets it
		var label, _ = model.New(h.db).GetFacilityNameForAudit(r.Context(), input.FacilityId)
		audit.Target(r.Context(), "booking", uuid.MustParse(response.Id), label)
		audit.Describe(r.Context(), "turn %s, with an account: %v", response.StartsAt.Format("2006-01-02 15:04"), response.HasAccount)
	}
	send(w, r, response, err)
}

func (h *counciladminHandler) CancelStaffBooking(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var bookingId, err = helpers.GetUUidPathValue("bookingId", r)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return
	}
	var input bookings.StaffCancelInput
	if !decode(w, r, &input) {
		return
	}
	send(w, r, nil, bookings.GetBookingsService(h.db).StaffCancel(r.Context(), tenantId, bookingId, input))
}

// AddEventParticipant: staff sign someone up for one of the council's events ({email} or {name, age}).
func (h *counciladminHandler) AddEventParticipant(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var bookingId, err = helpers.GetUUidPathValue("bookingId", r)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return
	}
	var input invites.StaffJoinInput
	if !decode(w, r, &input) {
		return
	}
	send(w, r, nil, invites.GetInvitesService(h.db).StaffAddToEvent(r.Context(), tenantId, bookingId, input))
}

// EventPeople: who is in one of the council's events (name, email, stars, resident or not).
func (h *counciladminHandler) EventPeople(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var bookingId, err = helpers.GetUUidPathValue("bookingId", r)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return
	}
	var response, peopleErr = invites.GetInvitesService(h.db).EventPeople(r.Context(), tenantId, bookingId)
	send(w, r, response, peopleErr)
}

// RemoveEventParticipant: staff take someone out of one of the council's events.
func (h *counciladminHandler) RemoveEventParticipant(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var bookingId, err = helpers.GetUUidPathValue("bookingId", r)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return
	}
	var participantId uuid.UUID
	if participantId, err = helpers.GetUUidPathValue("participantId", r); err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return
	}
	send(w, r, nil, invites.GetInvitesService(h.db).StaffRemoveFromEvent(r.Context(), tenantId, bookingId, participantId))
}

func (h *counciladminHandler) StaffCheckIn(w http.ResponseWriter, r *http.Request) {
	var tenantId, staffId, ok = h.councilAdminUser(w, r)
	if !ok {
		return
	}
	var bookingId, err = helpers.GetUUidPathValue("bookingId", r)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return
	}
	var response, checkinErr = GetCounciladminService(h.db).StaffCheckIn(r.Context(), tenantId, staffId, bookingId)
	send(w, r, response, checkinErr)
}

func (h *counciladminHandler) UndoStaffCheckIn(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var bookingId, err = helpers.GetUUidPathValue("bookingId", r)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return
	}
	var response, undoErr = GetCounciladminService(h.db).UndoStaffCheckIn(r.Context(), tenantId, bookingId)
	send(w, r, response, undoErr)
}

// The "Eventos" tab: the council's own activities, which residents join.

func (h *counciladminHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var response, err = bookings.GetBookingsService(h.db).ListEvents(r.Context(), tenantId)
	send(w, r, response, err)
}

// PreviewEvent: the bookings and closures in the way of an event at that time
// (?except=<event id> while moving one).
func (h *counciladminHandler) PreviewEvent(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var except uuid.NullUUID
	if raw := r.URL.Query().Get("except"); raw != "" {
		var id, err = uuid.Parse(raw)
		if err != nil {
			helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
			return
		}
		except = uuid.NullUUID{UUID: id, Valid: true}
	}
	var input bookings.EventInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = bookings.GetBookingsService(h.db).PreviewEvent(r.Context(), tenantId, input, except)
	send(w, r, response, err)
}

func (h *counciladminHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var tenantId, staffId, ok = h.councilAdminUser(w, r)
	if !ok {
		return
	}
	var input bookings.EventInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = bookings.GetBookingsService(h.db).CreateEvent(r.Context(), tenantId, staffId, input)
	if err == nil {
		audit.Target(r.Context(), "booking", uuid.MustParse(response.Id), response.Title)
		audit.Describe(r.Context(), "%s, %s – %s", response.FacilityName, response.StartsAt.Format("2006-01-02 15:04"), response.EndsAt.Format("2006-01-02 15:04"))
	}
	send(w, r, response, err)
}

func (h *counciladminHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var eventId, idErr = helpers.GetUUidPathValue("bookingId", r)
	if idErr != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_query_params")
		return
	}
	var input bookings.EventInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = bookings.GetBookingsService(h.db).UpdateEvent(r.Context(), tenantId, eventId, input)
	if err == nil {
		audit.Describe(r.Context(), "%s: %s – %s", response.Title, response.StartsAt.Format("2006-01-02 15:04"), response.EndsAt.Format("2006-01-02 15:04"))
	}
	send(w, r, response, err)
}
