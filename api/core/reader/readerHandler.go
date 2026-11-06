package reader

import (
	"database/sql"
	"encoding/json"
	"librequenda/core/checkin"
	"librequenda/db"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"strings"
	"sync"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

type readerHandler struct {
	db *sql.DB
}

var hOnce *sync.Once = new(sync.Once)
var hInstance *readerHandler

func ReaderHandler(db *sql.DB) *readerHandler {
	hOnce.Do(func() {
		hInstance = &readerHandler{db: db}
	})
	return hInstance
}

// device signs the reader in with its key. Otherwise it answers 401 and returns false.
func (h *readerHandler) device(w http.ResponseWriter, r *http.Request) (model.GetCheckinDeviceByKeyRow, bool) {
	var key, _ = strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !checkin.IsDeviceKey(key) {
		helpers.SendI18NErrorResponse(w, r, http.StatusUnauthorized, "errors.invalid_device_key")
		return model.GetCheckinDeviceByKeyRow{}, false
	}
	var device, err = model.New(h.db).GetCheckinDeviceByKey(r.Context(), checkin.HashDeviceKey(key))
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusUnauthorized, "errors.invalid_device_key")
		return model.GetCheckinDeviceByKeyRow{}, false
	}
	// the reader acts for its own council; a failed "last seen" never blocks a scan
	var touchErr = db.WithTenant(r.Context(), h.db, device.TenantID, func(queries *model.Queries) error {
		return queries.TouchCheckinDevice(r.Context(), device.ID)
	})
	if touchErr != nil {
		shared.Logger.Error("failed to record reader %s as seen: %v", device.ID, touchErr)
	}
	return device, true
}

func (h *readerHandler) GetReader(w http.ResponseWriter, r *http.Request) {
	var device, ok = h.device(w, r)
	if !ok {
		return
	}
	helpers.SendSuccessResponse(w, MapReader(device))
}

func (h *readerHandler) CheckIn(w http.ResponseWriter, r *http.Request) {
	var device, ok = h.device(w, r)
	if !ok {
		return
	}
	var body CheckinInput
	if err := json.NewDecoder(shared.JSONBody(w, r)).Decode(&body); err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.invalid_request_body")
		return
	}
	var response, err = GetReaderService(h.db).CheckIn(r.Context(), device, body.Code)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, err.Code, err.Message, err.Args...)
		return
	}
	helpers.SendSuccessResponse(w, response)
}
