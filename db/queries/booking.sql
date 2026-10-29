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
    cancelled_at = NOW(),
    cancelled_by = 'user'
WHERE
    id = sqlc.arg('id')
    AND user_id = sqlc.arg('user_id')
    AND status = 'confirmed'
RETURNING
    *;

-- name: ChangeBookingTime :one
-- Moves the owner's booking to another turn of the same facility; overlaps are
-- rejected by the exclusion constraint on booking.
UPDATE booking
SET
    starts_at  = sqlc.arg('starts_at'),
    ends_at    = sqlc.arg('ends_at'),
    updated_at = NOW()
WHERE
    id = sqlc.arg('id')
    AND user_id = sqlc.arg('user_id')
    AND status = 'confirmed'
RETURNING
    *;

-- name: LockOwnedBooking :one
-- Inside the change transaction: joins to this booking wait (they lock it too),
-- so nobody joins while the people coming are checked against the new time.
SELECT
    b.id,
    b.starts_at,
    b.status,
    f.min_cancel_hours
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
WHERE
    b.id = sqlc.arg('id')
    AND b.user_id = sqlc.arg('user_id')
FOR UPDATE OF b;

-- name: ListBookingsByUser :many
-- reader_at_start: a QR reader was there when the turn began, so a missing
-- check-in means the person did not show up.
SELECT
    b.id,
    b.starts_at,
    b.ends_at,
    b.status,
    b.checkin_token,
    b.checked_in_at,
    EXISTS (SELECT 1 FROM checkin_device d WHERE d.facility_id = f.id AND d.created_at <= b.starts_at)::BOOLEAN AS reader_at_start,
    -- the ways in the facility offers now, for the check-in button
    (EXISTS (SELECT 1 FROM checkin_device d WHERE d.facility_id = f.id) AND COALESCE(c.qr_reader, TRUE))::BOOLEAN AS has_reader,
    (COALESCE(c.printed_qr, FALSE) AND f.latitude IS NOT NULL)::BOOLEAN AS printed_qr,
    (COALESCE(c.entry_code, FALSE) AND f.latitude IS NOT NULL)::BOOLEAN AS entry_code,
    COALESCE(c.staff_confirm, FALSE)::BOOLEAN AS staff_confirm,
    f.id   AS facility_id,
    f.name AS facility_name,
    f.slug AS facility_slug,
    f.min_cancel_hours,
    t.slug AS tenant_slug,
    t.name AS tenant_name,
    facility_sport(f.id)::TEXT AS facility_sport,
    sqlc.embed(f) -- what the booking details show: image, address, rules
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
    LEFT JOIN facility_checkin c ON c.facility_id = f.id
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
    b.facility_id,
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
-- except_booking_id: the booking being moved, which does not count against itself.
SELECT EXISTS (
    SELECT 1
    FROM booking b
    WHERE
        b.status = 'confirmed'
        AND b.id IS DISTINCT FROM sqlc.narg('except_booking_id')
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

-- name: LockParticipantBookings :exec
-- Inside the change transaction: the people who joined wait like the owner
-- (LockUserBookings), in a fixed order.
SELECT pg_advisory_xact_lock(hashtextextended(p.user_id::TEXT, 0))
FROM booking_participant p
WHERE
    p.booking_id = sqlc.arg('booking_id')
    AND p.user_id IS NOT NULL
ORDER BY
    p.user_id;

-- name: ParticipantsHaveOverlappingBooking :one
-- Someone who joined with an account is somewhere else in the new time,
-- as owner or invited (the same rule as UserHasOverlappingBooking).
SELECT EXISTS (
    SELECT 1
    FROM booking_participant p
    JOIN booking b ON b.id <> p.booking_id
    WHERE
        p.booking_id = sqlc.arg('booking_id')
        AND p.user_id IS NOT NULL
        AND b.status = 'confirmed'
        AND b.starts_at < sqlc.arg('ends_at')
        AND b.ends_at   > sqlc.arg('starts_at')
        AND (
            b.user_id = p.user_id
            OR EXISTS (
                SELECT 1 FROM booking_participant o
                WHERE o.booking_id = b.id AND o.user_id = p.user_id
            )
        )
);

-- name: ListCalendarBookings :many
-- The .ics feed: the user's own bookings and the ones they joined, from a month
-- back. Cancelled ones stay in, so calendar apps that already have them drop them.
SELECT
    b.id,
    b.starts_at,
    b.ends_at,
    b.status,
    b.updated_at,
    f.name AS facility_name,
    f.address,
    f.latitude,
    f.longitude,
    t.name AS tenant_name
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
WHERE
    (
        b.user_id = sqlc.arg('user_id')
        OR b.id IN (SELECT p.booking_id FROM booking_participant p WHERE p.user_id = sqlc.arg('user_id'))
    )
    AND b.ends_at > NOW() - INTERVAL '30 days'
ORDER BY
    b.starts_at ASC;
