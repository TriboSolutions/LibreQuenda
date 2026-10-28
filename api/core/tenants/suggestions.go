package tenants

import (
	"cmp"
	"librequenda/model"
	"librequenda/schedule"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Turn suggestions: plain counting, no AI. How busy a turn usually is comes
// from the confirmed bookings of the last weeks at the same weekday and hour.

const (
	historyWeeks      = 8 // how far back "usually" looks
	maxQuietSlots     = 3
	maxAlternatives   = 3
	maxAlternateSlots = 3
)

type BusyLevel string

const (
	BusyQuiet    BusyLevel = "quiet"
	BusyModerate BusyLevel = "moderate"
	BusyBusy     BusyLevel = "busy"
)

type SuggestedSlotDto struct {
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
	Level    BusyLevel `json:"level"` // how busy this turn usually is at this facility
}

type AlternativeDto struct {
	Slug       string             `json:"slug"`
	Name       string             `json:"name"`
	Address    string             `json:"address"`
	DistanceKm *float64           `json:"distance_km"` // null when either facility has no map pin
	FreeSlots  int                `json:"free_slots"`  // on the same day
	Slots      []SuggestedSlotDto `json:"slots"`       // its quietest free turns
}

type SuggestionsDto struct {
	// Quiet: the facility's free turns of the day, quietest first. Empty when
	// there is no booking history to tell quiet from busy.
	Quiet []SuggestedSlotDto `json:"quiet"`
	// Alternatives: other facilities of the same sport in the council with free
	// turns that day, nearest first.
	Alternatives []AlternativeDto `json:"alternatives"`
}

// busyKey is a local weekday (ISO) and starting hour.
type busyKey struct {
	weekday int
	hour    int
}

// busyHistory is how many confirmed bookings started at each weekday and hour.
type busyHistory struct {
	counts map[busyKey]int64
	peak   int64
}

func busyHistoryByFacility(rows []model.ListBusyHoursByTenantRow) map[uuid.UUID]*busyHistory {
	var byFacility = map[uuid.UUID]*busyHistory{}
	for _, row := range rows {
		var h = byFacility[row.FacilityID]
		if h == nil {
			h = &busyHistory{counts: map[busyKey]int64{}}
			byFacility[row.FacilityID] = h
		}
		h.counts[busyKey{int(row.Weekday), int(row.Hour)}] += row.Bookings
		h.peak = max(h.peak, h.counts[busyKey{int(row.Weekday), int(row.Hour)}])
	}
	return byFacility
}

func (h *busyHistory) count(start time.Time, loc *time.Location) int64 {
	if h == nil {
		return 0
	}
	var local = start.In(loc)
	return h.counts[busyKey{int(schedule.ISOWeekday(local)), local.Hour()}]
}

// level compares a turn with the facility's busiest one: up to a third of it is
// quiet, up to two thirds moderate.
func (h *busyHistory) level(count int64) BusyLevel {
	if h == nil || h.peak == 0 || count*3 <= h.peak {
		return BusyQuiet
	}
	if count*3 <= h.peak*2 {
		return BusyModerate
	}
	return BusyBusy
}

// quietestFree returns up to n free slots, the usually quietest first and,
// between equally quiet ones, the earliest.
func quietestFree(slots []schedule.Slot, history *busyHistory, loc *time.Location, n int) []SuggestedSlotDto {
	type scored struct {
		slot  schedule.Slot
		count int64
	}
	var free []scored
	for _, s := range slots {
		if s.Status == schedule.SlotFree {
			free = append(free, scored{s, history.count(s.Start, loc)})
		}
	}
	slices.SortStableFunc(free, func(a, b scored) int {
		return cmp.Or(cmp.Compare(a.count, b.count), a.slot.Start.Compare(b.slot.Start))
	})
	var result = make([]SuggestedSlotDto, 0, min(n, len(free)))
	for _, f := range free[:min(n, len(free))] {
		result = append(result, SuggestedSlotDto{StartsAt: f.slot.Start, EndsAt: f.slot.End, Level: history.level(f.count)})
	}
	return result
}

// suggest builds the suggestions for `target` from the day's facilities of the council.
func suggest(target FacilityDayDto, facilities []FacilityDayDto, histories map[uuid.UUID]*busyHistory, loc *time.Location) SuggestionsDto {
	var result = SuggestionsDto{Quiet: []SuggestedSlotDto{}, Alternatives: []AlternativeDto{}}
	if history := histories[uuid.MustParse(target.Id)]; history != nil && history.peak > 0 {
		result.Quiet = quietestFree(target.Slots, history, loc, maxQuietSlots)
	}

	var sport = strings.TrimSpace(target.Sport)
	if sport == "" {
		return result
	}
	for _, f := range facilities {
		if f.Id == target.Id || !strings.EqualFold(strings.TrimSpace(f.Sport), sport) {
			continue
		}
		var slots = quietestFree(f.Slots, histories[uuid.MustParse(f.Id)], loc, maxAlternateSlots)
		if len(slots) == 0 {
			continue
		}
		var free int
		for _, s := range f.Slots {
			if s.Status == schedule.SlotFree {
				free++
			}
		}
		result.Alternatives = append(result.Alternatives, AlternativeDto{
			Slug:       f.Slug,
			Name:       f.Name,
			Address:    f.Address,
			DistanceKm: distanceKm(target, f),
			FreeSlots:  free,
			Slots:      slots,
		})
	}
	// nearest first; those without a map pin go last, by name
	slices.SortStableFunc(result.Alternatives, func(a, b AlternativeDto) int {
		if (a.DistanceKm == nil) != (b.DistanceKm == nil) {
			if a.DistanceKm == nil {
				return 1
			}
			return -1
		}
		if a.DistanceKm != nil && *a.DistanceKm != *b.DistanceKm {
			return cmp.Compare(*a.DistanceKm, *b.DistanceKm)
		}
		return strings.Compare(a.Name, b.Name)
	})
	result.Alternatives = result.Alternatives[:min(maxAlternatives, len(result.Alternatives))]
	return result
}

// distanceKm is the great-circle distance between two facilities, rounded to 100 m.
func distanceKm(a FacilityDayDto, b FacilityDayDto) *float64 {
	if a.Latitude == nil || a.Longitude == nil || b.Latitude == nil || b.Longitude == nil {
		return nil
	}
	const earthRadiusKm = 6371.0
	var lat1, lat2 = *a.Latitude * math.Pi / 180, *b.Latitude * math.Pi / 180
	var dLat = lat2 - lat1
	var dLon = (*b.Longitude - *a.Longitude) * math.Pi / 180
	var h = math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	var km = math.Round(2*earthRadiusKm*math.Asin(math.Sqrt(h))*10) / 10
	return &km
}
