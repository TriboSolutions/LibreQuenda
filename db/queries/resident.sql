-- Residents a council verified (tenant_resident): they book and join at its
-- residents_only facilities.

-- name: ListTenantResidents :many
-- The council's verified residents, narrowed by name or email (q, '' = all) and
-- sorted by name (A-Z) or by the most recent verification, a page at a time.
SELECT
    u.id         AS user_id,
    u.email,
    COALESCE(u.full_name, '')::TEXT AS full_name,
    r.verified_at,
    COALESCE(v.full_name, v.email, '')::TEXT AS verified_by_name,
    COUNT(*) OVER () AS total
FROM
    tenant_resident r
    JOIN app_user u ON u.id = r.user_id
    LEFT JOIN app_user v ON v.id = r.verified_by
WHERE
    r.tenant_id = sqlc.arg('tenant_id')
    AND (sqlc.arg('q')::TEXT = ''
         OR u.email ILIKE '%' || sqlc.arg('q')::TEXT || '%'
         OR u.full_name ILIKE '%' || sqlc.arg('q')::TEXT || '%')
ORDER BY
    CASE WHEN sqlc.arg('sort')::TEXT = 'recent' THEN r.verified_at END DESC,
    lower(COALESCE(u.full_name, u.email)) ASC,
    u.id
LIMIT sqlc.arg('page_size') OFFSET sqlc.arg('page_offset');

-- name: AddTenantResident :one
-- Verifies the person; verifying them again keeps the first verification.
-- Returns whether they were not verified yet.
WITH added AS (
    INSERT INTO tenant_resident (tenant_id, user_id, verified_by)
    VALUES (sqlc.arg('tenant_id'), sqlc.arg('user_id'), sqlc.narg('verified_by'))
    ON CONFLICT (tenant_id, user_id) DO NOTHING
    RETURNING user_id
)
SELECT EXISTS (SELECT 1 FROM added) AS added;

-- name: RemoveTenantResident :execrows
DELETE FROM tenant_resident
WHERE
    tenant_id = sqlc.arg('tenant_id')
    AND user_id = sqlc.arg('user_id');

-- name: IsTenantResident :one
-- Verified by the council, or verified by their role: the council's own
-- administrators, and the superadministrators everywhere, need no request.
SELECT (
    EXISTS (SELECT 1 FROM tenant_resident r WHERE r.tenant_id = sqlc.arg('tenant_id') AND r.user_id = sqlc.arg('user_id'))
    OR EXISTS (SELECT 1 FROM tenant_membership m WHERE m.tenant_id = sqlc.arg('tenant_id') AND m.user_id = sqlc.arg('user_id') AND m.role = 'admin')
    OR EXISTS (SELECT 1 FROM platform_admin p WHERE p.user_id = sqlc.arg('user_id'))
)::BOOLEAN AS verified;

-- name: ListResidentCouncilsByUser :many
-- The councils where the person counts as verified, for /me and their profile:
-- those that verified them, those they administer, and every active council
-- for a superadministrator (as IsTenantResident), each since the earliest.
-- by_role: only through that role, not verified by the council: the person
-- books there as its resident, but is the resident of one council at most.
SELECT
    t.slug,
    t.name,
    MIN(v.since)::TIMESTAMPTZ AS verified_at,
    (NOT BOOL_OR(v.verified))::BOOLEAN AS by_role
FROM (
    SELECT r.tenant_id, r.verified_at AS since, TRUE AS verified FROM tenant_resident r WHERE r.user_id = sqlc.arg('user_id')
    UNION ALL
    SELECT m.tenant_id, m.created_at, FALSE FROM tenant_membership m WHERE m.user_id = sqlc.arg('user_id') AND m.role = 'admin'
    UNION ALL
    SELECT t2.id, p.created_at, FALSE FROM platform_admin p CROSS JOIN tenant t2 WHERE p.user_id = sqlc.arg('user_id') AND t2.is_active = TRUE
) v
JOIN tenant t ON t.id = v.tenant_id
GROUP BY
    t.id, t.slug, t.name
ORDER BY
    t.name;

-- ============================================================
--  Requests to be verified (resident_request): deleted once decided,
--  and after 15 days at the latest (PurgeResidentRequests).
-- ============================================================

-- name: CreateResidentRequest :one
-- One pending request per person and council: a second one returns no row.
INSERT INTO resident_request (tenant_id, user_id, dni, full_name, postal_code, street)
VALUES (sqlc.arg('tenant_id'), sqlc.arg('user_id'), sqlc.arg('dni'), sqlc.arg('full_name'), sqlc.arg('postal_code'), sqlc.arg('street'))
ON CONFLICT (tenant_id, user_id) DO NOTHING
RETURNING id, created_at;

-- name: ClaimWeeklyResidentRequest :one
-- One request a week per person, whatever the council: takes this week's, or
-- returns no row when it was used (withdrawing a request does not give it back).
UPDATE app_user
SET
    resident_requested_at = NOW()
WHERE
    id = sqlc.arg('user_id')
    AND (resident_requested_at IS NULL OR resident_requested_at <= NOW() - INTERVAL '7 days')
RETURNING id;

-- name: GetResidentRequestedAt :one
SELECT
    resident_requested_at
FROM
    app_user
WHERE
    id = sqlc.arg('user_id');

-- name: NotifyCouncilAdminsOfResidentRequest :exec
-- One notice per council administrator (bell and email), with the name the
-- person gave; the DNI and address stay in the request alone.
INSERT INTO notification (user_id, type, payload)
SELECT m.user_id, 'resident_requested', jsonb_build_object('tenant_name', t.name, 'tenant_slug', t.slug, 'name', sqlc.arg('name')::TEXT)
FROM tenant t
JOIN tenant_membership m ON m.tenant_id = t.id AND m.role = 'admin'
WHERE t.id = sqlc.arg('tenant_id');

-- name: NotifyResidentRequestDecided :exec
-- Tells the person how the council decided (resident_verified or resident_rejected).
INSERT INTO notification (user_id, type, payload)
SELECT sqlc.arg('user_id'), sqlc.arg('type')::notification_type, jsonb_build_object('tenant_name', t.name, 'tenant_slug', t.slug)
FROM tenant t
WHERE t.id = sqlc.arg('tenant_id');

-- name: DeleteOwnResidentRequest :execrows
-- The person withdraws their request.
DELETE FROM resident_request
WHERE
    tenant_id = sqlc.arg('tenant_id')
    AND user_id = sqlc.arg('user_id');

-- name: ListResidentRequestsByUser :many
-- The person's pending requests, for /me and their data export.
SELECT
    t.slug,
    t.name,
    q.dni,
    q.full_name,
    q.postal_code,
    q.street,
    q.created_at
FROM
    resident_request q
    JOIN tenant t ON t.id = q.tenant_id
WHERE
    q.user_id = sqlc.arg('user_id')
ORDER BY
    q.created_at;

-- name: ListResidentRequests :many
-- The council's pending requests, the oldest first (it expires first), a page at a time.
SELECT
    q.id,
    q.user_id,
    u.email,
    COALESCE(u.full_name, '')::TEXT AS account_name,
    q.dni,
    q.full_name,
    q.postal_code,
    q.street,
    q.created_at,
    COUNT(*) OVER () AS total
FROM
    resident_request q
    JOIN app_user u ON u.id = q.user_id
WHERE
    q.tenant_id = sqlc.arg('tenant_id')
ORDER BY
    q.created_at ASC,
    q.id
LIMIT sqlc.arg('page_size') OFFSET sqlc.arg('page_offset');

-- name: CountResidentRequests :one
SELECT
    COUNT(*)
FROM
    resident_request
WHERE
    tenant_id = sqlc.arg('tenant_id');

-- name: TakeResidentRequest :one
-- Deletes the request as the administrator decides on it, returning whose it was.
DELETE FROM resident_request
WHERE
    id = sqlc.arg('id')
    AND tenant_id = sqlc.arg('tenant_id')
RETURNING user_id;

-- name: DeleteResidentRequestOfUser :exec
-- Verified by email from the Veciños tab: their pending request has done its job.
DELETE FROM resident_request
WHERE
    tenant_id = sqlc.arg('tenant_id')
    AND user_id = sqlc.arg('user_id');

-- name: PurgeResidentRequests :execrows
-- Requests nobody decided on: their identity data goes after the retention period.
DELETE FROM resident_request
WHERE
    created_at < NOW() - make_interval(days => sqlc.arg('days')::INT);

-- name: ListPendingResidentEmails :many
-- The email queue for resident notices (no booking, no facility): to the
-- council's administrators, or to the person verified or not. Through the
-- council's relay or else the environment's.
SELECT
    n.id,
    n.type,
    n.payload,
    u.email,
    u.full_name,
    u.preferred_language,
    u.email_notifications,
    t.name AS tenant_name,
    t.slug AS tenant_slug,
    t.id   AS tenant_id,
    t.contact_phone AS tenant_phone,
    s.config AS smtp_config
FROM
    notification n
    JOIN app_user u ON u.id = n.user_id
    JOIN tenant   t ON t.slug = n.payload->>'tenant_slug'
    LEFT JOIN tenant_smtp s ON s.tenant_id = t.id
WHERE
    n.emailed_at IS NULL
    AND n.type IN ('resident_requested', 'resident_verified', 'resident_rejected', 'resident_revoked')
    AND (sqlc.arg('env_relay')::BOOLEAN OR s.tenant_id IS NOT NULL)
ORDER BY
    n.created_at ASC
LIMIT sqlc.arg('max_rows');

-- name: ResidentCouncilsBySubject :one
-- For the facility lists of someone signed in: a verified resident of one council
-- cannot ask any other, so other councils' residents-only facilities are left
-- out. restricted is false for anyone not verified anywhere and for a
-- superadministrator; councils are where they book residents-only all the same
-- (theirs, and those they administer, as IsTenantResident).
SELECT
    (EXISTS (SELECT 1 FROM tenant_resident r WHERE r.user_id = u.id)
     AND NOT EXISTS (SELECT 1 FROM platform_admin p WHERE p.user_id = u.id))::BOOLEAN AS restricted,
    ARRAY(
        SELECT t.slug FROM tenant_resident r JOIN tenant t ON t.id = r.tenant_id WHERE r.user_id = u.id
        UNION
        SELECT t.slug FROM tenant_membership m JOIN tenant t ON t.id = m.tenant_id WHERE m.user_id = u.id AND m.role = 'admin'
    )::TEXT[] AS councils
FROM
    app_user u
WHERE
    u.idp_subject = sqlc.arg('idp_subject');

-- name: GetResidentCouncil :one
-- The one council that verified the person as its resident, if any.
SELECT
    t.slug,
    t.name
FROM
    tenant_resident r
    JOIN tenant t ON t.id = r.tenant_id
WHERE
    r.user_id = sqlc.arg('user_id');

-- name: CancelBookingsOfFormerResident :many
-- No longer a resident: their own upcoming bookings at the council's residents-only
-- facilities are cancelled by the council (what each notice to those who joined says).
UPDATE booking b
SET
    status       = 'cancelled',
    cancelled_at = NOW(),
    cancelled_by = 'council'
FROM
    facility f
    JOIN tenant t ON t.id = f.tenant_id
WHERE
    b.facility_id = f.id
    AND f.tenant_id = sqlc.arg('tenant_id')
    AND f.residents_only = TRUE
    AND b.user_id = sqlc.arg('user_id')
    AND b.status = 'confirmed'
    AND b.starts_at > NOW()
RETURNING
    b.id,
    b.facility_id,
    f.name AS facility_name,
    t.slug AS tenant_slug,
    t.name AS tenant_name,
    b.starts_at,
    b.ends_at,
    COALESCE(b.title, '')::TEXT AS title;

-- name: LeaveGamesOfFormerResident :many
-- …and they leave the upcoming public bookings and events they had joined there
-- (each one is told to them: where and when).
WITH left_games AS (
    DELETE FROM booking_participant p
    WHERE
        p.user_id = sqlc.arg('user_id')
        AND p.booking_id IN (
            SELECT b.id
            FROM booking b JOIN facility f ON f.id = b.facility_id
            WHERE f.tenant_id = sqlc.arg('tenant_id') AND f.residents_only = TRUE AND b.status = 'confirmed' AND b.starts_at > NOW()
        )
    RETURNING p.booking_id
)
SELECT
    b.id,
    b.facility_id,
    f.name AS facility_name,
    t.slug AS tenant_slug,
    t.name AS tenant_name,
    b.starts_at,
    b.ends_at,
    COALESCE(b.title, '')::TEXT AS title
FROM
    left_games l
    JOIN booking  b ON b.id = l.booking_id
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id;

-- name: NotifyResidentRevoked :exec
-- Tells the person (bell and email) that the council took the verification back,
-- and how many of their bookings went with it.
INSERT INTO notification (user_id, type, payload)
SELECT sqlc.arg('user_id'), 'resident_revoked', jsonb_build_object('tenant_name', t.name, 'tenant_slug', t.slug, 'cancelled', sqlc.arg('cancelled')::INT)
FROM tenant t
WHERE t.id = sqlc.arg('tenant_id');
