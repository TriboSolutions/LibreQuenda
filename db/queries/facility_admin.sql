-- ============================================================
--  COUNCIL ADMINISTRATION: facilities and their weekly schedule
-- ============================================================

-- name: IsTenantAdmin :one
SELECT EXISTS (
    SELECT 1
    FROM tenant_membership
    WHERE user_id = sqlc.arg('user_id')
      AND tenant_id = sqlc.arg('tenant_id')
      AND role = 'admin'
);

-- name: ListFacilitiesForAdmin :many
SELECT
    f.id,
    f.slug,
    f.name,
    facility_sport(f.id)::TEXT AS sport,
    ARRAY(
        SELECT tg.slug FROM facility_tag ft JOIN tag tg ON tg.id = ft.tag_id JOIN tag_category tc ON tc.id = tg.category_id
        WHERE ft.facility_id = f.id ORDER BY tc.position, tg.name
    )::TEXT[] AS tags,
    f.image_ref,
    f.is_active,
    (SELECT COUNT(*) FROM opening_hours oh WHERE oh.facility_id = f.id)::INT AS opening_window_count,
    (SELECT COUNT(*) FROM booking b WHERE b.facility_id = f.id AND b.status = 'confirmed' AND b.ends_at > NOW())::INT AS upcoming_booking_count
FROM
    facility f
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
ORDER BY
    f.is_active DESC,
    f.name ASC;

-- name: GetFacilityForAdmin :one
SELECT
    *
FROM
    facility
WHERE
    id = sqlc.arg('id')
    AND tenant_id = sqlc.arg('tenant_id');

-- name: CreateFacility :one
INSERT INTO facility (
    tenant_id, slug, name, description, address, latitude, longitude, capacity,
    accessibility_info, contact_phone, contact_email, image_ref, show_weather, is_active, slot_minutes,
    custom_rules, max_advance_hours, min_advance_minutes, min_cancel_hours, max_active_bookings_per_user,
    max_bookings_per_day, max_turns_in_a_row, max_bookings_per_week, weekly_opening, opens_weekday, opens_minute,
    no_show_penalty, no_show_limit, no_show_window_days, no_show_block_days,
    min_age, residents_only, usage_rules_text
) VALUES (
    sqlc.arg('tenant_id'), sqlc.arg('slug'), sqlc.arg('name'), sqlc.narg('description'),
    sqlc.narg('address'), sqlc.narg('latitude'), sqlc.narg('longitude'), sqlc.narg('capacity'),
    sqlc.narg('accessibility_info'), sqlc.narg('contact_phone'), sqlc.narg('contact_email'), sqlc.narg('image_ref'), sqlc.arg('show_weather'), sqlc.arg('is_active'), sqlc.arg('slot_minutes'),
    sqlc.arg('custom_rules'), sqlc.arg('max_advance_hours'), sqlc.arg('min_advance_minutes'), sqlc.arg('min_cancel_hours'),
    sqlc.arg('max_active_bookings_per_user'), sqlc.arg('max_bookings_per_day'), sqlc.arg('max_turns_in_a_row'),
    sqlc.arg('max_bookings_per_week'), sqlc.arg('weekly_opening'), sqlc.arg('opens_weekday'), sqlc.arg('opens_minute'),
    sqlc.arg('no_show_penalty'), sqlc.arg('no_show_limit'), sqlc.arg('no_show_window_days'), sqlc.arg('no_show_block_days'),
    sqlc.narg('min_age'), sqlc.arg('residents_only'), sqlc.narg('usage_rules_text')
)
RETURNING
    id;

-- name: UpdateFacility :one
UPDATE facility
SET
    slug                         = sqlc.arg('slug'),
    name                         = sqlc.arg('name'),
    description                  = sqlc.narg('description'),
    address                      = sqlc.narg('address'),
    latitude                     = sqlc.narg('latitude'),
    longitude                    = sqlc.narg('longitude'),
    capacity                     = sqlc.narg('capacity'),
    accessibility_info           = sqlc.narg('accessibility_info'),
    contact_phone                = sqlc.narg('contact_phone'),
    contact_email                = sqlc.narg('contact_email'),
    image_ref                    = sqlc.narg('image_ref'),
    show_weather                 = sqlc.arg('show_weather'),
    is_active                    = sqlc.arg('is_active'),
    slot_minutes                 = sqlc.arg('slot_minutes'),
    min_age                      = sqlc.narg('min_age'),
    residents_only               = sqlc.arg('residents_only'),
    usage_rules_text             = sqlc.narg('usage_rules_text')
WHERE
    id = sqlc.arg('id')
    AND tenant_id = sqlc.arg('tenant_id')
RETURNING
    id;

-- name: DeleteFacility :execrows
DELETE FROM facility
WHERE
    id = sqlc.arg('id')
    AND tenant_id = sqlc.arg('tenant_id');

-- name: CountBookingsByFacility :one
SELECT COUNT(*) FROM booking WHERE facility_id = sqlc.arg('facility_id');

-- name: DeleteOpeningHoursByFacility :exec
DELETE FROM opening_hours WHERE facility_id = sqlc.arg('facility_id');

-- name: InsertOpeningHours :exec
INSERT INTO opening_hours (facility_id, weekday, opens_at, closes_at)
VALUES (sqlc.arg('facility_id'), sqlc.arg('weekday'), sqlc.arg('opens_at'), sqlc.arg('closes_at'));

-- name: ListFacilityTagSlugs :many
SELECT
    t.slug
FROM
    facility_tag ft
    JOIN tag t ON t.id = ft.tag_id
    JOIN tag_category c ON c.id = t.category_id
WHERE
    ft.facility_id = sqlc.arg('facility_id')
ORDER BY
    c.position,
    t.name;

-- name: ClearFacilityTags :exec
DELETE FROM facility_tag WHERE facility_id = sqlc.arg('facility_id');

-- name: AddFacilityTags :execrows
-- The tags by slug; a slug not in the catalogue adds no row.
INSERT INTO facility_tag (facility_id, tag_id)
SELECT sqlc.arg('facility_id'), t.id FROM tag t WHERE t.slug = ANY(sqlc.arg('slugs')::TEXT[]);

-- name: EnsureTenantBookingRules :exec
-- Row-level security: runs as the council (db.WithTenant). Its booking window
-- with the defaults, the first time it is needed.
INSERT INTO tenant_booking_rules (tenant_id) VALUES (sqlc.arg('tenant_id')) ON CONFLICT (tenant_id) DO NOTHING;

-- name: GetTenantBookingRules :one
SELECT * FROM tenant_booking_rules WHERE tenant_id = sqlc.arg('tenant_id');

-- name: UpdateTenantBookingRules :exec
-- Row-level security: runs as the council (db.WithTenant).
UPDATE tenant_booking_rules
SET
    max_advance_hours            = sqlc.arg('max_advance_hours'),
    min_advance_minutes          = sqlc.arg('min_advance_minutes'),
    min_cancel_hours             = sqlc.arg('min_cancel_hours'),
    max_active_bookings_per_user = sqlc.arg('max_active_bookings_per_user'),
    max_bookings_per_day         = sqlc.arg('max_bookings_per_day'),
    max_turns_in_a_row           = sqlc.arg('max_turns_in_a_row'),
    max_bookings_per_week        = sqlc.arg('max_bookings_per_week'),
    max_active_bookings_council  = sqlc.arg('max_active_bookings_council'),
    weekly_opening               = sqlc.arg('weekly_opening'),
    opens_weekday                = sqlc.arg('opens_weekday'),
    opens_minute                 = sqlc.arg('opens_minute'),
    no_show_penalty              = sqlc.arg('no_show_penalty'),
    no_show_limit                = sqlc.arg('no_show_limit'),
    no_show_window_days          = sqlc.arg('no_show_window_days'),
    no_show_block_days           = sqlc.arg('no_show_block_days'),
    updated_at                   = NOW()
WHERE
    tenant_id = sqlc.arg('tenant_id');

-- name: ApplyTenantBookingRules :execrows
-- Row-level security: runs as the council (db.WithTenant). The council's window
-- copied to each of its facilities that does not keep its own.
UPDATE facility f
SET
    max_advance_hours            = r.max_advance_hours,
    min_advance_minutes          = r.min_advance_minutes,
    min_cancel_hours             = r.min_cancel_hours,
    max_active_bookings_per_user = r.max_active_bookings_per_user,
    max_bookings_per_day         = r.max_bookings_per_day,
    max_turns_in_a_row           = r.max_turns_in_a_row,
    max_bookings_per_week        = r.max_bookings_per_week,
    weekly_opening               = r.weekly_opening,
    opens_weekday                = r.opens_weekday,
    opens_minute                 = r.opens_minute,
    no_show_penalty              = r.no_show_penalty,
    no_show_limit                = r.no_show_limit,
    no_show_window_days          = r.no_show_window_days,
    no_show_block_days           = r.no_show_block_days,
    updated_at                   = NOW()
FROM
    tenant_booking_rules r
WHERE
    r.tenant_id = f.tenant_id
    AND f.tenant_id = sqlc.arg('tenant_id')
    AND f.custom_rules = FALSE;

-- name: UpdateFacilityBookingRules :execrows
-- Row-level security: runs as the council (db.WithTenant). The facility's
-- booking window, saved from its own tab: its own, or the council's copied in.
UPDATE facility
SET
    custom_rules                 = sqlc.arg('custom_rules'),
    max_advance_hours            = sqlc.arg('max_advance_hours'),
    min_advance_minutes          = sqlc.arg('min_advance_minutes'),
    min_cancel_hours             = sqlc.arg('min_cancel_hours'),
    max_active_bookings_per_user = sqlc.arg('max_active_bookings_per_user'),
    max_bookings_per_day         = sqlc.arg('max_bookings_per_day'),
    max_turns_in_a_row           = sqlc.arg('max_turns_in_a_row'),
    max_bookings_per_week        = sqlc.arg('max_bookings_per_week'),
    weekly_opening               = sqlc.arg('weekly_opening'),
    opens_weekday                = sqlc.arg('opens_weekday'),
    opens_minute                 = sqlc.arg('opens_minute'),
    no_show_penalty              = sqlc.arg('no_show_penalty'),
    no_show_limit                = sqlc.arg('no_show_limit'),
    no_show_window_days          = sqlc.arg('no_show_window_days'),
    no_show_block_days           = sqlc.arg('no_show_block_days'),
    updated_at                   = NOW()
WHERE
    id = sqlc.arg('id')
    AND tenant_id = sqlc.arg('tenant_id');

-- name: DuplicateFacility :one
-- A copy of one of the council's facilities under another name and address: every
-- setting, its booking window included, and switched off until it is reviewed.
INSERT INTO facility (
    tenant_id, slug, name, description, address, latitude, longitude, capacity,
    accessibility_info, contact_phone, contact_email, image_ref, show_weather, is_active, slot_minutes,
    custom_rules, max_advance_hours, min_advance_minutes, min_cancel_hours, max_active_bookings_per_user,
    max_bookings_per_day, max_turns_in_a_row, max_bookings_per_week, weekly_opening, opens_weekday, opens_minute,
    no_show_penalty, no_show_limit, no_show_window_days, no_show_block_days,
    min_age, residents_only, usage_rules_text
)
SELECT
    f.tenant_id, sqlc.arg('slug'), sqlc.arg('name'), f.description, f.address, f.latitude, f.longitude, f.capacity,
    f.accessibility_info, f.contact_phone, f.contact_email, f.image_ref, f.show_weather, FALSE, f.slot_minutes,
    f.custom_rules, f.max_advance_hours, f.min_advance_minutes, f.min_cancel_hours, f.max_active_bookings_per_user,
    f.max_bookings_per_day, f.max_turns_in_a_row, f.max_bookings_per_week, f.weekly_opening, f.opens_weekday, f.opens_minute,
    f.no_show_penalty, f.no_show_limit, f.no_show_window_days, f.no_show_block_days,
    f.min_age, f.residents_only, f.usage_rules_text
FROM
    facility f
WHERE
    f.id = sqlc.arg('source_id')
    AND f.tenant_id = sqlc.arg('tenant_id')
RETURNING
    id;

-- name: CopyFacilityTags :exec
INSERT INTO facility_tag (facility_id, tag_id)
SELECT sqlc.arg('facility_id'), t.tag_id FROM facility_tag t WHERE t.facility_id = sqlc.arg('source_id');

-- name: CopyOpeningHours :exec
INSERT INTO opening_hours (facility_id, weekday, opens_at, closes_at)
SELECT sqlc.arg('facility_id'), h.weekday, h.opens_at, h.closes_at FROM opening_hours h WHERE h.facility_id = sqlc.arg('source_id');

-- name: CopyFacilityCheckin :exec
-- The ways to check in and who to ask on site; the printed QR, the entry code and
-- the NFC tag get secrets of their own (the defaults), so the original's stay its own.
INSERT INTO facility_checkin (facility_id, printed_qr, entry_code, radius_m, flash_booking, staff_confirm, qr_reader, nfc, staff_contacts)
SELECT sqlc.arg('facility_id'), c.printed_qr, c.entry_code, c.radius_m, c.flash_booking, c.staff_confirm, c.qr_reader, c.nfc, c.staff_contacts
FROM facility_checkin c WHERE c.facility_id = sqlc.arg('source_id');
