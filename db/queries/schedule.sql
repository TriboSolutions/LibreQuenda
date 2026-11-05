-- name: ListActiveFacilitiesByTenant :many
SELECT
    id,
    slug,
    name,
    description,
    image_ref,
    show_weather,
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
    max_bookings_per_week,
    -- the council's cap across its facilities (tenant_booking_rules; 0 = none)
    COALESCE((SELECT r.max_active_bookings_council FROM tenant_booking_rules r WHERE r.tenant_id = facility.tenant_id), 0)::INT AS council_max_active,
    weekly_opening,
    opens_weekday,
    opens_minute,
    capacity,
    min_age,
    residents_only,
    usage_rules_text
FROM
    facility
WHERE
    facility.tenant_id = sqlc.arg('tenant_id')
    AND facility.is_active = TRUE
ORDER BY
    facility.name ASC;

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
    f.max_bookings_per_week,
    f.weekly_opening,
    f.opens_weekday,
    f.opens_minute,
    f.no_show_penalty,
    f.no_show_limit,
    f.no_show_window_days,
    f.no_show_block_days,
    f.capacity,
    f.min_age,
    f.residents_only,
    t.timezone
FROM
    facility f
    JOIN tenant t ON t.id = f.tenant_id
WHERE
    f.id = sqlc.arg('id')
    AND f.is_active = TRUE
    AND t.is_active = TRUE;

-- name: ListPublicBookingsByTenantRange :many
-- Public activity details only; participant contact data and surnames stay private.
-- Events have no owner: owner_name is '' and the capacity is the event's own, if set.
SELECT b.id, b.facility_id, b.starts_at, b.ends_at, b.public_description,
    b.kind, COALESCE(b.title, '')::TEXT AS title,
    COALESCE(b.event_capacity, f.capacity) AS capacity,
    split_part(trim(COALESCE(owner.full_name, '')), ' ', 1)::TEXT AS owner_name,
    ARRAY(SELECT split_part(trim(COALESCE(u.full_name, p.guest_name, '')), ' ', 1)
          FROM booking_participant p LEFT JOIN app_user u ON u.id = p.user_id
          WHERE p.booking_id = b.id ORDER BY p.joined_at, p.id)::TEXT[] AS people
FROM booking b
JOIN facility f ON f.id = b.facility_id
LEFT JOIN app_user owner ON owner.id = b.user_id
WHERE f.tenant_id = sqlc.arg('tenant_id')
  AND f.is_active = TRUE AND b.kind <> 'private' AND b.status = 'confirmed'
  AND b.ends_at > NOW()
  AND b.starts_at < sqlc.arg('range_end') AND b.ends_at > sqlc.arg('range_start')
ORDER BY b.starts_at, b.id;

-- name: ListJoinableKindsByFacility :many
-- Which facilities have upcoming public bookings or events with spots left, for
-- the facilities list's «type of booking» filter.
SELECT DISTINCT
    b.facility_id,
    b.kind
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
WHERE
    b.kind <> 'private'
    AND b.status = 'confirmed'
    AND b.starts_at > NOW()
    AND f.is_active = TRUE
    AND (SELECT COUNT(*) FROM booking_participant p WHERE p.booking_id = b.id)
        < COALESCE(b.event_capacity, f.capacity) - CASE WHEN b.kind = 'event' THEN 0 ELSE 1 END;
