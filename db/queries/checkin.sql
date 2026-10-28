-- name: ListCheckinDevices :many
SELECT
    d.id,
    d.name,
    d.key_hint,
    d.last_seen_at,
    d.created_at
FROM
    checkin_device d
    JOIN facility f ON f.id = d.facility_id
WHERE
    d.facility_id = sqlc.arg('facility_id')
    AND f.tenant_id = sqlc.arg('tenant_id')
ORDER BY
    d.created_at ASC;

-- name: CreateCheckinDevice :one
-- Only for a facility of the council; no row when it belongs to another one.
INSERT INTO checkin_device (
    facility_id,
    name,
    key_hash,
    key_hint
)
SELECT
    f.id,
    sqlc.arg('name'),
    sqlc.arg('key_hash'),
    sqlc.arg('key_hint')
FROM
    facility f
WHERE
    f.id = sqlc.arg('facility_id')
    AND f.tenant_id = sqlc.arg('tenant_id')
RETURNING
    id,
    name,
    key_hint,
    last_seen_at,
    created_at;

-- name: DeleteCheckinDevice :execrows
DELETE FROM checkin_device d
USING facility f
WHERE
    d.id = sqlc.arg('id')
    AND d.facility_id = sqlc.arg('facility_id')
    AND f.id = d.facility_id
    AND f.tenant_id = sqlc.arg('tenant_id');

-- name: GetCheckinDeviceByKey :one
-- Signs a reader in; TouchCheckinDevice then records that it was seen.
SELECT
    d.id,
    d.name,
    f.id       AS facility_id,
    f.name     AS facility_name,
    f.slug     AS facility_slug,
    t.id       AS tenant_id,
    t.name     AS tenant_name,
    t.slug     AS tenant_slug,
    t.timezone AS timezone
FROM
    checkin_device d
    JOIN facility f ON f.id = d.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
WHERE
    d.key_hash = sqlc.arg('key_hash');

-- name: TouchCheckinDevice :exec
-- Row-level security: runs as the reader's own council (db.WithTenant).
UPDATE checkin_device
SET
    last_seen_at = NOW()
WHERE
    id = sqlc.arg('id');

-- name: GetBookingByCheckinToken :one
SELECT
    b.id,
    b.facility_id,
    b.status,
    b.starts_at,
    b.ends_at,
    b.checked_in_at,
    f.name AS facility_name,
    u.full_name
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    LEFT JOIN app_user u ON u.id = b.user_id
WHERE
    b.checkin_token = sqlc.arg('checkin_token');

-- name: MarkBookingCheckedIn :one
-- The first scan wins; no row when the booking was already checked in.
UPDATE booking
SET
    checked_in_at        = NOW(),
    checked_in_device_id = sqlc.arg('device_id'),
    updated_at           = NOW()
WHERE
    id = sqlc.arg('id')
    AND checked_in_at IS NULL
RETURNING
    checked_in_at;

-- name: DeleteExpiredUserQrTokens :exec
-- Run before issuing a new personal QR, so a user keeps one or two rows at most.
DELETE FROM user_qr_token
WHERE
    user_id = sqlc.arg('user_id')
    AND expires_at <= NOW();

-- name: CreateUserQrToken :one
INSERT INTO user_qr_token (
    user_id,
    expires_at
) VALUES (
    sqlc.arg('user_id'),
    sqlc.arg('expires_at')
)
RETURNING
    token,
    expires_at;

-- name: UseUserQrToken :one
-- A personal QR works once and only until it expires; no row otherwise.
UPDATE user_qr_token q
SET
    used_at = NOW()
FROM
    app_user u
WHERE
    q.token = sqlc.arg('token')
    AND q.used_at IS NULL
    AND q.expires_at > NOW()
    AND u.id = q.user_id
RETURNING
    q.user_id,
    u.full_name;

-- name: ReleaseUserQrToken :exec
-- Gives a scanned personal QR back when the reader refused it, so it can be shown again.
UPDATE user_qr_token
SET
    used_at = NULL
WHERE
    token = sqlc.arg('token');

-- name: GetUserCheckinTokenAtFacility :one
-- The user's own confirmed booking at the facility that a reader would let in
-- now: not over, and starting before window_ends (now + the early window).
SELECT
    checkin_token
FROM
    booking
WHERE
    user_id = sqlc.arg('user_id')
    AND facility_id = sqlc.arg('facility_id')
    AND status = 'confirmed'
    AND ends_at > NOW()
    AND starts_at <= sqlc.arg('window_ends')
ORDER BY
    starts_at ASC
LIMIT 1;
