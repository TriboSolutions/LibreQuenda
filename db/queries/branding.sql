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
