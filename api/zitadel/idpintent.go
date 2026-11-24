package zitadel

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"librequenda/shared"
)

// A public identity provider over SAML (Cl@ve, Chave365) sends its attributes
// under names of its own, which Zitadel does not map to a person's name or email.
// zitadel/setup.sh registers the API as an Actions v2 target on the response of
// RetrieveIdentityProviderIntent, which the login reads to create or update the
// account; IdentityIntent fills in what the attributes say. The mapping and the
// target's signing key are in shared.ZITADEL_IDP_MAPPING_FILE, written by the setup.

// IDPMapping: for each field, the attribute names to try in order (see first).
type IDPMapping struct {
	SigningKey    string   `json:"signingKey"`
	IdpID         string   `json:"idpId"`
	GivenName     []string `json:"givenName"`
	FamilyName    []string `json:"familyName"`
	Email         []string `json:"email"`
	NIF           []string `json:"nif"`
	EmailVerified bool     `json:"emailVerified"`
}

// NIFMetadataKey: the Zitadel user metadata that keeps the NIF the identity provider vouched for.
const NIFMetadataKey = "nif"

var ErrBadSignature = errors.New("zitadel: the action's signature does not match")

// LoadIDPMapping returns nil when no public identity provider is set up.
func LoadIDPMapping() (*IDPMapping, error) {
	var raw, err = os.ReadFile(shared.ZITADEL_IDP_MAPPING_FILE)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var mapping IDPMapping
	if err := json.Unmarshal(raw, &mapping); err != nil {
		return nil, err
	}
	return &mapping, nil
}

// VerifySignature checks Zitadel's "ZITADEL-Signature: t=<unix>,v1=<hex hmac>"
// over "<t>.<body>", within five minutes.
func VerifySignature(header string, body []byte, key string, now time.Time) error {
	var timestamp int64
	var signatures [][]byte
	for _, part := range strings.Split(header, ",") {
		var name, value, _ = strings.Cut(part, "=")
		switch name {
		case "t":
			timestamp, _ = strconv.ParseInt(value, 10, 64)
		case "v1":
			if sig, err := hex.DecodeString(value); err == nil {
				signatures = append(signatures, sig)
			}
		}
	}
	if key == "" || timestamp == 0 || now.Sub(time.Unix(timestamp, 0)).Abs() > 5*time.Minute {
		return ErrBadSignature
	}
	var mac = hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10) + "."))
	mac.Write(body)
	var want = mac.Sum(nil)
	for _, sig := range signatures {
		if hmac.Equal(sig, want) {
			return nil
		}
	}
	return ErrBadSignature
}

// IdentityIntent rewrites the intent response in a call to the target, or
// returns nil to leave it as it is (another identity provider, nothing to add).
func (m *IDPMapping) IdentityIntent(call []byte) ([]byte, error) {
	var payload struct {
		Response map[string]any `json:"response"`
	}
	if err := json.Unmarshal(call, &payload); err != nil {
		return nil, err
	}
	var response = payload.Response
	var info, _ = response["idpInformation"].(map[string]any)
	if info == nil || info["idpId"] != m.IdpID {
		return nil, nil
	}
	var raw, _ = info["rawInformation"].(map[string]any)
	var attributes, _ = raw["attributes"].(map[string]any)

	var givenName = first(attributes, m.GivenName)
	var familyName = first(attributes, m.FamilyName)
	var email = first(attributes, m.Email)
	var nif = normaliseNIF(first(attributes, m.NIF))

	var metadata []any
	if nif != "" {
		metadata = []any{map[string]any{"key": NIFMetadataKey, "value": base64.StdEncoding.EncodeToString([]byte(nif))}}
	}
	var changed = false
	if create, ok := response["createUser"].(map[string]any); ok {
		var human, _ = create["human"].(map[string]any)
		// Without an email Zitadel cannot create the account: the login then asks the
		// person for their name and email, which it only does while the name is missing.
		if human != nil && email != "" && givenName != "" && familyName != "" {
			human["profile"] = map[string]any{"givenName": givenName, "familyName": familyName}
			var setEmail = map[string]any{"email": email, "sendCode": map[string]any{}}
			if m.EmailVerified {
				setEmail = map[string]any{"email": email, "isVerified": true}
			}
			human["email"] = setEmail
			// Zitadel's SAML link has no user name, which creating the account requires
			var links, _ = human["idpLinks"].([]any)
			for _, l := range links {
				if link, ok := l.(map[string]any); ok && (link["userName"] == nil || link["userName"] == "") {
					link["userName"] = email
				}
			}
			if metadata != nil {
				create["metadata"] = metadata
			}
			changed = true
		}
	}
	if update, ok := response["updateUser"].(map[string]any); ok && metadata != nil {
		update["metadata"] = metadata
		changed = true
	}
	if !changed {
		return nil, nil
	}
	return json.Marshal(response)
}

func values(attributes map[string]any, name string) []string {
	var list, _ = attributes[name].([]any)
	var out []string
	for _, v := range list {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

// first: the first alternative with a value; "a+b" joins a and b (a first and a second surname).
func first(attributes map[string]any, alternatives []string) string {
	for _, alternative := range alternatives {
		var parts []string
		for _, name := range strings.Split(alternative, "+") {
			if v := values(attributes, strings.TrimSpace(name)); len(v) > 0 {
				parts = append(parts, v[0])
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
	}
	return ""
}

// normaliseNIF: eIDAS sends "ES/ES/12345678Z" (issuing country, receiving
// country, identifier); some providers add an "IDES-" prefix.
func normaliseNIF(value string) string {
	if i := strings.LastIndex(value, "/"); i >= 0 {
		value = value[i+1:]
	}
	value = strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(value)), "IDES-")
	return strings.ReplaceAll(strings.ReplaceAll(value, "-", ""), " ", "")
}
