-- name: ListActiveFacilitiesByTenant :many
SELECT
    id,
    slug,
    name,
    description,
    image_ref,
    latitude,
    longitude,
    facility_sport(facility.id)::TEXT AS sport,
    ARRAY(
        SELECT tg.slug FROM facility_tag ft JOIN tag tg ON tg.id = ft.tag_id JOIN tag_category tc ON tc.id = tg.category_id
        WHERE ft.facility_id = facility.id ORDER BY tc.position, tg.name
    )::TEXT[] AS tags,
    address,
    contact_phone,
    contact_email,
    slot_minutes,
    max_advance_hours,
    min_advance_minutes,
    min_cancel_hours,
    max_active_bookings_per_user,
    max_bookings_per_day,
    max_turns_in_a_row,
    min_age,
    usage_rules_text
FROM
    facility
WHERE
    tenant_id = sqlc.arg('tenant_id')
    AND is_active = TRUE
ORDER BY
    name ASC;

-- name: ListOpeningHoursByTenantWeekday :many
SELECT
    oh.facility_id,
    oh.opens_at,
    oh.closes_at
FROM
    opening_hours oh
    JOIN facility f ON f.id = oh.facility_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
    AND oh.weekday = sqlc.arg('weekday')
ORDER BY
    oh.facility_id,
    oh.opens_at ASC;

-- name: ListClosuresByTenantRange :many
SELECT
    c.facility_id,
    c.starts_at,
    c.ends_at,
    c.reason
FROM
    closure c
    JOIN facility f ON f.id = c.facility_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
    AND c.starts_at < sqlc.arg('range_end')
    AND c.ends_at > sqlc.arg('range_start');

-- name: ListConfirmedBookingsByTenantRange :many
-- Busy intervals only: who booked is never exposed publicly.
SELECT
    b.facility_id,
    b.starts_at,
    b.ends_at
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
    AND b.status = 'confirmed'
    AND b.starts_at < sqlc.arg('range_end')
    AND b.ends_at > sqlc.arg('range_start');

-- name: GetFacilityForBooking :one
SELECT
    f.id,
    f.tenant_id,
    f.name,
    f.slot_minutes,
    f.max_advance_hours,
    f.min_advance_minutes,
    f.min_cancel_hours,
    f.max_active_bookings_per_user,
    f.max_bookings_per_day,
    f.max_turns_in_a_row,
    f.no_show_penalty,
    f.no_show_limit,
    f.no_show_window_days,
    f.no_show_block_days,
    f.min_age,
    t.timezone
FROM
    facility f
    JOIN tenant t ON t.id = f.tenant_id
WHERE
    f.id = sqlc.arg('id')
    AND f.is_active = TRUE
    AND t.is_active = TRUE;
