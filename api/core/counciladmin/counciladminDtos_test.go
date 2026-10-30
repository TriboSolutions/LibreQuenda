package counciladmin

import "testing"

func TestScheduleParse(t *testing.T) {
	var cases = []struct {
		name    string
		windows []WindowInput
		want    string
	}{
		{"empty week", nil, ""},
		{"two windows", []WindowInput{{1, "07:00", "10:00"}, {1, "16:00", "21:15"}}, ""},
		{"touching windows", []WindowInput{{2, "09:00", "14:00"}, {2, "14:00", "20:00"}}, ""},
		{"overlap", []WindowInput{{3, "09:00", "14:00"}, {3, "13:30", "20:00"}}, "errors.schedule_overlap"},
		{"same day other order", []WindowInput{{4, "16:00", "20:00"}, {4, "09:00", "16:30"}}, "errors.schedule_overlap"},
		{"closes before opening", []WindowInput{{5, "20:00", "09:00"}}, "errors.invalid_schedule"},
		{"bad time", []WindowInput{{6, "9", "10:00"}}, "errors.invalid_schedule"},
		{"bad weekday", []WindowInput{{8, "09:00", "10:00"}}, "errors.invalid_schedule"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var in = ScheduleInput{Windows: c.windows}
			if _, got := in.parse(); got != c.want {
				t.Errorf("parse() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestFacilityInputValidate(t *testing.T) {
	var lat, lng = 42.66, -8.11
	var valid = func() FacilityInput {
		return FacilityInput{Slug: "pista-de-padel-1", Name: "Pista de pádel 1", SlotMinutes: 90, Latitude: &lat, Longitude: &lng}
	}
	var cases = []struct {
		name   string
		change func(*FacilityInput)
		want   string
	}{
		{"valid", func(*FacilityInput) {}, ""},
		{"no map pin", func(in *FacilityInput) { in.Latitude, in.Longitude = nil, nil }, ""},
		{"half a pin", func(in *FacilityInput) { in.Longitude = nil }, "errors.invalid_coordinates"},
		{"reserved slug", func(in *FacilityInput) { in.Slug = "admin" }, "errors.slug_reserved"},
		{"slot not a multiple of 5", func(in *FacilityInput) { in.SlotMinutes = 47 }, "errors.invalid_slot_minutes"},
		{"image given as URL", func(in *FacilityInput) { in.ImageRef = "https://evil.example/a.png" }, "errors.invalid_image"},
		{"image of a council", func(in *FacilityInput) { in.ImageRef = "councils/a.png" }, "errors.invalid_image"},
		{"facility image", func(in *FacilityInput) { in.ImageRef = "facilities/a.png" }, ""},
		{"min age 16", func(in *FacilityInput) { var age int32 = 16; in.MinAge = &age }, ""},
		{"min age 0", func(in *FacilityInput) { var age int32 = 0; in.MinAge = &age }, "errors.invalid_min_age"},
		{"min age 81", func(in *FacilityInput) { var age int32 = 81; in.MinAge = &age }, "errors.invalid_min_age"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var in = valid()
			c.change(&in)
			in.normalise()
			if got := in.validate(); got != c.want {
				t.Errorf("validate() = %q, want %q", got, c.want)
			}
		})
	}
}

func goodRules() BookingRulesDto {
	return BookingRulesDto{MaxAdvanceHours: 7 * 24, MinAdvanceMinutes: 30, MinCancelHours: 4, MaxActiveBookingsPerUser: 2,
		MaxBookingsPerDay: 1, MaxTurnsInARow: 2, NoShowLimit: 3, NoShowWindowDays: 30, NoShowBlockDays: 7}
}

func TestBookingRulesValidate(t *testing.T) {
	if key := goodRules().validate(); key != "" {
		t.Fatalf("good rules: %q", key)
	}
	var noAdvance = goodRules()
	noAdvance.MaxAdvanceHours = 0
	if key := noAdvance.validate(); key != "errors.invalid_booking_window" {
		t.Errorf("0 hours ahead: %q", key)
	}
	var noBlock = goodRules()
	noBlock.NoShowBlockDays = 0
	if key := noBlock.validate(); key != "errors.invalid_no_show_penalty" {
		t.Errorf("no-show block of 0 days: %q", key)
	}
}
