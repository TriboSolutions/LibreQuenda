package checkin

import (
	"testing"

	"github.com/google/uuid"
)

func TestPayloadRoundTrip(t *testing.T) {
	var token = uuid.New()
	var parsed, ok = ParsePayload(" " + Payload(token) + "\n")
	if !ok || parsed != token {
		t.Fatalf("ParsePayload(Payload(%s)) = %s, %v", token, parsed, ok)
	}
	for _, text := range []string{"", "lq1:", "lq1:nope", "https://example.com", token.String(), "lq2:" + token.String()} {
		if _, ok := ParsePayload(text); ok {
			t.Errorf("ParsePayload(%q) accepted", text)
		}
	}
}

func TestDeviceKey(t *testing.T) {
	var key, hash, hint, err = NewDeviceKey()
	if err != nil {
		t.Fatal(err)
	}
	if !IsDeviceKey(key) {
		t.Errorf("IsDeviceKey(%q) = false", key)
	}
	if string(HashDeviceKey(key)) != string(hash) || hint != key[len(key)-4:] {
		t.Error("hash or hint do not match the key")
	}
	if IsDeviceKey("lqd_short") || IsDeviceKey("eyJhbGciOi") {
		t.Error("IsDeviceKey accepted a malformed key")
	}
}
