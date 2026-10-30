package counciladmin

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"io"
	"librequenda/model"
	"librequenda/shared"
	"librequenda/storage"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// --- Council data export --------------------------------------------------------
// Everything the council keeps in LibreQuenda, as a ZIP it can take to another
// system: one CSV per table (RFC 4180, UTF-8, comma), the facility images and
// the council logo as they were uploaded, and a datapackage.json (Frictionless
// Data Package, https://specs.frictionlessdata.io/) describing every column,
// key and link between the tables. Times are RFC 3339 in UTC; times of day are
// local to the council's time zone (council.csv).
// Left out: secrets (SMTP, device keys, QR and invite tokens). Resident
// identities only with ?personal=1 (people.csv, and the guests' names and
// ages); without it a person is just their account id, the same in every table.

type exportField struct {
	name        string
	kind        string // Frictionless type: string, integer, number, boolean, date, time, datetime
	description string
}

type exportForeignKey struct {
	field    string
	resource string // the table it points at, by its id
}

type exportTable struct {
	name        string // file name without .csv
	description string
	fields      []exportField
	foreignKeys []exportForeignKey
	rows        [][]string
}

// councilExport: the council's data, read in one snapshot.
type councilExport struct {
	generatedAt  time.Time
	personal     bool // residents' names, contact details and birth dates
	tenant       model.Tenant
	facilities   []model.ExportFacilitiesRow
	hours        []model.OpeningHour
	closures     []model.Closure
	devices      []model.ExportCheckinDevicesRow
	bookings     []model.ExportBookingsRow
	participants []model.ExportBookingParticipantsRow
	people       []model.ExportPeopleRow
}

// ExportCouncil reads everything first, so a problem still comes back before
// anything is written.
func (s *counciladminService) ExportCouncil(ctx context.Context, tenantId uuid.UUID, personal bool) (*councilExport, *shared.ErrorResponse) {
	var data = councilExport{generatedAt: time.Now().UTC(), personal: personal}
	var err = s.snapshot(ctx, func(q *model.Queries) error {
		var err error
		if data.tenant, err = q.ExportTenant(ctx, tenantId); err != nil {
			return err
		}
		if data.facilities, err = q.ExportFacilities(ctx, tenantId); err != nil {
			return err
		}
		if data.hours, err = q.ExportOpeningHours(ctx, tenantId); err != nil {
			return err
		}
		if data.closures, err = q.ExportClosures(ctx, tenantId); err != nil {
			return err
		}
		if data.devices, err = q.ExportCheckinDevices(ctx, tenantId); err != nil {
			return err
		}
		if data.bookings, err = q.ExportBookings(ctx, tenantId); err != nil {
			return err
		}
		if data.participants, err = q.ExportBookingParticipants(ctx, tenantId); err != nil || !personal {
			return err
		}
		data.people, err = q.ExportPeople(ctx, tenantId)
		return err
	})
	if err != nil {
		return nil, failed("errors.council_export_failed")
	}
	return &data, nil
}

// mediaPath: where a stored file goes inside the ZIP.
func mediaPath(ref string) string {
	return path.Join("media", path.Clean("/" + ref)[1:])
}

// WriteZip streams the export. Images that cannot be read are left out (their
// column stays empty) rather than losing the whole export over one file.
func (data *councilExport) WriteZip(ctx context.Context, w io.Writer) error {
	var archive = zip.NewWriter(w)
	var media = map[string]string{} // reference -> path inside the ZIP
	for _, ref := range data.mediaRefs() {
		if err := copyMedia(ctx, archive, ref); err != nil {
			shared.Logger.Error("council export %s: image %q left out: %s", data.tenant.Slug, ref, err.Error())
			continue
		}
		media[ref] = mediaPath(ref)
	}
	var tables = data.tables(media)
	for _, table := range tables {
		var file, err = archive.CreateHeader(&zip.FileHeader{Name: table.name + ".csv", Method: zip.Deflate, Modified: data.generatedAt})
		if err != nil {
			return err
		}
		var out = csv.NewWriter(file)
		var header = make([]string, len(table.fields))
		for i, f := range table.fields {
			header[i] = f.name
		}
		out.Write(header)
		out.WriteAll(table.rows)
		if err = out.Error(); err != nil {
			return err
		}
	}
	var descriptor, err = archive.CreateHeader(&zip.FileHeader{Name: "datapackage.json", Method: zip.Deflate, Modified: data.generatedAt})
	if err != nil {
		return err
	}
	var encoder = json.NewEncoder(descriptor)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err = encoder.Encode(data.datapackage(tables)); err != nil {
		return err
	}
	return archive.Close()
}

func (data *councilExport) mediaRefs() []string {
	var refs []string
	if data.tenant.LogoRef.Valid && storage.ValidRef(data.tenant.LogoRef.String) {
		refs = append(refs, data.tenant.LogoRef.String)
	}
	for _, row := range data.facilities {
		if ref := row.Facility.ImageRef; ref.Valid && storage.ValidRef(ref.String) {
			refs = append(refs, ref.String)
		}
	}
	return refs
}

func copyMedia(ctx context.Context, archive *zip.Writer, ref string) error {
	var object, err = storage.Get(ctx, ref)
	if err != nil {
		return err
	}
	defer object.Reader.Close()
	var file io.Writer
	// images are already compressed: stored as they are
	file, err = archive.CreateHeader(&zip.FileHeader{Name: mediaPath(ref), Method: zip.Store, Modified: object.LastModified})
	if err != nil {
		return err
	}
	_, err = io.Copy(file, object.Reader)
	return err
}

// datapackage: the descriptor, built from the same definitions as the CSVs so
// they cannot drift apart.
func (data *councilExport) datapackage(tables []exportTable) map[string]any {
	var resources = make([]map[string]any, 0, len(tables))
	for _, table := range tables {
		var fields = make([]map[string]string, 0, len(table.fields))
		for _, f := range table.fields {
			var field = map[string]string{"name": f.name, "type": f.kind, "description": f.description}
			if f.kind == "time" {
				field["format"] = "%H:%M"
			}
			fields = append(fields, field)
		}
		var schema = map[string]any{"fields": fields, "primaryKey": "id", "missingValues": []string{""}}
		if len(table.foreignKeys) > 0 {
			var keys = make([]map[string]any, 0, len(table.foreignKeys))
			for _, k := range table.foreignKeys {
				keys = append(keys, map[string]any{"fields": k.field, "reference": map[string]string{"resource": k.resource, "fields": "id"}})
			}
			schema["foreignKeys"] = keys
		}
		resources = append(resources, map[string]any{
			"name":        table.name,
			"path":        table.name + ".csv",
			"profile":     "tabular-data-resource",
			"format":      "csv",
			"mediatype":   "text/csv",
			"encoding":    "utf-8",
			"description": table.description,
			"schema":      schema,
		})
	}
	var description = "Facilities, schedules, closures, check-in readers and bookings of the council. Times are RFC 3339 in UTC; times of day are local to the council's time zone. "
	if data.personal {
		description += "Includes residents' personal data (people.csv, guests' names and ages): handle it under the GDPR."
	} else {
		description += "Resident names and contact details are not included: a person is their account id."
	}
	return map[string]any{
		"profile":     "tabular-data-package",
		"name":        "librequenda-" + data.tenant.Slug,
		"title":       data.tenant.Name + " (LibreQuenda)",
		"description": description,
		"created":     exportTime(data.generatedAt),
		"resources":   resources,
	}
}

// --- tables -------------------------------------------------------------------

func exportTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func exportNullTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return exportTime(t.Time)
}

func exportNullString(s sql.NullString) string {
	return s.String // "" when NULL
}

func exportNullInt(n sql.NullInt32) string {
	if !n.Valid {
		return ""
	}
	return strconv.Itoa(int(n.Int32))
}

func exportNullFloat(n sql.NullFloat64) string {
	if !n.Valid {
		return ""
	}
	return strconv.FormatFloat(n.Float64, 'f', -1, 64)
}

func exportNullDate(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(localDate)
}

func exportNullUUID(id uuid.NullUUID) string {
	if !id.Valid {
		return ""
	}
	return id.UUID.String()
}

// exportMedia: the image's path inside the ZIP, or "" when it has none or it was left out.
func exportMedia(ref sql.NullString, media map[string]string) string {
	if !ref.Valid {
		return ""
	}
	return media[ref.String]
}

func (data *councilExport) tables(media map[string]string) []exportTable {
	var t = data.tenant
	var council = exportTable{
		name:        "council",
		description: "The council (one row).",
		fields: []exportField{
			{"id", "string", "Council id (UUID)"},
			{"slug", "string", "Name in the web addresses: /<slug>"},
			{"name", "string", "Council name"},
			{"timezone", "string", "IANA time zone of opening hours and local dates"},
			{"contact_email", "string", "Public contact email"},
			{"contact_phone", "string", "Public contact phone"},
			{"logo_file", "string", "The logo inside this ZIP"},
			{"is_active", "boolean", "Open to residents"},
			{"created_at", "datetime", "Created"},
			{"updated_at", "datetime", "Last changed"},
		},
		rows: [][]string{{
			t.ID.String(), t.Slug, t.Name, t.Timezone, exportNullString(t.ContactEmail), exportNullString(t.ContactPhone),
			exportMedia(t.LogoRef, media), strconv.FormatBool(t.IsActive), exportTime(t.CreatedAt), exportTime(t.UpdatedAt),
		}},
	}

	var facilities = exportTable{
		name:        "facilities",
		description: "Bookable facilities (a court, a pool lane, a room…) with their usage rules.",
		fields: []exportField{
			{"id", "string", "Facility id (UUID)"},
			{"slug", "string", "Name in the web address: /<council>/<slug>"},
			{"name", "string", "Facility name"},
			{"description", "string", "Description"},
			{"sport", "string", "Its sports: the names of its tags in the first tag category"},
			{"tags", "string", "Its tags from the platform catalogue (slugs), separated by spaces"},
			{"address", "string", "Postal address"},
			{"latitude", "number", "WGS 84 latitude of the map pin"},
			{"longitude", "number", "WGS 84 longitude of the map pin"},
			{"capacity", "integer", "People per booking, the owner included; empty = not set"},
			{"accessibility_info", "string", "Accessibility information"},
			{"contact_phone", "string", "Its own phone; empty = the council's is shown"},
			{"contact_email", "string", "Its own email; empty = the council's is shown"},
			{"image_file", "string", "The facility image inside this ZIP"},
			{"is_active", "boolean", "Open for booking"},
			{"slot_minutes", "integer", "Length of a booking slot"},
			{"custom_rules", "boolean", "Whether the facility keeps its own booking window instead of the council's"},
			{"max_advance_hours", "integer", "How many hours ahead residents can book"},
			{"min_advance_minutes", "integer", "Minutes before the start the app still books a turn"},
			{"min_cancel_hours", "integer", "Hours before the start a booking can still be cancelled"},
			{"max_active_bookings_per_user", "integer", "Upcoming bookings a person may hold at once"},
			{"max_bookings_per_day", "integer", "Bookings a person may make per day here; 0 = no limit"},
			{"max_turns_in_a_row", "integer", "Back-to-back turns a person may book; 0 = no limit"},
			{"no_show_penalty", "boolean", "Whether too many no-shows block booking for a while"},
			{"no_show_limit", "integer", "No-shows that block booking"},
			{"no_show_window_days", "integer", "Days the no-shows are counted over"},
			{"no_show_block_days", "integer", "Days booking stays blocked"},
			{"min_age", "integer", "Minimum age; empty = none"},
			{"usage_rules_text", "string", "Usage rules residents accept when booking"},
			{"created_at", "datetime", "Created"},
			{"updated_at", "datetime", "Last changed"},
		},
		rows: make([][]string, 0, len(data.facilities)),
	}
	for _, row := range data.facilities {
		var f = row.Facility
		facilities.rows = append(facilities.rows, []string{
			f.ID.String(), f.Slug, f.Name, exportNullString(f.Description), row.Sport, strings.Join(row.Tags, " "), exportNullString(f.Address),
			exportNullFloat(f.Latitude), exportNullFloat(f.Longitude), exportNullInt(f.Capacity), exportNullString(f.AccessibilityInfo),
			exportNullString(f.ContactPhone), exportNullString(f.ContactEmail),
			exportMedia(f.ImageRef, media), strconv.FormatBool(f.IsActive), strconv.Itoa(int(f.SlotMinutes)),
			strconv.FormatBool(f.CustomRules), strconv.Itoa(int(f.MaxAdvanceHours)), strconv.Itoa(int(f.MinAdvanceMinutes)),
			strconv.Itoa(int(f.MinCancelHours)), strconv.Itoa(int(f.MaxActiveBookingsPerUser)), strconv.Itoa(int(f.MaxBookingsPerDay)),
			strconv.Itoa(int(f.MaxTurnsInARow)), strconv.FormatBool(f.NoShowPenalty), strconv.Itoa(int(f.NoShowLimit)),
			strconv.Itoa(int(f.NoShowWindowDays)), strconv.Itoa(int(f.NoShowBlockDays)),
			exportNullInt(f.MinAge), exportNullString(f.UsageRulesText), exportTime(f.CreatedAt), exportTime(f.UpdatedAt),
		})
	}

	var hours = exportTable{
		name:        "opening_hours",
		description: "Weekly schedule: one row per opening window; a weekday may have several.",
		fields: []exportField{
			{"id", "string", "Window id (UUID)"},
			{"facility_id", "string", "Facility"},
			{"weekday", "integer", "ISO weekday: 1 = Monday … 7 = Sunday"},
			{"opens_at", "time", "Opens, local time"},
			{"closes_at", "time", "Closes, local time"},
		},
		foreignKeys: []exportForeignKey{{"facility_id", "facilities"}},
		rows:        make([][]string, 0, len(data.hours)),
	}
	for _, h := range data.hours {
		hours.rows = append(hours.rows, []string{h.ID.String(), h.FacilityID.String(), strconv.Itoa(int(h.Weekday)), h.OpensAt.Format("15:04"), h.ClosesAt.Format("15:04")})
	}

	var closures = exportTable{
		name:        "closures",
		description: "Exceptions to the schedule: holidays, maintenance…",
		fields: []exportField{
			{"id", "string", "Closure id (UUID)"},
			{"facility_id", "string", "Facility"},
			{"starts_at", "datetime", "Closed from"},
			{"ends_at", "datetime", "Closed until"},
			{"reason", "string", "Reason given to residents"},
			{"created_at", "datetime", "Created"},
		},
		foreignKeys: []exportForeignKey{{"facility_id", "facilities"}},
		rows:        make([][]string, 0, len(data.closures)),
	}
	for _, c := range data.closures {
		closures.rows = append(closures.rows, []string{c.ID.String(), c.FacilityID.String(), exportTime(c.StartsAt), exportTime(c.EndsAt), exportNullString(c.Reason), exportTime(c.CreatedAt)})
	}

	var devices = exportTable{
		name:        "checkin_devices",
		description: "QR readers at the facility entrances. Their keys are not exported: a reader moved to another system needs a new one.",
		fields: []exportField{
			{"id", "string", "Reader id (UUID)"},
			{"facility_id", "string", "Facility"},
			{"name", "string", "Reader name"},
			{"key_hint", "string", "Last characters of its key"},
			{"last_seen_at", "datetime", "Last contact"},
			{"created_at", "datetime", "Created"},
		},
		foreignKeys: []exportForeignKey{{"facility_id", "facilities"}},
		rows:        make([][]string, 0, len(data.devices)),
	}
	for _, d := range data.devices {
		devices.rows = append(devices.rows, []string{d.ID.String(), d.FacilityID.String(), d.Name, d.KeyHint, exportNullTime(d.LastSeenAt), exportTime(d.CreatedAt)})
	}

	var bookings = exportTable{
		name:        "bookings",
		description: "Every booking, past, upcoming and cancelled.",
		fields: []exportField{
			{"id", "string", "Booking id (UUID)"},
			{"facility_id", "string", "Facility"},
			{"person_id", "string", "Account id of the person who booked; empty after they erased their account"},
			{"starts_at", "datetime", "Starts"},
			{"ends_at", "datetime", "Ends"},
			{"status", "string", "confirmed or cancelled"},
			{"rules_accepted_at", "datetime", "When the person accepted the usage rules"},
			{"cancelled_at", "datetime", "When it was cancelled"},
			{"cancelled_by", "string", "user (the person) or council (a closure or the facility deactivated)"},
			{"checked_in_at", "datetime", "When the QR was scanned at the entrance"},
			{"checked_in_device_id", "string", "Reader that scanned it"},
			{"checked_in_method", "string", "How the person got in: reader, printed_qr, entry_code or staff"},
			{"created_at", "datetime", "Booked"},
			{"updated_at", "datetime", "Last changed"},
		},
		foreignKeys: []exportForeignKey{{"facility_id", "facilities"}, {"checked_in_device_id", "checkin_devices"}},
		rows:        make([][]string, 0, len(data.bookings)),
	}
	if data.personal {
		bookings.foreignKeys = append(bookings.foreignKeys, exportForeignKey{"person_id", "people"})
	}
	for _, b := range data.bookings {
		var cancelledBy, checkedInMethod string
		if b.CancelledBy.Valid {
			cancelledBy = string(b.CancelledBy.BookingCanceller)
		}
		if b.CheckedInMethod.Valid {
			checkedInMethod = string(b.CheckedInMethod.CheckinMethod)
		}
		bookings.rows = append(bookings.rows, []string{
			b.ID.String(), b.FacilityID.String(), exportNullUUID(b.UserID), exportTime(b.StartsAt), exportTime(b.EndsAt),
			string(b.Status), exportNullTime(b.RulesAcceptedAt), exportNullTime(b.CancelledAt), cancelledBy,
			exportNullTime(b.CheckedInAt), exportNullUUID(b.CheckedInDeviceID), checkedInMethod, exportTime(b.CreatedAt), exportTime(b.UpdatedAt),
		})
	}

	var participants = exportTable{
		name:        "booking_participants",
		description: "People who joined a booking through its invite link; the owner is not a row here.",
		fields: []exportField{
			{"id", "string", "Participation id (UUID)"},
			{"booking_id", "string", "Booking"},
			{"person_id", "string", "Account id; empty for a guest"},
			{"is_guest", "boolean", "Joined without an account"},
			{"joined_at", "datetime", "Joined"},
		},
		foreignKeys: []exportForeignKey{{"booking_id", "bookings"}},
		rows:        make([][]string, 0, len(data.participants)),
	}
	if data.personal {
		participants.fields = append(participants.fields, exportField{"guest_name", "string", "Name a guest gave"}, exportField{"guest_age", "integer", "Age a guest gave"})
		participants.foreignKeys = append(participants.foreignKeys, exportForeignKey{"person_id", "people"})
	}
	for _, p := range data.participants {
		var row = []string{p.ID.String(), p.BookingID.String(), exportNullUUID(p.UserID), strconv.FormatBool(p.IsGuest), exportTime(p.JoinedAt)}
		if data.personal {
			var age string
			if p.GuestAge.Valid {
				age = strconv.Itoa(int(p.GuestAge.Int16))
			}
			row = append(row, exportNullString(p.GuestName), age)
		}
		participants.rows = append(participants.rows, row)
	}

	var tables = []exportTable{council, facilities, hours, closures, devices, bookings, participants}
	if !data.personal {
		return tables
	}
	var people = exportTable{
		name:        "people",
		description: "Residents who booked at the council or joined one of its bookings. Personal data: handle it under the GDPR.",
		fields: []exportField{
			{"id", "string", "Account id (UUID), the person_id of the other tables"},
			{"full_name", "string", "Full name"},
			{"email", "string", "Email"},
			{"email_verified", "boolean", "The email was verified"},
			{"phone", "string", "Phone"},
			{"birth_date", "date", "Birth date, for the facilities' minimum age"},
			{"preferred_language", "string", "Language of the emails: gl, es, en or ca"},
			{"created_at", "datetime", "Account created"},
		},
		rows: make([][]string, 0, len(data.people)),
	}
	for _, p := range data.people {
		people.rows = append(people.rows, []string{
			p.ID.String(), exportNullString(p.FullName), p.Email, strconv.FormatBool(p.EmailVerified), exportNullString(p.Phone),
			exportNullDate(p.BirthDate), exportNullString(p.PreferredLanguage), exportTime(p.CreatedAt),
		})
	}
	return append(tables, people)
}
