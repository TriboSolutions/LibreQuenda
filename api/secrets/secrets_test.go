package secrets

import (
	"errors"
	"librequenda/shared"
	"testing"
)

func withKey(t *testing.T, key string) {
	var old = shared.SECRETS_KEY
	shared.SECRETS_KEY = key
	t.Cleanup(func() { shared.SECRETS_KEY = old })
}

const testKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

func TestSealOpen(t *testing.T) {
	withKey(t, testKey)
	var lalin, vigo = []byte("lalin"), []byte("vigo")
	var sealed, err = Seal("s3cret", lalin)
	if err != nil {
		t.Fatal(err)
	}
	if plain, err := Open(sealed, lalin); err != nil || plain != "s3cret" {
		t.Errorf("round trip: got %q, %v", plain, err)
	}
	if _, err := Open(sealed, vigo); !errors.Is(err, ErrSealed) {
		t.Errorf("another council's value: got %v, want ErrSealed", err)
	}
	sealed[len(sealed)-1] ^= 1
	if _, err := Open(sealed, lalin); !errors.Is(err, ErrSealed) {
		t.Errorf("tampered: got %v, want ErrSealed", err)
	}
	var again, _ = Seal("s3cret", lalin)
	if string(again) == string(sealed) {
		t.Error("two seals of the same value should differ (random nonce)")
	}
}

func TestNoKey(t *testing.T) {
	for _, key := range []string{"", "abcd", "zz" + testKey[2:]} {
		withKey(t, key)
		if _, err := Seal("x", nil); !errors.Is(err, ErrNoKey) {
			t.Errorf("key %q: got %v, want ErrNoKey", key, err)
		}
	}
}
