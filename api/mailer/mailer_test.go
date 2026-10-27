package mailer

import (
	"bytes"
	"io"
	"mime"
	"net/mail"
	"strings"
	"testing"
)

func TestBuildParses(t *testing.T) {
	var from = &mail.Address{Name: "LibreQuenda", Address: "noreply@example.com"}
	var message = Message{
		To:      "vecina@example.com",
		Subject: "Lembrete: Pista de pádel",
		Text:    "Ola, Uxía",
		HTML:    `<p>Ola, Uxía</p><img src="cid:qr">`,
		Inline:  []Inline{{CID: "qr", ContentType: "image/png", Filename: "qr.png", Data: bytes.Repeat([]byte{1, 2, 3}, 100)}},
	}
	var raw, err = message.build(from)
	if err != nil {
		t.Fatal(err)
	}
	var parsed *mail.Message
	if parsed, err = mail.ReadMessage(bytes.NewReader(raw)); err != nil {
		t.Fatal(err)
	}
	var subject string
	if subject, err = new(mime.WordDecoder).DecodeHeader(parsed.Header.Get("Subject")); err != nil || subject != message.Subject {
		t.Errorf("subject = %q, %v", subject, err)
	}
	if !strings.HasPrefix(parsed.Header.Get("Content-Type"), "multipart/related") {
		t.Errorf("content type = %q", parsed.Header.Get("Content-Type"))
	}
	var body, _ = io.ReadAll(parsed.Body)
	for _, want := range []string{"Content-ID: <qr>", "text/plain", "text/html"} {
		if !bytes.Contains(body, []byte(want)) {
			t.Errorf("body lacks %q", want)
		}
	}
}
