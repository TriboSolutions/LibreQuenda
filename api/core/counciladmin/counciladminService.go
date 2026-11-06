package counciladmin

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/core/checkin"
	"librequenda/db"
	"librequenda/model"
	"librequenda/shared"
	"librequenda/storage"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type counciladminService struct {
	db *sql.DB
}

var sInstance *counciladminService
var sOnce *sync.Once = new(sync.Once)

func GetCounciladminService(db *sql.DB) *counciladminService {
	sOnce.Do(func() {
		sInstance = &counciladminService{db: db}
	})
	return sInstance
}

func failed(message string) *shared.ErrorResponse {
	return &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: message}
}

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func nullFloat(value *float64) sql.NullFloat64 {
	if value == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *value, Valid: true}
}

func nullInt(value *int32) sql.NullInt32 {
	if value == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: *value, Valid: true}
}

func floatPtr(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	return &value.Float64
}

func intPtr(value sql.NullInt32) *int32 {
	if !value.Valid {
		return nil
	}
	return &value.Int32
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

var errUnknownTag = errors.New("a tag is not in the catalogue")

// setFacilityTags replaces the facility's tags, inside the council's transaction.
// A slug the catalogue does not have (a superadministrator removed it meanwhile)
// fails the whole save.
func setFacilityTags(ctx context.Context, queries *model.Queries, facilityId uuid.UUID, slugs []string) error {
	if err := queries.ClearFacilityTags(ctx, facilityId); err != nil {
		return err
	}
	if len(slugs) == 0 {
		return nil
	}
	var added, err = queries.AddFacilityTags(ctx, model.AddFacilityTagsParams{FacilityID: facilityId, Slugs: slugs})
	if err != nil {
		return err
	}
	if added != int64(len(slugs)) {
		return errUnknownTag
	}
	return nil
}

func (s *counciladminService) ListFacilities(ctx context.Context, tenantId uuid.UUID) (*FacilitiesResponse, *shared.ErrorResponse) {
	var rows, err = model.New(s.db).ListFacilitiesForAdmin(ctx, tenantId)
	if err != nil {
		return nil, failed("errors.day_load_failed")
	}
	var response = &FacilitiesResponse{Facilities: make([]FacilitySummaryDto, 0, len(rows))}
	for _, f := range rows {
		response.Facilities = append(response.Facilities, FacilitySummaryDto{
			Id:                   f.ID.String(),
			Slug:                 f.Slug,
			Name:                 f.Name,
			Sport:                f.Sport,
			Tags:                 append([]string{}, f.Tags...),
			ImageUrl:             storage.URL(f.ImageRef.String),
			IsActive:             f.IsActive,
			OpeningWindowCount:   f.OpeningWindowCount,
			UpcomingBookingCount: f.UpcomingBookingCount,
		})
	}
	return response, nil
}

func (s *counciladminService) GetFacility(ctx context.Context, tenantId uuid.UUID, id uuid.UUID) (*FacilityDetailDto, *shared.ErrorResponse) {
	var queries = model.New(s.db)
	var f, err = queries.GetFacilityForAdmin(ctx, model.GetFacilityForAdminParams{ID: id, TenantID: tenantId})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	}
	if err != nil {
		return nil, failed("errors.day_load_failed")
	}
	var bookings int64
	bookings, err = queries.CountBookingsByFacility(ctx, f.ID)
	if err != nil {
		return nil, failed("errors.day_load_failed")
	}
	var toCancel int64
	toCancel, err = queries.CountBookingsToCancel(ctx, model.CountBookingsToCancelParams{FacilityID: f.ID, RangeStart: time.Now(), RangeEnd: farFuture})
	if err != nil {
		return nil, failed("errors.day_load_failed")
	}
	var closures []model.ListUpcomingClosuresRow
	closures, err = queries.ListUpcomingClosures(ctx, f.ID)
	if err != nil {
		return nil, failed("errors.day_load_failed")
	}
	var timezone string
	timezone, err = queries.GetFacilityTimezone(ctx, model.GetFacilityTimezoneParams{FacilityID: f.ID, TenantID: tenantId})
	if err != nil {
		return nil, failed("errors.day_load_failed")
	}
	var hours []model.ListOpeningHoursByFacilityRow
	hours, err = queries.ListOpeningHoursByFacility(ctx, f.ID)
	if err != nil {
		return nil, failed("errors.day_load_failed")
	}
	var tags []string
	tags, err = queries.ListFacilityTagSlugs(ctx, f.ID)
	if err != nil {
		return nil, failed("errors.day_load_failed")
	}
	if tags == nil {
		tags = []string{}
	}
	var detail = &FacilityDetailDto{
		ShowWeather:       f.ShowWeather,
		Id:                f.ID.String(),
		Slug:              f.Slug,
		Name:              f.Name,
		Description:       f.Description.String,
		Tags:              tags,
		Address:           f.Address.String,
		Latitude:          floatPtr(f.Latitude),
		Longitude:         floatPtr(f.Longitude),
		Capacity:          intPtr(f.Capacity),
		AccessibilityInfo: f.AccessibilityInfo.String,
		ContactPhone:      f.ContactPhone.String,
		ContactEmail:      f.ContactEmail.String,
		ImageRef:          f.ImageRef.String,
		ImageUrl:          storage.URL(f.ImageRef.String),
		IsActive:          f.IsActive,
		SlotMinutes:       f.SlotMinutes,
		CustomRules:       f.CustomRules,
		Rules:             rulesOfFacility(f),
		MinAge:            intPtr(f.MinAge),
		ResidentsOnly:     f.ResidentsOnly,
		UsageRules:        f.UsageRulesText.String,
		BookingCount:      bookings,
		BookingsToCancel:  toCancel,
		Schedule:          make([]WindowDto, 0, len(hours)),
		Closures:          make([]ClosureDto, 0, len(closures)),
		Timezone:          timezone,
	}
	for _, c := range closures {
		detail.Closures = append(detail.Closures, mapClosure(c.ID, c.StartsAt, c.EndsAt, c.Reason))
	}
	for _, h := range hours {
		detail.Schedule = append(detail.Schedule, WindowDto{Weekday: h.Weekday, OpensAt: h.OpensAt.Format("15:04"), ClosesAt: h.ClosesAt.Format("15:04")})
	}
	if council, rulesErr := s.councilRules(ctx, tenantId); rulesErr == nil {
		detail.CouncilRules = council
	}
	return detail, nil
}

func (s *counciladminService) CreateFacility(ctx context.Context, tenantId uuid.UUID, input FacilityInput) (*FacilityDetailDto, *shared.ErrorResponse) {
	input.normalise()
	if key := input.validate(); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	// only an image uploaded for this council
	if input.ImageRef != "" && !storage.Owned(input.ImageRef, tenantId, "facilities") {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.invalid_image"}
	}
	// a new facility starts with the council's booking window (its Reserva tab changes it)
	var rules, rulesErr = s.councilRules(ctx, tenantId)
	if rulesErr != nil {
		return nil, rulesErr
	}
	var windows []parsedWindow
	if input.Schedule != nil {
		var key string
		if windows, key = input.Schedule.parse(); key != "" {
			return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
		}
	}
	var id uuid.UUID
	var err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		var createErr error
		id, createErr = queries.CreateFacility(ctx, model.CreateFacilityParams{
			TenantID:                 tenantId,
			Slug:                     input.Slug,
			Name:                     input.Name,
			Description:              nullString(input.Description),
			Address:                  nullString(input.Address),
			Latitude:                 nullFloat(input.Latitude),
			Longitude:                nullFloat(input.Longitude),
			Capacity:                 nullInt(input.Capacity),
			AccessibilityInfo:        nullString(input.AccessibilityInfo),
			ContactPhone:             nullString(input.ContactPhone),
			ContactEmail:             nullString(input.ContactEmail),
			ImageRef:                 nullString(input.ImageRef),
			ShowWeather:              input.ShowWeather,
			IsActive:                 input.IsActive,
			SlotMinutes:              input.SlotMinutes,
			CustomRules:              false,
			MaxAdvanceHours:          rules.MaxAdvanceHours,
			MinAdvanceMinutes:        rules.MinAdvanceMinutes,
			MinCancelHours:           rules.MinCancelHours,
			MaxActiveBookingsPerUser: rules.MaxActiveBookingsPerUser,
			MaxBookingsPerDay:        rules.MaxBookingsPerDay,
			MaxTurnsInARow:           rules.MaxTurnsInARow,
			NoShowPenalty:            rules.NoShowPenalty,
			NoShowLimit:              rules.NoShowLimit,
			NoShowWindowDays:         rules.NoShowWindowDays,
			NoShowBlockDays:          rules.NoShowBlockDays,
			MaxBookingsPerWeek:       rules.MaxBookingsPerWeek,
			WeeklyOpening:            rules.WeeklyOpening,
			OpensWeekday:             rules.OpensWeekday,
			OpensMinute:              rules.OpensMinute,
			MinAge:                   nullInt(input.MinAge),
			ResidentsOnly:            input.ResidentsOnly,
			UsageRulesText:           nullString(input.UsageRules),
		})
		if createErr != nil {
			return createErr
		}
		if err := insertWindows(ctx, queries, id, windows); err != nil {
			return err
		}
		return setFacilityTags(ctx, queries, id, input.Tags)
	})
	if errors.Is(err, errUnknownTag) {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.invalid_tags"}
	}
	if isUniqueViolation(err) {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.facility_slug_taken"}
	}
	if err != nil {
		return nil, failed("errors.facility_save_failed")
	}
	return s.GetFacility(ctx, tenantId, id)
}

func (s *counciladminService) UpdateFacility(ctx context.Context, tenantId uuid.UUID, id uuid.UUID, input FacilityInput) (*FacilityDetailDto, *shared.ErrorResponse) {
	input.normalise()
	if key := input.validate(); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	// only an image uploaded for this council
	if input.ImageRef != "" && !storage.Owned(input.ImageRef, tenantId, "facilities") {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.invalid_image"}
	}
	var err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		var _, updateErr = queries.UpdateFacility(ctx, model.UpdateFacilityParams{
			ID:                id,
			TenantID:          tenantId,
			Slug:              input.Slug,
			Name:              input.Name,
			Description:       nullString(input.Description),
			Address:           nullString(input.Address),
			Latitude:          nullFloat(input.Latitude),
			Longitude:         nullFloat(input.Longitude),
			Capacity:          nullInt(input.Capacity),
			AccessibilityInfo: nullString(input.AccessibilityInfo),
			ContactPhone:      nullString(input.ContactPhone),
			ContactEmail:      nullString(input.ContactEmail),
			ImageRef:          nullString(input.ImageRef),
			ShowWeather:       input.ShowWeather,
			IsActive:          input.IsActive,
			SlotMinutes:       input.SlotMinutes,
			MinAge:            nullInt(input.MinAge),
			ResidentsOnly:     input.ResidentsOnly,
			UsageRulesText:    nullString(input.UsageRules),
		})
		if updateErr != nil {
			return updateErr
		}
		return setFacilityTags(ctx, queries, id, input.Tags)
	})
	if errors.Is(err, errUnknownTag) {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.invalid_tags"}
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	}
	if isUniqueViolation(err) {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.facility_slug_taken"}
	}
	if err != nil {
		return nil, failed("errors.facility_save_failed")
	}
	return s.GetFacility(ctx, tenantId, id)
}

// DeleteFacility removes a facility created by mistake; one with bookings is deactivated instead.
func (s *counciladminService) DeleteFacility(ctx context.Context, tenantId uuid.UUID, id uuid.UUID) *shared.ErrorResponse {
	var queries = model.New(s.db)
	// another council's facility is not found, whatever it holds
	var _, err = queries.GetFacilityForAdmin(ctx, model.GetFacilityForAdminParams{ID: id, TenantID: tenantId})
	if errors.Is(err, sql.ErrNoRows) {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	}
	if err != nil {
		return failed("errors.facility_delete_failed")
	}
	var bookings int64
	bookings, err = queries.CountBookingsByFacility(ctx, id)
	if err != nil {
		return failed("errors.facility_delete_failed")
	}
	if bookings > 0 {
		return &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.facility_has_bookings"}
	}
	var deleted int64
	err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		var deleteErr error
		deleted, deleteErr = queries.DeleteFacility(ctx, model.DeleteFacilityParams{ID: id, TenantID: tenantId})
		return deleteErr
	})
	if err != nil {
		return failed("errors.facility_delete_failed")
	}
	if deleted == 0 {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	}
	return nil
}

// ReplaceSchedule swaps the whole weekly schedule in one transaction.
func (s *counciladminService) ReplaceSchedule(ctx context.Context, tenantId uuid.UUID, id uuid.UUID, input ScheduleInput) (*FacilityDetailDto, *shared.ErrorResponse) {
	var windows, key = input.parse()
	if key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	if _, notFound := s.GetFacility(ctx, tenantId, id); notFound != nil {
		return nil, notFound
	}

	var err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		if err := queries.DeleteOpeningHoursByFacility(ctx, id); err != nil {
			return err
		}
		return insertWindows(ctx, queries, id, windows)
	})
	if err != nil {
		return nil, failed("errors.schedule_save_failed")
	}
	return s.GetFacility(ctx, tenantId, id)
}

// insertWindows adds the weekly hours to a facility, inside the council's transaction.
func insertWindows(ctx context.Context, queries *model.Queries, facilityId uuid.UUID, windows []parsedWindow) error {
	for _, w := range windows {
		if err := queries.InsertOpeningHours(ctx, model.InsertOpeningHoursParams{FacilityID: facilityId, Weekday: w.weekday, OpensAt: w.opens, ClosesAt: w.closes}); err != nil {
			return err
		}
	}
	return nil
}

func timePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

// --- check-in readers ------------------------------------------------------

func (s *counciladminService) ListDevices(ctx context.Context, tenantId uuid.UUID, facilityId uuid.UUID) (*DevicesResponse, *shared.ErrorResponse) {
	if _, notFound := s.GetFacility(ctx, tenantId, facilityId); notFound != nil {
		return nil, notFound
	}
	var rows, err = model.New(s.db).ListCheckinDevices(ctx, model.ListCheckinDevicesParams{FacilityID: facilityId, TenantID: tenantId})
	if err != nil {
		return nil, failed("errors.devices_load_failed")
	}
	var response = &DevicesResponse{Devices: make([]DeviceDto, 0, len(rows))}
	for _, d := range rows {
		response.Devices = append(response.Devices, DeviceDto{
			Id:         d.ID.String(),
			Name:       d.Name,
			KeyHint:    d.KeyHint,
			LastSeenAt: timePtr(d.LastSeenAt),
			CreatedAt:  d.CreatedAt,
		})
	}
	return response, nil
}

// CreateDevice registers a reader for the facility and returns its key, shown only this once.
func (s *counciladminService) CreateDevice(ctx context.Context, tenantId uuid.UUID, facilityId uuid.UUID, input DeviceInput) (*CreatedDeviceDto, *shared.ErrorResponse) {
	if key := input.validate(); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	var key, hash, hint, err = checkin.NewDeviceKey()
	if err != nil {
		return nil, failed("errors.device_save_failed")
	}
	var row model.CreateCheckinDeviceRow
	err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		var createErr error
		row, createErr = queries.CreateCheckinDevice(ctx, model.CreateCheckinDeviceParams{
			Name:       input.Name,
			KeyHash:    hash,
			KeyHint:    hint,
			FacilityID: facilityId,
			TenantID:   tenantId,
		})
		return createErr
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	}
	if err != nil {
		return nil, failed("errors.device_save_failed")
	}
	return &CreatedDeviceDto{
		DeviceDto: DeviceDto{Id: row.ID.String(), Name: row.Name, KeyHint: row.KeyHint, LastSeenAt: timePtr(row.LastSeenAt), CreatedAt: row.CreatedAt},
		Key:       key,
	}, nil
}

// DeleteDevice revokes a reader: its key stops working at once.
func (s *counciladminService) DeleteDevice(ctx context.Context, tenantId uuid.UUID, facilityId uuid.UUID, id uuid.UUID) *shared.ErrorResponse {
	var deleted int64
	var err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		var deleteErr error
		deleted, deleteErr = queries.DeleteCheckinDevice(ctx, model.DeleteCheckinDeviceParams{ID: id, FacilityID: facilityId, TenantID: tenantId})
		return deleteErr
	})
	if err != nil {
		return failed("errors.device_delete_failed")
	}
	if deleted == 0 {
		return &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.device_not_found"}
	}
	return nil
}
