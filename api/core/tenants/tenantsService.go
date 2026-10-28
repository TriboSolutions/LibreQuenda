package tenants

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/model"
	"librequenda/schedule"
	"librequenda/shared"
	"librequenda/storage"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type tenantsService struct {
	db *sql.DB
}

var sTenantsInstance *tenantsService
var sTenantsOnce *sync.Once = new(sync.Once)

func GetTenantsService(db *sql.DB) *tenantsService {
	sTenantsOnce.Do(func() {
		sTenantsInstance = &tenantsService{db: db}
	})
	return sTenantsInstance
}

func (s *tenantsService) GetTenants(ctx context.Context) (*TenantsResponse, *shared.ErrorResponse) {
	var rows, err = model.New(s.db).ListTenants(ctx)
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.tenants_load_failed"}
	}
	var response = &TenantsResponse{Tenants: make([]TenantDto, 0, len(rows))}
	for _, row := range rows {
		response.Tenants = append(response.Tenants, TenantDto{
			Slug:         row.Slug,
			Name:         row.Name,
			LogoUrl:      storage.URL(row.LogoRef.String),
			ContactEmail: row.ContactEmail.String,
			ContactPhone: row.ContactPhone.String,
		})
	}
	return response, nil
}

func (s *tenantsService) GetDay(ctx context.Context, slug string, dateParam string) (*DayResponse, *shared.ErrorResponse) {
	var day, _, _, err = s.loadDay(ctx, slug, dateParam)
	return day, err
}

// loadDay is GetDay plus the council's id and time zone, for the facility page.
func (s *tenantsService) loadDay(ctx context.Context, slug string, dateParam string) (*DayResponse, uuid.UUID, *time.Location, *shared.ErrorResponse) {
	var queries = model.New(s.db)
	var tenant, err = queries.GetTenantBySlug(ctx, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, uuid.Nil, nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.tenant_not_found"}
	}
	if err != nil {
		return nil, uuid.Nil, nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.day_load_failed"}
	}

	var loc, locErr = time.LoadLocation(tenant.Timezone)
	if locErr != nil {
		return nil, uuid.Nil, nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.day_load_failed"}
	}
	var now = time.Now().In(loc)
	var day = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	if dateParam != "" {
		day, err = time.ParseInLocation(time.DateOnly, dateParam, loc)
		if err != nil {
			return nil, uuid.Nil, nil, &shared.ErrorResponse{Code: http.StatusBadRequest, Message: "errors.invalid_query_params"}
		}
	}
	var dayEnd = day.AddDate(0, 0, 1)

	var facilities []model.ListActiveFacilitiesByTenantRow
	facilities, err = queries.ListActiveFacilitiesByTenant(ctx, tenant.ID)
	if err != nil {
		return nil, uuid.Nil, nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.day_load_failed"}
	}
	var hours []model.ListOpeningHoursByTenantWeekdayRow
	hours, err = queries.ListOpeningHoursByTenantWeekday(ctx, model.ListOpeningHoursByTenantWeekdayParams{TenantID: tenant.ID, Weekday: schedule.ISOWeekday(day)})
	if err != nil {
		return nil, uuid.Nil, nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.day_load_failed"}
	}
	var closures []model.ListClosuresByTenantRangeRow
	closures, err = queries.ListClosuresByTenantRange(ctx, model.ListClosuresByTenantRangeParams{TenantID: tenant.ID, RangeStart: day, RangeEnd: dayEnd})
	if err != nil {
		return nil, uuid.Nil, nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.day_load_failed"}
	}
	var bookings []model.ListConfirmedBookingsByTenantRangeRow
	bookings, err = queries.ListConfirmedBookingsByTenantRange(ctx, model.ListConfirmedBookingsByTenantRangeParams{TenantID: tenant.ID, RangeStart: day, RangeEnd: dayEnd})
	if err != nil {
		return nil, uuid.Nil, nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.day_load_failed"}
	}

	var windowsByFacility = map[uuid.UUID][]schedule.Window{}
	for _, h := range hours {
		windowsByFacility[h.FacilityID] = append(windowsByFacility[h.FacilityID], schedule.Window{Opens: h.OpensAt, Closes: h.ClosesAt})
	}
	var closuresByFacility = map[uuid.UUID][]model.ListClosuresByTenantRangeRow{}
	for _, c := range closures {
		closuresByFacility[c.FacilityID] = append(closuresByFacility[c.FacilityID], c)
	}
	var busyByFacility = map[uuid.UUID][]schedule.Interval{}
	for _, b := range bookings {
		busyByFacility[b.FacilityID] = append(busyByFacility[b.FacilityID], schedule.Interval{Start: b.StartsAt, End: b.EndsAt})
	}

	var response = &DayResponse{
		Tenant:     TenantDto{Slug: tenant.Slug, Name: tenant.Name, LogoUrl: storage.URL(tenant.LogoRef.String)},
		Timezone:   tenant.Timezone,
		Date:       day.Format(time.DateOnly),
		Now:        now,
		Facilities: make([]FacilityDayDto, 0, len(facilities)),
	}
	for _, f := range facilities {
		var windows = windowsByFacility[f.ID]
		var closureIntervals []schedule.Interval
		var closureDtos = []ClosureDto{}
		for _, c := range closuresByFacility[f.ID] {
			closureIntervals = append(closureIntervals, schedule.Interval{Start: c.StartsAt, End: c.EndsAt})
			closureDtos = append(closureDtos, ClosureDto{StartsAt: c.StartsAt.In(loc), EndsAt: c.EndsAt.In(loc), Reason: c.Reason.String})
		}
		var windowDtos = make([]WindowDto, 0, len(windows))
		for _, w := range windows {
			windowDtos = append(windowDtos, WindowDto{OpensAt: w.Opens.Format("15:04"), ClosesAt: w.Closes.Format("15:04")})
		}
		var slots = schedule.DaySlots(day, loc, schedule.Rules{SlotMinutes: int(f.SlotMinutes), MaxAdvanceDays: int(f.MaxAdvanceDays)}, windows, closureIntervals, busyByFacility[f.ID], now)
		if slots == nil {
			slots = []schedule.Slot{}
		}
		var latitude, longitude *float64
		if f.Latitude.Valid && f.Longitude.Valid {
			latitude, longitude = &f.Latitude.Float64, &f.Longitude.Float64
		}
		response.Facilities = append(response.Facilities, FacilityDayDto{
			Id:             f.ID.String(),
			Slug:           f.Slug,
			Name:           f.Name,
			Description:    f.Description.String,
			ImageUrl:       storage.URL(f.ImageRef.String),
			Latitude:       latitude,
			Longitude:      longitude,
			Sport:          f.Sport.String,
			Address:        f.Address.String,
			SlotMinutes:    f.SlotMinutes,
			MaxAdvanceDays: f.MaxAdvanceDays,
			MinCancelHours: f.MinCancelHours,
			MaxActive:      f.MaxActiveBookingsPerUser,
			MinAge:         minAge(f.MinAge),
			UsageRules:     f.UsageRulesText.String,
			Windows:        windowDtos,
			Closures:       closureDtos,
			Slots:          slots,
		})
	}
	return response, tenant.ID, loc, nil
}

// GetFacilityPage is one active facility of the council with its slots for the day and its weekly schedule.
func (s *tenantsService) GetFacilityPage(ctx context.Context, slug string, facilitySlug string, dateParam string) (*FacilityPageResponse, *shared.ErrorResponse) {
	var day, tenantId, loc, err = s.loadDay(ctx, slug, dateParam)
	if err != nil {
		return nil, err
	}
	for _, f := range day.Facilities {
		if f.Slug != facilitySlug {
			continue
		}
		var id = uuid.MustParse(f.Id)
		var hours, qErr = model.New(s.db).ListOpeningHoursByFacility(ctx, id)
		if qErr != nil {
			return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.day_load_failed"}
		}
		var schedule = make([]WeeklyWindowDto, 0, len(hours))
		for _, h := range hours {
			schedule = append(schedule, WeeklyWindowDto{Weekday: h.Weekday, OpensAt: h.OpensAt.Format("15:04"), ClosesAt: h.ClosesAt.Format("15:04")})
		}
		var now = time.Now()
		var busy, busyErr = model.New(s.db).ListBusyHoursByTenant(ctx, model.ListBusyHoursByTenantParams{
			Timezone:   day.Timezone,
			TenantID:   tenantId,
			RangeStart: now.AddDate(0, 0, -7*historyWeeks),
			RangeEnd:   now,
		})
		if busyErr != nil {
			return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.day_load_failed"}
		}
		return &FacilityPageResponse{
			Tenant:      day.Tenant,
			Timezone:    day.Timezone,
			Date:        day.Date,
			Now:         day.Now,
			Facility:    f,
			Schedule:    schedule,
			Suggestions: suggest(f, day.Facilities, busyHistoryByFacility(busy), loc),
		}, nil
	}
	return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
}

func minAge(value sql.NullInt32) *int32 {
	if !value.Valid {
		return nil
	}
	return &value.Int32
}
