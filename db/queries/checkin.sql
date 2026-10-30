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
    COALESCE(u.full_name, b.holder_name) AS full_name -- someone without an account, booked by staff
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    LEFT JOIN app_user u ON u.id = b.user_id
WHERE
    b.checkin_token = sqlc.arg('checkin_token');

-- name: MarkBookingCheckedIn :one
-- The first scan wins; no row when the booking was already checked in.
-- checked_in_by: the staff member, only when method is 'staff'.
UPDATE booking
SET
    checked_in_at        = NOW(),
    checked_in_device_id = sqlc.arg('device_id'),
    checked_in_method    = sqlc.arg('method'),
    checked_in_by        = sqlc.arg('checked_in_by'),
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

-- name: EnsureFacilityCheckin :exec
-- Row-level security: runs as the facility's council (db.WithTenant). Creates the
-- facility's check-in settings with the defaults the first time they are needed.
INSERT INTO facility_checkin (facility_id)
SELECT
    f.id
FROM
    facility f
WHERE
    f.id = sqlc.arg('facility_id')
    AND f.tenant_id = sqlc.arg('tenant_id')
ON CONFLICT (facility_id) DO NOTHING;

-- name: GetFacilityCheckin :one
SELECT
    c.printed_qr,
    c.qr_token,
    c.entry_code,
    c.code_secret,
    c.radius_m,
    c.flash_booking,
    c.staff_confirm,
    c.qr_reader,
    c.staff_contacts
FROM
    facility_checkin c
    JOIN facility f ON f.id = c.facility_id
WHERE
    c.facility_id = sqlc.arg('facility_id')
    AND f.tenant_id = sqlc.arg('tenant_id');

-- name: UpdateFacilityCheckin :one
-- Row-level security: runs as the facility's council (db.WithTenant).
UPDATE facility_checkin
SET
    printed_qr    = sqlc.arg('printed_qr'),
    entry_code    = sqlc.arg('entry_code'),
    radius_m      = sqlc.arg('radius_m'),
    flash_booking = sqlc.arg('flash_booking'),
    staff_confirm = sqlc.arg('staff_confirm'),
    qr_reader     = sqlc.arg('qr_reader'),
    staff_contacts = sqlc.arg('staff_contacts'),
    updated_at    = NOW()
WHERE
    facility_id = sqlc.arg('facility_id')
RETURNING
    printed_qr,
    qr_token,
    entry_code,
    code_secret,
    radius_m,
    flash_booking,
    staff_confirm,
    qr_reader,
    staff_contacts;

-- name: RotateFacilityQrToken :one
-- Row-level security: runs as the facility's council (db.WithTenant). The QRs
-- already printed stop working.
UPDATE facility_checkin
SET
    qr_token   = gen_random_uuid(),
    updated_at = NOW()
WHERE
    facility_id = sqlc.arg('facility_id')
RETURNING
    qr_token;

-- name: RotateFacilityCodeSecret :one
-- Row-level security: runs as the facility's council (db.WithTenant). The code
-- already printed stops working.
UPDATE facility_checkin
SET
    code_secret = uuid_send(gen_random_uuid()) || uuid_send(gen_random_uuid()),
    updated_at  = NOW()
WHERE
    facility_id = sqlc.arg('facility_id')
RETURNING
    code_secret;

-- name: GetFacilityQrReader :one
-- Whether the facility's QR readers let anyone in; on without settings.
SELECT
    COALESCE((SELECT c.qr_reader FROM facility_checkin c WHERE c.facility_id = sqlc.arg('facility_id')), TRUE)::BOOLEAN AS qr_reader;

-- name: GetFacilityFlashBooking :one
-- Whether a free turn can be booked on the spot at the entrance; on without settings.
SELECT
    COALESCE((SELECT c.flash_booking FROM facility_checkin c WHERE c.facility_id = sqlc.arg('facility_id')), TRUE)::BOOLEAN AS flash_booking;

-- name: GetFacilityEntrance :one
-- What a resident's phone needs to check in at /<council>/<facility>/entrada.
SELECT
    f.id,
    f.name,
    f.slug,
    f.latitude,
    f.longitude,
    t.name     AS tenant_name,
    t.slug     AS tenant_slug,
    t.timezone,
    COALESCE(c.printed_qr, FALSE)::BOOLEAN    AS printed_qr,
    c.qr_token,
    COALESCE(c.entry_code, FALSE)::BOOLEAN    AS entry_code,
    c.code_secret,
    COALESCE(c.radius_m, 100)::INT            AS radius_m,
    COALESCE(c.flash_booking, TRUE)::BOOLEAN  AS flash_booking
FROM
    facility f
    JOIN tenant t ON t.id = f.tenant_id
    LEFT JOIN facility_checkin c ON c.facility_id = f.id
WHERE
    t.slug = sqlc.arg('tenant_slug')
    AND f.slug = sqlc.arg('facility_slug')
    AND f.is_active = TRUE
    AND t.is_active = TRUE;

-- name: ListNearbyFacilities :many
-- What the phone can find by its location: the facilities that book with their
-- entry code, have a QR on the door (checked before the camera opens), or have
-- staff confirmation (someone to ask). The phone's
-- distance to each is checked in the API.
SELECT
    f.name,
    f.slug,
    f.latitude::FLOAT8  AS latitude,
    f.longitude::FLOAT8 AS longitude,
    t.name AS tenant_name,
    t.slug AS tenant_slug,
    c.radius_m,
    (c.entry_code AND c.flash_booking)::BOOLEAN AS entry_code,
    c.printed_qr,
    c.staff_confirm,
    c.staff_contacts,
    -- who to ask without staff contacts: the facility's, else the council's
    f.contact_phone AS facility_phone,
    f.contact_email AS facility_email,
    t.contact_phone AS tenant_phone,
    t.contact_email AS tenant_email
FROM
    facility f
    JOIN tenant t ON t.id = f.tenant_id
    JOIN facility_checkin c ON c.facility_id = f.id
WHERE
    ((c.entry_code AND c.flash_booking) OR c.printed_qr OR c.staff_confirm)
    AND f.latitude IS NOT NULL
    AND f.longitude IS NOT NULL
    AND f.is_active = TRUE
    AND t.is_active = TRUE;

-- name: GetFacilityStaffConfirm :one
-- Whether council staff can mark the entry from the calendar; off without settings.
SELECT
    COALESCE((SELECT c.staff_confirm FROM facility_checkin c WHERE c.facility_id = sqlc.arg('facility_id')), FALSE)::BOOLEAN AS staff_confirm;

-- name: UndoStaffCheckIn :execrows
-- Takes back an entry staff marked by mistake, only within the undo window
-- (marked after since). No row when it was not a staff entry or it is too late.
UPDATE booking b
SET
    checked_in_at        = NULL,
    checked_in_device_id = NULL,
    checked_in_method    = NULL,
    checked_in_by        = NULL,
    updated_at           = NOW()
FROM
    facility f
WHERE
    b.id = sqlc.arg('id')
    AND f.id = b.facility_id
    AND f.tenant_id = sqlc.arg('tenant_id')
    AND b.checked_in_method = 'staff'
    AND b.checked_in_at > sqlc.arg('since')::TIMESTAMPTZ;
