package counciladmin

import (
	"librequenda/core/bookings"
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
