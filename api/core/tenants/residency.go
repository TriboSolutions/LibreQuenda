package tenants

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"librequenda/auth"
	"librequenda/model"
)

// eligibility: the councils whose residents-only facilities a verified resident
// can book (their own, and those they administer). A person is a resident of one
// council only, so the others' residents-only facilities are no use to them and
// are left out of what they are offered. nil: no restriction (signed out, not
// verified anywhere, or a superadministrator).
type eligibility map[string]bool

// allows: the facility is for anyone, or for residents and the person is one there.
func (e eligibility) allows(council string, residentsOnly bool) bool {
	return e == nil || !residentsOnly || e[council]
}

// eligibilityOf: the restriction for the person behind subject (Zitadel's "sub").
func eligibilityOf(ctx context.Context, db *sql.DB, subject string) eligibility {
	var row, err = model.New(db).ResidentCouncilsBySubject(ctx, subject)
	if err != nil || !row.Restricted {
		return nil // unknown to the API yet, or nothing to restrict: everything shows
	}
	var councils = make(eligibility, len(row.Councils))
	for _, slug := range row.Councils {
		councils[slug] = true
	}
	return councils
}

// eligibilityFromRequest: on the public lists, for whoever is signed in (a bad or
// missing token is the same as signed out: the lists are public).
func eligibilityFromRequest(r *http.Request, db *sql.DB) eligibility {
	var token, ok = strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || token == "" {
		return nil
	}
	var claims, err = auth.VerifyAccessToken(token)
	if err != nil || claims.Subject == "" {
		return nil
	}
	return eligibilityOf(r.Context(), db, claims.Subject)
}

// ForResident narrows the filters for whoever the token's subject is (Zitadel's
// "sub"), as the public lists do for someone signed in.
func (f *Filters) ForResident(ctx context.Context, db *sql.DB, subject string) {
	f.Eligible = eligibilityOf(ctx, db, subject)
}
