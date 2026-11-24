package validate

import "testing"

func TestDNI(t *testing.T) {
	for _, ok := range []string{"12345678Z", "00000000T", "X1234567L", "Y1234567X", "Z1234567R"} {
		if !DNI(ok) {
			t.Errorf("DNI(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{"", "12345678A", "1234567Z", "12345678z", "X1234567A", "A1234567L", "12345678-Z", "123456789"} {
		if DNI(bad) {
			t.Errorf("DNI(%q) = true, want false", bad)
		}
	}
}

func TestPostalCode(t *testing.T) {
	for _, ok := range []string{"15001", "27002", "01001", "52001"} {
		if !PostalCode(ok) {
			t.Errorf("PostalCode(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{"", "00001", "53001", "1500", "150011", "15a01"} {
		if PostalCode(bad) {
			t.Errorf("PostalCode(%q) = true, want false", bad)
		}
	}
}

func TestPhone(t *testing.T) {
	for _, ok := range []string{"986000000", "+34 986 000 000", "(986) 00-00-00", "+" + "12345678901234567890123456789"} {
		if !Phone(ok, 30) {
			t.Errorf("Phone(%q, 30) = false, want true", ok)
		}
	}
	for _, bad := range []string{"", "12345", "986 000 000 ext. 2", "+" + "123456789012345678901234567890", "teléfono", "986/000/000"} {
		if Phone(bad, 30) {
			t.Errorf("Phone(%q, 30) = true, want false", bad)
		}
	}
}

func TestE164(t *testing.T) {
	for in, want := range map[string]string{
		"986000000": "+34986000000", "600 12 34 56": "+34600123456", "+34 986-000-000": "+34986000000",
		"0034986000000": "+34986000000", "+351 912 345 678": "+351912345678", "986.00.00.00": "+34986000000",
	} {
		if got, ok := E164(in); !ok || got != want {
			t.Errorf("E164(%q) = %q %v, want %q", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "12345", "+0123456789", "986 000 000 ext. 2", "+1234567890123456"} {
		if got, ok := E164(bad); ok {
			t.Errorf("E164(%q) = %q, want none", bad, got)
		}
	}
}
