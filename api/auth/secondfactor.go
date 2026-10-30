package auth

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"librequenda/shared"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Administrators sign in with a second factor (REQUIRE_ADMIN_MFA). Zitadel's JWT
// access token carries no "amr" claim; token introspection does ("pwd" for a
// password alone; "otp", "u2f", "mfa"… with a second factor), so admin routes
// introspect the token once and remember the answer until the token expires.
// Introspection needs the API application's credentials, written by zitadel-setup.

var ErrNoSecondFactor = errors.New("signed in without a second factor")

// HasSecondFactor: the sign-in's authentication methods include a second factor,
// or a passkey ("user": the device checked who it was, a factor on its own).
func HasSecondFactor(amr []string) bool {
	for _, method := range amr {
		switch method {
		case "mfa", "otp", "u2f", "user", "sms", "email", "hwk", "swk":
			return true
		}
	}
	return false
}

type introspectionCredentials struct {
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

type checkedToken struct {
	secondFactor bool
	until        time.Time
}

var checkedMu sync.Mutex
var checked = map[[32]byte]checkedToken{}

// RequireSecondFactor: nil when admins need no second factor here or this
// sign-in used one; ErrNoSecondFactor when it did not; another error when
// Zitadel could not be asked.
func RequireSecondFactor(ctx context.Context, session Session) error {
	if !shared.REQUIRE_ADMIN_MFA {
		return nil
	}
	var key = sha256.Sum256([]byte(session.AccessToken))
	var now = time.Now()
	checkedMu.Lock()
	var known, found = checked[key]
	checkedMu.Unlock()
	if !found || now.After(known.until) {
		var amr, expires, err = introspect(ctx, session.AccessToken)
		if err != nil {
			return err
		}
		known = checkedToken{secondFactor: HasSecondFactor(amr), until: expires}
		checkedMu.Lock()
		for k, v := range checked { // forget expired tokens as new ones come
			if now.After(v.until) {
				delete(checked, k)
			}
		}
		checked[key] = known
		checkedMu.Unlock()
	}
	if !known.secondFactor {
		return ErrNoSecondFactor
	}
	return nil
}

func introspect(ctx context.Context, accessToken string) ([]string, time.Time, error) {
	var content, err = os.ReadFile(shared.ZITADEL_INTROSPECTION_FILE)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("introspection credentials: %w", err)
	}
	var credentials introspectionCredentials
	if err = json.Unmarshal(content, &credentials); err != nil || credentials.ClientId == "" {
		return nil, time.Time{}, errors.New("introspection credentials: unreadable")
	}
	var req *http.Request
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, shared.ZITADEL_INTERNAL_URL+"/oauth/v2/introspect",
		strings.NewReader(url.Values{"token": {accessToken}}.Encode()))
	if err != nil {
		return nil, time.Time{}, err
	}
	req.SetBasicAuth(credentials.ClientId, credentials.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var res *http.Response
	if res, err = zitadelClient.Do(req); err != nil {
		return nil, time.Time{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, time.Time{}, fmt.Errorf("introspection returned %d", res.StatusCode)
	}
	var reply struct {
		Active bool     `json:"active"`
		Amr    []string `json:"amr"`
		Exp    int64    `json:"exp"`
	}
	if err = json.NewDecoder(res.Body).Decode(&reply); err != nil {
		return nil, time.Time{}, err
	}
	if !reply.Active {
		return nil, time.Time{}, ErrInvalidToken
	}
	return reply.Amr, time.Unix(reply.Exp, 0), nil
}
