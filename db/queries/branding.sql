-- ============================================================
--  BRANDING: the installation's own name and logo (a single row)
-- ============================================================

-- name: GetAppSettings :one
SELECT
    app_name,
    logo_ref
FROM
    app_settings
WHERE
    id;

-- name: SaveAppSettings :exec
INSERT INTO app_settings (id, app_name, logo_ref, updated_at)
VALUES (TRUE, sqlc.narg('app_name'), sqlc.narg('logo_ref'), NOW())
ON CONFLICT (id) DO UPDATE SET
    app_name = EXCLUDED.app_name,
    logo_ref = EXCLUDED.logo_ref,
    updated_at = NOW();

-- name: GetBookingPolicy :one
-- Platform-wide booking rules: one council at a time per person (on without a row).
SELECT
    COALESCE((SELECT a.one_council_per_account FROM app_settings a WHERE a.id), TRUE)::BOOLEAN AS one_council_per_account;

-- name: SaveBookingPolicy :exec
INSERT INTO app_settings (id, one_council_per_account, updated_at)
VALUES (TRUE, sqlc.arg('one_council_per_account'), NOW())
ON CONFLICT (id) DO UPDATE SET
    one_council_per_account = EXCLUDED.one_council_per_account,
    updated_at = NOW();
