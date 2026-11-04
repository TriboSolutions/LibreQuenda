-- ============================================================
--  RXPD / GDPR: the resident's data export and account erasure (/perfil)
-- ============================================================

-- name: ExportUser :one
SELECT
    id,
    email,
    full_name,
    email_verified,
    birth_date,
    phone,
    avatar_ref,
    preferred_language,
    email_reminders,
    email_notifications,
    show_recommendations,
    created_at,
    updated_at
FROM
    app_user
WHERE
    id = sqlc.arg('id');

-- name: ExportBookingsByUser :many
-- Every booking the user made, cancelled ones too.
SELECT
    b.id,
    t.name AS tenant_name,
    f.name AS facility_name,
    b.starts_at,
    b.ends_at,
    b.status,
    b.rules_accepted_at,
    b.public_description,
    b.cancelled_at,
    b.cancelled_by,
    b.checked_in_at,
    b.created_at,
    (SELECT COUNT(*) FROM booking_participant p WHERE p.booking_id = b.id) AS participant_count
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
WHERE
    b.user_id = sqlc.arg('user_id')
ORDER BY
    b.starts_at DESC;

-- name: ExportParticipationsByUser :many
-- Bookings of someone else the user joined through an invite link.
SELECT
    b.id,
    t.name AS tenant_name,
    f.name AS facility_name,
    b.starts_at,
    b.ends_at,
    b.status,
    p.joined_at
FROM
    booking_participant p
    JOIN booking  b ON b.id = p.booking_id
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
WHERE
    p.user_id = sqlc.arg('user_id')
ORDER BY
    b.starts_at DESC;

-- name: ExportNotificationsByUser :many
SELECT
    type,
    payload,
    created_at,
    read_at,
    emailed_at
FROM
    notification
WHERE
    user_id = sqlc.arg('user_id')
ORDER BY
    created_at DESC;

-- name: CancelUpcomingBookingsOfUser :execrows
-- Account erasure: the slots the user still held go back to everyone else.
UPDATE booking
SET
    status       = 'cancelled',
    cancelled_at = NOW(),
    cancelled_by = 'user'
WHERE
    user_id = sqlc.arg('user_id')
    AND status = 'confirmed'
    AND starts_at > NOW();

-- name: RevokeInviteLinksOfUser :exec
-- Account erasure: links to the user's bookings stop working.
UPDATE booking
SET
    invite_token = NULL,
    kind = 'private',
    public_description = ''
WHERE
    user_id = sqlc.arg('user_id');

-- name: DeleteUser :one
-- Account erasure. Bookings stay for the indicators with user_id set to NULL;
-- roles, notifications and the bookings the user joined go with the row
-- (ON DELETE in 01_tables.sql).
DELETE FROM app_user
WHERE
    id = sqlc.arg('id')
RETURNING
    avatar_ref;
