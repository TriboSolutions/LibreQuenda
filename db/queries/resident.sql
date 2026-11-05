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
SELECT EXISTS (
    SELECT 1 FROM tenant_resident
    WHERE tenant_id = sqlc.arg('tenant_id') AND user_id = sqlc.arg('user_id')
) AS verified;

-- name: ListResidentCouncilsByUser :many
-- The councils that verified the person, for their profile.
SELECT
    t.slug,
    t.name,
    r.verified_at
FROM
    tenant_resident r
    JOIN tenant t ON t.id = r.tenant_id
WHERE
    r.user_id = sqlc.arg('user_id')
ORDER BY
    t.name;
