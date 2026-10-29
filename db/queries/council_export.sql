-- ============================================================
--  COUNCIL DATA EXPORT: everything the council keeps in LibreQuenda, so it
--  can take it to another system (GET /tenants/{slug}/admin/export)
--  No secrets (SMTP, device keys, QR and invite tokens). Resident identities
--  only when the administrator asks for them (?personal=1); otherwise a person
--  is their account id.
-- ============================================================

-- name: ExportTenant :one
SELECT
    *
FROM
    tenant
WHERE
    id = sqlc.arg('tenant_id');

-- name: ExportFacilities :many
SELECT
    sqlc.embed(facility),
    facility_sport(facility.id)::TEXT AS sport,
    ARRAY(
        SELECT tg.slug FROM facility_tag ft JOIN tag tg ON tg.id = ft.tag_id
        WHERE ft.facility_id = facility.id ORDER BY tg.slug
    )::TEXT[] AS tags
FROM
    facility
WHERE
    tenant_id = sqlc.arg('tenant_id')
ORDER BY
    name ASC;

-- name: ExportOpeningHours :many
SELECT
    oh.id,
    oh.facility_id,
    oh.weekday,
    oh.opens_at,
    oh.closes_at
FROM
    opening_hours oh
    JOIN facility f ON f.id = oh.facility_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
ORDER BY
    f.name ASC,
    oh.weekday ASC,
    oh.opens_at ASC;

-- name: ExportClosures :many
SELECT
    c.id,
    c.facility_id,
    c.starts_at,
    c.ends_at,
    c.reason,
    c.created_at
FROM
    closure c
    JOIN facility f ON f.id = c.facility_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
ORDER BY
    c.starts_at ASC;

-- name: ExportCheckinDevices :many
SELECT
    d.id,
    d.facility_id,
    d.name,
    d.key_hint,
    d.last_seen_at,
    d.created_at
FROM
    checkin_device d
    JOIN facility f ON f.id = d.facility_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
ORDER BY
    f.name ASC,
    d.name ASC;

-- name: ExportBookings :many
-- Every booking, cancelled ones too.
SELECT
    b.id,
    b.facility_id,
    b.user_id,
    b.starts_at,
    b.ends_at,
    b.status,
    b.rules_accepted_at,
    b.cancelled_at,
    b.cancelled_by,
    b.checked_in_at,
    b.checked_in_device_id,
    b.checked_in_method,
    b.created_at,
    b.updated_at
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
ORDER BY
    b.starts_at ASC,
    b.id ASC;

-- name: ExportBookingParticipants :many
-- A guest's name and age are written only with ?personal=1.
SELECT
    p.id,
    p.booking_id,
    p.user_id,
    (p.user_id IS NULL)::BOOLEAN AS is_guest,
    p.guest_name,
    p.guest_age,
    p.joined_at
FROM
    booking_participant p
    JOIN booking  b ON b.id = p.booking_id
    JOIN facility f ON f.id = b.facility_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
ORDER BY
    b.starts_at ASC,
    p.joined_at ASC;

-- name: ExportPeople :many
-- Everyone who booked at the council or joined one of its bookings (?personal=1).
SELECT
    u.id,
    u.full_name,
    u.email,
    u.email_verified,
    u.phone,
    u.birth_date,
    u.preferred_language,
    u.created_at
FROM
    app_user u
WHERE
    u.id IN (
        SELECT b.user_id
        FROM booking b
        JOIN facility f ON f.id = b.facility_id
        WHERE f.tenant_id = sqlc.arg('tenant_id')
        UNION
        SELECT p.user_id
        FROM booking_participant p
        JOIN booking  b ON b.id = p.booking_id
        JOIN facility f ON f.id = b.facility_id
        WHERE f.tenant_id = sqlc.arg('tenant_id')
    )
ORDER BY
    u.id ASC;
