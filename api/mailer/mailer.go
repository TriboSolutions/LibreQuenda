// Package mailer sends the emails LibreQuenda writes itself (Zitadel sends its own).
package mailer

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"librequenda/secrets"
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

	"github.com/google/uuid"
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

// Relay is an SMTP server to send through: a council's own, or the one in the
// environment (SMTP_*). STARTTLS is used when the server offers it.
type Relay struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"` // e.g. Concello de Lalín <reservas@lalin.gal>
}

// SealRelay encrypts a council's whole relay for tenant_smtp.config: nothing
// about it is stored in clear.
func SealRelay(relay Relay, tenantId uuid.UUID) ([]byte, error) {
	var plain, err = json.Marshal(relay)
	if err != nil {
		return nil, err
	}
	return secrets.Seal(string(plain), tenantId[:])
}

// OpenRelay decrypts what SealRelay stored for the same council; nil for none.
func OpenRelay(sealed []byte, tenantId uuid.UUID) (*Relay, error) {
	if sealed == nil {
		return nil, nil
	}
	var plain, err = secrets.Open(sealed, tenantId[:])
	if err != nil {
		return nil, err
	}
	var relay Relay
	if err = json.Unmarshal([]byte(plain), &relay); err != nil {
		return nil, err
	}
	return &relay, nil
}

// ErrNoRelay: the council has no relay of its own and the environment has none either.
var ErrNoRelay = errors.New("no SMTP relay: the council has none and SMTP_HOST or SMTP_FROM is not set")

// EnvRelay is the relay in the environment, and whether it is set.
func EnvRelay() (Relay, bool) {
	var relay = Relay{
		Host:     shared.SMTP_HOST,
		Port:     shared.SMTP_PORT,
		Username: shared.SMTP_USERNAME,
		Password: shared.SMTP_PASSWORD,
		From:     shared.SMTP_FROM,
	}
	return relay, relay.Host != "" && relay.From != ""
}

// Pick: the council's relay when it has one (nil when it has not), else the
// environment's, else ErrNoRelay.
func Pick(council *Relay) (Relay, error) {
	if council != nil && council.Host != "" && council.From != "" {
		return *council, nil
	}
	if relay, ok := EnvRelay(); ok {
		return relay, nil
	}
	return Relay{}, ErrNoRelay
}

func Send(relay Relay, message Message) error {
	var from, err = mail.ParseAddress(relay.From)
	if err != nil {
		return fmt.Errorf("from address: %w", err)
	}
	var body []byte
	if body, err = message.build(from); err != nil {
		return err
	}
	return relay.send(from.Address, message.To, body)
}

// sendTimeout bounds a whole send, so a wrong host fails instead of hanging.
const sendTimeout = 30 * time.Second

// send is smtp.SendMail with a deadline.
func (relay Relay) send(from string, to string, body []byte) error {
	var conn, err = net.DialTimeout("tcp", net.JoinHostPort(relay.Host, relay.Port), sendTimeout)
	if err != nil {
		return err
	}
	if err = conn.SetDeadline(time.Now().Add(sendTimeout)); err != nil {
		conn.Close()
		return err
	}
	var client *smtp.Client
	if client, err = smtp.NewClient(conn, relay.Host); err != nil {
		conn.Close()
		return err
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err = client.StartTLS(&tls.Config{ServerName: relay.Host}); err != nil {
			return err
		}
	}
	if relay.Username != "" {
		// PlainAuth refuses to send the password without TLS, except to localhost
		if err = client.Auth(smtp.PlainAuth("", relay.Username, relay.Password, relay.Host)); err != nil {
			return err
		}
	}
	if err = client.Mail(from); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}
	var writer io.WriteCloser
	if writer, err = client.Data(); err != nil {
		return err
	}
	if _, err = writer.Write(body); err != nil {
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	return client.Quit()
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
