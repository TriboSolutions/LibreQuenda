package counciladmin

import (
	"context"
	"librequenda/db"
	"librequenda/model"
	"librequenda/shared"
	"net/http"

	"github.com/google/uuid"
)

// The booking window: the council's, which its facilities inherit, or a
// facility's own (custom_rules). Each facility keeps the values in force, so
// booking and the day view read them from the facility alone.

type BookingRulesDto struct {
	MaxAdvanceHours          int32 `json:"max_advance_hours"`            // how far ahead, from now
	MinAdvanceMinutes        int32 `json:"min_advance_minutes"`          // how close to the start the app still books
	MinCancelHours           int32 `json:"min_cancel_hours"`             // cancel or change until this long before
	MaxActiveBookingsPerUser int32 `json:"max_active_bookings_per_user"` // upcoming bookings per person
	MaxBookingsPerDay        int32 `json:"max_bookings_per_day"`         // per person and day; 0 = no limit
	MaxTurnsInARow           int32 `json:"max_turns_in_a_row"`           // back-to-back turns per person; 0 = no limit
	MaxBookingsPerWeek       int32 `json:"max_bookings_per_week"`        // per person and ISO week; 0 = no limit
	// instead of MaxAdvanceHours: next week's turns open all at once on OpensWeekday (ISO, 7 = Sunday) at OpensMinute
	WeeklyOpening bool  `json:"weekly_opening"`
	OpensWeekday  int16 `json:"opens_weekday"`
	OpensMinute   int32 `json:"opens_minute"` // minutes after local midnight
	// no-show penalty: after NoShowLimit no-shows in NoShowWindowDays, no booking for NoShowBlockDays
	NoShowPenalty    bool  `json:"no_show_penalty"`
	NoShowLimit      int32 `json:"no_show_limit"`
	NoShowWindowDays int32 `json:"no_show_window_days"`
	NoShowBlockDays  int32 `json:"no_show_block_days"`
}

// validate returns the i18n key of the first value out of range, or "".
func (r BookingRulesDto) validate() string {
	switch {
	case r.MaxAdvanceHours < 1 || r.MaxAdvanceHours > 365*24:
		return "errors.invalid_booking_window"
	case r.MinAdvanceMinutes < 0 || r.MinAdvanceMinutes > 24*60:
		return "errors.invalid_min_advance"
	case r.MinCancelHours < 0 || r.MinCancelHours > 720:
		return "errors.invalid_cancel_hours"
	case r.MaxActiveBookingsPerUser < 1 || r.MaxActiveBookingsPerUser > 50:
		return "errors.invalid_booking_limit"
	case r.MaxBookingsPerDay < 0 || r.MaxBookingsPerDay > 24, r.MaxTurnsInARow < 0 || r.MaxTurnsInARow > 24, r.MaxBookingsPerWeek < 0 || r.MaxBookingsPerWeek > 100:
		return "errors.invalid_booking_limit"
	case r.OpensWeekday < 1 || r.OpensWeekday > 7 || r.OpensMinute < 0 || r.OpensMinute > 24*60-1:
		return "errors.invalid_weekly_opening"
	case r.NoShowLimit < 1 || r.NoShowLimit > 20 || r.NoShowWindowDays < 1 || r.NoShowWindowDays > 365 || r.NoShowBlockDays < 1 || r.NoShowBlockDays > 365:
		return "errors.invalid_no_show_penalty"
	}
	return ""
}

func rulesOfFacility(f model.Facility) BookingRulesDto {
	return BookingRulesDto{
		MaxAdvanceHours: f.MaxAdvanceHours, MinAdvanceMinutes: f.MinAdvanceMinutes, MinCancelHours: f.MinCancelHours,
		MaxActiveBookingsPerUser: f.MaxActiveBookingsPerUser, MaxBookingsPerDay: f.MaxBookingsPerDay, MaxTurnsInARow: f.MaxTurnsInARow,
		NoShowPenalty: f.NoShowPenalty, NoShowLimit: f.NoShowLimit, NoShowWindowDays: f.NoShowWindowDays, NoShowBlockDays: f.NoShowBlockDays,
		MaxBookingsPerWeek: f.MaxBookingsPerWeek, WeeklyOpening: f.WeeklyOpening, OpensWeekday: f.OpensWeekday, OpensMinute: f.OpensMinute,
	}
}

func rulesOfCouncil(r model.TenantBookingRule) BookingRulesDto {
	return BookingRulesDto{
		MaxAdvanceHours: r.MaxAdvanceHours, MinAdvanceMinutes: r.MinAdvanceMinutes, MinCancelHours: r.MinCancelHours,
		MaxActiveBookingsPerUser: r.MaxActiveBookingsPerUser, MaxBookingsPerDay: r.MaxBookingsPerDay, MaxTurnsInARow: r.MaxTurnsInARow,
		NoShowPenalty: r.NoShowPenalty, NoShowLimit: r.NoShowLimit, NoShowWindowDays: r.NoShowWindowDays, NoShowBlockDays: r.NoShowBlockDays,
		MaxBookingsPerWeek: r.MaxBookingsPerWeek, WeeklyOpening: r.WeeklyOpening, OpensWeekday: r.OpensWeekday, OpensMinute: r.OpensMinute,
	}
}

// councilRules creates the council's window with the defaults the first time, and reads it.
func (s *counciladminService) councilRules(ctx context.Context, tenantId uuid.UUID) (BookingRulesDto, *shared.ErrorResponse) {
	var rules model.TenantBookingRule
	var err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		if ensureErr := queries.EnsureTenantBookingRules(ctx, tenantId); ensureErr != nil {
			return ensureErr
		}
		var getErr error
		rules, getErr = queries.GetTenantBookingRules(ctx, tenantId)
		return getErr
	})
	if err != nil {
		return BookingRulesDto{}, failed("errors.booking_rules_failed")
	}
	return rulesOfCouncil(rules), nil
}

func (s *counciladminService) GetBookingRules(ctx context.Context, tenantId uuid.UUID) (*BookingRulesDto, *shared.ErrorResponse) {
	var rules, err = s.councilRules(ctx, tenantId)
	if err != nil {
		return nil, err
	}
	return &rules, nil
}

// SaveBookingRules: the council's window, copied at once to every facility that inherits it.
func (s *counciladminService) SaveBookingRules(ctx context.Context, tenantId uuid.UUID, input BookingRulesDto) (*BookingRulesDto, *shared.ErrorResponse) {
	if key := input.validate(); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	if _, err := s.councilRules(ctx, tenantId); err != nil {
		return nil, err
	}
	var err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		if updateErr := queries.UpdateTenantBookingRules(ctx, model.UpdateTenantBookingRulesParams{
			TenantID: tenantId, MaxAdvanceHours: input.MaxAdvanceHours, MinAdvanceMinutes: input.MinAdvanceMinutes,
			MinCancelHours: input.MinCancelHours, MaxActiveBookingsPerUser: input.MaxActiveBookingsPerUser,
			MaxBookingsPerDay: input.MaxBookingsPerDay, MaxTurnsInARow: input.MaxTurnsInARow, NoShowPenalty: input.NoShowPenalty,
			MaxBookingsPerWeek: input.MaxBookingsPerWeek, WeeklyOpening: input.WeeklyOpening, OpensWeekday: input.OpensWeekday, OpensMinute: input.OpensMinute,
			NoShowLimit: input.NoShowLimit, NoShowWindowDays: input.NoShowWindowDays, NoShowBlockDays: input.NoShowBlockDays,
		}); updateErr != nil {
			return updateErr
		}
		var _, applyErr = queries.ApplyTenantBookingRules(ctx, tenantId)
		return applyErr
	})
	if err != nil {
		return nil, failed("errors.booking_rules_failed")
	}
	return &input, nil
}

// FacilityRulesInput: a facility's booking window, from its Reserva tab.
type FacilityRulesInput struct {
	CustomRules bool            `json:"custom_rules"` // false: the council's (Rules is ignored)
	Rules       BookingRulesDto `json:"rules"`
}

// SaveFacilityRules: the facility's own window, or back to the council's.
func (s *counciladminService) SaveFacilityRules(ctx context.Context, tenantId uuid.UUID, facilityId uuid.UUID, input FacilityRulesInput) (*FacilityDetailDto, *shared.ErrorResponse) {
	var rules = input.Rules
	if input.CustomRules {
		if key := rules.validate(); key != "" {
			return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
		}
	} else {
		var council, err = s.councilRules(ctx, tenantId)
		if err != nil {
			return nil, err
		}
		rules = council
	}
	var updated int64
	var err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		var updateErr error
		updated, updateErr = queries.UpdateFacilityBookingRules(ctx, model.UpdateFacilityBookingRulesParams{
			ID: facilityId, TenantID: tenantId, CustomRules: input.CustomRules,
			MaxAdvanceHours: rules.MaxAdvanceHours, MinAdvanceMinutes: rules.MinAdvanceMinutes, MinCancelHours: rules.MinCancelHours,
			MaxActiveBookingsPerUser: rules.MaxActiveBookingsPerUser, MaxBookingsPerDay: rules.MaxBookingsPerDay, MaxTurnsInARow: rules.MaxTurnsInARow,
			NoShowPenalty: rules.NoShowPenalty, NoShowLimit: rules.NoShowLimit, NoShowWindowDays: rules.NoShowWindowDays, NoShowBlockDays: rules.NoShowBlockDays,
			MaxBookingsPerWeek: rules.MaxBookingsPerWeek, WeeklyOpening: rules.WeeklyOpening, OpensWeekday: rules.OpensWeekday, OpensMinute: rules.OpensMinute,
		})
		return updateErr
	})
	if err != nil {
		return nil, failed("errors.booking_rules_failed")
	}
	if updated == 0 {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	}
	return s.GetFacility(ctx, tenantId, facilityId)
}
