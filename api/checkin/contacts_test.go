package checkin

import (
	"strings"
	"testing"
)

func TestCleanStaffContacts(t *testing.T) {
	var clean, key = CleanStaffContacts([]StaffContact{{Name: "  Marta ", Phone: " 986 000 000", Place: "Conserxería "}})
	if key != "" || clean[0] != (StaffContact{Name: "Marta", Phone: "986 000 000", Place: "Conserxería"}) {
		t.Fatalf("trimmed: %+v %q", clean, key)
	}
	if _, key = CleanStaffContacts([]StaffContact{{Phone: "986"}}); key != "errors.invalid_staff_contact" {
		t.Errorf("a contact needs a name: %q", key)
	}
	if _, key = CleanStaffContacts([]StaffContact{{Name: strings.Repeat("a", 81)}}); key != "errors.invalid_staff_contact" {
		t.Errorf("name too long: %q", key)
	}
	if _, key = CleanStaffContacts(make([]StaffContact, MaxStaffContacts+1)); key != "errors.too_many_staff_contacts" {
		t.Errorf("too many: %q", key)
	}
}

func TestStaffContactsReadsBrokenAsEmpty(t *testing.T) {
	if got := StaffContacts([]byte("not json")); got == nil || len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
	if got := StaffContacts([]byte(`[{"name":"Marta","phone":"","place":"Conserxería"}]`)); len(got) != 1 || got[0].Place != "Conserxería" {
		t.Fatalf("got %#v", got)
	}
}

func TestStaffOrFallback(t *testing.T) {
	var staff = []byte(`[{"name":"Marta","phone":"986 1","place":"Conserxería"}]`)
	if got := StaffOrFallback(staff, "Pavillón", "986 2", "", "Concello", "986 3", ""); len(got) != 1 || got[0].Name != "Marta" {
		t.Errorf("staff first: %+v", got)
	}
	if got := StaffOrFallback(nil, "Pavillón", "986 2", "", "Concello", "986 3", "c@c.gal"); len(got) != 1 || got[0] != (StaffContact{Name: "Pavillón", Phone: "986 2"}) {
		t.Errorf("then the facility's own: %+v", got)
	}
	if got := StaffOrFallback([]byte("[]"), "Pavillón", "", "", "Concello", "986 3", "c@c.gal"); len(got) != 1 || got[0] != (StaffContact{Name: "Concello", Phone: "986 3", Email: "c@c.gal"}) {
		t.Errorf("then the council's: %+v", got)
	}
	if got := StaffOrFallback(nil, "Pavillón", "", "", "Concello", "", ""); got == nil || len(got) != 0 {
		t.Errorf("none at all: %+v", got)
	}
}
