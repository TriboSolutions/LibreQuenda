package mailer

import (
	"bytes"
	"errors"
	"librequenda/shared"
	"testing"

	"github.com/google/uuid"
)

func TestPick(t *testing.T) {
	var council = &Relay{Host: "smtp.lalin.gal", Port: "587", From: "reservas@lalin.gal"}
	var withEnv = func(host string, from string) {
		var oldHost, oldFrom = shared.SMTP_HOST, shared.SMTP_FROM
		shared.SMTP_HOST, shared.SMTP_FROM = host, from
		t.Cleanup(func() { shared.SMTP_HOST, shared.SMTP_FROM = oldHost, oldFrom })
	}

	withEnv("smtp.example.gal", "noreply@example.gal")
	if relay, err := Pick(council); err != nil || relay.Host != "smtp.lalin.gal" {
		t.Errorf("the council's relay first: got %+v, %v", relay, err)
	}
	if relay, err := Pick(nil); err != nil || relay.Host != "smtp.example.gal" {
		t.Errorf("no council relay, the environment's: got %+v, %v", relay, err)
	}

	withEnv("", "")
	if relay, err := Pick(council); err != nil || relay.Host != "smtp.lalin.gal" {
		t.Errorf("the council's relay without the environment's: got %+v, %v", relay, err)
	}
	if _, err := Pick(nil); !errors.Is(err, ErrNoRelay) {
		t.Errorf("neither: got %v, want ErrNoRelay", err)
	}
}

func TestSealRelay(t *testing.T) {
	var old = shared.SECRETS_KEY
	shared.SECRETS_KEY = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	t.Cleanup(func() { shared.SECRETS_KEY = old })

	var lalin, vigo = uuid.New(), uuid.New()
	var relay = Relay{Host: "smtp.lalin.gal", Port: "587", Username: "reservas", Password: "s3cret", From: "Concello de Lalín <reservas@lalin.gal>"}
	var sealed, err = SealRelay(relay, lalin)
	if err != nil {
		t.Fatal(err)
	}
	for _, clear := range []string{relay.Host, relay.Username, relay.Password, "reservas@lalin.gal"} {
		if bytes.Contains(sealed, []byte(clear)) {
			t.Errorf("%q is in the sealed config in clear", clear)
		}
	}
	if opened, err := OpenRelay(sealed, lalin); err != nil || *opened != relay {
		t.Errorf("round trip: got %+v, %v", opened, err)
	}
	if _, err := OpenRelay(sealed, vigo); err == nil {
		t.Error("another council's config should not open")
	}
	if opened, err := OpenRelay(nil, lalin); opened != nil || err != nil {
		t.Errorf("no config: got %+v, %v", opened, err)
	}
}
