package agent

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"librequenda/auth"
	"librequenda/core/bookings"
	"librequenda/core/invites"
	"librequenda/core/me"
	"librequenda/core/tenants"
	"librequenda/mailer"
	"librequenda/model"
	"librequenda/schedule"
	"librequenda/shared"
	"librequenda/validate"

	"github.com/angelbarreiros/Penguin/router/helpers"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// agent: the MCP server's tools, on the same services the web and the app use, so
// they keep every rule (booking window, limits, residents only, age, overlaps…).
type agent struct {
	db       *sql.DB
	grants   *grants
	codes    *phoneCodes
	voiceKey string                                      // VOICE_API_KEY: "" without a voice line
	verify   func(string) (*jwt.RegisteredClaims, error) // an access token's claims (auth.VerifyAccessToken)
	send     func(context.Context, mailer.Message) error // the phone sign-in code (sendCodeEmail)
}

// failure: a refusal from a service, in English (the agent says it in the person's language).
func failure(e *shared.ErrorResponse) error {
	return errors.New(helpers.GetI18nInstance().TranslateWithLanguageWithVars(e.Message, "en", e.Args...))
}

// --- what anyone can look up -------------------------------------------------------

type SearchIn struct {
	Date      string   `json:"date,omitempty" jsonschema:"the day, YYYY-MM-DD (the councils' local time, Europe/Madrid); today when empty"`
	Query     string   `json:"query,omitempty" jsonschema:"words to find in the facility's name, its council, its sport or its tags, e.g. 'padel lalin'"`
	Council   string   `json:"council,omitempty" jsonschema:"only this council, by its slug (from list_councils)"`
	Sport     string   `json:"sport,omitempty" jsonschema:"only facilities with this sport tag slug, e.g. 'padel', 'tenis', 'natacion'"`
	From      string   `json:"from,omitempty" jsonschema:"only turns starting at or after this time, HH:MM"`
	Until     string   `json:"until,omitempty" jsonschema:"only turns starting before this time, HH:MM"`
	Latitude  *float64 `json:"latitude,omitempty" jsonschema:"the person's position, to sort by distance (with longitude)"`
	Longitude *float64 `json:"longitude,omitempty"`
	RadiusKm  *float64 `json:"radius_km,omitempty" jsonschema:"only facilities within this distance of the position"`
	MinRating *float64 `json:"min_rating,omitempty" jsonschema:"only facilities residents rated at least this on average, 1 to 5 (one nobody rated is left out)"`
}

type Turn struct {
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
}

type Game struct {
	BookingId   string    `json:"booking_id" jsonschema:"for join_public_booking"`
	Kind        string    `json:"kind" jsonschema:"public (someone's booking open to others) or event (the council's)"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	SpotsLeft   int       `json:"spots_left"`
}

type FacilityFound struct {
	FacilityId    string   `json:"facility_id" jsonschema:"for book_turn"`
	Council       string   `json:"council" jsonschema:"the council's slug, for facility_details"`
	CouncilName   string   `json:"council_name"`
	Facility      string   `json:"facility" jsonschema:"the facility's slug, for facility_details"`
	Name          string   `json:"name"`
	Sport         string   `json:"sport,omitempty"`
	Address       string   `json:"address,omitempty"`
	DistanceKm    *float64 `json:"distance_km,omitempty"`
	Rating        *float64 `json:"rating,omitempty" jsonschema:"residents' average stars, 1 to 5"`
	RatingCount   int64    `json:"rating_count,omitempty"`
	ResidentsOnly bool     `json:"residents_only" jsonschema:"only the council's verified residents book or join"`
	MinAge        *int32   `json:"min_age,omitempty"`
	FreeTurns     []Turn   `json:"free_turns"`
	PublicGames   []Game   `json:"public_games" jsonschema:"public bookings and events with spots left, to join"`
}

type SearchOut struct {
	Date       string          `json:"date"`
	Facilities []FacilityFound `json:"facilities"`
}

func freeTurns(slots []schedule.Slot) []Turn {
	var out = []Turn{}
	for _, slot := range slots {
		if slot.Status == schedule.SlotFree {
			out = append(out, Turn{slot.Start, slot.End})
		}
	}
	return out
}

func games(turns []tenants.PublicTurnDto, now time.Time) []Game {
	var out = []Game{}
	for _, t := range turns {
		var left = int(t.Capacity) - t.PeopleCount
		if t.StartsAt.After(now) && left > 0 {
			out = append(out, Game{BookingId: t.Id, Kind: t.Kind, Title: t.Title, Description: t.Description, StartsAt: t.StartsAt, EndsAt: t.EndsAt, SpotsLeft: left})
		}
	}
	return out
}

func (a *agent) search(ctx context.Context, req *mcp.CallToolRequest, in SearchIn) (*mcp.CallToolResult, SearchOut, error) {
	var values = url.Values{}
	for key, value := range map[string]string{"q": in.Query, "council": in.Council, "sport": in.Sport, "from": in.From, "until": in.Until} {
		if value != "" {
			values.Set(key, value)
		}
	}
	if in.MinRating != nil {
		values.Set("min_rating", strconv.FormatFloat(*in.MinRating, 'f', -1, 64))
	}
	if in.Latitude != nil && in.Longitude != nil {
		values.Set("lat", strconv.FormatFloat(*in.Latitude, 'f', -1, 64))
		values.Set("lon", strconv.FormatFloat(*in.Longitude, 'f', -1, 64))
		if in.RadiusKm != nil {
			values.Set("radius_km", strconv.FormatFloat(*in.RadiusKm, 'f', -1, 64))
		}
	}
	var filters, ok = tenants.FiltersFromValues(values, true)
	var near, nearOk = tenants.NearFromValues(values)
	if !ok || !nearOk {
		return nil, SearchOut{}, errors.New("a filter is not valid: dates are YYYY-MM-DD, times HH:MM, slugs lower case with dashes, and a position needs both latitude and longitude")
	}
	// signed in here: a resident of one council is not offered the others' residents-only facilities
	if subject := a.subject(ctx, req); subject != "" {
		filters.ForResident(ctx, a.db, subject)
	}
	var day, err = tenants.GetTenantsService(a.db).GetFacilitiesDay(ctx, in.Date, near, filters, helpers.PaginationParams{Page: 1, PageSize: 50})
	if err != nil {
		return nil, SearchOut{}, failure(err)
	}
	var now = time.Now()
	var out = SearchOut{Date: day.Date, Facilities: []FacilityFound{}}
	for _, f := range day.Facilities {
		var found = FacilityFound{
			FacilityId: f.Id, Council: f.Council.Slug, CouncilName: f.Council.Name, Facility: f.Slug, Name: f.Name, Sport: f.Sport,
			Address: f.Address, DistanceKm: f.DistanceKm, ResidentsOnly: f.ResidentsOnly, MinAge: f.MinAge,
			FreeTurns: freeTurns(f.Slots), PublicGames: games(f.PublicBookings, now),
		}
		if f.Rating != nil {
			found.Rating, found.RatingCount = &f.Rating.Average, f.Rating.Count
		}
		out.Facilities = append(out.Facilities, found)
	}
	return nil, out, nil
}

type CouncilsOut struct {
	Councils []tenants.TenantDto `json:"councils"`
}

func (a *agent) councils(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, CouncilsOut, error) {
	var list, err = tenants.GetTenantsService(a.db).GetTenants(ctx)
	if err != nil {
		return nil, CouncilsOut{}, failure(err)
	}
	return nil, CouncilsOut{Councils: list.Tenants}, nil
}

type DetailsIn struct {
	Council  string `json:"council" jsonschema:"the council's slug"`
	Facility string `json:"facility" jsonschema:"the facility's slug"`
	Date     string `json:"date,omitempty" jsonschema:"the day, YYYY-MM-DD; today when empty"`
}

type DetailsOut struct {
	FacilityId     string   `json:"facility_id" jsonschema:"for book_turn"`
	Name           string   `json:"name"`
	CouncilName    string   `json:"council_name"`
	Description    string   `json:"description,omitempty"`
	Address        string   `json:"address,omitempty"`
	Date           string   `json:"date"`
	FreeTurns      []Turn   `json:"free_turns"`
	PublicGames    []Game   `json:"public_games"`
	UsageRules     string   `json:"usage_rules" jsonschema:"the facility's rules: show them to the person and have them accept before book_turn"`
	SlotMinutes    int32    `json:"slot_minutes"`
	MinCancelHours int32    `json:"min_cancel_hours" jsonschema:"free cancellation until this many hours before the start"`
	Capacity       int32    `json:"capacity" jsonschema:"people per booking, owner included; 0 without a limit"`
	MinAge         *int32   `json:"min_age,omitempty"`
	ResidentsOnly  bool     `json:"residents_only"`
	Accessibility  string   `json:"accessibility,omitempty"`
	Rating         *float64 `json:"rating,omitempty" jsonschema:"residents' average stars, 1 to 5"`
}

func (a *agent) details(ctx context.Context, _ *mcp.CallToolRequest, in DetailsIn) (*mcp.CallToolResult, DetailsOut, error) {
	var page, err = tenants.GetTenantsService(a.db).GetFacilityPage(ctx, in.Council, in.Facility, in.Date)
	if err != nil {
		return nil, DetailsOut{}, failure(err)
	}
	var f = page.Facility
	var out = DetailsOut{
		FacilityId: f.Id, Name: f.Name, CouncilName: page.Tenant.Name, Description: f.Description, Address: f.Address, Date: page.Date,
		FreeTurns: freeTurns(f.Slots), PublicGames: games(f.PublicBookings, time.Now()), UsageRules: f.UsageRules,
		SlotMinutes: f.SlotMinutes, MinCancelHours: f.MinCancelHours, Capacity: f.Capacity, MinAge: f.MinAge, ResidentsOnly: f.ResidentsOnly,
	}
	if page.Rating != nil && page.Rating.Count > 0 {
		var average = page.Rating.Average
		out.Rating = &average
	}
	return nil, out, nil
}

type EventsOut struct {
	Events []tenants.UpcomingEventDto `json:"events"`
}

func (a *agent) events(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, EventsOut, error) {
	var list, err = tenants.GetTenantsService(a.db).UpcomingEvents(ctx, nil, nil)
	if err != nil {
		return nil, EventsOut{}, failure(err)
	}
	return nil, EventsOut{Events: list.Events}, nil
}

// --- signing in --------------------------------------------------------------------

type SignInOut struct {
	Status    string `json:"status" jsonschema:"signed_in; or approve: show the person url and code, wait for them to approve, then call sign_in again"`
	URL       string `json:"url,omitempty" jsonschema:"where the person approves, on the installation's own sign-in page (any device)"`
	Code      string `json:"code,omitempty" jsonschema:"the code that page shows, for the person to check it matches"`
	ExpiresAt string `json:"expires_at,omitempty"`
	Name      string `json:"name,omitempty"`
	Email     string `json:"email,omitempty"`
}

func sessionId(req *mcp.CallToolRequest) string {
	if req == nil || req.Session == nil {
		return ""
	}
	return req.Session.ID()
}

func (a *agent) signIn(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, SignInOut, error) {
	if bearer(req) != "" {
		var p, err = a.person(ctx, req, false)
		if err != nil {
			return nil, SignInOut{}, err
		}
		var profile, profileErr = model.New(a.db).GetUserProfile(ctx, p.id)
		if profileErr != nil {
			return nil, SignInOut{}, errors.New("your account could not be read; try again in a moment")
		}
		return nil, SignInOut{Status: "signed_in", Name: profile.FullName.String, Email: profile.Email}, nil
	}
	var session = sessionId(req)
	if session == "" {
		return nil, SignInOut{}, errors.New("this MCP client keeps no session (Mcp-Session-Id): signing in needs one")
	}
	if phone := a.callerPhone(req); phone != "" {
		var signedIn = a.grants.caller(session, phone)
		if signedIn == nil {
			return nil, SignInOut{}, errPhoneNotSignedIn
		}
		return nil, SignInOut{Status: "signed_in", Name: signedIn.name, Email: signedIn.email}, nil
	}
	var g = a.grants.of(session)
	g.mu.Lock()
	defer g.mu.Unlock()
	if token, err := g.token(ctx); err == nil {
		var account, accountErr = a.account(ctx, token)
		if accountErr != nil {
			return nil, SignInOut{}, accountErr
		}
		return nil, SignInOut{Status: "signed_in", Name: account.FullName, Email: account.Email}, nil
	}
	if !g.pending() {
		if err := g.start(ctx); err != nil {
			return nil, SignInOut{}, err
		}
	}
	return nil, SignInOut{Status: "approve", URL: g.verifyURL, Code: g.userCode, ExpiresAt: g.deviceExpires.Format(time.RFC3339)}, nil
}

func (a *agent) signOut(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct{}, error) {
	if session := sessionId(req); session != "" {
		var g = a.grants.of(session)
		g.mu.Lock()
		g.signOut(ctx)
		g.mu.Unlock()
		a.grants.forget(session)
	}
	return nil, struct{}{}, nil
}

// account: who the token belongs to, synced into the API as when they open the web.
func (a *agent) account(ctx context.Context, token string) (*me.MeResponse, error) {
	var claims, err = a.verify(token)
	if err != nil {
		return nil, errNotSignedIn
	}
	var account, syncErr = me.GetMeService(a.db).SyncCurrentUser(ctx, auth.Session{Subject: claims.Subject, AccessToken: token})
	if syncErr != nil {
		return nil, failure(syncErr)
	}
	return account, nil
}

// header: one of the HTTP request's headers, "" without (or without HTTP, as in the tests).
func header(req *mcp.CallToolRequest, name string) string {
	if req == nil || req.Extra == nil || req.Extra.Header == nil {
		return ""
	}
	return strings.TrimSpace(req.Extra.Header.Get(name))
}

// bearer: the person's own access token sent with the request (Authorization: Bearer).
func bearer(req *mcp.CallToolRequest) string {
	var token, _ = strings.CutPrefix(header(req, "Authorization"), "Bearer ")
	return strings.TrimSpace(token)
}

var errBadToken = errors.New("the access token sent with this request (Authorization: Bearer) is not valid or has expired: send a fresh one, or none to sign in with sign_in")

// person: who a request acts for.
type person struct {
	id      uuid.UUID // uuid.Nil: nobody
	subject string    // Zitadel's id
	byPhone bool      // a phone caller who signed in with an emailed code
}

// person: whoever this request acts for: the access token sent with it, else the
// phone caller's account they proved with a code (phone_verify_code), else this
// MCP session's sign-in (sign_in). poll: a sign-in under way may be asked whether
// it was approved. Nobody is no error; an access token that does not hold is.
func (a *agent) person(ctx context.Context, req *mcp.CallToolRequest, poll bool) (person, error) {
	if token := bearer(req); token != "" {
		var claims, err = a.verify(token)
		if err != nil {
			return person{}, errBadToken
		}
		return a.personOf(ctx, claims.Subject, token)
	}
	var session = sessionId(req)
	if session == "" {
		return person{}, nil
	}
	if phone := a.callerPhone(req); phone != "" {
		if signedIn := a.grants.caller(session, phone); signedIn != nil {
			return person{id: signedIn.id, subject: signedIn.subject, byPhone: true}, nil
		}
	}
	var g = a.grants.of(session)
	g.mu.Lock()
	defer g.mu.Unlock()
	if !poll && g.access == "" && g.refresh == "" {
		return person{}, nil
	}
	var token, err = g.token(ctx)
	if err != nil {
		return person{}, nil
	}
	var claims, claimsErr = a.verify(token)
	if claimsErr != nil {
		return person{}, nil
	}
	return a.personOf(ctx, claims.Subject, token)
}

// personOf: the account of a Zitadel user, synced into the API as when they open the web.
func (a *agent) personOf(ctx context.Context, subject string, token string) (person, error) {
	var id, err = me.CurrentUserId(ctx, a.db, auth.Session{Subject: subject, AccessToken: token})
	if err != nil {
		return person{}, failure(err)
	}
	return person{id: id, subject: subject}, nil
}

// signedIn: whoever this request acts for, or why there is nobody.
func (a *agent) signedIn(ctx context.Context, req *mcp.CallToolRequest) (person, error) {
	var p, err = a.person(ctx, req, true)
	switch {
	case err != nil:
		return person{}, err
	case p.id != uuid.Nil:
		return p, nil
	case a.callerPhone(req) != "":
		return person{}, errPhoneNotSignedIn
	case a.webGuest(req):
		return person{}, errWebGuest
	}
	return person{}, errNotSignedIn
}

// user: the signed-in person's id, or why there is nobody.
func (a *agent) user(ctx context.Context, req *mcp.CallToolRequest) (uuid.UUID, error) {
	var p, err = a.signedIn(ctx, req)
	return p.id, err
}

// subject: Zitadel's id of whoever is signed in, "" when nobody (never asks to sign in).
func (a *agent) subject(ctx context.Context, req *mcp.CallToolRequest) string {
	var p, _ = a.person(ctx, req, false)
	return p.subject
}

// --- the person's bookings -----------------------------------------------------------

type Booking struct {
	BookingId   string    `json:"booking_id"`
	Facility    string    `json:"facility"`
	Council     string    `json:"council"`
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	Kind        string    `json:"kind" jsonschema:"private, public or event"`
	Title       string    `json:"title,omitempty"`
	Role        string    `json:"role" jsonschema:"owner, or participant (joined someone else's)"`
	CanCancel   bool      `json:"can_cancel"`
	CancelUntil time.Time `json:"cancel_until"`
	People      int       `json:"people" jsonschema:"everyone in it"`
	HolderName  string    `json:"holder_name,omitempty" jsonschema:"booked on the phone without an account: whose name it is under"`
	// book_turn only
	Confirmation string `json:"confirmation,omitempty" jsonschema:"email: the person gets it by email; read_out: nobody to email (a phone caller without an account), read its details out to them; sms: texted to the caller"`
}

func booking(b bookings.BookingDto) Booking {
	var people = b.ParticipantCount
	if b.Kind != "event" {
		people++ // the owner
	}
	return Booking{
		BookingId: b.Id, Facility: b.FacilityName, Council: b.TenantName, StartsAt: b.StartsAt, EndsAt: b.EndsAt,
		Kind: b.Kind, Title: b.Title, Role: b.Role, CanCancel: b.CanCancel, CancelUntil: b.CancelUntil, People: people,
	}
}

type MyBookingsOut struct {
	Upcoming []Booking `json:"upcoming" jsonschema:"soonest first"`
}

func (a *agent) myBookings(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, MyBookingsOut, error) {
	var userId, phone, err = a.booker(ctx, req)
	if err != nil {
		return nil, MyBookingsOut{}, err
	}
	if userId == uuid.Nil {
		var theirs, listErr = bookings.GetBookingsService(a.db).PhoneBookings(ctx, phone)
		if listErr != nil {
			return nil, MyBookingsOut{}, failure(listErr)
		}
		var out = MyBookingsOut{Upcoming: []Booking{}}
		for _, b := range theirs {
			var item = booking(b.BookingDto)
			item.HolderName = b.HolderName
			out.Upcoming = append(out.Upcoming, item)
		}
		return nil, out, nil
	}
	var list, listErr = bookings.GetBookingsService(a.db).GetMyBookings(ctx, userId, helpers.PaginationParams{Page: 1, PageSize: 1})
	if listErr != nil {
		return nil, MyBookingsOut{}, failure(listErr)
	}
	var out = MyBookingsOut{Upcoming: []Booking{}}
	for _, b := range list.Upcoming {
		out.Upcoming = append(out.Upcoming, booking(b))
	}
	return nil, out, nil
}

type BookIn struct {
	FacilityId  string `json:"facility_id" jsonschema:"from search_facilities or facility_details"`
	StartsAt    string `json:"starts_at" jsonschema:"the turn's start exactly as listed in free_turns (RFC 3339)"`
	AcceptRules bool   `json:"accept_rules" jsonschema:"true only after showing the person the facility's usage_rules and them accepting"`
	Public      bool   `json:"public,omitempty" jsonschema:"open it to others, who can join until it is full"`
	Description string `json:"description,omitempty" jsonschema:"for a public booking: what they will do, for those who join (up to 1000 characters)"`
	HolderName  string `json:"holder_name,omitempty" jsonschema:"on the phone line without signing in: the caller's name, to book for them and their number (up to 80 characters)"`
	HolderPhone string `json:"holder_phone,omitempty" jsonschema:"on a web call without signing in: the phone number the person gives (9 digits in Spain, or with its country code)"`
}

func (a *agent) book(ctx context.Context, req *mcp.CallToolRequest, in BookIn) (*mcp.CallToolResult, Booking, error) {
	if !in.AcceptRules {
		return nil, Booking{}, errors.New("show the person the facility's usage_rules (facility_details) and book with accept_rules true only once they accept them")
	}
	var facility, facilityErr = uuid.Parse(in.FacilityId)
	var start, startErr = time.Parse(time.RFC3339, in.StartsAt)
	if facilityErr != nil || startErr != nil {
		return nil, Booking{}, errors.New("facility_id must be the facility's id and starts_at the turn's start as listed (RFC 3339)")
	}
	var body = bookings.CreateBookingDto{FacilityId: facility, StartsAt: start, AcceptRules: true, Public: in.Public, PublicDescription: in.Description}
	if problems := body.Validate(); len(problems) > 0 {
		return nil, Booking{}, errors.New(strings.Join(problems, "; "))
	}
	var p, err = a.person(ctx, req, true)
	if err != nil {
		return nil, Booking{}, err
	}
	var phone = a.callerPhone(req)
	if p.id == uuid.Nil && phone != "" {
		return a.bookByPhone(ctx, phone, in.HolderName, body)
	}
	if p.id == uuid.Nil && a.webGuest(req) {
		var given, ok = validate.E164(in.HolderPhone)
		if !ok {
			return nil, Booking{}, errors.New("not signed in, on a web call: ask the person for a phone number (9 digits in Spain, or with its country code) and book with holder_phone and holder_name")
		}
		return a.bookByPhone(ctx, given, in.HolderName, body)
	}
	if p.id == uuid.Nil {
		return nil, Booking{}, errNotSignedIn
	}
	var service = bookings.GetBookingsService(a.db)
	var made *bookings.BookingDto
	var bookErr *shared.ErrorResponse
	if phone != "" {
		made, bookErr = service.PhoneBooking(ctx, bookings.Account(p.id), body)
	} else {
		made, bookErr = service.CreateBooking(ctx, p.id, body)
	}
	if bookErr != nil {
		return nil, Booking{}, failure(bookErr)
	}
	var out = booking(*made)
	// as the web shows it: with its council and how many are in it
	var list, _ = service.GetMyBookings(ctx, p.id, helpers.PaginationParams{Page: 1, PageSize: 1})
	if list != nil {
		for _, b := range list.Upcoming {
			if b.Id == made.Id {
				out = booking(b)
			}
		}
	}
	out.Confirmation = "email"
	return nil, out, nil
}

// booker: whose bookings a request works on: the signed-in person's (userId), or on
// the phone line without signing in, the caller's number's (phone).
func (a *agent) booker(ctx context.Context, req *mcp.CallToolRequest) (uuid.UUID, string, error) {
	var p, err = a.person(ctx, req, true)
	if err != nil {
		return uuid.Nil, "", err
	}
	if p.id != uuid.Nil {
		return p.id, "", nil
	}
	if phone := a.callerPhone(req); phone != "" {
		return uuid.Nil, phone, nil
	}
	if a.webGuest(req) {
		return uuid.Nil, "", errWebGuest
	}
	return uuid.Nil, "", errNotSignedIn
}

type BookingIn struct {
	BookingId string `json:"booking_id" jsonschema:"from my_bookings, or a public game's booking_id"`
}

type Done struct {
	Done bool `json:"done"`
}

func (a *agent) cancel(ctx context.Context, req *mcp.CallToolRequest, in BookingIn) (*mcp.CallToolResult, Done, error) {
	var id, parseErr = uuid.Parse(in.BookingId)
	if parseErr != nil {
		return nil, Done{}, errors.New("booking_id is not a booking's id")
	}
	var userId, phone, err = a.booker(ctx, req)
	if err != nil {
		return nil, Done{}, err
	}
	var cancelErr *shared.ErrorResponse
	if userId == uuid.Nil {
		cancelErr = bookings.GetBookingsService(a.db).CancelPhoneBooking(ctx, phone, id)
	} else {
		cancelErr = bookings.GetBookingsService(a.db).CancelBooking(ctx, userId, id)
	}
	if cancelErr != nil {
		return nil, Done{}, failure(cancelErr)
	}
	return nil, Done{Done: true}, nil
}

func (a *agent) join(ctx context.Context, req *mcp.CallToolRequest, in BookingIn) (*mcp.CallToolResult, Done, error) {
	var id, parseErr = uuid.Parse(in.BookingId)
	if parseErr != nil {
		return nil, Done{}, errors.New("booking_id is not a booking's id")
	}
	var userId, err = a.user(ctx, req)
	if err != nil {
		return nil, Done{}, err
	}
	if joinErr := invites.GetInvitesService(a.db).JoinPublic(ctx, userId, id); joinErr != nil {
		return nil, Done{}, failure(joinErr)
	}
	return nil, Done{Done: true}, nil
}
