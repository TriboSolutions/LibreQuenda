package counciladmin

import (
	"context"
	"database/sql"
	"encoding/csv"
	"io"
	"librequenda/model"
	"librequenda/schedule"
	"librequenda/shared"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// snapshot runs `read` in one read-only transaction, so every figure comes from the same moment.
func (s *counciladminService) snapshot(ctx context.Context, read func(*model.Queries) error) error {
	var tx, err = s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	return read(model.New(tx))
}

// indicatorsPeriod resolves the council's time zone and the period asked for.
func (s *counciladminService) indicatorsPeriod(ctx context.Context, tenantId uuid.UUID, query IndicatorsQuery) (*time.Location, indicatorsRange, *shared.ErrorResponse) {
	var timezone, err = model.New(s.db).GetTenantTimezone(ctx, tenantId)
	if err != nil {
		return nil, indicatorsRange{}, failed("errors.indicators_failed")
	}
	var loc, locErr = time.LoadLocation(timezone)
	if locErr != nil {
		return nil, indicatorsRange{}, failed("errors.indicators_failed")
	}
	var r, key = query.parse(loc, time.Now())
	if key != "" {
		return nil, r, &shared.ErrorResponse{Code: http.StatusBadRequest, Message: key}
	}
	return loc, r, nil
}

// Indicators: the council dashboard, computed live from the bookings, and the
// same figures for the period it is compared with.
func (s *counciladminService) Indicators(ctx context.Context, tenantId uuid.UUID, query IndicatorsQuery) (*IndicatorsDto, *shared.ErrorResponse) {
	var loc, r, problem = s.indicatorsPeriod(ctx, tenantId, query)
	if problem != nil {
		return nil, problem
	}
	var response = IndicatorsDto{Bucket: r.bucket, Timezone: loc.String(), GeneratedAt: time.Now(), Compare: r.compare}
	var found = true
	var err = s.snapshot(ctx, func(q *model.Queries) error {
		var hours, err = q.ListOpeningHoursByTenant(ctx, tenantId)
		if err != nil {
			return err
		}
		if response.PeriodFiguresDto, err = periodFigures(ctx, q, tenantId, loc, r, hours); err != nil {
			return err
		}
		if found = !r.facility.Valid || len(response.Facilities) > 0; !found {
			return nil
		}
		if previous, ok := r.comparison(); ok {
			var figures PeriodFiguresDto
			if figures, err = periodFigures(ctx, q, tenantId, loc, previous, hours); err != nil {
				return err
			}
			response.Previous = &figures
		}
		var busy []model.IndicatorsBusyHoursRow
		busy, err = q.IndicatorsBusyHours(ctx, model.IndicatorsBusyHoursParams{Timezone: loc.String(), TenantID: tenantId, FacilityID: r.facility, RangeStart: r.start, RangeEnd: r.end})
		response.BusyHours = make([]BusyHourDto, 0, len(busy))
		for _, b := range busy {
			response.BusyHours = append(response.BusyHours, BusyHourDto{Weekday: b.Weekday, Hour: b.Hour, Bookings: b.Bookings})
		}
		return err
	})
	if err != nil {
		return nil, failed("errors.indicators_failed")
	}
	if !found {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	}
	return &response, nil
}

// periodFigures: every figure of one period, inside the caller's snapshot.
func periodFigures(ctx context.Context, q *model.Queries, tenantId uuid.UUID, loc *time.Location, r indicatorsRange, hours []model.ListOpeningHoursByTenantRow) (PeriodFiguresDto, error) {
	var figures = PeriodFiguresDto{From: r.from.Format(localDate), To: r.to.Format(localDate)}
	var facilities, err = q.IndicatorsByFacility(ctx, model.IndicatorsByFacilityParams{TenantID: tenantId, FacilityID: r.facility, RangeStart: r.start, RangeEnd: r.end})
	if err != nil {
		return figures, err
	}
	var series []model.IndicatorsSeriesRow
	if series, err = q.IndicatorsSeries(ctx, model.IndicatorsSeriesParams{Bucket: r.bucket, Timezone: loc.String(), TenantID: tenantId, FacilityID: r.facility, RangeStart: r.start, RangeEnd: r.end}); err != nil {
		return figures, err
	}
	var weekdays []model.IndicatorsByWeekdayRow
	if weekdays, err = q.IndicatorsByWeekday(ctx, model.IndicatorsByWeekdayParams{Timezone: loc.String(), TenantID: tenantId, FacilityID: r.facility, RangeStart: r.start, RangeEnd: r.end}); err != nil {
		return figures, err
	}
	var habits model.IndicatorsHabitsRow
	if habits, err = q.IndicatorsHabits(ctx, model.IndicatorsHabitsParams{TenantID: tenantId, FacilityID: r.facility, RangeStart: r.start, RangeEnd: r.end}); err != nil {
		return figures, err
	}
	if figures.Totals.ActiveUsers, err = q.IndicatorsActiveUsers(ctx, model.IndicatorsActiveUsersParams{TenantID: tenantId, FacilityID: r.facility, RangeStart: r.start, RangeEnd: r.end}); err != nil {
		return figures, err
	}
	var closures []model.ListClosuresByTenantRangeRow
	if closures, err = q.ListClosuresByTenantRange(ctx, model.ListClosuresByTenantRangeParams{TenantID: tenantId, RangeStart: r.start, RangeEnd: r.end}); err != nil {
		return figures, err
	}

	var open = openMinutesByDay(r, loc, facilities, hours, closures)
	figures.Facilities = make([]FacilityIndicatorsDto, 0, len(facilities))
	for _, f := range facilities {
		var minutes int
		for _, m := range open[f.ID] {
			minutes += m
		}
		var item = mapFacilityIndicators(f, minutes)
		figures.Totals.add(item.IndicatorCountsDto)
		figures.Facilities = append(figures.Facilities, item)
	}
	figures.Totals.rates()
	if habits.Bookings > 0 {
		figures.Totals.AvgPartySize = &habits.AvgPartySize
		figures.Totals.MedianLeadHours = &habits.MedianLeadHours
	}
	figures.Series = fillSeries(r, series)
	figures.Weekdays = weekdayOccupancy(open, weekdays)
	return figures, nil
}

// weekdayOccupancy adds up the open time of every facility per weekday and sets
// the booked time against it.
func weekdayOccupancy(open map[uuid.UUID]map[string]int, booked []model.IndicatorsByWeekdayRow) []WeekdayOccupancyDto {
	var days = make([]WeekdayOccupancyDto, 7)
	for i := range days {
		days[i].Weekday = int32(i + 1)
	}
	for _, byDay := range open {
		for date, minutes := range byDay {
			var day, _ = time.Parse(localDate, date)
			days[schedule.ISOWeekday(day)-1].OpenHours += float64(minutes) / 60
		}
	}
	for _, b := range booked {
		if b.Weekday >= 1 && b.Weekday <= 7 {
			days[b.Weekday-1].BookedHours = float64(b.BookedMinutes) / 60
		}
	}
	for i := range days {
		days[i].Occupancy = ratio(days[i].BookedHours, days[i].OpenHours)
		if days[i].Occupancy != nil && *days[i].Occupancy > 1 {
			var full = 1.0
			days[i].Occupancy = &full
		}
	}
	return days
}

func mapFacilityIndicators(f model.IndicatorsByFacilityRow, openMinutes int) FacilityIndicatorsDto {
	var item = FacilityIndicatorsDto{
		ID:        f.ID,
		Name:      f.Name,
		IsActive:  f.IsActive,
		HasReader: f.HasReader,
		IndicatorCountsDto: IndicatorCountsDto{
			Bookings:           f.Bookings,
			Confirmed:          f.Confirmed,
			CancelledByUser:    f.CancelledByUser,
			CancelledByCouncil: f.CancelledByCouncil,
			BookedHours:        float64(f.BookedMinutes) / 60,
			OpenHours:          float64(openMinutes) / 60,
		},
	}
	// without a reader nobody checks in, so a missing check-in says nothing
	if f.HasReader {
		item.Finished = f.Finished
		item.Attended = f.Attended
	}
	item.rates()
	return item
}

// openMinutesByDay: per facility and local day ("2006-01-02"), the bookable time.
func openMinutesByDay(r indicatorsRange, loc *time.Location, facilities []model.IndicatorsByFacilityRow, hours []model.ListOpeningHoursByTenantRow, closures []model.ListClosuresByTenantRangeRow) map[uuid.UUID]map[string]int {
	type weekdayKey struct {
		facility uuid.UUID
		weekday  int16
	}
	var windows = map[weekdayKey][]schedule.Window{}
	for _, h := range hours {
		var key = weekdayKey{h.FacilityID, h.Weekday}
		windows[key] = append(windows[key], schedule.Window{Opens: h.OpensAt, Closes: h.ClosesAt})
	}
	var closed = map[uuid.UUID][]schedule.Interval{}
	for _, c := range closures {
		closed[c.FacilityID] = append(closed[c.FacilityID], schedule.Interval{Start: c.StartsAt, End: c.EndsAt})
	}
	var open = make(map[uuid.UUID]map[string]int, len(facilities))
	for _, f := range facilities {
		open[f.ID] = map[string]int{}
	}
	for day := r.from; !day.After(r.to); day = day.AddDate(0, 0, 1) {
		var weekday = schedule.ISOWeekday(day)
		for _, f := range facilities {
			open[f.ID][day.Format(localDate)] = schedule.OpenMinutes(day, loc, int(f.SlotMinutes), windows[weekdayKey{f.ID, weekday}], closed[f.ID])
		}
	}
	return open
}

// fillSeries answers every period of the range, the empty ones too, so the chart has no gaps.
func fillSeries(r indicatorsRange, rows []model.IndicatorsSeriesRow) []PeriodBookingsDto {
	var byPeriod = make(map[string]model.IndicatorsSeriesRow, len(rows))
	for _, row := range rows {
		byPeriod[row.Period.Format(localDate)] = row
	}
	var series []PeriodBookingsDto
	var last = periodStart(r.to, r.bucket)
	for p := periodStart(r.from, r.bucket); !p.After(last); p = nextPeriod(p, r.bucket) {
		var key = p.Format(localDate)
		var row = byPeriod[key]
		series = append(series, PeriodBookingsDto{Period: key, Confirmed: row.Confirmed, Cancelled: row.Cancelled})
	}
	return series
}

func nextPeriod(p time.Time, bucket string) time.Time {
	switch bucket {
	case "week":
		return p.AddDate(0, 0, 7)
	case "month":
		return p.AddDate(0, 1, 0)
	}
	return p.AddDate(0, 0, 1)
}

// --- CSV export ---------------------------------------------------------------
// Every export is anonymous: totals per facility, day, weekday and hour, or
// bookings-per-person band, never a row per booking or per person. Counts of 1
// or 2 come out as "<3", and figures worked out from them (hours, rates) are
// left empty, so a quiet slot does not point at whoever uses it.
// Column names are fixed English identifiers, the same in every language, so
// scripts and spreadsheets keep working. excel = what Spanish-locale Excel opens
// directly: ';' between fields and a decimal comma.

const minShownCount = 3

// few: a count small enough to single someone out (0 says nothing about anybody).
func few(n int64) bool {
	return n > 0 && n < minShownCount
}

type csvFormat struct {
	excel bool
}

func (f csvFormat) writer(w io.Writer) *csv.Writer {
	io.WriteString(w, "\xef\xbb\xbf") // byte order mark: Excel then reads UTF-8 accents correctly
	var out = csv.NewWriter(w)
	if f.excel {
		out.Comma = ';'
	}
	return out
}

func (f csvFormat) count(n int64) string {
	if few(n) {
		return "<3"
	}
	return strconv.FormatInt(n, 10)
}

func (f csvFormat) number(value float64, decimals int) string {
	var text = strconv.FormatFloat(value, 'f', decimals, 64)
	if f.excel {
		text = strings.Replace(text, ".", ",", 1)
	}
	return text
}

func (f csvFormat) rate(value *float64) string {
	if value == nil {
		return ""
	}
	return f.number(*value, 4)
}

// unless leaves a derived figure empty when it would reveal a hidden count.
func unless(hide bool, value string) string {
	if hide {
		return ""
	}
	return value
}

// writeSummaryCSV: one row per facility, the same figures as the dashboard table.
func writeSummaryCSV(w io.Writer, format csvFormat, data *IndicatorsDto) error {
	var out = format.writer(w)
	out.Write([]string{
		"facility", "from", "to", "bookings", "confirmed", "cancelled_by_user", "cancelled_by_council", "cancellation_rate",
		"booked_hours", "open_hours", "occupancy", "finished_with_reader", "attended", "no_show_rate",
	})
	for _, f := range data.Facilities {
		var cancelled = f.CancelledByUser + f.CancelledByCouncil
		out.Write([]string{
			f.Name, data.From, data.To,
			format.count(f.Bookings),
			format.count(f.Confirmed),
			format.count(f.CancelledByUser),
			format.count(f.CancelledByCouncil),
			unless(few(f.Bookings) || few(cancelled), format.rate(f.CancellationRate)),
			unless(few(f.Confirmed), format.number(f.BookedHours, 2)),
			format.number(f.OpenHours, 2),
			unless(few(f.Confirmed), format.rate(f.Occupancy)),
			format.count(f.Finished),
			format.count(f.Attended),
			unless(few(f.Finished) || few(f.Finished-f.Attended), format.rate(f.NoShowRate)),
		})
	}
	out.Flush()
	return out.Error()
}

// writeDailyCSV: one row per facility and day of the period, the empty days too.
func writeDailyCSV(w io.Writer, format csvFormat, r indicatorsRange, facilities []model.IndicatorsByFacilityRow, open map[uuid.UUID]map[string]int, rows []model.IndicatorsDailyRow) error {
	type dayKey struct {
		facility uuid.UUID
		day      string
	}
	var byDay = make(map[dayKey]model.IndicatorsDailyRow, len(rows))
	for _, row := range rows {
		byDay[dayKey{row.FacilityID, row.Day.Format(localDate)}] = row
	}
	var out = format.writer(w)
	out.Write([]string{
		"facility", "date", "bookings", "confirmed", "cancelled_by_user", "cancelled_by_council",
		"booked_hours", "open_hours", "occupancy", "checked_in", "invited_people",
	})
	for _, f := range facilities {
		for day := r.from; !day.After(r.to); day = day.AddDate(0, 0, 1) {
			var date = day.Format(localDate)
			var row = byDay[dayKey{f.ID, date}]
			var openHours = float64(open[f.ID][date]) / 60
			var bookedHours = float64(row.BookedMinutes) / 60
			var checkedIn string // no reader, no check-ins to count
			if f.HasReader {
				checkedIn = format.count(row.CheckedIn)
			}
			out.Write([]string{
				f.Name, date,
				format.count(row.Bookings),
				format.count(row.Confirmed),
				format.count(row.CancelledByUser),
				format.count(row.CancelledByCouncil),
				unless(few(row.Confirmed), format.number(bookedHours, 2)),
				format.number(openHours, 2),
				unless(few(row.Confirmed), format.rate(ratio(bookedHours, openHours))),
				checkedIn,
				format.count(row.Invited),
			})
		}
	}
	out.Flush()
	return out.Error()
}

// writeBusyHoursCSV: confirmed bookings per facility, weekday (ISO: 1 = Monday)
// and starting hour, added up over the whole period.
func writeBusyHoursCSV(w io.Writer, format csvFormat, rows []model.IndicatorsBusyHoursByFacilityRow) error {
	var out = format.writer(w)
	out.Write([]string{"facility", "weekday", "hour", "bookings"})
	for _, row := range rows {
		out.Write([]string{row.FacilityName, strconv.Itoa(int(row.Weekday)), strconv.Itoa(int(row.Hour)), format.count(row.Bookings)})
	}
	out.Flush()
	return out.Error()
}

// A band of bookings per person: max 0 = no upper limit.
type usageBand struct {
	min, max int32
	people   int64
}

// usageBands groups "n people had k bookings" into 1, 2-4, 5-9 and 10 or more.
func usageBands(rows []model.IndicatorsBookingsPerPersonRow) []usageBand {
	var bands = []usageBand{{min: 1, max: 1}, {min: 2, max: 4}, {min: 5, max: 9}, {min: 10}}
	for _, row := range rows {
		for i := range bands {
			if row.Bookings >= bands[i].min && (bands[i].max == 0 || row.Bookings <= bands[i].max) {
				bands[i].people += row.People
				break
			}
		}
	}
	return bands
}

// writeUsageCSV: how many people booked how often (as owner or invited).
func writeUsageCSV(w io.Writer, format csvFormat, r indicatorsRange, bands []usageBand) error {
	var out = format.writer(w)
	out.Write([]string{"from", "to", "min_bookings", "max_bookings", "people"})
	for _, b := range bands {
		var max string // open-ended
		if b.max > 0 {
			max = strconv.Itoa(int(b.max))
		}
		out.Write([]string{r.from.Format(localDate), r.to.Format(localDate), strconv.Itoa(int(b.min)), max, format.count(b.people)})
	}
	out.Flush()
	return out.Error()
}

// ExportIndicators writes the anonymous CSV of `kind`: "summary", "daily",
// "hours" or "usage". Problems come back before anything is written, so the
// handler can still answer JSON.
func (s *counciladminService) ExportIndicators(ctx context.Context, tenantId uuid.UUID, query IndicatorsQuery, kind string, format csvFormat) (func(io.Writer) error, *shared.ErrorResponse) {
	if kind == "summary" {
		query.Compare = "none" // the file holds the period asked for only
		var data, problem = s.Indicators(ctx, tenantId, query)
		if problem != nil {
			return nil, problem
		}
		return func(w io.Writer) error { return writeSummaryCSV(w, format, data) }, nil
	}
	if kind != "daily" && kind != "hours" && kind != "usage" {
		return nil, &shared.ErrorResponse{Code: http.StatusBadRequest, Message: "errors.invalid_query_params"}
	}
	var loc, r, problem = s.indicatorsPeriod(ctx, tenantId, query)
	if problem != nil {
		return nil, problem
	}
	var write func(io.Writer) error
	var err = s.snapshot(ctx, func(q *model.Queries) error {
		var facilities, err = q.IndicatorsByFacility(ctx, model.IndicatorsByFacilityParams{TenantID: tenantId, FacilityID: r.facility, RangeStart: r.start, RangeEnd: r.end})
		if err != nil || (r.facility.Valid && len(facilities) == 0) {
			return err // no facility: answered below
		}
		switch kind {
		case "daily":
			var hours, hoursErr = q.ListOpeningHoursByTenant(ctx, tenantId)
			if hoursErr != nil {
				return hoursErr
			}
			var closures, closuresErr = q.ListClosuresByTenantRange(ctx, model.ListClosuresByTenantRangeParams{TenantID: tenantId, RangeStart: r.start, RangeEnd: r.end})
			if closuresErr != nil {
				return closuresErr
			}
			var rows, rowsErr = q.IndicatorsDaily(ctx, model.IndicatorsDailyParams{Timezone: loc.String(), TenantID: tenantId, FacilityID: r.facility, RangeStart: r.start, RangeEnd: r.end})
			var open = openMinutesByDay(r, loc, facilities, hours, closures)
			write = func(w io.Writer) error { return writeDailyCSV(w, format, r, facilities, open, rows) }
			return rowsErr
		case "hours":
			var rows, rowsErr = q.IndicatorsBusyHoursByFacility(ctx, model.IndicatorsBusyHoursByFacilityParams{Timezone: loc.String(), TenantID: tenantId, FacilityID: r.facility, RangeStart: r.start, RangeEnd: r.end})
			write = func(w io.Writer) error { return writeBusyHoursCSV(w, format, rows) }
			return rowsErr
		default:
			var rows, rowsErr = q.IndicatorsBookingsPerPerson(ctx, model.IndicatorsBookingsPerPersonParams{TenantID: tenantId, FacilityID: r.facility, RangeStart: r.start, RangeEnd: r.end})
			var bands = usageBands(rows)
			write = func(w io.Writer) error { return writeUsageCSV(w, format, r, bands) }
			return rowsErr
		}
	})
	if err != nil {
		return nil, failed("errors.indicators_failed")
	}
	if write == nil {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	}
	return write, nil
}
