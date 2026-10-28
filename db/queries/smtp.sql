-- name: GetTenantSmtp :one
SELECT
    config,
    updated_at
FROM
    tenant_smtp
WHERE
    tenant_id = sqlc.arg('tenant_id');

-- name: UpsertTenantSmtp :exec
INSERT INTO tenant_smtp (tenant_id, config)
VALUES (
    sqlc.arg('tenant_id'),
    sqlc.arg('config')
)
ON CONFLICT (tenant_id) DO UPDATE
SET
    config = EXCLUDED.config,
    updated_at = NOW();

-- name: DeleteTenantSmtp :execrows
-- Back to the relay in the environment.
DELETE FROM tenant_smtp
WHERE
    tenant_id = sqlc.arg('tenant_id');

-- name: GetTenantName :one
SELECT
    name
FROM
    tenant
WHERE
    id = sqlc.arg('id');
