-- name: GetTenantBySlug :one
SELECT
    id,
    slug,
    name,
    timezone,
    contact_email,
    contact_phone,
    contact_address,
    contact_person,
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
    contact_phone,
    contact_address,
    contact_person
FROM
    tenant
WHERE
    is_active = TRUE
ORDER BY
    name ASC;
