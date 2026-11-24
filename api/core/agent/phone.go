package agent

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"librequenda/branding"
	"librequenda/core/bookings"
	"librequenda/cronJobs"
	"librequenda/mailer"
	"librequenda/model"
	"librequenda/shared"
	"librequenda/validate"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The phone line: the voice service answers calls and uses these tools for the
// caller. It is trusted with VOICE_API_KEY (X-Voice-Key) and sends the caller's
// number with every request (X-Caller-Phone). A caller signs in with a code emailed
// to the one account whose profile has that number, kept in memory for the MCP
// session like a sign_in; or books with just their name, for their number.

const (
	codeTTL      = 10 * time.Minute
	codeAttempts = 5           // wrong codes before it is thrown away
	codeEvery    = time.Minute // between two codes to one number
	codesPerHour = 5           // codes to one number
	codeDigits   = 6
)

var (
	errNotPhone         = errors.New("this tool is only for the phone line, in a call through the voice service")
	errPhoneNotSignedIn = errors.New("not signed in: on the phone, call phone_send_code to email a code to the caller's account and phone_verify_code with it; without an account, book_turn with holder_name books for the caller's number")
)

// SMSConfirmation texts a phone caller without an account their booking.
// TODO(sms): nil until the installation has an SMS provider; set it at start-up
// to send one, and book_turn answers confirmation "sms" instead of "read_out".
var SMSConfirmation func(ctx context.Context, phone string, booking Booking) error

// callerPhone: the caller's number (E.164) when the voice service sent this request
// for a phone call, "" otherwise (no voice line, a wrong key, or no valid number).
func (a *agent) callerPhone(req *mcp.CallToolRequest) string {
	if a.voiceKey == "" {
		return ""
	}
	var key = header(req, "X-Voice-Key")
	if subtle.ConstantTimeCompare([]byte(key), []byte(a.voiceKey)) != 1 {
		return ""
	}
	var phone, ok = validate.E164(header(req, "X-Caller-Phone"))
	if !ok {
		return ""
	}
	return phone
}

// caller: the account a phone caller proved with a code.
type caller struct {
	id                   uuid.UUID
	subject, name, email string
}

// caller: who signed in on this MCP session by phone, for that same number; nil when nobody.
func (g *grants) caller(session, phone string) *caller {
	var one = g.of(session)
	one.mu.Lock()
	defer one.mu.Unlock()
	if one.phone != phone {
		return nil
	}
	return one.caller
}

// phoneCode: the code emailed for one number, and when the last ones were sent.
type phoneCode struct {
	caller
	code     string
	expires  time.Time
	attempts int
	sent     []time.Time // within the last hour
}

type phoneCodes struct {
	mu sync.Mutex
	by map[string]*phoneCode
}

// allow: whether another code may go to this number now; it counts it when so.
func (c *phoneCodes) allow(phone string, now time.Time) (*phoneCode, bool) {
	var one = c.by[phone]
	if one == nil {
		one = &phoneCode{}
		c.by[phone] = one
	}
	var recent = one.sent[:0]
	for _, at := range one.sent {
		if now.Sub(at) < time.Hour {
			recent = append(recent, at)
		}
	}
	one.sent = recent
	if len(recent) >= codesPerHour || (len(recent) > 0 && now.Sub(recent[len(recent)-1]) < codeEvery) {
		return one, false
	}
	one.sent = append(one.sent, now)
	return one, true
}

func newCode() (string, error) {
	var n, err = rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", codeDigits, n.Int64()), nil
}

// emailHint: enough of an address for its owner to know it, "m***@gmail.com".
func emailHint(email string) string {
	var local, domain, ok = strings.Cut(email, "@")
	if !ok || local == "" {
		return "***"
	}
	var first, _ = utf8.DecodeRuneInString(local)
	return string(first) + "***@" + domain
}

// sendCodeEmail: through the installation's relay (a call belongs to no council).
func sendCodeEmail(_ context.Context, message mailer.Message) error {
	var relay, err = mailer.Pick(nil)
	if err != nil {
		mailer.Record(mailer.Relay{}, message, err)
		return err
	}
	return mailer.Send(relay, message)
}

type SendCodeOut struct {
	Found     bool   `json:"found" jsonschema:"an account has the caller's number in its profile"`
	Status    string `json:"status" jsonschema:"sent: ask the caller for the code in their email; no_account: book with holder_name instead; several_accounts or email_unverified: the account cannot be told apart or reached, book with holder_name or sign in on the web"`
	EmailHint string `json:"email_hint,omitempty" jsonschema:"where the code went, e.g. m***@gmail.com, for the caller to check that inbox"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

func (a *agent) phoneSendCode(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, SendCodeOut, error) {
	var phone = a.callerPhone(req)
	if phone == "" {
		return nil, SendCodeOut{}, errNotPhone
	}
	if sessionId(req) == "" {
		return nil, SendCodeOut{}, errors.New("this MCP client keeps no session (Mcp-Session-Id): signing in needs one")
	}
	var digits = phone[1:] // E.164: "+" and digits
	var rows, err = model.New(a.db).ListUsersByPhoneEnd(ctx, digits[max(0, len(digits)-9):])
	if err != nil {
		return nil, SendCodeOut{}, errors.New("the accounts could not be looked up; try again in a moment")
	}
	var found []model.ListUsersByPhoneEndRow
	for _, row := range rows {
		if theirs, ok := validate.E164(row.Phone); ok && theirs == phone {
			found = append(found, row)
		}
	}
	switch {
	case len(found) == 0:
		return nil, SendCodeOut{Status: "no_account"}, nil
	case len(found) > 1:
		return nil, SendCodeOut{Found: true, Status: "several_accounts"}, nil
	case !found[0].EmailVerified:
		return nil, SendCodeOut{Found: true, Status: "email_unverified"}, nil
	}
	var account = found[0]
	var code string
	if code, err = newCode(); err != nil {
		return nil, SendCodeOut{}, err
	}
	var now = time.Now()
	a.codes.mu.Lock()
	var pending, allowed = a.codes.allow(phone, now)
	if allowed {
		pending.caller = caller{id: account.ID, subject: account.IdpSubject, name: account.FullName, email: account.Email}
		pending.code, pending.expires, pending.attempts = code, now.Add(codeTTL), 0
	}
	a.codes.mu.Unlock()
	if !allowed {
		return nil, SendCodeOut{}, errors.New("a code was sent to this number moments ago, or too many in the last hour: ask the caller to check their email, or wait")
	}
	var message = cronJobs.PhoneCodeEmail(branding.ForEmail(ctx, a.db), account.Email, account.FullName, account.PreferredLanguage, code, int(codeTTL/time.Minute))
	if err = a.send(ctx, message); err != nil {
		shared.Logger.Error("mcp: phone code email: %v", err)
		return nil, SendCodeOut{}, errors.New("the email with the code could not be sent; the caller can book with their name instead (holder_name)")
	}
	return nil, SendCodeOut{Found: true, Status: "sent", EmailHint: emailHint(account.Email), ExpiresAt: now.Add(codeTTL).Format(time.RFC3339)}, nil
}

type VerifyCodeIn struct {
	Code string `json:"code" jsonschema:"the 6 digits the caller reads out from the email"`
}

func (a *agent) phoneVerifyCode(ctx context.Context, req *mcp.CallToolRequest, in VerifyCodeIn) (*mcp.CallToolResult, SignInOut, error) {
	var phone = a.callerPhone(req)
	if phone == "" {
		return nil, SignInOut{}, errNotPhone
	}
	var session = sessionId(req)
	if session == "" {
		return nil, SignInOut{}, errors.New("this MCP client keeps no session (Mcp-Session-Id): signing in needs one")
	}
	var code = strings.Join(strings.Fields(in.Code), "")
	a.codes.mu.Lock()
	var pending = a.codes.by[phone]
	if pending == nil || pending.code == "" || time.Now().After(pending.expires) {
		a.codes.mu.Unlock()
		return nil, SignInOut{}, errors.New("no code is waiting for this number, or it expired: call phone_send_code for a new one")
	}
	if subtle.ConstantTimeCompare([]byte(code), []byte(pending.code)) != 1 {
		pending.attempts++
		var left = codeAttempts - pending.attempts
		if left <= 0 {
			pending.code = ""
		}
		a.codes.mu.Unlock()
		if left <= 0 {
			return nil, SignInOut{}, errors.New("wrong code too many times: it no longer works, call phone_send_code for a new one")
		}
		return nil, SignInOut{}, fmt.Errorf("wrong code: %d tries left", left)
	}
	var signedIn = pending.caller
	pending.code = ""
	a.codes.mu.Unlock()

	var g = a.grants.of(session)
	g.mu.Lock()
	g.phone, g.caller = phone, &signedIn
	g.mu.Unlock()
	return nil, SignInOut{Status: "signed_in", Name: signedIn.name, Email: signedIn.email}, nil
}

// bookByPhone: a booking for a phone caller without an account, under their name
// and number; they hear the details, as there is no email to send them to.
func (a *agent) bookByPhone(ctx context.Context, phone, name string, body bookings.CreateBookingDto) (*mcp.CallToolResult, Booking, error) {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" || utf8.RuneCountInString(name) > 80 {
		return nil, Booking{}, errors.New("not signed in: ask the caller's name and book with holder_name (up to 80 characters), or sign them in with phone_send_code")
	}
	if body.Public {
		return nil, Booking{}, errors.New("a public booking needs an account (others join it from theirs): book it private, or sign the caller in with phone_send_code")
	}
	var service = bookings.GetBookingsService(a.db)
	var made, err = service.PhoneBooking(ctx, bookings.Booker{Name: name, Phone: phone}, body)
	if err != nil {
		return nil, Booking{}, failure(err)
	}
	var out = booking(*made)
	if theirs, listErr := service.PhoneBookings(ctx, phone); listErr == nil {
		for _, b := range theirs {
			if b.Id == made.Id {
				out = booking(b.BookingDto)
			}
		}
	}
	out.HolderName = name
	out.Confirmation = "read_out"
	if SMSConfirmation != nil {
		if smsErr := SMSConfirmation(ctx, phone, out); smsErr == nil {
			out.Confirmation = "sms"
		} else {
			shared.Logger.Error("mcp: booking %s confirmation by SMS: %v", out.BookingId, smsErr)
		}
	}
	return nil, out, nil
}
