// Package mailer sends the emails LibreQuenda writes itself (Zitadel sends its own).
package mailer

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"librequenda/model"
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

	// for the email registry (LOG_DIR/mail-*.log): what kind of email, and whose council
	Kind     string
	TenantID uuid.UUID
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

// Send sends the message through the relay, and writes it down in the email
// registry whatever happens.
func Send(relay Relay, message Message) error {
	var err = sendMessage(relay, message)
	Record(relay, message, err)
	return err
}

func sendMessage(relay Relay, message Message) error {
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

// mailRecord is a line of the email registry: who, what, through which relay,
// and how it went (never the body).
type mailRecord struct {
	Time    string `json:"time"`
	Status  string `json:"status"` // sent | failed
	Kind    string `json:"kind,omitempty"`
	Tenant  string `json:"tenant_id,omitempty"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	From    string `json:"from,omitempty"`
	Relay   string `json:"relay,omitempty"`
	Error   string `json:"error,omitempty"`
}

// registry: where Record writes the emails down (the email_log table); set by UseRegistry.
var registry *sql.DB

// UseRegistry: the database the emails are written down in, from main.
func UseRegistry(db *sql.DB) { registry = db }

// Record writes an email down in the registry (email_log, which the
// superadministrators read) and in the API's log: sent, or why not. Also for
// the ones that never reached a relay.
func Record(relay Relay, message Message, sendErr error) {
	var record = mailRecord{
		Time:    time.Now().Format(time.RFC3339),
		Status:  "sent",
		Kind:    message.Kind,
		To:      message.To,
		Subject: message.Subject,
		From:    relay.From,
	}
	if message.TenantID != uuid.Nil {
		record.Tenant = message.TenantID.String()
	}
	if relay.Host != "" {
		record.Relay = net.JoinHostPort(relay.Host, relay.Port)
	}
	if sendErr != nil {
		record.Status = "failed"
		record.Error = sendErr.Error()
		shared.Logger.Warn("email %s to %s not sent: %v", message.Kind, message.To, sendErr)
	} else {
		shared.Logger.Info("email %s sent to %s via %s", message.Kind, message.To, record.Relay)
	}
	if registry == nil {
		return
	}
	var ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var err = model.New(registry).InsertEmailLog(ctx, model.InsertEmailLogParams{
		Status:    record.Status,
		Kind:      truncate(record.Kind, 64),
		TenantID:  uuid.NullUUID{UUID: message.TenantID, Valid: message.TenantID != uuid.Nil},
		Recipient: truncate(record.To, 320),
		Subject:   record.Subject,
		Sender:    truncate(record.From, 320),
		Relay:     truncate(record.Relay, 255),
		Error:     record.Error,
	})
	if err != nil {
		// the API's log keeps it anyway
		shared.Logger.Error("email log: %v", err)
	}
}

func truncate(s string, max int) string {
	if len([]rune(s)) <= max {
		return s
	}
	return string([]rune(s)[:max])
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
