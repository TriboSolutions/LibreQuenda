package checkin

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// StaffContact: someone residents can ask on site to be checked in or to book a
// free turn. Not an account: only what the council chooses to show.
type StaffContact struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Place string `json:"place"`           // where to find them, e.g. "Reception, ground floor"
	Email string `json:"email,omitempty"` // the fallback only: the facility's or council's
}

const (
	MaxStaffContacts = 10
	maxContactName   = 80
	maxContactPhone  = 30
	maxContactPlace  = 120
)

// StaffContacts reads the stored list; a broken one reads as empty.
func StaffContacts(raw []byte) []StaffContact {
	var contacts = []StaffContact{}
	if len(raw) > 0 && json.Unmarshal(raw, &contacts) != nil {
		return []StaffContact{}
	}
	return contacts
}

// CleanStaffContacts trims each contact and checks the limits; the error is an i18n key.
func CleanStaffContacts(contacts []StaffContact) ([]StaffContact, string) {
	if len(contacts) > MaxStaffContacts {
		return nil, "errors.too_many_staff_contacts"
	}
	var clean = make([]StaffContact, 0, len(contacts))
	for _, c := range contacts {
		c = StaffContact{Name: strings.TrimSpace(c.Name), Phone: strings.TrimSpace(c.Phone), Place: strings.TrimSpace(c.Place)}
		if c.Name == "" || utf8.RuneCountInString(c.Name) > maxContactName ||
			utf8.RuneCountInString(c.Phone) > maxContactPhone || utf8.RuneCountInString(c.Place) > maxContactPlace {
			return nil, "errors.invalid_staff_contact"
		}
		clean = append(clean, c)
	}
	return clean, ""
}

// StaffOrFallback: the staff contacts, or without any, the facility's own contact,
// else the council's, as everywhere else; empty when there is none of them.
func StaffOrFallback(raw []byte, facilityName, facilityPhone, facilityEmail, councilName, councilPhone, councilEmail string) []StaffContact {
	if contacts := StaffContacts(raw); len(contacts) > 0 {
		return contacts
	}
	var name, phone, email = facilityName, strings.TrimSpace(facilityPhone), strings.TrimSpace(facilityEmail)
	if phone == "" && email == "" {
		name, phone, email = councilName, strings.TrimSpace(councilPhone), strings.TrimSpace(councilEmail)
	}
	if phone == "" && email == "" {
		return []StaffContact{}
	}
	return []StaffContact{{Name: name, Phone: phone, Email: email}}
}
