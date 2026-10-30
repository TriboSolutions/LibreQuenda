package me

import (
	"errors"
	"librequenda/auth"
	"librequenda/shared"
	"net/http"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

// AdminSecondFactor lets an administrator through when they signed in with a
// second factor, or when REQUIRE_ADMIN_MFA is off. Otherwise it answers 403 with
// X-Mfa-Required, which the web turns into how to set one up, and returns false.
// Called by every admin check once the role is known.
func AdminSecondFactor(w http.ResponseWriter, r *http.Request, session auth.Session) bool {
	var err = auth.RequireSecondFactor(r.Context(), session)
	switch {
	case err == nil:
		return true
	case errors.Is(err, auth.ErrNoSecondFactor):
		w.Header().Set("X-Mfa-Required", "1")
		helpers.SendI18NErrorResponse(w, r, http.StatusForbidden, "errors.mfa_required")
	default:
		shared.Logger.Error("second factor check for %s: %v", session.Subject, err)
		helpers.SendI18NErrorResponse(w, r, http.StatusBadGateway, "errors.mfa_check_failed")
	}
	return false
}
