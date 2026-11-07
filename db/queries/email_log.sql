-- name: InsertEmailLog :exec
-- One email LibreQuenda tried to send, sent or not.
INSERT INTO email_log (status, kind, tenant_id, recipient, subject, sender, relay, error)
VALUES (sqlc.arg('status'), sqlc.arg('kind'), sqlc.narg('tenant_id'), sqlc.arg('recipient'), sqlc.arg('subject'),
        sqlc.arg('sender'), sqlc.arg('relay'), sqlc.arg('error'));

-- name: ListEmailLog :many
-- The emails, newest first, one page before `before` (NULL: the newest).
-- q: any part of the address, the subject or the error; status, kind and the
-- council exactly; since and until bound the time.
SELECT
    e.id,
    e.at,
    e.status,
    e.kind,
    e.tenant_id,
    COALESCE(t.slug, '')::TEXT AS tenant_slug,
    COALESCE(t.name, '')::TEXT AS tenant_name,
    e.recipient,
    e.subject,
    e.sender,
    e.relay,
    e.error
FROM
    email_log e
    LEFT JOIN tenant t ON t.id = e.tenant_id
WHERE
    (sqlc.narg('q')::TEXT IS NULL
        OR e.recipient ILIKE '%' || sqlc.narg('q') || '%'
        OR e.subject ILIKE '%' || sqlc.narg('q') || '%'
        OR e.error ILIKE '%' || sqlc.narg('q') || '%')
    AND (sqlc.narg('status')::TEXT IS NULL OR e.status = sqlc.narg('status'))
    AND (sqlc.narg('kind')::TEXT IS NULL OR e.kind = sqlc.narg('kind'))
    AND (sqlc.narg('tenant_id')::UUID IS NULL OR e.tenant_id = sqlc.narg('tenant_id'))
    AND (sqlc.narg('since')::TIMESTAMPTZ IS NULL OR e.at >= sqlc.narg('since'))
    AND (sqlc.narg('until')::TIMESTAMPTZ IS NULL OR e.at < sqlc.narg('until'))
    AND (sqlc.narg('before')::TIMESTAMPTZ IS NULL OR e.at < sqlc.narg('before'))
ORDER BY
    e.at DESC
LIMIT sqlc.arg('max_rows');

-- name: ListEmailLogKinds :many
-- The kinds there are in the log, for the filter.
SELECT DISTINCT kind FROM email_log WHERE kind <> '' ORDER BY kind;

-- name: CountEmailLogSince :one
-- How many went and how many failed since a moment (the page's summary).
SELECT
    COUNT(*) FILTER (WHERE status = 'sent')::BIGINT AS sent,
    COUNT(*) FILTER (WHERE status = 'failed')::BIGINT AS failed
FROM email_log
WHERE at >= sqlc.arg('since');

-- name: PurgeEmailLog :one
-- Emails older than the retention period go; returns how many.
SELECT purge_email_log(make_interval(days => sqlc.arg('days')::INT))::BIGINT AS purged;
