package tenants

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/angelbarreiros/Penguin/router/helpers"
	"librequenda/model"
	"librequenda/rating"
	"librequenda/schedule"
	"librequenda/shared"
	"librequenda/storage"
	"net/http"
	"slices"
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

// GetTenants lists the active councils: who they are and how to contact them,
// for the council filter and the switcher. Public, so nothing about how
// they use LibreQuenda (bookings, residents) goes here.
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

// GetTenant is one active council, for the pages about it.
func (s *tenantsService) GetTenant(ctx context.Context, slug string) (*TenantDto, *shared.ErrorResponse) {
	var row, err = model.New(s.db).GetTenantBySlug(ctx, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.tenant_not_found"}
	}
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.tenants_load_failed"}
	}
	return &TenantDto{
		Slug:         row.Slug,
		Name:         row.Name,
		LogoUrl:      storage.URL(row.LogoRef.String),
		ContactEmail: row.ContactEmail.String,
		ContactPhone: row.ContactPhone.String,
	}, nil
}

// GetFacilities is every active facility of every active council, for the home
// page's list and map. It reuses the day view, a few queries per
// council; a council that fails to load is left out rather than failing the page.
func (s *tenantsService) GetFacilities(ctx context.Context, near *Near, filters Filters, page helpers.PaginationParams) (*FacilitiesResponse, *shared.ErrorResponse) {
	var tenants, err = model.New(s.db).ListTenants(ctx)
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.tenants_load_failed"}
	}
	var names = s.tagNames(ctx)
	var joinable = map[string][]string{}
	var kinds, kindsErr = model.New(s.db).ListJoinableKindsByFacility(ctx)
	if kindsErr != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.tenants_load_failed"}
	}
	for _, row := range kinds {
		joinable[row.FacilityID.String()] = append(joinable[row.FacilityID.String()], string(row.Kind))
	}
	var response = &FacilitiesResponse{Facilities: []FacilityCardDto{}}
	for _, tenant := range tenants {
		var day, _, _, dayErr = s.loadDay(ctx, tenant.Slug, "")
		if dayErr != nil {
			continue
		}
		for _, f := range day.Facilities {
			var free = 0
			for _, slot := range f.Slots {
				if slot.Status == schedule.SlotFree {
					free++
				}
			}
			var km = distanceKm(near, f.Latitude, f.Longitude)
			var open = joinable[f.Id]
			if open == nil {
				open = []string{}
			}
			slices.Sort(open) // "event" before "public", whatever the database's order
			if !near.within(km) || !filters.matches(f.Name, day.Tenant.Slug, day.Tenant.Name, f.Sport, f.Tags, names) || !filters.joinable(open) {
				continue
			}
			response.Facilities = append(response.Facilities, FacilityCardDto{
				Id:         f.Id,
				Slug:       f.Slug,
				Name:       f.Name,
				Sport:      f.Sport,
				Tags:       f.Tags,
				ImageUrl:   f.ImageUrl,
				Address:    f.Address,
				Latitude:   f.Latitude,
				Longitude:  f.Longitude,
				FreeToday:  free,
				Council:    day.Tenant,
				Joinable:   open,
				DistanceKm: km,
			})
		}
	}
	if near != nil {
		byDistance(response.Facilities, func(f FacilityCardDto) *float64 { return f.DistanceKm })
	}
	response.Pagination = shared.NewPageDto(page, int64(len(response.Facilities)))
	response.Facilities = shared.Slice(response.Facilities, page)
	return response, nil
}

// tagNames: every tag's name by slug, for searching words in them; empty if it fails.
func (s *tenantsService) tagNames(ctx context.Context) map[string]string {
	var names = map[string]string{}
	if tags, err := model.New(s.db).ListTags(ctx); err == nil {
		for _, tag := range tags {
			names[tag.Slug] = tag.Name
		}
	}
	return names
}

// GetFacilitiesDay: every council's facilities with their turns for one day
// (?date=, today by default), nearest first with ?lat=&lon=. The search's list.
func (s *tenantsService) GetFacilitiesDay(ctx context.Context, dateParam string, near *Near, filters Filters, page helpers.PaginationParams) (*FacilitiesDayResponse, *shared.ErrorResponse) {
	var tenants, err = model.New(s.db).ListTenants(ctx)
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.tenants_load_failed"}
	}
	var names = s.tagNames(ctx)
	var now = time.Now()
	var response = &FacilitiesDayResponse{Facilities: []FacilityAnyDayDto{}}
	for _, tenant := range tenants {
		var day, _, loc, dayErr = s.loadDay(ctx, tenant.Slug, dateParam)
		if dayErr != nil {
			if dayErr.Code == http.StatusBadRequest {
				return nil, dayErr // a malformed date is the same for every council
			}
			continue // one council down still shows the others
		}
		response.Date = day.Date
		for _, f := range day.Facilities {
			var km = distanceKm(near, f.Latitude, f.Longitude)
			if !near.within(km) || !filters.matches(f.Name, day.Tenant.Slug, day.Tenant.Name, f.Sport, f.Tags, names) {
				continue
			}
			var kept, any = filters.narrowDay(f, loc, now)
			if !any {
				continue
			}
			f = kept
			response.Facilities = append(response.Facilities, FacilityAnyDayDto{
				FacilityDayDto: f,
				Council:        day.Tenant,
				DistanceKm:     km,
			})
		}
	}
	if near != nil {
		byDistance(response.Facilities, func(f FacilityAnyDayDto) *float64 { return f.DistanceKm })
	}
	response.Pagination = shared.NewPageDto(page, int64(len(response.Facilities)))
	response.Facilities = shared.Slice(response.Facilities, page)
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

	var publicRows, publicErr = queries.ListPublicBookingsByTenantRange(ctx, model.ListPublicBookingsByTenantRangeParams{
		TenantID: tenant.ID, RangeStart: day, RangeEnd: dayEnd,
	})
	if publicErr != nil {
		return nil, uuid.Nil, nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.day_load_failed"}
	}
	var publicByFacility = map[uuid.UUID][]PublicTurnDto{}
	for _, row := range publicRows {
		var people = append([]string{row.OwnerName}, row.People...)
		if row.Kind == model.BookingKindEvent {
			people = row.People
		}
		publicByFacility[row.FacilityID] = append(publicByFacility[row.FacilityID], PublicTurnDto{
			Id: row.ID.String(), Kind: string(row.Kind), Title: row.Title, StartsAt: row.StartsAt, EndsAt: row.EndsAt,
			Description: row.PublicDescription, Capacity: row.Capacity.Int32,
			People: people,
		})
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
		Tenant: TenantDto{
			Slug:         tenant.Slug,
			Name:         tenant.Name,
			LogoUrl:      storage.URL(tenant.LogoRef.String),
			ContactEmail: tenant.ContactEmail.String,
			ContactPhone: tenant.ContactPhone.String,
		},
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
		var rules = schedule.Rules{
			SlotMinutes: int(f.SlotMinutes),
			MaxAdvance:  time.Duration(f.MaxAdvanceHours) * time.Hour,
			MinAdvance:  time.Duration(f.MinAdvanceMinutes) * time.Minute,
		}
		if f.WeeklyOpening {
			rules.Weekly = &schedule.WeeklyOpening{Weekday: int(f.OpensWeekday), Minute: int(f.OpensMinute)}
		}
		var slots = schedule.DaySlots(day, loc, rules, windows, closureIntervals, busyByFacility[f.ID], now)
		if slots == nil {
			slots = []schedule.Slot{}
		}
		var latitude, longitude *float64
		if f.Latitude.Valid && f.Longitude.Valid {
			latitude, longitude = &f.Latitude.Float64, &f.Longitude.Float64
		}
		response.Facilities = append(response.Facilities, FacilityDayDto{
			Id:                f.ID.String(),
			PublicBookings:    append([]PublicTurnDto{}, publicByFacility[f.ID]...),
			Slug:              f.Slug,
			Name:              f.Name,
			Description:       f.Description.String,
			ImageUrl:          storage.URL(f.ImageRef.String),
			Latitude:          latitude,
			Longitude:         longitude,
			Sport:             f.Sport,
			Tags:              append([]string{}, f.Tags...),
			ShowWeather:       f.ShowWeather,
			Address:           f.Address.String,
			ContactPhone:      f.ContactPhone.String,
			ContactEmail:      f.ContactEmail.String,
			SlotMinutes:       f.SlotMinutes,
			MaxAdvanceHours:   f.MaxAdvanceHours,
			MinAdvanceMinutes: f.MinAdvanceMinutes,
			MinCancelHours:    f.MinCancelHours,
			MaxActive:         f.MaxActiveBookingsPerUser,
			MaxPerDay:         f.MaxBookingsPerDay,
			MaxInARow:         f.MaxTurnsInARow,
			MaxPerWeek:        f.MaxBookingsPerWeek,
			CouncilMaxActive:  f.CouncilMaxActive,
			WeeklyOpening:     weeklyOpening(f.WeeklyOpening, f.OpensWeekday, f.OpensMinute),
			Capacity:          f.Capacity.Int32,
			MinAge:            minAge(f.MinAge),
			ResidentsOnly:     f.ResidentsOnly,
			UsageRules:        f.UsageRulesText.String,
			Windows:           windowDtos,
			Closures:          closureDtos,
			Slots:             slots,
		})
	}
	return response, tenant.ID, loc, nil
}

// GetFacilityPage is one active facility of the council with its slots for the day and its weekly schedule.
func (s *tenantsService) GetFacilityPage(ctx context.Context, slug string, facilitySlug string, dateParam string) (*FacilityPageResponse, *shared.ErrorResponse) {
	var day, _, _, err = s.loadDay(ctx, slug, dateParam)
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
		var stars, starsErr = model.New(s.db).GetFacilityRatingSummary(ctx, id)
		if starsErr != nil {
			return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.day_load_failed"}
		}
		return &FacilityPageResponse{
			Rating:   rating.Of(stars.Average, stars.Count),
			Tenant:   day.Tenant,
			Timezone: day.Timezone,
			Date:     day.Date,
			Now:      day.Now,
			Facility: f,
			Schedule: schedule,
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

// weeklyOpening: when next week's turns open, for the facility page; nil with a rolling window.
func weeklyOpening(on bool, weekday int16, minute int32) *WeeklyOpeningDto {
	if !on {
		return nil
	}
	return &WeeklyOpeningDto{Weekday: weekday, At: fmt.Sprintf("%02d:%02d", minute/60, minute%60)}
}
