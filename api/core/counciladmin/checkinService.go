package counciladmin

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"librequenda/checkin"
	"librequenda/db"
	"librequenda/model"
	"librequenda/shared"
	"net/http"

	"github.com/google/uuid"
)

// checkinSettings creates the facility's settings with the defaults the first time, and reads them.
func (s *counciladminService) checkinSettings(ctx context.Context, tenantId uuid.UUID, facilityId uuid.UUID) (model.GetFacilityCheckinRow, *shared.ErrorResponse) {
	var row model.GetFacilityCheckinRow
	var err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		if ensureErr := queries.EnsureFacilityCheckin(ctx, model.EnsureFacilityCheckinParams{FacilityID: facilityId, TenantID: tenantId}); ensureErr != nil {
			return ensureErr
		}
		var getErr error
		row, getErr = queries.GetFacilityCheckin(ctx, model.GetFacilityCheckinParams{FacilityID: facilityId, TenantID: tenantId})
		return getErr
	})
	if errors.Is(err, sql.ErrNoRows) {
		return row, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	}
	if err != nil {
		return row, failed("errors.checkin_settings_failed")
	}
	return row, nil
}

func mapCheckinSettings(row model.GetFacilityCheckinRow, hasLocation bool) *CheckinSettingsDto {
	return &CheckinSettingsDto{
		PrintedQr:     row.PrintedQr,
		QrToken:       row.QrToken.String(),
		Nfc:           row.Nfc,
		NfcToken:      row.NfcToken.String(),
		EntryCode:     row.EntryCode,
		Code:          checkin.EntryCode(row.CodeSecret),
		RadiusM:       int(row.RadiusM),
		FlashBooking:  row.FlashBooking,
		StaffConfirm:  row.StaffConfirm,
		QrReader:      row.QrReader,
		StaffContacts: checkin.StaffContacts(row.StaffContacts),
		HasLocation:   hasLocation,
	}
}

func (s *counciladminService) GetCheckinSettings(ctx context.Context, tenantId uuid.UUID, facilityId uuid.UUID) (*CheckinSettingsDto, *shared.ErrorResponse) {
	var facility, notFound = s.GetFacility(ctx, tenantId, facilityId)
	if notFound != nil {
		return nil, notFound
	}
	var row, err = s.checkinSettings(ctx, tenantId, facilityId)
	if err != nil {
		return nil, err
	}
	return mapCheckinSettings(row, facility.Latitude != nil), nil
}

// SaveCheckinSettings: the printed QR and the entry code need the facility's pin,
// since the phone's location is checked against it.
func (s *counciladminService) SaveCheckinSettings(ctx context.Context, tenantId uuid.UUID, facilityId uuid.UUID, input CheckinSettingsInput) (*CheckinSettingsDto, *shared.ErrorResponse) {
	var facility, notFound = s.GetFacility(ctx, tenantId, facilityId)
	if notFound != nil {
		return nil, notFound
	}
	var hasLocation = facility.Latitude != nil
	if key := input.validate(hasLocation); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	var contacts, contactsErr = checkin.CleanStaffContacts(input.StaffContacts)
	if contactsErr != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: contactsErr}
	}
	var contactsJson, _ = json.Marshal(contacts)
	if _, err := s.checkinSettings(ctx, tenantId, facilityId); err != nil {
		return nil, err
	}
	var row model.UpdateFacilityCheckinRow
	var err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		var updateErr error
		row, updateErr = queries.UpdateFacilityCheckin(ctx, model.UpdateFacilityCheckinParams{
			PrintedQr:     input.PrintedQr,
			Nfc:           input.Nfc,
			EntryCode:     input.EntryCode,
			RadiusM:       int32(input.RadiusM),
			FlashBooking:  input.FlashBooking,
			StaffConfirm:  input.StaffConfirm,
			QrReader:      input.QrReader,
			StaffContacts: contactsJson,
			FacilityID:    facilityId,
		})
		return updateErr
	})
	if err != nil {
		return nil, failed("errors.checkin_settings_failed")
	}
	return mapCheckinSettings(model.GetFacilityCheckinRow(row), hasLocation), nil
}

// RotateCheckinQr: a new address for the printed QR; the copies already on doors stop working.
func (s *counciladminService) RotateCheckinQr(ctx context.Context, tenantId uuid.UUID, facilityId uuid.UUID) (*CheckinSettingsDto, *shared.ErrorResponse) {
	var facility, notFound = s.GetFacility(ctx, tenantId, facilityId)
	if notFound != nil {
		return nil, notFound
	}
	var row, settingsErr = s.checkinSettings(ctx, tenantId, facilityId)
	if settingsErr != nil {
		return nil, settingsErr
	}
	var err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		var rotateErr error
		row.QrToken, rotateErr = queries.RotateFacilityQrToken(ctx, facilityId)
		return rotateErr
	})
	if err != nil {
		return nil, failed("errors.checkin_settings_failed")
	}
	return mapCheckinSettings(row, facility.Latitude != nil), nil
}

// RotateCheckinNfc: a new address for the NFC tags; the tags already written stop working.
func (s *counciladminService) RotateCheckinNfc(ctx context.Context, tenantId uuid.UUID, facilityId uuid.UUID) (*CheckinSettingsDto, *shared.ErrorResponse) {
	var facility, notFound = s.GetFacility(ctx, tenantId, facilityId)
	if notFound != nil {
		return nil, notFound
	}
	var row, settingsErr = s.checkinSettings(ctx, tenantId, facilityId)
	if settingsErr != nil {
		return nil, settingsErr
	}
	var err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		var rotateErr error
		row.NfcToken, rotateErr = queries.RotateFacilityNfcToken(ctx, facilityId)
		return rotateErr
	})
	if err != nil {
		return nil, failed("errors.checkin_settings_failed")
	}
	return mapCheckinSettings(row, facility.Latitude != nil), nil
}

// RotateCheckinCode: a new entry code; the one already printed stops working.
func (s *counciladminService) RotateCheckinCode(ctx context.Context, tenantId uuid.UUID, facilityId uuid.UUID) (*CheckinSettingsDto, *shared.ErrorResponse) {
	var facility, notFound = s.GetFacility(ctx, tenantId, facilityId)
	if notFound != nil {
		return nil, notFound
	}
	var row, settingsErr = s.checkinSettings(ctx, tenantId, facilityId)
	if settingsErr != nil {
		return nil, settingsErr
	}
	var err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		var rotateErr error
		row.CodeSecret, rotateErr = queries.RotateFacilityCodeSecret(ctx, facilityId)
		return rotateErr
	})
	if err != nil {
		return nil, failed("errors.checkin_settings_failed")
	}
	return mapCheckinSettings(row, facility.Latitude != nil), nil
}
