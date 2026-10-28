-- Council staff booking for residents who phone or come to the desk, from the
-- "Reservas" tab of the council administration.

-- name: GetUserByEmailForStaff :one
-- The account staff book for. Emails are not unique: a verified one wins.
SELECT
    id,
    email,
    full_name
FROM
    app_user
WHERE
    lower(email) = lower(sqlc.arg('email'))
ORDER BY
    email_verified DESC,
    created_at ASC
LIMIT 1;

-- name: CreateStaffBooking :one
-- For an account (user_id) or for someone without one (holder_*). Overlaps are
-- rejected by the exclusion constraint on booking.
INSERT INTO booking (
    facility_id,
    user_id,
    starts_at,
    ends_at,
    rules_accepted_at,
    channel,
    booked_by,
    holder_name,
    holder_phone
) VALUES (
    sqlc.arg('facility_id'),
    sqlc.narg('user_id'),
    sqlc.arg('starts_at'),
    sqlc.arg('ends_at'),
    NOW(),
    'staff',
    sqlc.arg('booked_by'),
    sqlc.narg('holder_name'),
    sqlc.narg('holder_phone')
)
RETURNING
    *;

-- name: ListFacilityBookingsForStaff :many
-- The confirmed bookings of one facility of the council in a range, with who
-- holds each one and how to reach them.
SELECT
    b.id,
    b.starts_at,
    b.ends_at,
    b.channel,
    b.checked_in_at,
    b.checkin_token,
    b.user_id,
    u.full_name,
    u.email,
    u.phone,
    b.holder_name,
    b.holder_phone,
    (SELECT COUNT(*) FROM booking_participant p WHERE p.booking_id = b.id) AS participant_count
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    LEFT JOIN app_user u ON u.id = b.user_id
WHERE
    b.facility_id = sqlc.arg('facility_id')
    AND f.tenant_id = sqlc.arg('tenant_id')
    AND b.status = 'confirmed'
    AND b.starts_at < sqlc.arg('range_end')
    AND b.ends_at > sqlc.arg('range_start')
ORDER BY
    b.starts_at ASC;

-- name: GetBookingForCouncil :one
SELECT
    b.id,
    b.status,
    b.starts_at,
    b.ends_at,
    b.user_id,
    f.id   AS facility_id,
    f.name AS facility_name,
    t.slug AS tenant_slug,
    t.name AS tenant_name
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
WHERE
    b.id = sqlc.arg('id')
    AND f.tenant_id = sqlc.arg('tenant_id');

-- name: CancelBookingByCouncil :execrows
-- Staff cancel one booking before it starts; no row when it was cancelled or started meanwhile.
UPDATE booking
SET
    status       = 'cancelled',
    cancelled_at = NOW(),
    cancelled_by = 'council'
WHERE
    id = sqlc.arg('id')
    AND status = 'confirmed'
    AND starts_at > NOW();

-- name: ForgetBookingHolders :execrows
-- RXPD: the name and phone of someone without an account are kept until 30 days
-- after their booking ends, then removed. The booking stays for the indicators.
UPDATE booking
SET
    holder_name  = NULL,
    holder_phone = NULL
WHERE
    holder_name IS NOT NULL
    AND ends_at < NOW() - INTERVAL '30 days';
