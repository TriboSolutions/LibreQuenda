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
