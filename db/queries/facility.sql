-- name: ListFacilitiesByTenant :many
SELECT
    id,
    name,
    description,
    sport,
    address,
    image_ref
FROM
    facility
WHERE
    tenant_id = sqlc.arg('tenant_id')
    AND is_active = TRUE
    AND (
        sqlc.narg('search')::TEXT IS NULL
        OR unaccent(lower(name)) LIKE '%' || unaccent(lower(sqlc.narg('search')::TEXT)) || '%'
    )
ORDER BY
    name ASC;

-- name: GetFacility :one
SELECT
    *
FROM
    facility
WHERE
    id = sqlc.arg('id')
    AND tenant_id = sqlc.arg('tenant_id');

-- name: ListOpeningHoursByFacility :many
SELECT
    weekday,
    opens_at,
    closes_at
FROM
    opening_hours
WHERE
    facility_id = sqlc.arg('facility_id')
ORDER BY
    weekday ASC,
    opens_at ASC;
