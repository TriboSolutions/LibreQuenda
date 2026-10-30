package me

import (
	"librequenda/auth"
	"librequenda/shared"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

func TestAdminSecondFactorOff(t *testing.T) {
	helpers.GetI18nInstance(helpers.SetDefaultLocale("gl"), helpers.SetDirectory("../../i18n"))
	var previous = shared.REQUIRE_ADMIN_MFA
	shared.REQUIRE_ADMIN_MFA = false
	t.Cleanup(func() { shared.REQUIRE_ADMIN_MFA = previous })
	var w = httptest.NewRecorder()
	if !AdminSecondFactor(w, httptest.NewRequest(http.MethodGet, "/x", nil), auth.Session{AccessToken: "any"}) || w.Code != http.StatusOK {
		t.Errorf("off: let through, got %d", w.Code)
	}
}

func TestAdminSecondFactorUnavailable(t *testing.T) {
	helpers.GetI18nInstance(helpers.SetDefaultLocale("gl"), helpers.SetDirectory("../../i18n"))
	var previous, file = shared.REQUIRE_ADMIN_MFA, shared.ZITADEL_INTROSPECTION_FILE
	shared.REQUIRE_ADMIN_MFA, shared.ZITADEL_INTROSPECTION_FILE = true, "/nonexistent/introspection.json"
	t.Cleanup(func() { shared.REQUIRE_ADMIN_MFA, shared.ZITADEL_INTROSPECTION_FILE = previous, file })
	var w = httptest.NewRecorder()
	if AdminSecondFactor(w, httptest.NewRequest(http.MethodGet, "/x", nil), auth.Session{AccessToken: "any"}) || w.Code != http.StatusBadGateway {
		t.Errorf("Zitadel cannot be asked: refuse with 502, got %d", w.Code)
	}
}
