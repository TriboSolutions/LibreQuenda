-- name: ListFacilitiesByTenant :many
SELECT
    id,
    name,
    description,
    facility_sport(id)::TEXT AS sport,
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

-- name: GetFacilityWeatherPlace :one
-- Where to read the forecast of an active facility: its council's municipality
-- (INE code, for MeteoGalicia) and its explicit forecast preference.
SELECT
    t.municipality_code,
    t.timezone,
    f.show_weather
FROM
    facility f
    JOIN tenant t ON t.id = f.tenant_id
WHERE
    t.slug = sqlc.arg('tenant_slug')
    AND f.slug = sqlc.arg('facility_slug')
    AND f.is_active = TRUE
    AND t.is_active = TRUE;
