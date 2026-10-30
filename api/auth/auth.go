package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"librequenda/shared"
	"net/http"
	"os"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid access token")

// Zitadel resolves the instance from the Host header, so internal calls
// (http://zitadel-api:8080) must carry the public domain.
type hostTransport struct {
	host string
	base http.RoundTripper
}

func (t *hostTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var req *http.Request = r.Clone(r.Context())
	req.Host = t.host
	return t.base.RoundTrip(req)
}

var zitadelClient *http.Client
var publicKeys keyfunc.Keyfunc
var audience string

type runtimeConfig struct {
	ProjectId string `json:"projectId"`
}

// InitializeAuth loads the expected token audience and Zitadel's public signing keys.
// The keys are cached and refreshed in the background (hourly, and on unknown key ids).
func InitializeAuth() {
	if shared.OIDC_ISSUER == "" {
		panic("OIDC_ISSUER environment variable is not set")
	}
	if shared.ZITADEL_DOMAIN == "" {
		panic("ZITADEL_DOMAIN environment variable is not set")
	}
	if shared.ZITADEL_INTERNAL_URL == "" {
		panic("ZITADEL_INTERNAL_URL environment variable is not set")
	}
	if shared.RUNTIME_CONFIG_PATH == "" {
		panic("RUNTIME_CONFIG_PATH environment variable is not set")
	}

	zitadelClient = &http.Client{
		Timeout:   10 * time.Second,
		Transport: &hostTransport{host: shared.ZITADEL_DOMAIN, base: http.DefaultTransport},
	}
	audience = loadAudience()
	publicKeys = loadPublicKeys()
}

func loadAudience() string {
	var err error
	maxAttempts := 10
	for i := range maxAttempts {
		var content []byte
		content, err = os.ReadFile(shared.RUNTIME_CONFIG_PATH)
		if err == nil {
			var config runtimeConfig
			err = json.Unmarshal(content, &config)
			if err == nil && config.ProjectId != "" {
				shared.Logger.Info("token audience loaded: project %s", config.ProjectId)
				return config.ProjectId
			}
			if err == nil {
				err = errors.New("projectId is empty")
			}
		}
		wait := 2*i + 1
		shared.Logger.Error("could not read %s, retrying in %d seconds: %s", shared.RUNTIME_CONFIG_PATH, wait, err.Error())
		time.Sleep(time.Duration(wait) * time.Second)
	}
	panic("could not load token audience: " + err.Error())
}

func loadPublicKeys() keyfunc.Keyfunc {
	var jwksURL string = shared.ZITADEL_INTERNAL_URL + "/oauth/v2/keys"
	var failOnFirstRequest bool = false
	var keys keyfunc.Keyfunc
	var err error
	maxAttempts := 10
	for i := range maxAttempts {
		keys, err = keyfunc.NewDefaultOverrideCtx(context.Background(), []string{jwksURL}, keyfunc.Override{
			Client:                    zitadelClient,
			NoErrorReturnFirstHTTPReq: &failOnFirstRequest,
		})
		if err == nil {
			var loaded, readErr = keys.Storage().KeyReadAll(context.Background())
			if readErr == nil && len(loaded) > 0 {
				shared.Logger.Info("zitadel public keys loaded: %d key(s)", len(loaded))
				return keys
			}
			err = errors.Join(errors.New("no public keys returned"), readErr)
		}
		wait := 2*i + 1
		shared.Logger.Error("could not load zitadel public keys, retrying in %d seconds: %s", wait, err.Error())
		time.Sleep(time.Duration(wait) * time.Second)
	}
	panic("could not load zitadel public keys: " + err.Error())
}

// VerifyAccessToken checks signature, issuer, audience and expiry locally (no call to Zitadel).
func VerifyAccessToken(raw string) (*jwt.RegisteredClaims, error) {
	var claims *jwt.RegisteredClaims = &jwt.RegisteredClaims{}
	var _, err = jwt.ParseWithClaims(raw, claims, publicKeys.Keyfunc,
		jwt.WithIssuer(shared.OIDC_ISSUER),
		jwt.WithAudience(audience),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{"RS256"}),
	)
	if err != nil {
		return nil, errors.Join(ErrInvalidToken, err)
	}
	if claims.Subject == "" {
		return nil, errors.Join(ErrInvalidToken, errors.New("missing sub"))
	}
	return claims, nil
}

type UserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"` // the photo on the sign-in account, "" without one
}

// FetchUserInfo asks Zitadel for the profile of the token's user (name, email),
// which the access token itself does not carry.
func FetchUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	var req, err = http.NewRequestWithContext(ctx, http.MethodGet, shared.ZITADEL_INTERNAL_URL+"/oidc/v1/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	var res *http.Response
	res, err = zitadelClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo returned %d", res.StatusCode)
	}

	var info UserInfo
	err = json.NewDecoder(res.Body).Decode(&info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}
