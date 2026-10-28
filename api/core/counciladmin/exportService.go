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
// Left out: secrets (SMTP, device keys, QR and invite tokens) and resident
// identities. A person is their account id, the same in every table; their
// name and email are theirs to export (GET /me/export).

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
	tenant       model.Tenant
	facilities   []model.Facility
	hours        []model.OpeningHour
	closures     []model.Closure
	devices      []model.ExportCheckinDevicesRow
	bookings     []model.ExportBookingsRow
	participants []model.ExportBookingParticipantsRow
}

// ExportCouncil reads everything first, so a problem still comes back before
// anything is written.
func (s *counciladminService) ExportCouncil(ctx context.Context, tenantId uuid.UUID) (*councilExport, *shared.ErrorResponse) {
	var data = councilExport{generatedAt: time.Now().UTC()}
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
		data.participants, err = q.ExportBookingParticipants(ctx, tenantId)
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
	for _, f := range data.facilities {
		if f.ImageRef.Valid && storage.ValidRef(f.ImageRef.String) {
			refs = append(refs, f.ImageRef.String)
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
	return map[string]any{
		"profile":     "tabular-data-package",
		"name":        "librequenda-" + data.tenant.Slug,
		"title":       data.tenant.Name + " (LibreQuenda)",
		"description": "Facilities, schedules, closures, check-in readers and bookings of the council. Times are RFC 3339 in UTC; times of day are local to the council's time zone. Resident names and emails are not included: a person is their account id.",
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
			{"contact_address", "string", "Public contact address"},
			{"contact_person", "string", "Person in charge of the facilities"},
			{"logo_file", "string", "The logo inside this ZIP"},
			{"is_active", "boolean", "Open to residents"},
			{"created_at", "datetime", "Created"},
			{"updated_at", "datetime", "Last changed"},
		},
		rows: [][]string{{
			t.ID.String(), t.Slug, t.Name, t.Timezone, exportNullString(t.ContactEmail), exportNullString(t.ContactPhone),
			exportNullString(t.ContactAddress), exportNullString(t.ContactPerson),
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
			{"sport", "string", "Sport or use"},
			{"address", "string", "Postal address"},
			{"latitude", "number", "WGS 84 latitude of the map pin"},
			{"longitude", "number", "WGS 84 longitude of the map pin"},
			{"capacity", "integer", "People per booking, the owner included; empty = not set"},
			{"accessibility_info", "string", "Accessibility information"},
			{"image_file", "string", "The facility image inside this ZIP"},
			{"is_active", "boolean", "Open for booking"},
			{"slot_minutes", "integer", "Length of a booking slot"},
			{"max_advance_days", "integer", "How many days ahead residents can book"},
			{"min_cancel_hours", "integer", "Hours before the start a booking can still be cancelled"},
			{"max_active_bookings_per_user", "integer", "Upcoming bookings a person may hold at once"},
			{"min_age", "integer", "Minimum age; empty = none"},
			{"usage_rules_text", "string", "Usage rules residents accept when booking"},
			{"created_at", "datetime", "Created"},
			{"updated_at", "datetime", "Last changed"},
		},
		rows: make([][]string, 0, len(data.facilities)),
	}
	for _, f := range data.facilities {
		facilities.rows = append(facilities.rows, []string{
			f.ID.String(), f.Slug, f.Name, exportNullString(f.Description), exportNullString(f.Sport), exportNullString(f.Address),
			exportNullFloat(f.Latitude), exportNullFloat(f.Longitude), exportNullInt(f.Capacity), exportNullString(f.AccessibilityInfo),
			exportMedia(f.ImageRef, media), strconv.FormatBool(f.IsActive), strconv.Itoa(int(f.SlotMinutes)),
			strconv.Itoa(int(f.MaxAdvanceDays)), strconv.Itoa(int(f.MinCancelHours)), strconv.Itoa(int(f.MaxActiveBookingsPerUser)),
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
			{"created_at", "datetime", "Booked"},
			{"updated_at", "datetime", "Last changed"},
		},
		foreignKeys: []exportForeignKey{{"facility_id", "facilities"}, {"checked_in_device_id", "checkin_devices"}},
		rows:        make([][]string, 0, len(data.bookings)),
	}
	for _, b := range data.bookings {
		var cancelledBy string
		if b.CancelledBy.Valid {
			cancelledBy = string(b.CancelledBy.BookingCanceller)
		}
		bookings.rows = append(bookings.rows, []string{
			b.ID.String(), b.FacilityID.String(), exportNullUUID(b.UserID), exportTime(b.StartsAt), exportTime(b.EndsAt),
			string(b.Status), exportNullTime(b.RulesAcceptedAt), exportNullTime(b.CancelledAt), cancelledBy,
			exportNullTime(b.CheckedInAt), exportNullUUID(b.CheckedInDeviceID), exportTime(b.CreatedAt), exportTime(b.UpdatedAt),
		})
	}

	var participants = exportTable{
		name:        "booking_participants",
		description: "People who joined a booking through its invite link; the owner is not a row here.",
		fields: []exportField{
			{"id", "string", "Participation id (UUID)"},
			{"booking_id", "string", "Booking"},
			{"person_id", "string", "Account id; empty for a guest"},
			{"is_guest", "boolean", "Joined without an account (name and age not exported)"},
			{"joined_at", "datetime", "Joined"},
		},
		foreignKeys: []exportForeignKey{{"booking_id", "bookings"}},
		rows:        make([][]string, 0, len(data.participants)),
	}
	for _, p := range data.participants {
		participants.rows = append(participants.rows, []string{p.ID.String(), p.BookingID.String(), exportNullUUID(p.UserID), strconv.FormatBool(p.IsGuest), exportTime(p.JoinedAt)})
	}

	return []exportTable{council, facilities, hours, closures, devices, bookings, participants}
}
