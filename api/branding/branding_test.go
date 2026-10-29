package branding

import "testing"

func TestName(t *testing.T) {
	if got := (Branding{}).Name(); got != DefaultName {
		t.Errorf("default name = %q", got)
	}
	if got := (Branding{CustomName: "Reservas Lalín"}).Name(); got != "Reservas Lalín" {
		t.Errorf("custom name = %q", got)
	}
}
