package invites

import "testing"

func TestStaffJoinInputValidate(t *testing.T) {
	var cases = []struct {
		in   StaffJoinInput
		want string
	}{
		{StaffJoinInput{Email: " Ana@Example.org "}, ""},
		{StaffJoinInput{Name: "  Xan  Pérez ", Age: 34}, ""},
		{StaffJoinInput{}, "errors.staff_booking_holder"},
		{StaffJoinInput{Email: "ana@example.org", Name: "Ana", Age: 30}, "errors.staff_booking_holder"},
		{StaffJoinInput{Email: "Ana <ana@example.org>"}, "errors.invalid_email"},
		{StaffJoinInput{Name: "Xan"}, "errors.invalid_guest_age"},
		{StaffJoinInput{Name: "Xan", Age: 121}, "errors.invalid_guest_age"},
	}
	for _, c := range cases {
		var in = c.in
		if got := in.Validate(); got != c.want {
			t.Errorf("%+v: Validate() = %q, want %q", c.in, got, c.want)
		}
	}
	var in = StaffJoinInput{Name: "  Xan  Pérez ", Age: 34}
	if in.Validate(); in.Name != "Xan Pérez" {
		t.Errorf("name not tidied: %q", in.Name)
	}
}
