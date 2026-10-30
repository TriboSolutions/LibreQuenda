-- name: InsertAuditEntry :exec
-- One entry of the audit log. The actor and their role then come from the
-- signed-in account's subject (none: the system); the council's name is kept
-- in case the council is deleted later.
INSERT INTO audit_log (actor_id, actor_role, tenant_id, tenant_label, action, target_type, target_id, target_user_id, target_label, summary)
SELECT
    u.id,
    CASE
        WHEN sqlc.narg('subject')::TEXT IS NULL THEN 'system'
        WHEN EXISTS (SELECT 1 FROM platform_admin pa WHERE pa.user_id = u.id) THEN 'superadmin'
        WHEN EXISTS (SELECT 1 FROM tenant_membership m WHERE m.user_id = u.id AND m.tenant_id = sqlc.narg('tenant_id')) THEN 'council_admin'
        ELSE 'resident'
    END,
    sqlc.narg('tenant_id'),
    COALESCE(sqlc.narg('tenant_label')::TEXT, (SELECT t.name FROM tenant t WHERE t.id = sqlc.narg('tenant_id'))),
    sqlc.arg('action'),
    sqlc.narg('target_type'),
    sqlc.narg('target_id'),
    sqlc.narg('target_user_id'),
    sqlc.narg('target_label'),
    sqlc.arg('summary')
FROM
    (SELECT 1) AS one
    LEFT JOIN app_user u ON u.idp_subject = sqlc.narg('subject');

-- name: ListAuditEntries :many
-- The log, newest first, one page before `before` (NULL: the newest). Inside
-- db.WithTenant only that council's entries are visible (row security).
-- Names come from the accounts as they are now: an erased one has none.
SELECT
    a.id,
    a.at,
    a.actor_id,
    a.actor_role,
    COALESCE(actor.full_name, actor.email, '')::TEXT AS actor_name,
    a.tenant_id,
    COALESCE(t.slug, '')::TEXT AS tenant_slug,
    COALESCE(t.name, a.tenant_label, '')::TEXT AS tenant_name,
    a.action,
    COALESCE(a.target_type, '')::TEXT AS target_type,
    a.target_id,
    a.target_user_id,
    COALESCE(target.full_name, target.email, '')::TEXT AS target_user_name,
    COALESCE(a.target_label, '')::TEXT AS target_label,
    (a.target_type = 'facility' AND EXISTS (SELECT 1 FROM facility f WHERE f.id = a.target_id))::BOOLEAN AS target_exists,
    a.summary
FROM
    audit_log a
    LEFT JOIN app_user actor ON actor.id = a.actor_id
    LEFT JOIN app_user target ON target.id = a.target_user_id
    LEFT JOIN tenant t ON t.id = a.tenant_id
WHERE
    (sqlc.narg('tenant_id')::UUID IS NULL OR a.tenant_id = sqlc.narg('tenant_id'))
    AND (sqlc.narg('actor_id')::UUID IS NULL OR a.actor_id = sqlc.narg('actor_id'))
    AND (sqlc.narg('action')::TEXT IS NULL OR a.action = sqlc.narg('action') OR a.action LIKE sqlc.narg('action') || '.%')
    AND (sqlc.narg('since')::TIMESTAMPTZ IS NULL OR a.at >= sqlc.narg('since'))
    AND (sqlc.narg('until')::TIMESTAMPTZ IS NULL OR a.at < sqlc.narg('until'))
    AND (sqlc.narg('before')::TIMESTAMPTZ IS NULL OR a.at < sqlc.narg('before'))
ORDER BY
    a.at DESC
LIMIT sqlc.arg('max_rows');

-- name: PurgeAuditLog :one
-- Entries older than the retention period go; returns how many.
SELECT purge_audit_log(make_interval(days => sqlc.arg('days')::INT))::BIGINT AS purged;

-- name: GetTenantIdBySlug :one
SELECT id, name FROM tenant WHERE slug = sqlc.arg('slug');

-- name: GetFacilityNameForAudit :one
SELECT name FROM facility WHERE id = sqlc.arg('id');

-- name: GetUserIdByEmailForAudit :one
-- The account a superadministrator was just given, to name it in the log by id.
SELECT id FROM app_user WHERE lower(email) = lower(sqlc.arg('email')) ORDER BY created_at LIMIT 1;
