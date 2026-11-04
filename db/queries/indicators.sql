-- Council indicators ("indicadores básicos"): live aggregates over bookings.
-- A booking belongs to the period its start falls in: range_start <= starts_at < range_end.
-- facility_id NULL = every facility of the council.

-- name: IndicatorsByFacility :many
-- One row per facility, also the ones with no bookings in the period.
-- finished / attended: past confirmed bookings that began while the facility
-- took check-ins (a QR reader switched on, the code, the QR on the door or
-- staff; see facility_checkin_period), and those checked in. Outside those
-- stretches nobody could check in, so those bookings say nothing about no-shows.
-- takes_checkin: it took check-ins at some point of the period.
-- ratings: the residents' stars for the facility after the bookings of the
-- period (a rating belongs to the period its booking began in, as the booking),
-- how many, and how many of each from 1 to 5.
SELECT
    f.id,
    f.name,
    f.slot_minutes,
    f.is_active,
    EXISTS (
        SELECT 1 FROM facility_checkin_period p
        WHERE p.facility_id = f.id AND p.starts_at < sqlc.arg('range_end')::TIMESTAMPTZ AND (p.ends_at IS NULL OR p.ends_at > sqlc.arg('range_start')::TIMESTAMPTZ)
    )::BOOLEAN AS takes_checkin,
    COUNT(b.id) AS bookings,
    COUNT(b.id) FILTER (WHERE b.status = 'confirmed') AS confirmed,
    COUNT(b.id) FILTER (WHERE b.cancelled_by = 'user') AS cancelled_by_user,
    COUNT(b.id) FILTER (WHERE b.cancelled_by = 'council') AS cancelled_by_council,
    COALESCE(SUM(EXTRACT(EPOCH FROM b.ends_at - b.starts_at) / 60) FILTER (WHERE b.status = 'confirmed'), 0)::BIGINT AS booked_minutes,
    -- attendance: events are left out, nobody checks in to them
    COUNT(b.id) FILTER (WHERE b.status = 'confirmed' AND b.kind <> 'event' AND b.ends_at <= NOW() AND facility_took_checkin(f.id, b.starts_at)) AS finished,
    COUNT(b.id) FILTER (WHERE b.status = 'confirmed' AND b.kind <> 'event' AND b.ends_at <= NOW() AND facility_took_checkin(f.id, b.starts_at) AND b.checked_in_at IS NOT NULL) AS attended,
    r.ratings::BIGINT AS ratings,
    r.stars_1::BIGINT AS stars_1,
    r.stars_2::BIGINT AS stars_2,
    r.stars_3::BIGINT AS stars_3,
    r.stars_4::BIGINT AS stars_4,
    r.stars_5::BIGINT AS stars_5
FROM
    facility f
    LEFT JOIN booking b ON b.facility_id = f.id
        AND b.starts_at >= sqlc.arg('range_start')
        AND b.starts_at < sqlc.arg('range_end')
    -- aggregated apart: joined with the bookings it would count each rating once per booking
    CROSS JOIN LATERAL (
        SELECT
            COUNT(*) AS ratings,
            COUNT(*) FILTER (WHERE fr.stars = 1) AS stars_1,
            COUNT(*) FILTER (WHERE fr.stars = 2) AS stars_2,
            COUNT(*) FILTER (WHERE fr.stars = 3) AS stars_3,
            COUNT(*) FILTER (WHERE fr.stars = 4) AS stars_4,
            COUNT(*) FILTER (WHERE fr.stars = 5) AS stars_5
        FROM facility_rating fr
        JOIN booking rb ON rb.id = fr.booking_id
        WHERE fr.facility_id = f.id
          AND rb.starts_at >= sqlc.arg('range_start')
          AND rb.starts_at < sqlc.arg('range_end')
    ) r
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
    AND (sqlc.narg('facility_id')::UUID IS NULL OR f.id = sqlc.narg('facility_id'))
GROUP BY
    f.id, r.ratings, r.stars_1, r.stars_2, r.stars_3, r.stars_4, r.stars_5
ORDER BY
    f.name ASC;

-- name: IndicatorsSeries :many
-- Bookings per period; bucket is 'day', 'week' or 'month', in the council's time zone.
SELECT
    date_trunc(sqlc.arg('bucket')::TEXT, b.starts_at AT TIME ZONE sqlc.arg('timezone')::TEXT)::DATE AS period,
    COUNT(*) FILTER (WHERE b.status = 'confirmed') AS confirmed,
    COUNT(*) FILTER (WHERE b.status = 'cancelled') AS cancelled
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
    AND (sqlc.narg('facility_id')::UUID IS NULL OR f.id = sqlc.narg('facility_id'))
    AND b.starts_at >= sqlc.arg('range_start')
    AND b.starts_at < sqlc.arg('range_end')
GROUP BY
    1
ORDER BY
    1;

-- name: IndicatorsRatingSeries :many
-- The facilities' stars per period, by when the rated booking began, in the council's time zone.
SELECT
    date_trunc(sqlc.arg('bucket')::TEXT, b.starts_at AT TIME ZONE sqlc.arg('timezone')::TEXT)::DATE AS period,
    COUNT(*) AS ratings,
    COALESCE(SUM(fr.stars), 0)::BIGINT AS rating_stars
FROM
    facility_rating fr
    JOIN booking b ON b.id = fr.booking_id
    JOIN facility f ON f.id = fr.facility_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
    AND (sqlc.narg('facility_id')::UUID IS NULL OR f.id = sqlc.narg('facility_id'))
    AND b.starts_at >= sqlc.arg('range_start')
    AND b.starts_at < sqlc.arg('range_end')
GROUP BY
    1
ORDER BY
    1;

-- name: IndicatorsBusyHours :many
-- Confirmed bookings by local weekday (ISO: 1 = Monday) and starting hour.
SELECT
    EXTRACT(ISODOW FROM b.starts_at AT TIME ZONE sqlc.arg('timezone')::TEXT)::INT AS weekday,
    EXTRACT(HOUR FROM b.starts_at AT TIME ZONE sqlc.arg('timezone')::TEXT)::INT AS hour,
    COUNT(*) AS bookings
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
    AND (sqlc.narg('facility_id')::UUID IS NULL OR f.id = sqlc.narg('facility_id'))
    AND b.status = 'confirmed'
    AND b.starts_at >= sqlc.arg('range_start')
    AND b.starts_at < sqlc.arg('range_end')
GROUP BY
    1, 2;

-- name: IndicatorsByWeekday :many
-- Confirmed booked time per local weekday (ISO: 1 = Monday), for occupancy by weekday.
SELECT
    EXTRACT(ISODOW FROM b.starts_at AT TIME ZONE sqlc.arg('timezone')::TEXT)::INT AS weekday,
    COALESCE(SUM(EXTRACT(EPOCH FROM b.ends_at - b.starts_at) / 60), 0)::BIGINT AS booked_minutes
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
    AND (sqlc.narg('facility_id')::UUID IS NULL OR f.id = sqlc.narg('facility_id'))
    AND b.status = 'confirmed'
    AND b.starts_at >= sqlc.arg('range_start')
    AND b.starts_at < sqlc.arg('range_end')
GROUP BY
    1;

-- name: IndicatorsHabits :one
-- How people book: the average group (owner + invited, guests too) and the
-- median time between booking and playing, over the confirmed bookings.
SELECT
    COUNT(*) AS bookings,
    COALESCE(AVG(1 + invited.people), 0)::FLOAT8 AS avg_party_size,
    COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM b.starts_at - b.created_at) / 3600), 0)::FLOAT8 AS median_lead_hours
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    CROSS JOIN LATERAL (SELECT COUNT(*) AS people FROM booking_participant bp WHERE bp.booking_id = b.id) invited
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
    AND (sqlc.narg('facility_id')::UUID IS NULL OR f.id = sqlc.narg('facility_id'))
    AND b.status = 'confirmed'
    AND b.starts_at >= sqlc.arg('range_start')
    AND b.starts_at < sqlc.arg('range_end');

-- name: IndicatorsActiveUsers :one
-- Distinct people with a confirmed booking in the period: owners and those who
-- joined through an invite. Guests have no account and are not counted.
WITH bookings AS (
    SELECT
        b.id,
        b.user_id
    FROM
        booking b
        JOIN facility f ON f.id = b.facility_id
    WHERE
        f.tenant_id = sqlc.arg('tenant_id')
        AND (sqlc.narg('facility_id')::UUID IS NULL OR f.id = sqlc.narg('facility_id'))
        AND b.status = 'confirmed'
        AND b.starts_at >= sqlc.arg('range_start')
        AND b.starts_at < sqlc.arg('range_end')
)
SELECT
    COUNT(DISTINCT people.user_id) AS active_users
FROM (
    SELECT user_id FROM bookings
    UNION ALL
    SELECT bp.user_id FROM booking_participant bp JOIN bookings ON bookings.id = bp.booking_id
) people
WHERE
    people.user_id IS NOT NULL;

-- name: ListOpeningHoursByTenant :many
-- The whole weekly schedule, to work out each facility's open hours in a period.
SELECT
    oh.facility_id,
    oh.weekday,
    oh.opens_at,
    oh.closes_at
FROM
    opening_hours oh
    JOIN facility f ON f.id = oh.facility_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
ORDER BY
    oh.facility_id,
    oh.weekday,
    oh.opens_at ASC;

-- name: IndicatorsDaily :many
-- Per facility and local day, for the anonymous daily export.
SELECT
    b.facility_id,
    (b.starts_at AT TIME ZONE sqlc.arg('timezone')::TEXT)::DATE AS day,
    COUNT(*) AS bookings,
    COUNT(*) FILTER (WHERE b.status = 'confirmed') AS confirmed,
    COUNT(*) FILTER (WHERE b.cancelled_by = 'user') AS cancelled_by_user,
    COUNT(*) FILTER (WHERE b.cancelled_by = 'council') AS cancelled_by_council,
    COALESCE(SUM(EXTRACT(EPOCH FROM b.ends_at - b.starts_at) / 60) FILTER (WHERE b.status = 'confirmed'), 0)::BIGINT AS booked_minutes,
    COUNT(*) FILTER (WHERE b.status = 'confirmed' AND b.checked_in_at IS NOT NULL) AS checked_in,
    COALESCE(SUM(invited.people) FILTER (WHERE b.status = 'confirmed'), 0)::BIGINT AS invited
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    CROSS JOIN LATERAL (SELECT COUNT(*) AS people FROM booking_participant bp WHERE bp.booking_id = b.id) invited
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
    AND (sqlc.narg('facility_id')::UUID IS NULL OR f.id = sqlc.narg('facility_id'))
    AND b.starts_at >= sqlc.arg('range_start')
    AND b.starts_at < sqlc.arg('range_end')
GROUP BY
    1, 2;

-- name: IndicatorsBusyHoursByFacility :many
-- The heatmap per facility, for the anonymous busy-hours export.
SELECT
    f.name AS facility_name,
    EXTRACT(ISODOW FROM b.starts_at AT TIME ZONE sqlc.arg('timezone')::TEXT)::INT AS weekday,
    EXTRACT(HOUR FROM b.starts_at AT TIME ZONE sqlc.arg('timezone')::TEXT)::INT AS hour,
    COUNT(*) AS bookings
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
    AND (sqlc.narg('facility_id')::UUID IS NULL OR f.id = sqlc.narg('facility_id'))
    AND b.status = 'confirmed'
    AND b.starts_at >= sqlc.arg('range_start')
    AND b.starts_at < sqlc.arg('range_end')
GROUP BY
    1, 2, 3
ORDER BY
    1, 2, 3;

-- name: IndicatorsBookingsPerPerson :many
-- How many people had 1, 2, 3… confirmed bookings in the period, as owner or
-- invited: only the distribution leaves the database, never who.
WITH bookings AS (
    SELECT
        b.id,
        b.user_id
    FROM
        booking b
        JOIN facility f ON f.id = b.facility_id
    WHERE
        f.tenant_id = sqlc.arg('tenant_id')
        AND (sqlc.narg('facility_id')::UUID IS NULL OR f.id = sqlc.narg('facility_id'))
        AND b.status = 'confirmed'
        AND b.starts_at >= sqlc.arg('range_start')
        AND b.starts_at < sqlc.arg('range_end')
),
per_person AS (
    SELECT
        people.user_id,
        COUNT(*) AS bookings
    FROM (
        SELECT user_id FROM bookings
        UNION ALL
        SELECT bp.user_id FROM booking_participant bp JOIN bookings ON bookings.id = bp.booking_id
    ) people
    WHERE
        people.user_id IS NOT NULL
    GROUP BY
        people.user_id
)
SELECT
    bookings::INT AS bookings,
    COUNT(*) AS people
FROM
    per_person
GROUP BY
    1
ORDER BY
    1;

-- name: GetTenantTimezone :one
SELECT
    timezone
FROM
    tenant
WHERE
    id = sqlc.arg('id');
