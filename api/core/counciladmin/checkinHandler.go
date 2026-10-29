package counciladmin

import (
	"net/http"

	"github.com/google/uuid"
)

// --- check-in from the resident's phone (Entrada tab) -----------------------

func (h *counciladminHandler) GetCheckinSettings(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var id uuid.UUID
	if id, ok = facilityId(w, r); !ok {
		return
	}
	var response, err = GetCounciladminService(h.db).GetCheckinSettings(r.Context(), tenantId, id)
	send(w, r, response, err)
}

func (h *counciladminHandler) SaveCheckinSettings(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var id uuid.UUID
	if id, ok = facilityId(w, r); !ok {
		return
	}
	var input CheckinSettingsInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetCounciladminService(h.db).SaveCheckinSettings(r.Context(), tenantId, id, input)
	send(w, r, response, err)
}

func (h *counciladminHandler) RotateCheckinQr(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var id uuid.UUID
	if id, ok = facilityId(w, r); !ok {
		return
	}
	var response, err = GetCounciladminService(h.db).RotateCheckinQr(r.Context(), tenantId, id)
	send(w, r, response, err)
}

func (h *counciladminHandler) RotateCheckinCode(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var id uuid.UUID
	if id, ok = facilityId(w, r); !ok {
		return
	}
	var response, err = GetCounciladminService(h.db).RotateCheckinCode(r.Context(), tenantId, id)
	send(w, r, response, err)
}
