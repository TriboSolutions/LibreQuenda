package zitadel

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

const (
	eidasGiven  = "http://eidas.europa.eu/attributes/naturalperson/CurrentGivenName"
	eidasFamily = "http://eidas.europa.eu/attributes/naturalperson/CurrentFamilyName"
	eidasID     = "http://eidas.europa.eu/attributes/naturalperson/PersonIdentifier"
)

var mapping = IDPMapping{
	IdpID:      "361",
	GivenName:  []string{"FirstName", eidasGiven},
	FamilyName: []string{"FirstSurname+SecondSurname", eidasFamily},
	Email:      []string{"Email"},
	NIF:        []string{eidasID},
}

// a call as Zitadel sends it for a new person (protojson of RetrieveIdentityProviderIntentResponse)
func intentCall(idpID string, attributes map[string][]string, action string) []byte {
	var attrs, _ = json.Marshal(attributes)
	var userAction = `"createUser":{"human":{"profile":{"givenName":"","familyName":""},"email":{"email":"","sendCode":{}},"idpLinks":[{"idpId":"` + idpID + `","userId":"abc"}]}}`
	if action == "update" {
		userAction = `"userId":"u1","updateUser":{"userId":"u1","human":{"profile":{}}}`
	}
	return []byte(fmt.Sprintf(`{"fullMethod":"/zitadel.user.v2.UserService/RetrieveIdentityProviderIntent","response":{"details":{"sequence":"12"},"idpInformation":{"idpId":"%s","userId":"abc","rawInformation":{"id":"abc","attributes":%s}},%s}}`, idpID, attrs, userAction))
}

func TestIdentityIntentNewPerson(t *testing.T) {
	var out, err = mapping.IdentityIntent(intentCall("361", map[string][]string{
		eidasGiven:      {"Uxía"},
		"FirstSurname":  {"Castro"},
		"SecondSurname": {"Pazos"},
		"Email":         {"uxia@example.gal"},
		eidasID:         {"ES/ES/12345678z"},
	}, "create"))
	if err != nil || out == nil {
		t.Fatalf("IdentityIntent = %s, %v", out, err)
	}
	var got struct {
		Details    map[string]any `json:"details"`
		CreateUser struct {
			Human struct {
				Profile  map[string]string `json:"profile"`
				Email    map[string]any    `json:"email"`
				IdpLinks []any             `json:"idpLinks"`
			} `json:"human"`
			Metadata []map[string]string `json:"metadata"`
		} `json:"createUser"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	var human = got.CreateUser.Human
	if human.Profile["givenName"] != "Uxía" || human.Profile["familyName"] != "Castro Pazos" {
		t.Errorf("profile = %v", human.Profile)
	}
	if human.Email["email"] != "uxia@example.gal" || human.Email["sendCode"] == nil || human.Email["isVerified"] != nil {
		t.Errorf("an email the provider does not vouch for is verified by code: %v", human.Email)
	}
	if len(human.IdpLinks) != 1 || got.Details["sequence"] != "12" {
		t.Error("the rest of the response is kept")
	}
	if link, _ := human.IdpLinks[0].(map[string]any); link["userName"] != "uxia@example.gal" || link["userId"] != "abc" {
		t.Errorf("the link needs a user name: %v", link)
	}
	var nif, _ = base64.StdEncoding.DecodeString(got.CreateUser.Metadata[0]["value"])
	if got.CreateUser.Metadata[0]["key"] != NIFMetadataKey || string(nif) != "12345678Z" {
		t.Errorf("metadata = %v (%s)", got.CreateUser.Metadata, nif)
	}

	var verified = mapping
	verified.EmailVerified = true
	out, _ = verified.IdentityIntent(intentCall("361", map[string][]string{"FirstName": {"Uxía"}, "FirstSurname": {"Castro"}, "Email": {"uxia@example.gal"}}, "create"))
	var trusted struct {
		CreateUser struct {
			Human struct {
				Email map[string]any `json:"email"`
			} `json:"human"`
		} `json:"createUser"`
	}
	if err := json.Unmarshal(out, &trusted); err != nil {
		t.Fatal(err)
	}
	if email := trusted.CreateUser.Human.Email; email["isVerified"] != true || email["sendCode"] != nil {
		t.Errorf("email = %v", email)
	}
}

func TestIdentityIntentLeavesAlone(t *testing.T) {
	var cases = map[string][]byte{
		"another provider (Google)": intentCall("999", map[string][]string{"FirstName": {"Uxía"}, "FirstSurname": {"Castro"}, "Email": {"u@x.gal"}}, "create"),
		// the login then asks for the name and email itself
		"no email":                 intentCall("361", map[string][]string{"FirstName": {"Uxía"}, "FirstSurname": {"Castro"}}, "create"),
		"no surname":               intentCall("361", map[string][]string{"FirstName": {"Uxía"}, "Email": {"u@x.gal"}}, "create"),
		"existing account, no NIF": intentCall("361", map[string][]string{"FirstName": {"Uxía"}}, "update"),
	}
	for name, call := range cases {
		if out, err := mapping.IdentityIntent(call); out != nil || err != nil {
			t.Errorf("%s: %s, %v", name, out, err)
		}
	}
}

func TestIdentityIntentExistingAccountKeepsNIF(t *testing.T) {
	var out, _ = mapping.IdentityIntent(intentCall("361", map[string][]string{eidasID: {"ES/ES/12345678Z"}}, "update"))
	var got struct {
		UserID     string `json:"userId"`
		UpdateUser struct {
			Metadata []map[string]string `json:"metadata"`
		} `json:"updateUser"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if got.UserID != "u1" || len(got.UpdateUser.Metadata) != 1 || got.UpdateUser.Metadata[0]["key"] != NIFMetadataKey {
		t.Errorf("got %s", out)
	}
}

func TestFirst(t *testing.T) {
	var attributes = map[string]any{"FirstSurname": []any{"Castro"}, eidasFamily: []any{"Castro Pazos"}, "Empty": []any{" "}}
	var cases = []struct {
		alternatives []string
		want         string
	}{
		{[]string{"FirstSurname+SecondSurname", eidasFamily}, "Castro"}, // a single surname
		{[]string{"Empty", eidasFamily}, "Castro Pazos"},
		{[]string{"Missing"}, ""},
	}
	for _, c := range cases {
		if got := first(attributes, c.alternatives); got != c.want {
			t.Errorf("first(%v) = %q, want %q", c.alternatives, got, c.want)
		}
	}
}

func TestNormaliseNIF(t *testing.T) {
	var cases = map[string]string{
		"ES/ES/12345678Z": "12345678Z",
		"12345678-z":      "12345678Z",
		"IDES-X1234567L":  "X1234567L",
		" 12345678Z ":     "12345678Z",
		"":                "",
	}
	for in, want := range cases {
		if got := normaliseNIF(in); got != want {
			t.Errorf("normaliseNIF(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestVerifySignature(t *testing.T) {
	var body = []byte(`{"fullMethod":"x"}`)
	var now = time.Unix(1767225600, 0)
	var sign = func(key string, at time.Time) string {
		var mac = hmac.New(sha256.New, []byte(key))
		fmt.Fprintf(mac, "%d.%s", at.Unix(), body)
		return fmt.Sprintf("t=%d,v1=%s", at.Unix(), hex.EncodeToString(mac.Sum(nil)))
	}
	if err := VerifySignature(sign("k", now), body, "k", now.Add(time.Minute)); err != nil {
		t.Errorf("a good signature: %v", err)
	}
	var bad = map[string]string{
		"another key": sign("other", now),
		"too old":     sign("k", now.Add(-10*time.Minute)),
		"none":        "",
		"garbage":     "t=x,v1=zz",
	}
	for name, header := range bad {
		if err := VerifySignature(header, body, "k", now); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if err := VerifySignature(sign("", now), body, "", now); err == nil {
		t.Error("no key configured: accepted")
	}
}
