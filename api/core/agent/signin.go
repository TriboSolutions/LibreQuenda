package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"librequenda/auth"
	"librequenda/shared"
)

// An agent signs the person in with the OAuth device flow (RFC 8628) against Zitadel:
// sign_in shows a link and a code, the person approves on the installation's own
// sign-in page, and the tokens stay here, in memory, for that MCP session only. They
// never reach the agent. A restart of the API, or a new session, asks again.

// grant: one MCP session's sign-in, pending or done.
type grant struct {
	mu sync.Mutex
	// pending: the device flow under way
	deviceCode, userCode, verifyURL string
	deviceExpires                   time.Time
	interval                        time.Duration
	nextPoll                        time.Time
	// done: the person's tokens
	access, refresh string
	accessExpires   time.Time
	lastUsed        time.Time
}

// grants: by MCP session id. Forgotten after a day without use.
type grants struct {
	mu sync.Mutex
	by map[string]*grant
}

const forgetAfter = 24 * time.Hour

func (g *grants) of(session string) *grant {
	g.mu.Lock()
	defer g.mu.Unlock()
	var now = time.Now()
	for id, one := range g.by {
		if now.Sub(one.lastUsed) > forgetAfter {
			delete(g.by, id)
		}
	}
	var one = g.by[session]
	if one == nil {
		one = &grant{}
		g.by[session] = one
	}
	one.lastUsed = now
	return one
}

func (g *grants) forget(session string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.by, session)
}

var errNotSignedIn = errors.New("not signed in: call sign_in first, show the person the link and code it gives, and call sign_in again once they approved")

// tokenReply: Zitadel's token and device authorization endpoints.
type tokenReply struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	// device authorization
	DeviceCode string `json:"device_code"`
	UserCode   string `json:"user_code"`
	Interval   int    `json:"interval"`
}

var zitadel = &http.Client{Timeout: 10 * time.Second}

// post: a form to Zitadel, inside the network, as if through the public address
// (Zitadel tells its instance, and the tokens' issuer, by the host and scheme).
func post(ctx context.Context, path string, form url.Values) (tokenReply, error) {
	var reply tokenReply
	var req, err = http.NewRequestWithContext(ctx, http.MethodPost, shared.ZITADEL_INTERNAL_URL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return reply, err
	}
	req.Host = shared.ZITADEL_DOMAIN
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if strings.HasPrefix(shared.OIDC_ISSUER, "https://") {
		req.Header.Set("X-Forwarded-Proto", "https")
	}
	var res *http.Response
	if res, err = zitadel.Do(req); err != nil {
		return reply, err
	}
	defer res.Body.Close()
	if err = json.NewDecoder(http.MaxBytesReader(nil, res.Body, 64*1024)).Decode(&reply); err != nil {
		return reply, err
	}
	if reply.Error == "" && res.StatusCode >= 400 {
		reply.Error = res.Status
	}
	return reply, nil
}

func scope() string {
	return "openid profile email offline_access urn:zitadel:iam:org:project:id:" + auth.ProjectId() + ":aud"
}

// start asks Zitadel for a code; the link goes to the installation's own sign-in page.
func (g *grant) start(ctx context.Context) error {
	if auth.MCPClientId() == "" {
		return errors.New("signing in agents is not set up on this installation (zitadel-setup creates the «LibreQuenda MCP» app)")
	}
	var reply, err = post(ctx, "/oauth/v2/device_authorization", url.Values{"client_id": {auth.MCPClientId()}, "scope": {scope()}})
	if err != nil || reply.Error != "" || reply.DeviceCode == "" {
		shared.Logger.Error("mcp: device authorization: %v %s", err, reply.Error)
		return errors.New("the sign-in service did not answer; try again in a moment")
	}
	g.deviceCode, g.userCode = reply.DeviceCode, reply.UserCode
	g.verifyURL = strings.TrimRight(shared.OIDC_ISSUER, "/") + "/ui/v2/login/device?user_code=" + url.QueryEscape(reply.UserCode)
	g.deviceExpires = time.Now().Add(time.Duration(max(reply.ExpiresIn, 60)) * time.Second)
	g.interval = time.Duration(max(reply.Interval, 5)) * time.Second
	g.nextPoll = time.Time{}
	return nil
}

func (g *grant) pending() bool {
	return g.deviceCode != "" && time.Now().Before(g.deviceExpires)
}

func (g *grant) keep(reply tokenReply) {
	g.access = reply.AccessToken
	if reply.RefreshToken != "" {
		g.refresh = reply.RefreshToken
	}
	g.accessExpires = time.Now().Add(time.Duration(max(reply.ExpiresIn, 60)) * time.Second)
	g.deviceCode, g.userCode, g.verifyURL = "", "", ""
}

// token: a valid access token, polling a pending sign-in or renewing an expired one;
// errNotSignedIn when there is none to be had.
func (g *grant) token(ctx context.Context) (string, error) {
	var now = time.Now()
	if g.access != "" && now.Before(g.accessExpires.Add(-30*time.Second)) {
		return g.access, nil
	}
	if g.refresh != "" {
		var reply, err = post(ctx, "/oauth/v2/token", url.Values{"client_id": {auth.MCPClientId()}, "grant_type": {"refresh_token"}, "refresh_token": {g.refresh}})
		if err == nil && reply.AccessToken != "" {
			g.keep(reply)
			return g.access, nil
		}
		g.access, g.refresh = "", "" // revoked or too old: sign in again
	}
	if g.pending() && !now.Before(g.nextPoll) {
		g.nextPoll = now.Add(g.interval) // Zitadel asks not to be polled faster
		var reply, err = post(ctx, "/oauth/v2/token", url.Values{
			"client_id":   {auth.MCPClientId()},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {g.deviceCode},
		})
		if err == nil && reply.AccessToken != "" {
			g.keep(reply)
			return g.access, nil
		}
		if reply.Error == "access_denied" || reply.Error == "expired_token" {
			g.deviceCode = ""
		}
	}
	return "", errNotSignedIn
}

// signOut revokes the refresh token at Zitadel (best effort) and forgets it all.
func (g *grant) signOut(ctx context.Context) {
	if g.refresh != "" {
		_, _ = post(ctx, "/oauth/v2/revoke", url.Values{"client_id": {auth.MCPClientId()}, "token": {g.refresh}, "token_type_hint": {"refresh_token"}})
	}
	g.access, g.refresh, g.accessExpires = "", "", time.Time{}
	g.deviceCode, g.userCode, g.verifyURL = "", "", ""
}
