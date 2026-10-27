-- name: SetTenant :exec
-- Names the council for the rest of the transaction (row-level security,
-- db/init/05_row_security.sql). Only inside a transaction: it resets on commit.
SELECT set_config('app.tenant_id', sqlc.arg('tenant_id')::TEXT, TRUE);
