package counciladmin

import (
	"bytes"
	"librequenda/model"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIndicatorsQueryParse(t *testing.T) {
	var madrid, _ = time.LoadLocation("Europe/Madrid")
	var now = time.Date(2026, 10, 28, 23, 30, 0, 0, madrid) // 22:30 UTC, still the 28th in Madrid
	var cases = []struct {
		name   string
		in     IndicatorsQuery
		want   string
		from   string
		bucket string
	}{
		{"default: the last 30 days", IndicatorsQuery{}, "", "2026-09-29", "day"},
		{"a quarter", IndicatorsQuery{From: "2026-07-01", To: "2026-09-30"}, "", "2026-07-01", "week"},
		{"a year", IndicatorsQuery{From: "2025-10-29", To: "2026-10-28"}, "", "2025-10-29", "month"},
		{"one day", IndicatorsQuery{From: "2026-10-28", To: "2026-10-28"}, "", "2026-10-28", "day"},
		{"the last 90 days", IndicatorsQuery{Days: 90}, "", "2026-07-31", "week"},
		{"the last 7 days", IndicatorsQuery{Days: 7}, "", "2026-10-22", "day"},
		{"too many days", IndicatorsQuery{Days: 400}, "errors.invalid_indicators_range", "", ""},
		{"days that are not a number", IndicatorsQuery{Days: -1}, "errors.invalid_indicators_range", "", ""},
		{"end before start", IndicatorsQuery{From: "2026-10-28", To: "2026-10-01"}, "errors.invalid_indicators_range", "", ""},
		{"longer than a year", IndicatorsQuery{From: "2025-01-01", To: "2026-10-28"}, "errors.invalid_indicators_range", "", ""},
		{"bad date", IndicatorsQuery{From: "28/10/2026"}, "errors.invalid_indicators_range", "", ""},
		{"bad facility", IndicatorsQuery{Facility: "pavillon"}, "errors.invalid_query_params", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var r, got = c.in.parse(madrid, now)
			if got != c.want {
				t.Fatalf("parse() = %q, want %q", got, c.want)
			}
			if got != "" {
				return
			}
			if r.from.Format(localDate) != c.from || r.bucket != c.bucket {
				t.Errorf("from %s bucket %s, want %s %s", r.from.Format(localDate), r.bucket, c.from, c.bucket)
			}
			if !r.end.Equal(r.to.AddDate(0, 0, 1)) || r.start.Location() != madrid {
				t.Errorf("the range must run from local midnight to the midnight after the last day: %v - %v", r.start, r.end)
			}
		})
	}
}

func TestDayCountAcrossDST(t *testing.T) {
	var madrid, _ = time.LoadLocation("Europe/Madrid")
	// the clocks go back on 25 October 2026: that day has 25 hours
	var from = time.Date(2026, 10, 24, 0, 0, 0, 0, madrid)
	var to = time.Date(2026, 10, 26, 0, 0, 0, 0, madrid)
	if got := dayCount(from, to); got != 3 {
		t.Errorf("dayCount() = %d, want 3", got)
	}
}

func TestFillSeries(t *testing.T) {
	var madrid, _ = time.LoadLocation("Europe/Madrid")
	var day = func(s string) time.Time { var d, _ = time.Parse(localDate, s); return d }
	// Thursday 1 to Wednesday 14 October, by week: the weeks of 28 Sep, 5 Oct and 12 Oct
	var r, _ = IndicatorsQuery{From: "2026-10-01", To: "2026-10-14"}.parse(madrid, time.Now())
	r.bucket = "week"
	var rows = []model.IndicatorsSeriesRow{{Period: day("2026-10-05"), Confirmed: 7, Cancelled: 1}}
	// 3 ratings, 13 stars, in the last week only
	var rated = []model.IndicatorsRatingSeriesRow{{Period: day("2026-10-12"), Ratings: 3, RatingStars: 13}}
	var series = fillSeries(r, rows, rated)
	type point struct {
		period               string
		confirmed, cancelled int64
		ratings              int64
		average              float64 // 0: none (null)
	}
	var want = []point{{"2026-09-28", 0, 0, 0, 0}, {"2026-10-05", 7, 1, 0, 0}, {"2026-10-12", 0, 0, 3, 13.0 / 3}}
	if len(series) != len(want) {
		t.Fatalf("got %+v, want %+v", series, want)
	}
	for i, w := range want {
		var got = series[i]
		var average float64
		if got.RatingAverage != nil {
			average = *got.RatingAverage
		}
		if got.Period != w.period || got.Confirmed != w.confirmed || got.Cancelled != w.cancelled || got.Ratings != w.ratings || average != w.average {
			t.Errorf("period %d: got %+v (average %v), want %+v", i, got, average, w)
		}
		if w.ratings == 0 && got.RatingAverage != nil {
			t.Errorf("period %d: nobody rated, the average must be null", i)
		}
	}
}

func TestFacilityIndicatorRates(t *testing.T) {
	var row = model.IndicatorsByFacilityRow{
		ID: uuid.New(), Name: "Pavillón", SlotMinutes: 60, TakesCheckin: true,
		Bookings: 10, Confirmed: 8, CancelledByUser: 1, CancelledByCouncil: 1,
		BookedMinutes: 8 * 60, Finished: 4, Attended: 3,
	}
	var item = mapFacilityIndicators(row, 40*60)
	if *item.CancellationRate != 0.2 || *item.Occupancy != 0.2 || *item.NoShowRate != 0.25 {
		t.Errorf("cancellation %v occupancy %v no-show %v, want 0.2 0.2 0.25", *item.CancellationRate, *item.Occupancy, *item.NoShowRate)
	}

	row.TakesCheckin = false
	item = mapFacilityIndicators(row, 0)
	if item.NoShowRate != nil || item.Occupancy != nil {
		t.Errorf("no way to check in or no opening hours: the rates must be null, got %v %v", item.NoShowRate, item.Occupancy)
	}

	if item.RatingAverage != nil {
		t.Errorf("nobody rated: the average must be null, got %v", *item.RatingAverage)
	}

	row.BookedMinutes = 50 * 60 // booked before the schedule was cut down
	item = mapFacilityIndicators(row, 40*60)
	if *item.Occupancy != 1 {
		t.Errorf("occupancy must not pass 100%%, got %v", *item.Occupancy)
	}
}

func TestRatingFigures(t *testing.T) {
	// two facilities: 2 ratings (5 and 4) and 3 ratings (1, 3 and 3)
	var a = mapFacilityIndicators(model.IndicatorsByFacilityRow{ID: uuid.New(), Ratings: 2, Stars4: 1, Stars5: 1}, 0)
	var b = mapFacilityIndicators(model.IndicatorsByFacilityRow{ID: uuid.New(), Ratings: 3, Stars1: 1, Stars3: 2}, 0)
	if *a.RatingAverage != 4.5 || *b.RatingAverage != 7.0/3 {
		t.Errorf("averages %v %v, want 4.5 and 2.33", *a.RatingAverage, *b.RatingAverage)
	}
	// the council's: every rating counts the same, not the facilities' averages averaged
	var totals IndicatorTotalsDto
	totals.add(a.IndicatorCountsDto)
	totals.add(b.IndicatorCountsDto)
	totals.rates()
	if totals.Ratings != 5 || *totals.RatingAverage != 16.0/5 || totals.RatingStars != [5]int64{1, 0, 2, 1, 1} {
		t.Errorf("totals %d ratings, average %v, by stars %v; want 5, 3.2, [1 0 2 1 1]", totals.Ratings, *totals.RatingAverage, totals.RatingStars)
	}
}

func TestSummaryCSV(t *testing.T) {
	var occupancy, rating = 0.125, 4.25
	var data = &IndicatorsDto{From: "2026-10-01", To: "2026-10-28", Facilities: []FacilityIndicatorsDto{{
		Name:               "Pista de pádel 1",
		IndicatorCountsDto: IndicatorCountsDto{Bookings: 3, Confirmed: 3, BookedHours: 4.5, OpenHours: 36, Occupancy: &occupancy, Ratings: 4, RatingAverage: &rating},
	}}}
	var out bytes.Buffer
	if err := writeSummaryCSV(&out, csvFormat{}, data); err != nil {
		t.Fatal(err)
	}
	var text = out.String()
	if !strings.HasPrefix(text, "\xef\xbb\xbf") {
		t.Error("missing the UTF-8 byte order mark")
	}
	if !strings.Contains(text, "Pista de pádel 1,2026-10-01,2026-10-28,3,3,0,0,,4.50,36.00,0.1250,0,0,,4,4.25\n") {
		t.Errorf("unexpected row:\n%s", text)
	}

	out.Reset()
	writeSummaryCSV(&out, csvFormat{excel: true}, data)
	if !strings.Contains(out.String(), "Pista de pádel 1;2026-10-01;2026-10-28;3;3;0;0;;4,50;36,00;0,1250;0;0;;4;4,25\n") {
		t.Errorf("Excel format must use ';' and a decimal comma:\n%s", out.String())
	}
}

func TestSummaryCSVHidesSmallCounts(t *testing.T) {
	var occupancy, cancellation, rating = 0.05, 0.0625, 5.0
	var data = &IndicatorsDto{From: "2026-10-01", To: "2026-10-28", Facilities: []FacilityIndicatorsDto{{
		Name: "Pista de tenis",
		IndicatorCountsDto: IndicatorCountsDto{
			Bookings: 32, Confirmed: 30, CancelledByUser: 2, BookedHours: 30, OpenHours: 600, Occupancy: &occupancy, CancellationRate: &cancellation,
			Finished: 2, Attended: 2, Ratings: 2, RatingAverage: &rating,
		},
	}}}
	var out bytes.Buffer
	writeSummaryCSV(&out, csvFormat{}, data)
	// 2 cancellations: "<3", and the rate that would give them away stays empty; the same with 2 ratings
	var want = "Pista de tenis,2026-10-01,2026-10-28,32,30,<3,0,,30.00,600.00,0.0500,<3,<3,,<3,\n"
	if !strings.HasSuffix(out.String(), want) {
		t.Errorf("got:\n%s\nwant the row:\n%s", out.String(), want)
	}
}

func TestDailyCSV(t *testing.T) {
	var madrid, _ = time.LoadLocation("Europe/Madrid")
	var r, _ = IndicatorsQuery{From: "2026-10-26", To: "2026-10-27"}.parse(madrid, time.Now())
	var pavilion = model.IndicatorsByFacilityRow{ID: uuid.New(), Name: "Pavillón", TakesCheckin: true}
	var pool = model.IndicatorsByFacilityRow{ID: uuid.New(), Name: "Piscina"}
	var day = func(s string) time.Time { var d, _ = time.Parse(localDate, s); return d }
	var open = map[uuid.UUID]map[string]int{
		pavilion.ID: {"2026-10-26": 600, "2026-10-27": 600},
		pool.ID:     {"2026-10-26": 300, "2026-10-27": 300},
	}
	var rows = []model.IndicatorsDailyRow{
		{FacilityID: pavilion.ID, Day: day("2026-10-26"), Bookings: 6, Confirmed: 5, CancelledByUser: 1, BookedMinutes: 300, CheckedIn: 4, Invited: 7},
		{FacilityID: pool.ID, Day: day("2026-10-27"), Bookings: 1, Confirmed: 1, BookedMinutes: 45},
	}
	var out bytes.Buffer
	writeDailyCSV(&out, csvFormat{}, r, []model.IndicatorsByFacilityRow{pavilion, pool}, open, rows)
	var lines = strings.Split(strings.TrimSpace(out.String()), "\n")
	var want = []string{
		"Pavillón,2026-10-26,6,5,<3,0,5.00,10.00,0.5000,4,7",
		"Pavillón,2026-10-27,0,0,0,0,0.00,10.00,0.0000,0,0", // empty days are listed too
		"Piscina,2026-10-26,0,0,0,0,0.00,5.00,0.0000,,0",    // no way to check in: no check-in figure
		"Piscina,2026-10-27,<3,<3,0,0,,5.00,,,0",            // one booking: nothing that gives it away
	}
	if len(lines) != len(want)+1 {
		t.Fatalf("got %d lines:\n%s", len(lines), out.String())
	}
	for i, w := range want {
		if lines[i+1] != w {
			t.Errorf("row %d:\n got  %s\n want %s", i, lines[i+1], w)
		}
	}
}

func TestUsageBands(t *testing.T) {
	var bands = usageBands([]model.IndicatorsBookingsPerPersonRow{{Bookings: 1, People: 9}, {Bookings: 3, People: 2}, {Bookings: 4, People: 2}, {Bookings: 12, People: 1}})
	var want = []usageBand{{1, 1, 9}, {2, 4, 4}, {5, 9, 0}, {10, 0, 1}}
	for i := range want {
		if bands[i] != want[i] {
			t.Errorf("band %d: got %+v, want %+v", i, bands[i], want[i])
		}
	}
	var madrid, _ = time.LoadLocation("Europe/Madrid")
	var r, _ = IndicatorsQuery{From: "2026-10-01", To: "2026-10-28"}.parse(madrid, time.Now())
	var out bytes.Buffer
	writeUsageCSV(&out, csvFormat{excel: true}, r, bands)
	if !strings.Contains(out.String(), "2026-10-01;2026-10-28;10;;<3\n") || !strings.Contains(out.String(), "2026-10-01;2026-10-28;2;4;4\n") {
		t.Errorf("unexpected usage CSV:\n%s", out.String())
	}
}

func TestBusyHoursCSV(t *testing.T) {
	var out bytes.Buffer
	writeBusyHoursCSV(&out, csvFormat{}, []model.IndicatorsBusyHoursByFacilityRow{{FacilityName: "Pista de pádel 1", Weekday: 2, Hour: 19, Bookings: 11}, {FacilityName: "Pista de pádel 1", Weekday: 7, Hour: 10, Bookings: 1}})
	if !strings.HasSuffix(out.String(), "Pista de pádel 1,2,19,11\nPista de pádel 1,7,10,<3\n") {
		t.Errorf("unexpected busy-hours CSV:\n%s", out.String())
	}
}

func TestComparisonPeriod(t *testing.T) {
	var madrid, _ = time.LoadLocation("Europe/Madrid")
	var cases = []struct {
		in       IndicatorsQuery
		from, to string
		ok       bool
	}{
		{IndicatorsQuery{From: "2026-10-01", To: "2026-10-28"}, "2026-09-03", "2026-09-30", true}, // the 28 days just before
		{IndicatorsQuery{From: "2026-10-01", To: "2026-10-28", Compare: "year"}, "2025-10-01", "2025-10-28", true},
		{IndicatorsQuery{From: "2026-03-01", To: "2026-03-31"}, "2026-01-29", "2026-02-28", true}, // 31 days back, across February
		{IndicatorsQuery{From: "2026-10-01", To: "2026-10-28", Compare: "none"}, "", "", false},
	}
	for _, c := range cases {
		var r, key = c.in.parse(madrid, time.Now())
		if key != "" {
			t.Fatalf("%+v: %s", c.in, key)
		}
		var p, ok = r.comparison()
		if ok != c.ok {
			t.Fatalf("%+v: comparison ok = %v, want %v", c.in, ok, c.ok)
		}
		if !ok {
			continue
		}
		if p.from.Format(localDate) != c.from || p.to.Format(localDate) != c.to || !p.end.Equal(p.to.AddDate(0, 0, 1)) || p.bucket != r.bucket {
			t.Errorf("%+v: got %s - %s (%s), want %s - %s", c.in, p.from.Format(localDate), p.to.Format(localDate), p.bucket, c.from, c.to)
		}
	}
	if _, key := (IndicatorsQuery{Compare: "last-month"}).parse(madrid, time.Now()); key != "errors.invalid_query_params" {
		t.Errorf("unknown comparison: got %q", key)
	}
}

func TestWeekdayOccupancy(t *testing.T) {
	var pavilion, pool = uuid.New(), uuid.New()
	// Monday 26 and Tuesday 27 October 2026
	var open = map[uuid.UUID]map[string]int{
		pavilion: {"2026-10-26": 600, "2026-10-27": 600},
		pool:     {"2026-10-26": 300, "2026-10-27": 0},
	}
	var days = weekdayOccupancy(open, []model.IndicatorsByWeekdayRow{{Weekday: 1, BookedMinutes: 450}, {Weekday: 2, BookedMinutes: 900}})
	if len(days) != 7 || days[0].OpenHours != 15 || *days[0].Occupancy != 0.5 {
		t.Errorf("Monday: got %+v", days[0])
	}
	if *days[1].Occupancy != 1 {
		t.Errorf("Tuesday: occupancy must not pass 100%%, got %v", *days[1].Occupancy)
	}
	if days[6].Occupancy != nil {
		t.Errorf("Sunday is never open: occupancy must be null, got %v", *days[6].Occupancy)
	}
}
