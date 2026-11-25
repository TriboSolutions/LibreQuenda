-- ============================================================
--  OPEN DATA (datos abiertos): the council's public DCAT-AP-ES catalog
--  (GET /tenants/{slug}/open-data/...) and its settings
--  (GET/PUT /tenants/{slug}/admin/open-data). Nothing personal: the
--  facilities as residents see them, and their use counted by month.
-- ============================================================

-- name: GetTenantOpenData :one
SELECT
    enabled,
    dir3_code,
    issued_at,
    updated_at
FROM
    tenant_open_data
WHERE
    tenant_id = sqlc.arg('tenant_id');

-- name: UpsertTenantOpenData :exec
-- issued_at: the first time it is turned on, kept afterwards.
INSERT INTO tenant_open_data (tenant_id, enabled, dir3_code, issued_at)
VALUES (
    sqlc.arg('tenant_id'),
    sqlc.arg('enabled'),
    sqlc.arg('dir3_code'),
    CASE WHEN sqlc.arg('enabled')::BOOLEAN THEN NOW() END
)
ON CONFLICT (tenant_id) DO UPDATE
SET
    enabled = EXCLUDED.enabled,
    dir3_code = EXCLUDED.dir3_code,
    issued_at = COALESCE(tenant_open_data.issued_at, EXCLUDED.issued_at),
    updated_at = NOW();

-- name: GetPublishedCatalog :one
-- An active council that publishes its open data.
SELECT
    t.id,
    t.slug,
    t.name,
    t.timezone,
    t.contact_email,
    t.contact_phone,
    t.municipality_code,
    o.dir3_code,
    o.issued_at,
    -- the latest change to what the catalog lists
    GREATEST(
        o.updated_at,
        t.updated_at,
        (SELECT MAX(f.updated_at) FROM facility f WHERE f.tenant_id = t.id)
    )::TIMESTAMPTZ AS modified_at
FROM
    tenant t
    JOIN tenant_open_data o ON o.tenant_id = t.id
WHERE
    t.slug = sqlc.arg('slug')
    AND t.is_active = TRUE
    AND o.enabled = TRUE;

-- name: OpenDataFacilities :many
-- The active facilities, with what their public page shows.
SELECT
    f.id,
    f.slug,
    f.name,
    f.description,
    f.address,
    f.latitude,
    f.longitude,
    f.capacity,
    f.accessibility_info,
    f.slot_minutes,
    f.residents_only,
    f.min_age,
    ARRAY(
        SELECT tg.name FROM facility_tag ft JOIN tag tg ON tg.id = ft.tag_id
        WHERE ft.facility_id = f.id ORDER BY tg.name
    )::TEXT[] AS tags,
    ARRAY(
        SELECT oh.weekday || ' ' || to_char(oh.opens_at, 'HH24:MI') || '-' || to_char(oh.closes_at, 'HH24:MI')
        FROM opening_hours oh
        WHERE oh.facility_id = f.id ORDER BY oh.weekday, oh.opens_at
    )::TEXT[] AS opening_hours
FROM
    facility f
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
    AND f.is_active = TRUE
ORDER BY
    f.name ASC;

-- name: OpenDataMonthlyUse :many
-- Each active facility's use by month (in the council's time zone), from
-- range_start (the first day of a month) up to range_end (the first of the
-- current month): only finished months, whose figures no longer change.
-- Counts only: no person, no booking.
SELECT
    f.id AS facility_id,
    f.name AS facility_name,
    to_char(m.month, 'YYYY-MM')::TEXT AS month,
    COUNT(b.id) FILTER (WHERE b.status = 'confirmed' AND b.kind <> 'event') AS bookings,
    COUNT(b.id) FILTER (WHERE b.status = 'confirmed' AND b.kind = 'event') AS events,
    COUNT(b.id) FILTER (WHERE b.status = 'cancelled') AS cancelled,
    (COALESCE(SUM(EXTRACT(EPOCH FROM b.ends_at - b.starts_at)) FILTER (WHERE b.status = 'confirmed'), 0) / 3600)::NUMERIC(10, 1)::TEXT AS booked_hours
FROM
    facility f
    CROSS JOIN generate_series(sqlc.arg('range_start')::DATE, sqlc.arg('range_end')::DATE - 1, INTERVAL '1 month') AS m(month)
    LEFT JOIN booking b ON b.facility_id = f.id
        AND b.starts_at >= (m.month::TIMESTAMP AT TIME ZONE sqlc.arg('timezone')::TEXT)
        AND b.starts_at < ((m.month + INTERVAL '1 month')::TIMESTAMP AT TIME ZONE sqlc.arg('timezone')::TEXT)
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
    AND f.is_active = TRUE
GROUP BY
    f.id,
    f.name,
    m.month
ORDER BY
    m.month ASC,
    f.name ASC;
