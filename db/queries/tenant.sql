-- name: GetTenantBySlug :one
SELECT
    id,
    slug,
    name,
    timezone,
    contact_email,
    contact_phone,
    logo_ref
FROM
    tenant
WHERE
    slug = sqlc.arg('slug')
    AND is_active = TRUE;

-- name: ListTenants :many
SELECT
    id,
    slug,
    name,
    logo_ref,
    contact_email,
    contact_phone
FROM
    tenant
WHERE
    is_active = TRUE
ORDER BY
    name ASC;

-- name: ListTenantStats :many
-- The home page's council cards: active facilities, their sports, and how much
-- the council's residents have used LibreQuenda so far.
SELECT
    t.id,
    COUNT(DISTINCT f.id)::INT AS facilities,
    (
        SELECT COALESCE(STRING_AGG(DISTINCT tg.name, ',' ORDER BY tg.name), '')
        FROM facility af
        JOIN facility_tag ft ON ft.facility_id = af.id
        JOIN tag tg ON tg.id = ft.tag_id
        WHERE af.tenant_id = t.id
          AND af.is_active = TRUE
          AND tg.category_id = (SELECT c.id FROM tag_category c ORDER BY c.position, c.name LIMIT 1)
    )::TEXT AS sports,
    COUNT(b.id)::INT AS bookings,
    COUNT(DISTINCT b.user_id)::INT AS residents
FROM
    tenant t
    LEFT JOIN facility f ON f.tenant_id = t.id
        AND f.is_active = TRUE
    LEFT JOIN booking b ON b.facility_id = f.id
        AND b.status = 'confirmed'
WHERE
    t.is_active = TRUE
GROUP BY
    t.id;
