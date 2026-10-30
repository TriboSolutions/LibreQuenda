package auth

import (
	"context"
	"encoding/json"
	"errors"
	"librequenda/shared"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHasSecondFactor(t *testing.T) {
	var cases = []struct {
		amr  []string
		want bool
	}{
		{[]string{"pwd"}, false},              // what Zitadel reports for a password alone
		{nil, false},                          // a token from before, or none
		{[]string{"pwd", "otp", "mfa"}, true}, // password and an authenticator app
		{[]string{"pwd", "u2f"}, true},        // password and a security key
		{[]string{"user"}, true},              // a passkey
	}
	for _, c := range cases {
		if got := HasSecondFactor(c.amr); got != c.want {
			t.Errorf("HasSecondFactor(%v) = %v, want %v", c.amr, got, c.want)
		}
	}
}

// fakeZitadel answers introspection as Zitadel does, counting the calls.
func fakeZitadel(t *testing.T, amrByToken map[string][]string, calls *int) {
	var server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*calls++
		if id, secret, ok := r.BasicAuth(); !ok || id != "client" || secret != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = r.ParseForm()
		var amr, known = amrByToken[r.PostForm.Get("token")]
		_ = json.NewEncoder(w).Encode(map[string]any{"active": known, "amr": amr, "exp": time.Now().Add(10 * time.Minute).Unix()})
	}))
	t.Cleanup(server.Close)
	var file = filepath.Join(t.TempDir(), "introspection.json")
	if err := os.WriteFile(file, []byte(`{"clientId":"client","clientSecret":"secret"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var previous = []any{shared.REQUIRE_ADMIN_MFA, shared.ZITADEL_INTERNAL_URL, shared.ZITADEL_INTROSPECTION_FILE, zitadelClient}
	shared.REQUIRE_ADMIN_MFA, shared.ZITADEL_INTERNAL_URL, shared.ZITADEL_INTROSPECTION_FILE, zitadelClient = true, server.URL, file, server.Client()
	t.Cleanup(func() {
		shared.REQUIRE_ADMIN_MFA = previous[0].(bool)
		shared.ZITADEL_INTERNAL_URL = previous[1].(string)
		shared.ZITADEL_INTROSPECTION_FILE = previous[2].(string)
		zitadelClient, _ = previous[3].(*http.Client)
	})
}

func TestRequireSecondFactor(t *testing.T) {
	var calls int
	fakeZitadel(t, map[string][]string{"password-only": {"pwd"}, "with-otp": {"pwd", "otp", "mfa"}}, &calls)
	var ctx = context.Background()

	if err := RequireSecondFactor(ctx, Session{AccessToken: "with-otp"}); err != nil {
		t.Errorf("with a second factor: %v", err)
	}
	if err := RequireSecondFactor(ctx, Session{AccessToken: "with-otp"}); err != nil || calls != 1 {
		t.Errorf("the answer is remembered until the token expires: err %v, %d calls", err, calls)
	}
	if err := RequireSecondFactor(ctx, Session{AccessToken: "password-only"}); !errors.Is(err, ErrNoSecondFactor) {
		t.Errorf("password only: %v", err)
	}
	if err := RequireSecondFactor(ctx, Session{AccessToken: "revoked"}); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("an inactive token: %v", err)
	}

	shared.REQUIRE_ADMIN_MFA = false
	if err := RequireSecondFactor(ctx, Session{AccessToken: "password-only"}); err != nil {
		t.Errorf("not required here: %v", err)
	}
}
