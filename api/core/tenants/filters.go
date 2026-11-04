package tenants

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"librequenda/model"
	"librequenda/schedule"
)

// Filters: what the facilities search narrows by, from the query string. Every
// value is checked; a bad one is a 400, never quietly ignored.
type Filters struct {
	Words   []string // ?q=: every word somewhere in the facility's name, its council's name or slug, its sport, or any of its tags (name or slug); accents and case aside
	Council string   // ?council=: a council's slug
	Tags    []string // ?sport= and ?tags=a,b: the facility has them all
	Booking []string // ?booking=public,event: it has any of these to join (FacilityCardDto.Joinable); /facilities only
	// /facilities/day only
	From, Until string // ?from=HH:MM&until=HH:MM: turns starting in [from, until), the council's time
	Public      bool   // ?public=true: someone's public booking with spots left, instead of free turns
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var clockPattern = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

const (
	maxQueryLength = 100
	maxTags        = 20
)

// filtersFromQuery reads and checks the filters; ok false on any bad value.
func filtersFromQuery(r *http.Request, withDay bool) (Filters, bool) {
	var q = r.URL.Query()
	var f Filters
	if text := strings.TrimSpace(q.Get("q")); text != "" {
		if len([]rune(text)) > maxQueryLength {
			return f, false
		}
		f.Words = strings.Fields(fold(text))
	}
	if council := q.Get("council"); council != "" {
		if !validSlug(council) {
			return f, false
		}
		f.Council = council
	}
	if sport := q.Get("sport"); sport != "" {
		if !validSlug(sport) {
			return f, false
		}
		f.Tags = append(f.Tags, sport)
	}
	if tags := q.Get("tags"); tags != "" {
		var list = strings.Split(tags, ",")
		if len(list) > maxTags {
			return f, false
		}
		for _, tag := range list {
			if !validSlug(tag) {
				return f, false
			}
			f.Tags = append(f.Tags, tag)
		}
	}
	if booking := q.Get("booking"); booking != "" {
		if withDay {
			return f, false
		}
		for _, kind := range strings.Split(booking, ",") {
			if kind != string(model.BookingKindPublic) && kind != string(model.BookingKindEvent) {
				return f, false
			}
			f.Booking = append(f.Booking, kind)
		}
	}
	var dayOnly = q.Get("from") != "" || q.Get("until") != "" || q.Get("public") != ""
	if dayOnly && !withDay {
		return f, false
	}
	for _, clock := range []struct {
		key string
		to  *string
	}{{"from", &f.From}, {"until", &f.Until}} {
		if text := q.Get(clock.key); text != "" {
			if !clockPattern.MatchString(text) {
				return f, false
			}
			*clock.to = text
		}
	}
	if f.From != "" && f.Until != "" && f.From >= f.Until {
		return f, false
	}
	switch q.Get("public") {
	case "", "false":
	case "true":
		f.Public = true
	default:
		return f, false
	}
	return f, true
}

func validSlug(s string) bool { return len(s) <= 64 && slugPattern.MatchString(s) }

// fold: lower case without accents, so «Pádel» matches «padel».
func fold(text string) string {
	var stripped, _, _ = transform.String(transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC), text)
	return strings.ToLower(stripped)
}

// matches: the facility passes the words, council and tags (names: the tags' names by slug).
func (f Filters) matches(name, council, councilName, sport string, tags []string, names map[string]string) bool {
	if f.Council != "" && council != f.Council {
		return false
	}
	var has = map[string]bool{}
	for _, tag := range tags {
		has[tag] = true
	}
	for _, tag := range f.Tags {
		if !has[tag] {
			return false
		}
	}
	if len(f.Words) > 0 {
		// all of it: the facility, its council, its sport and every tag it has
		var parts = []string{name, councilName, council, sport}
		for _, tag := range tags {
			parts = append(parts, names[tag], tag, strings.ReplaceAll(tag, "-", " "))
		}
		var text = fold(strings.Join(parts, " "))
		for _, word := range f.Words {
			if !strings.Contains(text, word) {
				return false
			}
		}
	}
	return true
}

// joinable: the facility has one of the kinds asked for (any, without ?booking=).
func (f Filters) joinable(kinds []string) bool {
	if len(f.Booking) == 0 {
		return true
	}
	for _, want := range f.Booking {
		if slices.Contains(kinds, want) {
			return true
		}
	}
	return false
}

// inWindow: a turn starting at, in the council's time, inside [from, until).
func (f Filters) inWindow(at time.Time, loc *time.Location) bool {
	var clock = at.In(loc).Format("15:04")
	return (f.From == "" || clock >= f.From) && (f.Until == "" || clock < f.Until)
}

// narrowDay keeps, of a facility's day, only what the search asked for: the free
// turns in the window, or the public bookings with spots left in it. ok false
// when nothing is left, and the facility drops out of the list.
func (f Filters) narrowDay(day FacilityDayDto, loc *time.Location, now time.Time) (FacilityDayDto, bool) {
	var timed = f.From != "" || f.Until != ""
	if f.Public {
		var games = []PublicTurnDto{}
		for _, g := range day.PublicBookings {
			if g.StartsAt.After(now) && int(g.Capacity) > len(g.People) && f.inWindow(g.StartsAt, loc) {
				games = append(games, g)
			}
		}
		day.PublicBookings = games
		return day, len(games) > 0
	}
	if !timed {
		return day, true
	}
	var slots = []schedule.Slot{}
	for _, s := range day.Slots {
		if s.Status == schedule.SlotFree && f.inWindow(s.Start, loc) {
			slots = append(slots, s)
		}
	}
	day.Slots = slots
	return day, len(slots) > 0
}
