-- ============================================================
--  CLOSURES AND COUNCIL CANCELLATIONS
--  A closure (works, maintenance, an event) cancels the bookings inside it;
--  deactivating a facility cancels its upcoming bookings only if the council
--  asks for it. Either way each owner gets a "sorry" notice with the reason.
--  Writes run as that council (row-level security on closure).
-- ============================================================

-- name: GetFacilityTimezone :one
SELECT
    t.timezone
FROM
    facility f
    JOIN tenant t ON t.id = f.tenant_id
WHERE
    f.id = sqlc.arg('facility_id')
    AND f.tenant_id = sqlc.arg('tenant_id');

-- name: ListUpcomingClosures :many
SELECT
    id,
    starts_at,
    ends_at,
    reason
FROM
    closure
WHERE
    facility_id = sqlc.arg('facility_id')
    AND ends_at > NOW()
ORDER BY
    starts_at ASC;

-- name: CreateClosure :one
INSERT INTO closure (
    facility_id,
    starts_at,
    ends_at,
    reason
) VALUES (
    sqlc.arg('facility_id'),
    sqlc.arg('starts_at'),
    sqlc.arg('ends_at'),
    sqlc.arg('reason')
)
RETURNING
    id,
    starts_at,
    ends_at,
    reason;

-- name: DeleteClosure :execrows
-- Reopens the slots; bookings it cancelled stay cancelled.
DELETE FROM closure c
USING facility f
WHERE
    c.id = sqlc.arg('id')
    AND c.facility_id = sqlc.arg('facility_id')
    AND f.id = c.facility_id
    AND f.tenant_id = sqlc.arg('tenant_id');

-- name: CountBookingsToCancel :one
-- What CancelBookingsInRange would cancel: confirmed bookings not yet started
-- that overlap the range.
SELECT
    COUNT(*)
FROM
    booking
WHERE
    facility_id = sqlc.arg('facility_id')
    AND status = 'confirmed'
    AND starts_at > NOW()
    AND starts_at < sqlc.arg('range_end')
    AND ends_at   > sqlc.arg('range_start');

-- name: CancelBookingsInRange :one
-- Cancels them and leaves a "booking_cancelled_by_council" notice (and email) for
-- each owner and everyone with an account who joined; answers how many bookings.
-- closed_from / closed_until are set for a closure, NULL when the facility is
-- deactivated.
WITH cancelled AS (
    UPDATE booking b
    SET
        status       = 'cancelled',
        cancelled_at = NOW(),
        cancelled_by = 'council'
    WHERE
        b.facility_id = sqlc.arg('facility_id')
        AND b.status = 'confirmed'
        AND b.starts_at > NOW()
        AND b.starts_at < sqlc.arg('range_end')
        AND b.ends_at   > sqlc.arg('range_start')
    RETURNING
        b.id,
        b.user_id,
        b.starts_at,
        b.ends_at
),
told AS (
INSERT INTO notification (
    user_id,
    booking_id,
    type,
    payload
)
SELECT
    x.user_id,
    c.id,
    'booking_cancelled_by_council',
    jsonb_build_object(
        'facility_id',   f.id,
        'facility_name', f.name,
        'tenant_slug',   t.slug,
        'starts_at',     c.starts_at,
        'ends_at',       c.ends_at,
        'reason',        sqlc.arg('reason')::TEXT,
        'closed_from',   sqlc.narg('closed_from')::TIMESTAMPTZ,
        'closed_until',  sqlc.narg('closed_until')::TIMESTAMPTZ
    )
FROM
    cancelled c
    JOIN facility f ON f.id = sqlc.arg('facility_id')
    JOIN tenant   t ON t.id = f.tenant_id
    CROSS JOIN LATERAL (
        SELECT c.user_id
        UNION
        SELECT p.user_id FROM booking_participant p WHERE p.booking_id = c.id
    ) x
WHERE
    x.user_id IS NOT NULL
RETURNING 1
)
SELECT COUNT(*) FROM cancelled;

-- name: SetFacilityActive :execrows
UPDATE facility
SET
    is_active = sqlc.arg('is_active')
WHERE
    id = sqlc.arg('id')
    AND tenant_id = sqlc.arg('tenant_id');
