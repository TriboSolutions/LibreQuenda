-- name: CreateBookingReminders :execrows
-- 15 minutes before the start: one reminder per confirmed booking, to its owner
-- (people who joined through an invite get no notices).
INSERT INTO notification (
    user_id,
    booking_id,
    type,
    payload
)
SELECT
    b.user_id,
    b.id,
    'booking_reminder',
    jsonb_build_object(
        'facility_id',   f.id,
        'facility_name', f.name,
        'tenant_slug',   t.slug,
        'starts_at',     b.starts_at,
        'ends_at',       b.ends_at
    )
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
WHERE
    b.status = 'confirmed'
    AND b.user_id IS NOT NULL
    AND b.starts_at > NOW()
    AND b.starts_at <= NOW() + INTERVAL '15 minutes'
    AND NOT EXISTS (
        SELECT 1
        FROM notification n
        WHERE n.booking_id = b.id
          AND n.type = 'booking_reminder'
    );

-- name: CreateNotification :exec
INSERT INTO notification (
    user_id,
    booking_id,
    type,
    payload
) VALUES (
    sqlc.arg('user_id'),
    sqlc.arg('booking_id'),
    sqlc.arg('type'),
    sqlc.arg('payload')
);

-- name: ListPendingEmails :many
-- The email queue: every notification is also an email, sent through the
-- council's relay or else the environment's.
SELECT
    n.id,
    n.type,
    n.payload,
    u.email,
    u.full_name,
    u.preferred_language,
    u.email_reminders,
    b.status,
    b.starts_at,
    b.ends_at,
    b.checkin_token,
    f.name     AS facility_name,
    f.slug     AS facility_slug,
    f.address  AS facility_address,
    t.name     AS tenant_name,
    t.slug     AS tenant_slug,
    t.timezone AS timezone,
    t.id       AS tenant_id,
    s.config AS smtp_config,
    -- the ways in the facility takes, each explained in the email
    (EXISTS (SELECT 1 FROM checkin_device d WHERE d.facility_id = f.id) AND COALESCE(c.qr_reader, TRUE))::BOOLEAN AS has_reader,
    (COALESCE(c.entry_code, FALSE) AND f.latitude IS NOT NULL)::BOOLEAN AS entry_code,
    (COALESCE(c.printed_qr, FALSE) AND f.latitude IS NOT NULL)::BOOLEAN AS printed_qr,
    COALESCE(c.staff_confirm, FALSE)::BOOLEAN AS staff_confirm
FROM
    notification n
    JOIN app_user u ON u.id = n.user_id
    JOIN booking  b ON b.id = n.booking_id
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
    LEFT JOIN tenant_smtp s ON s.tenant_id = t.id
    LEFT JOIN facility_checkin c ON c.facility_id = f.id
WHERE
    n.emailed_at IS NULL
    -- with no relay in the environment, a council without its own waits
    AND (sqlc.arg('env_relay')::BOOLEAN OR s.tenant_id IS NOT NULL)
ORDER BY
    n.created_at ASC
LIMIT sqlc.arg('max_rows');

-- name: MarkNotificationEmailed :exec
-- Leaves the email queue, sent or skipped (reminders off, booking cancelled or started).
UPDATE notification
SET
    emailed_at = NOW()
WHERE
    id = sqlc.arg('id');

-- name: ListNotificationsByUser :many
-- The bell: the most recent notices first.
SELECT
    id,
    type,
    payload,
    created_at,
    read_at
FROM
    notification
WHERE
    user_id = sqlc.arg('user_id')
ORDER BY
    created_at DESC
LIMIT sqlc.arg('max_rows');

-- name: CountUnreadNotifications :one
SELECT
    COUNT(*)
FROM
    notification
WHERE
    user_id = sqlc.arg('user_id')
    AND read_at IS NULL;

-- name: MarkNotificationsRead :exec
UPDATE notification
SET
    read_at = NOW()
WHERE
    user_id = sqlc.arg('user_id')
    AND read_at IS NULL;
