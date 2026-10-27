-- name: UpsertUserBySubject :one
-- Called on every login with the claims of the verified JWT.
INSERT INTO app_user (
    idp_subject,
    email,
    full_name,
    email_verified
) VALUES (
    sqlc.arg('idp_subject'),
    sqlc.arg('email'),
    sqlc.narg('full_name'),
    sqlc.arg('email_verified')
)
ON CONFLICT (idp_subject) DO UPDATE SET
    email          = EXCLUDED.email,
    full_name      = EXCLUDED.full_name,
    email_verified = EXCLUDED.email_verified
RETURNING
    id,
    idp_subject,
    email,
    full_name;

-- name: GetUserTenantRoles :many
SELECT
    m.tenant_id,
    t.slug AS tenant_slug,
    t.name AS tenant_name,
    m.role
FROM
    tenant_membership m
    JOIN tenant t ON t.id = m.tenant_id
WHERE
    m.user_id = sqlc.arg('user_id')
ORDER BY
    t.name ASC;

-- name: HasTenantRole :one
SELECT EXISTS (
    SELECT 1
    FROM tenant_membership
    WHERE user_id   = sqlc.arg('user_id')
      AND tenant_id = sqlc.arg('tenant_id')
      AND role      = sqlc.arg('role')
);

-- name: GetUserIdBySubject :one
SELECT
    id
FROM
    app_user
WHERE
    idp_subject = sqlc.arg('idp_subject');

-- name: GetUserProfile :one
SELECT
    id,
    email,
    full_name,
    email_verified,
    birth_date,
    phone,
    avatar_ref,
    preferred_language,
    email_reminders
FROM
    app_user
WHERE
    id = sqlc.arg('id');

-- name: UpdateUserProfile :exec
UPDATE app_user
SET
    birth_date         = sqlc.arg('birth_date'),
    phone              = sqlc.narg('phone'),
    avatar_ref         = sqlc.narg('avatar_ref'),
    preferred_language = sqlc.narg('preferred_language'),
    email_reminders    = sqlc.arg('email_reminders')
WHERE
    id = sqlc.arg('id');

-- name: HasBirthDate :one
SELECT (birth_date IS NOT NULL)::BOOLEAN AS complete FROM app_user WHERE id = sqlc.arg('id');

-- name: GetUserBirthDate :one
SELECT birth_date FROM app_user WHERE id = sqlc.arg('id');
