package counciladmin

import (
	"time"

	"github.com/google/uuid"
)

// The period the indicators cover: local dates in the council's time zone, both included.
type IndicatorsQuery struct {
	From     string // "2026-10-01"; empty = the last Days days up to To
	To       string // empty = today
	Days     int    // used without From; 0 = 30
	Facility string // empty = every facility of the council
	Compare  string // "previous" (default): the same number of days just before; "year": the same dates a year earlier; "none"
}

const localDate = "2006-01-02"
const maxIndicatorDays = 366

type indicatorsRange struct {
	from       time.Time // midnight of the first day
	to         time.Time // midnight of the last day
	start, end time.Time // [start, end): what the queries filter on
	bucket     string    // 'day', 'week' or 'month'
	facility   uuid.NullUUID
	compare    string // "previous", "year" or "none"
}

// comparison: the period the figures are set against, with the same bucket so
// the charts line up; false when no comparison was asked for.
func (r indicatorsRange) comparison() (indicatorsRange, bool) {
	var c = r
	switch r.compare {
	case "previous":
		c.to = r.from.AddDate(0, 0, -1)
		c.from = c.to.AddDate(0, 0, 1-dayCount(r.from, r.to))
	case "year":
		c.from = r.from.AddDate(-1, 0, 0)
		c.to = r.to.AddDate(-1, 0, 0)
	default:
		return c, false
	}
	c.start = c.from
	c.end = c.to.AddDate(0, 0, 1)
	return c, true
}

// parse answers the period, or the i18n key of the problem.
func (q IndicatorsQuery) parse(loc *time.Location, now time.Time) (indicatorsRange, string) {
	var r indicatorsRange
	var y, m, d = now.In(loc).Date()
	r.to = time.Date(y, m, d, 0, 0, 0, 0, loc)
	var err error
	if q.To != "" {
		if r.to, err = time.ParseInLocation(localDate, q.To, loc); err != nil {
			return r, "errors.invalid_indicators_range"
		}
	}
	if q.Days == 0 {
		q.Days = 30
	}
	if q.Days < 1 || q.Days > maxIndicatorDays {
		return r, "errors.invalid_indicators_range"
	}
	r.from = r.to.AddDate(0, 0, 1-q.Days)
	if q.From != "" {
		if r.from, err = time.ParseInLocation(localDate, q.From, loc); err != nil {
			return r, "errors.invalid_indicators_range"
		}
	}
	var days = dayCount(r.from, r.to)
	if days < 1 || days > maxIndicatorDays {
		return r, "errors.invalid_indicators_range"
	}
	switch q.Compare {
	case "":
		r.compare = "previous"
	case "previous", "year", "none":
		r.compare = q.Compare
	default:
		return r, "errors.invalid_query_params"
	}
	if q.Facility != "" {
		var id, idErr = uuid.Parse(q.Facility)
		if idErr != nil {
			return r, "errors.invalid_query_params"
		}
		r.facility = uuid.NullUUID{UUID: id, Valid: true}
	}
	r.start = r.from
	r.end = r.to.AddDate(0, 0, 1)
	// enough bars to read, never hundreds
	switch {
	case days <= 45:
		r.bucket = "day"
	case days <= 190:
		r.bucket = "week"
	default:
		r.bucket = "month"
	}
	return r, ""
}

// dayCount: calendar days from `from` to `to`, both included (DST days are still one day).
func dayCount(from time.Time, to time.Time) int {
	var a = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	var b = time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(b.Sub(a).Hours()/24) + 1
}

// periodStart: the first day of the bucket `day` falls in, as date_trunc does (weeks start on Monday).
func periodStart(day time.Time, bucket string) time.Time {
	var y, m, d = day.Date()
	switch bucket {
	case "week":
		var back = (int(day.Weekday()) + 6) % 7
		return time.Date(y, m, d-back, 0, 0, 0, 0, time.UTC)
	case "month":
		return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
	}
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// The figures shared by a facility and the whole council.
type IndicatorCountsDto struct {
	Bookings           int64    `json:"bookings"` // confirmed + cancelled
	Confirmed          int64    `json:"confirmed"`
	CancelledByUser    int64    `json:"cancelled_by_user"`
	CancelledByCouncil int64    `json:"cancelled_by_council"`
	CancellationRate   *float64 `json:"cancellation_rate"` // 0-1; null without bookings
	BookedHours        float64  `json:"booked_hours"`
	OpenHours          float64  `json:"open_hours"` // bookable time, closures taken out
	Occupancy          *float64 `json:"occupancy"`  // booked ÷ open, 0-1; null when never open
	Finished           int64    `json:"finished"`   // past confirmed bookings, at facilities with a QR reader
	Attended           int64    `json:"attended"`   // of those, checked in at the door
	NoShowRate         *float64 `json:"no_show_rate"`
	// the residents' stars for the facility after the period's bookings
	Ratings       int64    `json:"ratings"`
	RatingAverage *float64 `json:"rating_average"` // 1-5; null without ratings
	RatingStars   [5]int64 `json:"rating_stars"`   // how many gave 1, 2, 3, 4 and 5 stars
}

type FacilityIndicatorsDto struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	IsActive     bool      `json:"is_active"`
	TakesCheckin bool      `json:"takes_checkin"` // a reader, the code, the door QR or staff
	IndicatorCountsDto
}

type IndicatorTotalsDto struct {
	IndicatorCountsDto
	ActiveUsers     int64    `json:"active_users"`      // people with a confirmed booking, as owner or invited
	AvgPartySize    *float64 `json:"avg_party_size"`    // people per confirmed booking (owner + invited); null without bookings
	MedianLeadHours *float64 `json:"median_lead_hours"` // how long before playing people book; null without bookings
}

// Occupancy by local weekday, to see which days a facility fills.
type WeekdayOccupancyDto struct {
	Weekday     int32    `json:"weekday"` // ISO: 1 = Monday
	BookedHours float64  `json:"booked_hours"`
	OpenHours   float64  `json:"open_hours"`
	Occupancy   *float64 `json:"occupancy"` // null when closed that weekday
}

// The figures of one period: the one asked for, or the one it is compared with.
type PeriodFiguresDto struct {
	From       string                  `json:"from"`
	To         string                  `json:"to"`
	Totals     IndicatorTotalsDto      `json:"totals"`
	Facilities []FacilityIndicatorsDto `json:"facilities"`
	Series     []PeriodBookingsDto     `json:"series"`
	Weekdays   []WeekdayOccupancyDto   `json:"weekdays"`
}

type PeriodBookingsDto struct {
	Period        string   `json:"period"` // first day of the day, week or month
	Confirmed     int64    `json:"confirmed"`
	Cancelled     int64    `json:"cancelled"`
	Ratings       int64    `json:"ratings"`        // the facilities' stars after the bookings that began in it
	RatingAverage *float64 `json:"rating_average"` // 1-5; null without ratings
}

type BusyHourDto struct {
	Weekday  int32 `json:"weekday"` // ISO: 1 = Monday
	Hour     int32 `json:"hour"`
	Bookings int64 `json:"bookings"`
}

type IndicatorsDto struct {
	PeriodFiguresDto
	Bucket      string            `json:"bucket"`
	Timezone    string            `json:"timezone"`
	GeneratedAt time.Time         `json:"generated_at"`
	BusyHours   []BusyHourDto     `json:"busy_hours"`
	Compare     string            `json:"compare"`  // "previous", "year" or "none"
	Previous    *PeriodFiguresDto `json:"previous"` // the comparison period; null with compare "none"
}

// add sums another facility into the council totals; rates are worked out afterwards.
func (c *IndicatorCountsDto) add(o IndicatorCountsDto) {
	c.Bookings += o.Bookings
	c.Confirmed += o.Confirmed
	c.CancelledByUser += o.CancelledByUser
	c.CancelledByCouncil += o.CancelledByCouncil
	c.BookedHours += o.BookedHours
	c.OpenHours += o.OpenHours
	c.Finished += o.Finished
	c.Attended += o.Attended
	c.Ratings += o.Ratings
	for i := range c.RatingStars {
		c.RatingStars[i] += o.RatingStars[i]
	}
}

// rates fills the ratios from the counts.
func (c *IndicatorCountsDto) rates() {
	c.CancellationRate = ratio(float64(c.CancelledByUser+c.CancelledByCouncil), float64(c.Bookings))
	c.NoShowRate = ratio(float64(c.Finished-c.Attended), float64(c.Finished))
	c.Occupancy = ratio(c.BookedHours, c.OpenHours)
	c.RatingAverage = ratio(float64(starsSum(c.RatingStars)), float64(c.Ratings))
	if c.Occupancy != nil && *c.Occupancy > 1 {
		// bookings made before the schedule was cut down
		var full = 1.0
		c.Occupancy = &full
	}
}

func ratio(part float64, whole float64) *float64 {
	if whole <= 0 {
		return nil
	}
	var r = part / whole
	return &r
}

// starsSum: the stars given altogether, from how many gave each.
func starsSum(byStars [5]int64) int64 {
	var sum int64
	for i, n := range byStars {
		sum += int64(i+1) * n
	}
	return sum
}
