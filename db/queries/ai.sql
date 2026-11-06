-- name: GetAISettings :one
-- The analytics assistant's language model; no row until a superadministrator sets it.
SELECT enabled, provider, base_url, model, api_key, updated_at FROM ai_settings WHERE id;

-- name: SaveAISettings :exec
-- api_key NULL keeps the one saved (the form never shows it back).
INSERT INTO ai_settings (id, enabled, provider, base_url, model, api_key, updated_at)
VALUES (TRUE, sqlc.arg('enabled'), sqlc.arg('provider'), sqlc.arg('base_url'), sqlc.arg('model'), sqlc.narg('api_key'), NOW())
ON CONFLICT (id) DO UPDATE SET
    enabled    = EXCLUDED.enabled,
    provider   = EXCLUDED.provider,
    base_url   = EXCLUDED.base_url,
    model      = EXCLUDED.model,
    api_key    = COALESCE(EXCLUDED.api_key, ai_settings.api_key),
    updated_at = NOW();
