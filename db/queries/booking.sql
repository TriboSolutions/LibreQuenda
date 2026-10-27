-- name: CreateBooking :one
-- Overlaps are rejected by the exclusion constraint on booking.
INSERT INTO booking (
    facility_id,
    user_id,
    starts_at,
    ends_at,
    rules_accepted_at
) VALUES (
    sqlc.arg('facility_id'),
    sqlc.arg('user_id'),
    sqlc.arg('starts_at'),
    sqlc.arg('ends_at'),
    NOW()
)
RETURNING
    *;

-- name: CancelBooking :one
UPDATE booking
SET
    status       = 'cancelled',
    cancelled_at = NOW()
WHERE
    id = sqlc.arg('id')
    AND user_id = sqlc.arg('user_id')
    AND status = 'confirmed'
RETURNING
    *;

-- name: ListBookingsByUser :many
SELECT
    b.id,
    b.starts_at,
    b.ends_at,
    b.status,
    b.checkin_token,
    b.checked_in_at,
    f.id   AS facility_id,
    f.name AS facility_name,
    f.min_cancel_hours,
    t.slug AS tenant_slug,
    t.name AS tenant_name
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
WHERE
    b.user_id = sqlc.arg('user_id')
ORDER BY
    b.starts_at DESC;

-- name: ListConfirmedBookingsInRange :many
-- Used to compute availability.
SELECT
    starts_at,
    ends_at
FROM
    booking
WHERE
    facility_id = sqlc.arg('facility_id')
    AND status = 'confirmed'
    AND starts_at < sqlc.arg('range_end')
    AND ends_at   > sqlc.arg('range_start')
ORDER BY
    starts_at ASC;

-- name: CountActiveBookingsByUserFacility :one
SELECT
    COUNT(*)
FROM
    booking
WHERE
    user_id = sqlc.arg('user_id')
    AND facility_id = sqlc.arg('facility_id')
    AND status = 'confirmed'
    AND ends_at > NOW();

-- name: GetBookingForUser :one
SELECT
    b.id,
    b.starts_at,
    b.ends_at,
    b.status,
    f.min_cancel_hours
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
WHERE
    b.id = sqlc.arg('id')
    AND b.user_id = sqlc.arg('user_id');

-- name: LockUserBookings :exec
-- Inside a transaction: one person's bookings and joins wait for each other,
-- so two requests at once cannot both pass the overlap check.
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg('user_id')::UUID::TEXT, 0));

-- name: UserHasOverlappingBooking :one
-- A person is in one place at a time: their own confirmed bookings and the
-- ones they joined through an invite, in any facility and any council.
SELECT EXISTS (
    SELECT 1
    FROM booking b
    WHERE
        b.status = 'confirmed'
        AND b.starts_at < sqlc.arg('ends_at')
        AND b.ends_at   > sqlc.arg('starts_at')
        AND (
            b.user_id = sqlc.arg('user_id')
            OR EXISTS (
                SELECT 1 FROM booking_participant p
                WHERE p.booking_id = b.id AND p.user_id = sqlc.arg('user_id')
            )
        )
);
