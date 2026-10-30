-- name: AdoptSeedTester :exec
-- Local testing only: the seed's tester (db/init/generate_data/tester.sql)
-- becomes the account that signs in with its verified email, the first time.
-- A database without seed data has no such row, so this changes nothing there.
UPDATE app_user
SET
    idp_subject = sqlc.arg('idp_subject'),
    updated_at  = NOW()
WHERE
    idp_subject = 'seed-tester'
    AND lower(email) = lower(sqlc.arg('email'))
    AND NOT EXISTS (SELECT 1 FROM app_user a WHERE a.idp_subject = sqlc.arg('idp_subject'));

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
    email_reminders,
    show_recommendations
FROM
    app_user
WHERE
    id = sqlc.arg('id');

-- name: UpdateUserProfile :exec
UPDATE app_user
SET
    birth_date         = sqlc.arg('birth_date'),
    phone              = sqlc.narg('phone'),
    avatar_ref         = sqlc.narg('avatar_ref')
WHERE
    id = sqlc.arg('id');

-- name: UpdateUserPreferences :exec
-- The profile's preferences, saved on their own: no birth date needed.
UPDATE app_user
SET
    preferred_language   = sqlc.narg('preferred_language'),
    email_reminders      = sqlc.arg('email_reminders'),
    show_recommendations = sqlc.arg('show_recommendations')
WHERE
    id = sqlc.arg('id');

-- name: SetShowRecommendations :exec
-- "For you" on the home page on or off, also from the section itself.
UPDATE app_user SET show_recommendations = sqlc.arg('show') WHERE id = sqlc.arg('id');

-- name: GetShowRecommendations :one
SELECT show_recommendations FROM app_user WHERE id = sqlc.arg('id');

-- name: HasBirthDate :one
SELECT (birth_date IS NOT NULL)::BOOLEAN AS complete FROM app_user WHERE id = sqlc.arg('id');

-- name: GetUserBirthDate :one
SELECT birth_date FROM app_user WHERE id = sqlc.arg('id');

-- ============================================================
--  Calendar feed (.ics): calendar apps fetch it with the token in the address
-- ============================================================

-- name: GetCalendarToken :one
SELECT calendar_token FROM app_user WHERE id = sqlc.arg('id');

-- name: ResetCalendarToken :one
-- Turns the feed on, or gives it a new address: the old one stops working.
UPDATE app_user
SET
    calendar_token = gen_random_uuid()
WHERE
    id = sqlc.arg('id')
RETURNING
    calendar_token;

-- name: ClearCalendarToken :exec
UPDATE app_user SET calendar_token = NULL WHERE id = sqlc.arg('id');

-- name: GetUserIdByCalendarToken :one
SELECT id FROM app_user WHERE calendar_token = sqlc.arg('calendar_token');
