package tenants

import (
	"context"
	"librequenda/auth"
	"librequenda/checkin"
	"librequenda/core/bookings"
	"librequenda/core/me"
	"librequenda/model"
	"librequenda/rating"
	"librequenda/schedule"
	"librequenda/shared"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
)

// The home page's "For you", for a signed-in resident: worked out on reading
// from their own bookings of the last weeks and today's free turns. Nothing is
// stored about them beyond the bookings themselves.

const (
	historyDays        = 120 // how far back their bookings count
	maxUsualToday      = 3
	maxFavourites      = 3
	maxSimilar         = 3
	maxPublic          = 3
	maxRecommendations = 8
	publicAheadDays    = 14 // public bookings shown up to this far ahead
	nearKm             = 10 // a public booking this close to the person counts as near
)

// Kinds of recommendation, most relevant first.
const (
	RecommendUsualToday = "usual_today"    // what they book on this weekday at this time, free today
	RecommendFavourite  = "favourite"      // a facility they book often, with free turns today
	RecommendSimilar    = "similar"        // a facility they have not tried, alike in tags to theirs, free today
	RecommendPublic     = "public_booking" // someone's public booking with spots left, at a facility of theirs, alike or near
)

type RecommendationDto struct {
	Kind     string          `json:"kind"`
	Facility FacilityCardDto `json:"facility"`
	// usual_today: that free turn; favourite and similar: the next free turn today
	StartsAt time.Time `json:"starts_at"`
	Times    int64     `json:"times,omitempty"`   // usual_today: bookings that weekday and time; favourite: bookings there
	Weekday  int32     `json:"weekday,omitempty"` // usual_today: ISO, 1 = Monday
	// similar: the facility of theirs it is like, and the tags they share
	Because    string   `json:"because,omitempty"`
	SharedTags []string `json:"shared_tags,omitempty"`
	// public_booking: the booking to join; StartsAt is its start
	Booking *PublicBookingDto `json:"booking,omitempty"`
}

// PublicBookingDto: what «For you» shows of someone's public booking: never who
// they are beyond a first name.
type PublicBookingDto struct {
	Kind           string    `json:"kind"`  // "public", or "event": the council's, with a title and no owner
	Title          string    `json:"title"` // events only
	Description    string    `json:"description"`
	Id             string    `json:"id"`
	StartsAt       time.Time `json:"starts_at"`
	EndsAt         time.Time `json:"ends_at"`
	SpotsLeft      int       `json:"spots_left"`
	OwnerFirstName string    `json:"owner_first_name"` // "" when unknown
	// what others gave the owner after public bookings; null until someone did
	OwnerRating *rating.Summary `json:"owner_rating"`
}

// Position: where the person is, when the browser shares it; checked, never stored.
type Position struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

type RecommendationsResponse struct {
	Recommendations []RecommendationDto `json:"recommendations"`
}

// todayFacility: an active facility with today's turns, in its council's zone.
type todayFacility struct {
	card  FacilityCardDto
	slots []schedule.Slot
	loc   *time.Location
}

func (s *tenantsService) Recommendations(ctx context.Context, session auth.Session, position Position) (*RecommendationsResponse, *shared.ErrorResponse) {
	var userId, userErr = me.CurrentUserId(ctx, s.db, session)
	if userErr != nil {
		return nil, userErr
	}
	var response = &RecommendationsResponse{Recommendations: []RecommendationDto{}}
	// turned off by the person: nothing is worked out
	if show, err := model.New(s.db).GetShowRecommendations(ctx, userId); err != nil || !show {
		return response, nil
	}
	var now = time.Now()
	var patterns, err = model.New(s.db).ListUserBookingPatterns(ctx, model.ListUserBookingPatternsParams{
		UserID: uuid.NullUUID{UUID: userId, Valid: true},
		Since:  now.AddDate(0, 0, -historyDays),
	})
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.recommendations_failed"}
	}
	var public []model.ListJoinablePublicBookingsRow
	public, err = model.New(s.db).ListJoinablePublicBookings(ctx, model.ListJoinablePublicBookingsParams{
		UserID: uuid.NullUUID{UUID: userId, Valid: true},
		Until:  now.AddDate(0, 0, publicAheadDays),
	})
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.recommendations_failed"}
	}
	if len(patterns) == 0 && len(public) == 0 {
		return response, nil
	}
	var today = s.today(ctx)
	var out = append(recommend(patterns, today, now), publicBookings(public, patterns, today, position)...)
	if len(out) > maxRecommendations {
		out = out[:maxRecommendations]
	}
	// how far each one is, from where the person said they are
	if position.Latitude != nil && position.Longitude != nil {
		var near = &Near{Latitude: *position.Latitude, Longitude: *position.Longitude}
		for i := range out {
			out[i].Facility.DistanceKm = distanceKm(near, out[i].Facility.Latitude, out[i].Facility.Longitude)
		}
	}
	response.Recommendations = out
	return response, nil
}

// publicBookings: the public bookings at a facility the person books, one alike
// in tags to theirs, or one near them; soonest first, one card per booking.
func publicBookings(rows []model.ListJoinablePublicBookingsRow, patterns []model.ListUserBookingPatternsRow, today []todayFacility, position Position) []RecommendationDto {
	var byId = make(map[string]todayFacility, len(today))
	for _, f := range today {
		byId[f.card.Id] = f
	}
	var theirs = make(map[string]bool)
	for _, p := range patterns {
		theirs[p.FacilityID.String()] = true
	}
	var relevant = func(f todayFacility) bool {
		if theirs[f.card.Id] {
			return true
		}
		for id := range theirs {
			if mine, ok := byId[id]; ok && len(sharedTags(f.card.Tags, mine.card.Tags)) > 0 {
				return true
			}
		}
		if position.Latitude != nil && position.Longitude != nil && f.card.Latitude != nil && f.card.Longitude != nil {
			return checkin.DistanceMetres(*position.Latitude, *position.Longitude, *f.card.Latitude, *f.card.Longitude) <= nearKm*1000
		}
		return false
	}
	var out = []RecommendationDto{}
	for _, row := range rows { // soonest first
		var f, ok = byId[row.FacilityID.String()]
		var spots = bookings.Places(row.Capacity, row.Kind) - int(row.ParticipantCount)
		if !ok || spots <= 0 || !relevant(f) {
			continue
		}
		out = append(out, RecommendationDto{
			Kind:     RecommendPublic,
			Facility: f.card,
			StartsAt: row.StartsAt,
			Booking:  &PublicBookingDto{Kind: string(row.Kind), Title: row.Title, Id: row.ID.String(), StartsAt: row.StartsAt, EndsAt: row.EndsAt, SpotsLeft: spots, OwnerFirstName: row.OwnerFirstName, OwnerRating: rating.Of(row.OwnerRatingAverage, row.OwnerRatingCount), Description: row.PublicDescription},
		})
		if len(out) == maxPublic {
			break
		}
	}
	return out
}

// today: every active facility with its turns today, as the home page's list
// has them; a council that fails to load is left out.
func (s *tenantsService) today(ctx context.Context) []todayFacility {
	var tenants, err = model.New(s.db).ListTenants(ctx)
	if err != nil {
		return nil
	}
	var facilities []todayFacility
	for _, tenant := range tenants {
		var day, _, loc, dayErr = s.loadDay(ctx, tenant.Slug, "")
		if dayErr != nil {
			continue
		}
		for _, f := range day.Facilities {
			facilities = append(facilities, todayFacility{
				card: FacilityCardDto{
					Id: f.Id, Slug: f.Slug, Name: f.Name, Sport: f.Sport, Tags: f.Tags, ImageUrl: f.ImageUrl,
					Address: f.Address, Latitude: f.Latitude, Longitude: f.Longitude, FreeToday: countFree(f.Slots), Council: day.Tenant, ResidentsOnly: f.ResidentsOnly, Joinable: []string{},
				},
				slots: f.Slots,
				loc:   loc,
			})
		}
	}
	return facilities
}

func countFree(slots []schedule.Slot) int {
	var free = 0
	for _, slot := range slots {
		if slot.Status == schedule.SlotFree {
			free++
		}
	}
	return free
}

// nextFree: the first free turn still to start; false when none is left today.
func nextFree(slots []schedule.Slot, now time.Time) (schedule.Slot, bool) {
	for _, slot := range slots {
		if slot.Status == schedule.SlotFree && slot.Start.After(now) {
			return slot, true
		}
	}
	return schedule.Slot{}, false
}

// recommend: first their usual turns that fall today and are free, then the
// facilities they book most with free turns today, then facilities they have
// not tried that share tags with theirs. One card per facility.
func recommend(patterns []model.ListUserBookingPatternsRow, today []todayFacility, now time.Time) []RecommendationDto {
	var byId = make(map[string]todayFacility, len(today))
	for _, f := range today {
		byId[f.card.Id] = f
	}
	var totals = make(map[string]int64)
	for _, p := range patterns {
		totals[p.FacilityID.String()] += p.Bookings
	}
	var shown = make(map[string]bool)
	var out []RecommendationDto

	// 1. their usual turn today: same weekday and starting time, still free
	var usual = append([]model.ListUserBookingPatternsRow{}, patterns...)
	sort.SliceStable(usual, func(i, j int) bool { return usual[i].Bookings > usual[j].Bookings })
	var count = 0
	for _, p := range usual {
		var f, ok = byId[p.FacilityID.String()]
		if !ok || shown[f.card.Id] || count == maxUsualToday || int32(isoWeekday(now.In(f.loc))) != p.Weekday {
			continue
		}
		for _, slot := range f.slots {
			if slot.Status == schedule.SlotFree && slot.Start.After(now) && slot.Start.In(f.loc).Format("15:04") == p.StartTime {
				out = append(out, RecommendationDto{Kind: RecommendUsualToday, Facility: f.card, StartsAt: slot.Start, Times: p.Bookings, Weekday: p.Weekday})
				shown[f.card.Id] = true
				count++
				break
			}
		}
	}

	// 2. the facilities they book most, with a free turn left today
	var mine = make([]string, 0, len(totals))
	for id := range totals {
		mine = append(mine, id)
	}
	sort.Slice(mine, func(i, j int) bool {
		if totals[mine[i]] != totals[mine[j]] {
			return totals[mine[i]] > totals[mine[j]]
		}
		return mine[i] < mine[j]
	})
	count = 0
	for _, id := range mine {
		var f, ok = byId[id]
		if !ok || shown[id] || count == maxFavourites {
			continue
		}
		if slot, free := nextFree(f.slots, now); free {
			out = append(out, RecommendationDto{Kind: RecommendFavourite, Facility: f.card, StartsAt: slot.Start, Times: totals[id]})
			shown[id] = true
			count++
		}
	}

	// 3. alike to theirs (shared tags) and not tried yet, with a free turn today
	type candidate struct {
		f       todayFacility
		shared  []string
		because string
		slot    schedule.Slot
	}
	var candidates []candidate
	for _, f := range today {
		if _, tried := totals[f.card.Id]; tried {
			continue
		}
		var slot, free = nextFree(f.slots, now)
		if !free {
			continue
		}
		var best candidate
		for _, id := range mine {
			var theirs, ok = byId[id]
			if !ok {
				continue
			}
			if shared := sharedTags(f.card.Tags, theirs.card.Tags); len(shared) > len(best.shared) {
				best = candidate{f: f, shared: shared, because: theirs.card.Name, slot: slot}
			}
		}
		if len(best.shared) > 0 {
			candidates = append(candidates, best)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool { return len(candidates[i].shared) > len(candidates[j].shared) })
	for i, c := range candidates {
		if i == maxSimilar {
			break
		}
		out = append(out, RecommendationDto{Kind: RecommendSimilar, Facility: c.f.card, StartsAt: c.slot.Start, Because: c.because, SharedTags: c.shared})
	}

	if out == nil {
		out = []RecommendationDto{}
	}
	return out
}

func isoWeekday(t time.Time) int {
	if t.Weekday() == time.Sunday {
		return 7
	}
	return int(t.Weekday())
}

func sharedTags(a, b []string) []string {
	var in = make(map[string]bool, len(b))
	for _, tag := range b {
		in[tag] = true
	}
	var shared []string
	for _, tag := range a {
		if in[tag] {
			shared = append(shared, tag)
		}
	}
	return shared
}
