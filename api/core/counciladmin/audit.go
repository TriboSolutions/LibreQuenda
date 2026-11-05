package counciladmin

import (
	"context"
	"librequenda/audit"
	"slices"
)

// What the audit log says changed: field names and non-personal values. Texts
// the council writes (descriptions, rules, reasons) are reported as changed,
// not copied; staff contacts are counted, not named.

func describeFacility(ctx context.Context, before, after *FacilityDetailDto) {
	if before == nil || after == nil {
		return
	}
	audit.Changed(ctx, "name", before.Name, after.Name)
	audit.Changed(ctx, "slug", before.Slug, after.Slug)
	audit.Changed(ctx, "slot_minutes", before.SlotMinutes, after.SlotMinutes)
	audit.Changed(ctx, "capacity", intOrNone(before.Capacity), intOrNone(after.Capacity))
	audit.Changed(ctx, "min_age", intOrNone(before.MinAge), intOrNone(after.MinAge))
	audit.Changed(ctx, "residents_only", before.ResidentsOnly, after.ResidentsOnly)
	for _, field := range []struct {
		name          string
		before, after string
	}{
		{"description", before.Description, after.Description},
		{"address", before.Address, after.Address},
		{"usage_rules", before.UsageRules, after.UsageRules},
		{"accessibility_info", before.AccessibilityInfo, after.AccessibilityInfo},
		{"contact_phone", before.ContactPhone, after.ContactPhone},
		{"contact_email", before.ContactEmail, after.ContactEmail},
		{"image", before.ImageRef, after.ImageRef},
	} {
		if field.before != field.after {
			audit.Describe(ctx, "%s changed", field.name)
		}
	}
	if !floatEqual(before.Latitude, after.Latitude) || !floatEqual(before.Longitude, after.Longitude) {
		audit.Describe(ctx, "map pin changed")
	}
	if !slices.Equal(before.Tags, after.Tags) {
		audit.Describe(ctx, "tags: %v → %v", before.Tags, after.Tags)
	}
}

func describeRules(ctx context.Context, before, after BookingRulesDto) {
	audit.Changed(ctx, "max_advance_hours", before.MaxAdvanceHours, after.MaxAdvanceHours)
	audit.Changed(ctx, "min_advance_minutes", before.MinAdvanceMinutes, after.MinAdvanceMinutes)
	audit.Changed(ctx, "min_cancel_hours", before.MinCancelHours, after.MinCancelHours)
	audit.Changed(ctx, "max_active_bookings_per_user", before.MaxActiveBookingsPerUser, after.MaxActiveBookingsPerUser)
	audit.Changed(ctx, "max_bookings_per_day", before.MaxBookingsPerDay, after.MaxBookingsPerDay)
	audit.Changed(ctx, "max_turns_in_a_row", before.MaxTurnsInARow, after.MaxTurnsInARow)
	audit.Changed(ctx, "max_bookings_per_week", before.MaxBookingsPerWeek, after.MaxBookingsPerWeek)
	audit.Changed(ctx, "weekly_opening", before.WeeklyOpening, after.WeeklyOpening)
	audit.Changed(ctx, "opens_weekday", before.OpensWeekday, after.OpensWeekday)
	audit.Changed(ctx, "opens_minute", before.OpensMinute, after.OpensMinute)
	audit.Changed(ctx, "no_show_penalty", before.NoShowPenalty, after.NoShowPenalty)
	audit.Changed(ctx, "no_show_limit", before.NoShowLimit, after.NoShowLimit)
	audit.Changed(ctx, "no_show_window_days", before.NoShowWindowDays, after.NoShowWindowDays)
	audit.Changed(ctx, "no_show_block_days", before.NoShowBlockDays, after.NoShowBlockDays)
}

func describeCheckin(ctx context.Context, before, after *CheckinSettingsDto) {
	if before == nil || after == nil {
		return
	}
	audit.Changed(ctx, "qr_reader", before.QrReader, after.QrReader)
	audit.Changed(ctx, "printed_qr", before.PrintedQr, after.PrintedQr)
	audit.Changed(ctx, "nfc", before.Nfc, after.Nfc)
	audit.Changed(ctx, "entry_code", before.EntryCode, after.EntryCode)
	audit.Changed(ctx, "staff_confirm", before.StaffConfirm, after.StaffConfirm)
	audit.Changed(ctx, "flash_booking", before.FlashBooking, after.FlashBooking)
	audit.Changed(ctx, "radius_m", before.RadiusM, after.RadiusM)
	audit.Changed(ctx, "staff_contacts", len(before.StaffContacts), len(after.StaffContacts))
}

func intOrNone(v *int32) any {
	if v == nil {
		return "none"
	}
	return *v
}

func floatEqual(a, b *float64) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}
