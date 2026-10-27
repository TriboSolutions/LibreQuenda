-- ============================================================
--  INVITES: the owner of a booking shares a link and others
--  fill the facility's capacity, with an account or as guests.
-- ============================================================

-- name: GetOwnedBookingForInvite :one
SELECT
    b.id,
    b.starts_at,
    b.ends_at,
    b.status,
    b.invite_token,
    f.capacity
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
WHERE
    b.id = sqlc.arg('id')
    AND b.user_id = sqlc.arg('owner_id');

-- name: SetBookingInviteToken :one
-- A new token makes the previous link stop working.
UPDATE booking
SET
    invite_token = gen_random_uuid()
WHERE
    id = sqlc.arg('id')
    AND user_id = sqlc.arg('owner_id')
    AND status = 'confirmed'
RETURNING
    invite_token;

-- name: GetInviteByToken :one
-- What the invite page shows: no personal data beyond the owner's name.
SELECT
    b.id,
    b.user_id  AS owner_id,
    b.starts_at,
    b.ends_at,
    b.status,
    f.name     AS facility_name,
    f.slug     AS facility_slug,
    f.capacity,
    f.min_age,
    t.slug     AS tenant_slug,
    t.name     AS tenant_name,
    t.timezone,
    u.full_name AS owner_name,
    (SELECT COUNT(*) FROM booking_participant p WHERE p.booking_id = b.id) AS participant_count
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
    LEFT JOIN app_user u ON u.id = b.user_id
WHERE
    b.invite_token = sqlc.arg('invite_token');

-- name: LockBookingByInviteToken :one
-- Inside the join transaction: joins to the same booking wait for each other,
-- so the last free spot is taken once.
SELECT
    b.id,
    b.user_id AS owner_id,
    b.starts_at,
    b.ends_at,
    b.status,
    f.capacity,
    f.min_age,
    t.timezone
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
WHERE
    b.invite_token = sqlc.arg('invite_token')
FOR UPDATE OF b;

-- name: CountParticipants :one
SELECT
    COUNT(*)
FROM
    booking_participant
WHERE
    booking_id = sqlc.arg('booking_id');

-- name: AddUserParticipant :one
-- A second join by the same person is rejected by uq_booking_participant_user.
INSERT INTO booking_participant (
    booking_id,
    user_id
) VALUES (
    sqlc.arg('booking_id'),
    sqlc.arg('user_id')
)
RETURNING
    id;

-- name: AddGuestParticipant :one
INSERT INTO booking_participant (
    booking_id,
    guest_name,
    guest_age
) VALUES (
    sqlc.arg('booking_id'),
    sqlc.arg('guest_name'),
    sqlc.arg('guest_age')
)
RETURNING
    id;

-- name: ListParticipantsByBooking :many
SELECT
    p.id,
    p.user_id,
    COALESCE(u.full_name, u.email, p.guest_name, '')::TEXT AS name,
    p.guest_age,
    p.joined_at
FROM
    booking_participant p
    LEFT JOIN app_user u ON u.id = p.user_id
WHERE
    p.booking_id = sqlc.arg('booking_id')
ORDER BY
    p.joined_at ASC;

-- name: RemoveParticipant :execrows
-- Only the owner of the booking removes someone.
DELETE FROM booking_participant p
USING booking b
WHERE
    p.id = sqlc.arg('id')
    AND p.booking_id = b.id
    AND b.id = sqlc.arg('booking_id')
    AND b.user_id = sqlc.arg('owner_id');

-- name: GetParticipation :one
SELECT
    b.id,
    b.starts_at,
    b.status
FROM
    booking_participant p
    JOIN booking b ON b.id = p.booking_id
WHERE
    p.booking_id = sqlc.arg('booking_id')
    AND p.user_id = sqlc.arg('user_id');

-- name: LeaveBooking :execrows
DELETE FROM booking_participant
WHERE
    booking_id = sqlc.arg('booking_id')
    AND user_id = sqlc.arg('user_id');

-- name: ListParticipationsByUser :many
-- Bookings someone else made that this user joined.
SELECT
    b.id,
    b.starts_at,
    b.ends_at,
    b.status,
    b.checked_in_at,
    f.id   AS facility_id,
    f.name AS facility_name,
    f.capacity,
    t.slug AS tenant_slug,
    t.name AS tenant_name,
    COALESCE(u.full_name, '')::TEXT AS owner_name,
    (SELECT COUNT(*) FROM booking_participant c WHERE c.booking_id = b.id) AS participant_count
FROM
    booking_participant p
    JOIN booking  b ON b.id = p.booking_id
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
    LEFT JOIN app_user u ON u.id = b.user_id
WHERE
    p.user_id = sqlc.arg('user_id')
ORDER BY
    b.starts_at DESC;

-- name: CountParticipantsByBookings :many
-- Spots filled on the user's own bookings, for the bookings list.
SELECT
    b.id,
    f.capacity,
    COUNT(p.id) AS participant_count
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    LEFT JOIN booking_participant p ON p.booking_id = b.id
WHERE
    b.id = ANY(sqlc.arg('booking_ids')::UUID[])
GROUP BY
    b.id,
    f.capacity;

-- name: ListPeopleByBookings :many
-- Who joined each booking of the bookings list. Names only: never the email, never the age.
SELECT
    p.booking_id,
    p.user_id,
    COALESCE(u.full_name, p.guest_name, '')::TEXT AS public_name
FROM
    booking_participant p
    LEFT JOIN app_user u ON u.id = p.user_id
WHERE
    p.booking_id = ANY(sqlc.arg('booking_ids')::UUID[])
ORDER BY
    p.joined_at ASC;
