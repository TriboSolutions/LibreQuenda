-- Events: the council's own activities («Fiesta del baloncesto», «Torneo de
-- fútbol 5v5»), from the "Eventos" tab of the council administration. An event
-- is a booking of kind 'event' with no owner: any start and end, not the turns,
-- and none of the per-person limits. It may not overlap a confirmed booking (the
-- exclusion constraint on booking) nor a closure; residents join it like a
-- public booking.

-- name: ListBookingsInTheWay :many
-- The confirmed bookings an event would overlap, for staff to cancel them first.
SELECT
    b.id,
    b.starts_at,
    b.ends_at,
    b.kind,
    COALESCE(b.title, '')::TEXT AS title,
    COALESCE(NULLIF(u.full_name, ''), u.email, b.holder_name, '')::TEXT AS holder
FROM
    booking b
    LEFT JOIN app_user u ON u.id = b.user_id
WHERE
    b.facility_id = sqlc.arg('facility_id')
    AND b.status = 'confirmed'
    AND b.starts_at < sqlc.arg('ends_at')
    AND b.ends_at > sqlc.arg('starts_at')
    AND (sqlc.narg('except_id')::UUID IS NULL OR b.id <> sqlc.narg('except_id'))
ORDER BY
    b.starts_at;

-- name: ListClosuresInTheWay :many
SELECT
    c.starts_at,
    c.ends_at,
    c.reason
FROM
    closure c
WHERE
    c.facility_id = sqlc.arg('facility_id')
    AND c.starts_at < sqlc.arg('ends_at')
    AND c.ends_at > sqlc.arg('starts_at')
ORDER BY
    c.starts_at;

-- name: CreateEvent :one
INSERT INTO booking (
    facility_id,
    starts_at,
    ends_at,
    channel,
    booked_by,
    kind,
    title,
    public_description,
    event_capacity
) VALUES (
    sqlc.arg('facility_id'),
    sqlc.arg('starts_at'),
    sqlc.arg('ends_at'),
    'staff',
    sqlc.arg('booked_by'),
    'event',
    sqlc.arg('title'),
    sqlc.arg('description'),
    sqlc.narg('event_capacity')
)
RETURNING
    id;

-- name: LockEvent :one
-- Before changing an event: joins wait, so the capacity is not lowered under who joined.
SELECT
    b.id,
    b.facility_id,
    b.starts_at,
    b.ends_at,
    b.status,
    f.capacity AS facility_capacity,
    t.timezone,
    (SELECT COUNT(*) FROM booking_participant p WHERE p.booking_id = b.id) AS participant_count
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
WHERE
    b.id = sqlc.arg('id')
    AND b.kind = 'event'
    AND f.tenant_id = sqlc.arg('tenant_id')
FOR UPDATE OF b;

-- name: UpdateEvent :execrows
UPDATE booking
SET
    title              = sqlc.arg('title'),
    public_description = sqlc.arg('description'),
    event_capacity     = sqlc.narg('event_capacity'),
    starts_at          = sqlc.arg('starts_at'),
    ends_at            = sqlc.arg('ends_at'),
    updated_at         = NOW()
WHERE
    id = sqlc.arg('id')
    AND kind = 'event'
    AND status = 'confirmed';

-- name: ListEventsByTenant :many
-- The council's confirmed events that have not ended, by start.
SELECT
    b.id,
    b.facility_id,
    f.name AS facility_name,
    b.starts_at,
    b.ends_at,
    b.status,
    b.title,
    b.public_description,
    b.event_capacity,
    COALESCE(b.event_capacity, f.capacity) AS capacity,
    (SELECT COUNT(*) FROM booking_participant p WHERE p.booking_id = b.id) AS participant_count,
    t.timezone
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
    AND b.kind = 'event'
    AND b.status = 'confirmed'
    AND b.ends_at > NOW()
ORDER BY
    b.starts_at,
    b.id;

-- name: GetEvent :one
SELECT
    b.id,
    b.facility_id,
    f.name AS facility_name,
    b.starts_at,
    b.ends_at,
    b.status,
    b.title,
    b.public_description,
    b.event_capacity,
    COALESCE(b.event_capacity, f.capacity) AS capacity,
    (SELECT COUNT(*) FROM booking_participant p WHERE p.booking_id = b.id) AS participant_count,
    t.timezone
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
WHERE
    b.id = sqlc.arg('id')
    AND f.tenant_id = sqlc.arg('tenant_id')
    AND b.kind = 'event';

-- name: ListParticipantAccounts :many
-- Who joined with an account, to tell them the council cancelled it.
SELECT
    p.user_id
FROM
    booking_participant p
WHERE
    p.booking_id = sqlc.arg('booking_id')
    AND p.user_id IS NOT NULL;

-- name: ListUpcomingEvents :many
-- The app's home: every council's events that have not started and still have
-- spots, soonest first (a full one is left out, like a full public booking).
-- Public, like the facilities list.
SELECT
    b.id,
    COALESCE(b.title, '')::TEXT AS title,
    b.public_description,
    b.starts_at,
    b.ends_at,
    COALESCE(b.event_capacity, f.capacity) AS capacity,
    (SELECT COUNT(*) FROM booking_participant p WHERE p.booking_id = b.id) AS participant_count,
    f.id AS facility_id,
    f.slug AS facility_slug,
    f.name AS facility_name,
    f.image_ref,
    f.latitude,
    f.longitude,
    f.residents_only,
    t.slug AS tenant_slug,
    t.name AS tenant_name
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
WHERE
    b.kind = 'event'
    AND b.status = 'confirmed'
    AND b.starts_at > NOW()
    AND f.is_active = TRUE
    AND t.is_active = TRUE
    AND (SELECT COUNT(*) FROM booking_participant p WHERE p.booking_id = b.id) < COALESCE(b.event_capacity, f.capacity)
ORDER BY
    b.starts_at,
    b.id
LIMIT 50;
