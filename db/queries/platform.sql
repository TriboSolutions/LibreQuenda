-- ============================================================
--  SUPERADMINISTRATORS
-- ============================================================

-- name: IsPlatformAdmin :one
SELECT EXISTS (
    SELECT 1 FROM platform_admin WHERE user_id = sqlc.arg('user_id')
);

-- name: AddPlatformAdmin :exec
INSERT INTO platform_admin (user_id)
VALUES (sqlc.arg('user_id'))
ON CONFLICT (user_id) DO NOTHING;

-- name: RemovePlatformAdmin :execrows
DELETE FROM platform_admin
WHERE user_id = sqlc.arg('user_id');

-- name: CountPlatformAdmins :one
SELECT COUNT(*) FROM platform_admin;

-- name: ListPlatformAdmins :many
SELECT
    u.id,
    u.email,
    u.full_name,
    pa.created_at
FROM
    platform_admin pa
    JOIN app_user u ON u.id = pa.user_id
ORDER BY
    pa.created_at ASC;

-- name: GetVerifiedUserByEmail :one
SELECT
    id,
    email,
    full_name
FROM
    app_user
WHERE
    lower(email) = lower(sqlc.arg('email'))
    AND email_verified = TRUE
LIMIT 1;


-- ============================================================
--  COUNCILS (superadmin)
-- ============================================================

-- name: ListTenantsForPlatform :many
SELECT
    t.id,
    t.slug,
    t.name,
    t.timezone,
    t.contact_email,
    t.contact_phone,
    t.logo_ref,
    t.is_active,
    t.created_at,
    (SELECT COUNT(*) FROM facility f WHERE f.tenant_id = t.id AND f.is_active)::INT AS facility_count,
    (SELECT COUNT(*) FROM tenant_membership m WHERE m.tenant_id = t.id AND m.role = 'admin')::INT AS admin_count,
    (SELECT COUNT(*) FROM tenant_invitation i WHERE i.tenant_id = t.id AND i.accepted_at IS NULL)::INT AS pending_invitation_count,
    (SELECT COUNT(*) FROM booking b JOIN facility f ON f.id = b.facility_id WHERE f.tenant_id = t.id)::INT AS booking_count
FROM
    tenant t
ORDER BY
    t.name ASC;

-- name: GetTenantForPlatform :one
SELECT
    id,
    slug,
    name,
    timezone,
    contact_email,
    contact_phone,
    logo_ref,
    is_active
FROM
    tenant
WHERE
    slug = sqlc.arg('slug');

-- name: CreateTenant :one
INSERT INTO tenant (
    slug,
    name,
    timezone,
    contact_email,
    contact_phone,
    logo_ref
) VALUES (
    sqlc.arg('slug'),
    sqlc.arg('name'),
    sqlc.arg('timezone'),
    sqlc.narg('contact_email'),
    sqlc.narg('contact_phone'),
    sqlc.narg('logo_ref')
)
RETURNING
    id,
    slug;

-- name: UpdateTenant :one
UPDATE tenant
SET
    slug          = sqlc.arg('slug'),
    name          = sqlc.arg('name'),
    timezone      = sqlc.arg('timezone'),
    contact_email = sqlc.narg('contact_email'),
    contact_phone = sqlc.narg('contact_phone'),
    logo_ref      = sqlc.narg('logo_ref'),
    is_active     = sqlc.arg('is_active')
WHERE
    id = sqlc.arg('id')
RETURNING
    id,
    slug;

-- name: DeleteTenant :execrows
DELETE FROM tenant
WHERE id = sqlc.arg('id');

-- name: CountBookingsByTenant :one
SELECT
    COUNT(*)
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id');


-- ============================================================
--  COUNCIL ADMINISTRATORS AND INVITATIONS
-- ============================================================

-- name: ListTenantMembers :many
SELECT
    u.id AS user_id,
    u.email,
    u.full_name,
    m.role,
    m.created_at
FROM
    tenant_membership m
    JOIN app_user u ON u.id = m.user_id
WHERE
    m.tenant_id = sqlc.arg('tenant_id')
ORDER BY
    m.created_at ASC;

-- name: RemoveTenantMember :execrows
DELETE FROM tenant_membership
WHERE
    tenant_id = sqlc.arg('tenant_id')
    AND user_id = sqlc.arg('user_id')
    AND role = sqlc.arg('role');

-- name: ListPendingInvitations :many
SELECT
    id,
    email,
    role,
    created_at
FROM
    tenant_invitation
WHERE
    tenant_id = sqlc.arg('tenant_id')
    AND accepted_at IS NULL
ORDER BY
    created_at ASC;

-- name: CreateInvitation :one
-- Returns no row when the same invitation is already pending.
INSERT INTO tenant_invitation (
    tenant_id,
    email,
    role,
    invited_by
) VALUES (
    sqlc.arg('tenant_id'),
    lower(sqlc.arg('email')),
    sqlc.arg('role'),
    sqlc.narg('invited_by')
)
ON CONFLICT (tenant_id, lower(email), role) WHERE accepted_at IS NULL DO NOTHING
RETURNING
    id,
    email,
    role,
    created_at;

-- name: DeleteInvitation :execrows
DELETE FROM tenant_invitation
WHERE
    id = sqlc.arg('id')
    AND tenant_id = sqlc.arg('tenant_id')
    AND accepted_at IS NULL;

-- name: AcceptPendingInvitations :execrows
-- Turns every pending invitation for this (verified) email into a role, and
-- leaves a notice in the bell for each one. The notice is not emailed: the
-- invitation email already said it.
WITH accepted AS (
    UPDATE tenant_invitation
    SET accepted_at = NOW()
    WHERE lower(email) = lower(sqlc.arg('email')) AND accepted_at IS NULL
    RETURNING tenant_id, role
),
granted AS (
    INSERT INTO tenant_membership (user_id, tenant_id, role)
    SELECT sqlc.arg('user_id')::UUID, tenant_id, role FROM accepted
    ON CONFLICT DO NOTHING
    RETURNING tenant_id
)
INSERT INTO notification (user_id, type, payload, emailed_at)
SELECT
    sqlc.arg('user_id')::UUID,
    'council_admin_granted',
    jsonb_build_object('tenant_name', t.name, 'tenant_slug', t.slug),
    NOW()
FROM
    granted g
    JOIN tenant t ON t.id = g.tenant_id;

-- name: ListPendingInvitationEmails :many
-- Invitations still to email, sent whether or not the person has an account yet.
SELECT
    i.id,
    i.email,
    t.name AS tenant_name,
    t.slug AS tenant_slug,
    u.full_name,
    u.preferred_language,
    t.id AS tenant_id,
    s.config AS smtp_config
FROM
    tenant_invitation i
    JOIN tenant t ON t.id = i.tenant_id
    LEFT JOIN app_user u ON lower(u.email) = lower(i.email)
    LEFT JOIN tenant_smtp s ON s.tenant_id = t.id
WHERE
    i.emailed_at IS NULL
    AND i.accepted_at IS NULL
    AND (sqlc.arg('env_relay')::BOOLEAN OR s.tenant_id IS NOT NULL)
ORDER BY
    i.created_at ASC
LIMIT sqlc.arg('max_rows');

-- name: MarkInvitationEmailed :exec
UPDATE tenant_invitation
SET
    emailed_at = NOW()
WHERE
    id = sqlc.arg('id');

-- name: ListPendingInvitationsForEmail :many
-- What this person is invited to and has not received yet: the role waits for a
-- verified email, and the web says so.
SELECT
    t.name AS tenant_name,
    t.slug AS tenant_slug
FROM
    tenant_invitation i
    JOIN tenant t ON t.id = i.tenant_id
WHERE
    lower(i.email) = lower(sqlc.arg('email'))
    AND i.accepted_at IS NULL
ORDER BY
    i.created_at ASC;
