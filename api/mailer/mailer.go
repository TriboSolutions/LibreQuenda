// Package mailer sends the emails LibreQuenda writes itself (Zitadel sends its own).
package mailer

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"librequenda/shared"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

// Inline is an image the HTML shows with <img src="cid:CID">.
type Inline struct {
	CID         string
	ContentType string
	Filename    string
	Data        []byte
}

type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
	Inline  []Inline
}

// Enabled: SMTP is configured.
func Enabled() bool {
	return shared.SMTP_HOST != "" && shared.SMTP_FROM != ""
}

func Send(message Message) error {
	var from, err = mail.ParseAddress(shared.SMTP_FROM)
	if err != nil {
		return fmt.Errorf("SMTP_FROM: %w", err)
	}
	var body []byte
	if body, err = message.build(from); err != nil {
		return err
	}
	var auth smtp.Auth
	if shared.SMTP_USERNAME != "" {
		auth = smtp.PlainAuth("", shared.SMTP_USERNAME, shared.SMTP_PASSWORD, shared.SMTP_HOST)
	}
	return smtp.SendMail(net.JoinHostPort(shared.SMTP_HOST, shared.SMTP_PORT), auth, from.Address, []string{message.To}, body)
}

// build writes the MIME message: multipart/related (the HTML and its inline
// images) around multipart/alternative (plain text and HTML).
func (m Message) build(from *mail.Address) ([]byte, error) {
	var buffer bytes.Buffer
	var related = multipart.NewWriter(&buffer)
	var domain = from.Address[strings.LastIndex(from.Address, "@")+1:]

	var headers = []string{
		"From: " + from.String(),
		"To: " + m.To,
		"Subject: " + mime.QEncoding.Encode("utf-8", m.Subject),
		"Date: " + time.Now().Format(time.RFC1123Z),
		"Message-ID: <" + randomId() + "@" + domain + ">",
		"MIME-Version: 1.0",
		`Content-Type: multipart/related; type="multipart/alternative"; boundary=` + related.Boundary(),
	}
	buffer.WriteString(strings.Join(headers, "\r\n") + "\r\n\r\n")

	var alternativeBuffer bytes.Buffer
	var alternative = multipart.NewWriter(&alternativeBuffer)
	if err := writeText(alternative, "text/plain; charset=utf-8", m.Text); err != nil {
		return nil, err
	}
	if err := writeText(alternative, "text/html; charset=utf-8", m.HTML); err != nil {
		return nil, err
	}
	if err := alternative.Close(); err != nil {
		return nil, err
	}
	var part, err = related.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"multipart/alternative; boundary=" + alternative.Boundary()},
	})
	if err != nil {
		return nil, err
	}
	if _, err = part.Write(alternativeBuffer.Bytes()); err != nil {
		return nil, err
	}

	for _, image := range m.Inline {
		part, err = related.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {image.ContentType},
			"Content-Transfer-Encoding": {"base64"},
			"Content-ID":                {"<" + image.CID + ">"},
			"Content-Disposition":       {`inline; filename="` + image.Filename + `"`},
		})
		if err != nil {
			return nil, err
		}
		if err = writeBase64(part, image.Data); err != nil {
			return nil, err
		}
	}
	if err = related.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func writeText(writer *multipart.Writer, contentType string, text string) error {
	var part, err = writer.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {contentType},
		"Content-Transfer-Encoding": {"quoted-printable"},
	})
	if err != nil {
		return err
	}
	var encoder = quotedprintable.NewWriter(part)
	if _, err = encoder.Write([]byte(text)); err != nil {
		return err
	}
	return encoder.Close()
}

// writeBase64 wraps lines at 76 characters, as MIME requires.
func writeBase64(part interface{ Write([]byte) (int, error) }, data []byte) error {
	var encoded = base64.StdEncoding.EncodeToString(data)
	for len(encoded) > 76 {
		if _, err := part.Write([]byte(encoded[:76] + "\r\n")); err != nil {
			return err
		}
		encoded = encoded[76:]
	}
	_, err := part.Write([]byte(encoded + "\r\n"))
	return err
}

func randomId() string {
	var id = make([]byte, 16)
	_, _ = rand.Read(id)
	return hex.EncodeToString(id)
}
