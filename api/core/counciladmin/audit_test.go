package counciladmin

import (
	"context"
	"librequenda/audit"
	"librequenda/core/checkin"
	"strings"
	"testing"
)

func TestDescribeKeepsPersonalDataOut(t *testing.T) {
	var ctx, entry = audit.Recording(context.Background())
	var before = &FacilityDetailDto{Name: "Pavillón", SlotMinutes: 60, Description: "Old text", ContactPhone: "986 000 000"}
	var after = &FacilityDetailDto{Name: "Pavillón", SlotMinutes: 90, Description: "New text", ContactPhone: "986 111 111"}
	describeFacility(ctx, before, after)
	var summary = entry().Summary
	for _, want := range []string{"slot_minutes: 60 → 90", "description changed", "contact_phone changed"} {
		if !strings.Contains(summary, want) {
			t.Errorf("summary %q lacks %q", summary, want)
		}
	}
	for _, leaked := range []string{"New text", "986 111 111"} {
		if strings.Contains(summary, leaked) {
			t.Errorf("summary %q copies a value: %q", summary, leaked)
		}
	}
}

func TestDescribeRulesAndCheckin(t *testing.T) {
	var ctx, entry = audit.Recording(context.Background())
	var rules = goodRules()
	var changed = rules
	changed.MaxBookingsPerDay, changed.NoShowPenalty = 2, true
	describeRules(ctx, rules, changed)
	describeCheckin(ctx,
		&CheckinSettingsDto{QrReader: true, StaffContacts: []checkin.StaffContact{{Name: "Marta"}}},
		&CheckinSettingsDto{QrReader: false, StaffContacts: []checkin.StaffContact{{Name: "Marta"}, {Name: "Xan"}}})
	var summary = entry().Summary
	for _, want := range []string{"max_bookings_per_day: 1 → 2", "no_show_penalty: false → true", "qr_reader: true → false", "staff_contacts: 1 → 2"} {
		if !strings.Contains(summary, want) {
			t.Errorf("summary %q lacks %q", summary, want)
		}
	}
	if strings.Contains(summary, "Marta") || strings.Contains(summary, "Xan") {
		t.Errorf("staff contacts are counted, not named: %q", summary)
	}
}
