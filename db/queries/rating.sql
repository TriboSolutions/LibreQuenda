-- name: CreateRatingRequests :execrows
-- Once a confirmed booking ends: one notice to each person in it with an account
-- (the owner and who joined), asking for the stars. In the app only: emailed_at
-- is set so the email queue skips it. A booking nobody checked in to, where
-- someone could have, is a no-show and asks nothing; neither do bookings that
-- ended more than a day ago (the job catches up after a restart, no further).
INSERT INTO notification (
    user_id,
    booking_id,
    type,
    payload,
    emailed_at
)
SELECT
    x.user_id,
    b.id,
    'rating_requested',
    jsonb_build_object(
        'facility_id',   f.id,
        'facility_name', f.name,
        'facility_slug', f.slug,
        'tenant_slug',   t.slug,
        'starts_at',     b.starts_at,
        'ends_at',       b.ends_at,
        'is_public',     b.is_public
    ),
    NOW()
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
    CROSS JOIN LATERAL (
        SELECT b.user_id
        UNION
        SELECT p.user_id FROM booking_participant p WHERE p.booking_id = b.id
    ) x
WHERE
    b.status = 'confirmed'
    AND b.ends_at <= NOW()
    AND b.ends_at > NOW() - INTERVAL '1 day'
    AND x.user_id IS NOT NULL
    AND NOT (b.checked_in_at IS NULL AND facility_took_checkin(f.id, b.starts_at))
    AND NOT EXISTS (
        SELECT 1
        FROM notification n
        WHERE n.booking_id = b.id
          AND n.type = 'rating_requested'
          AND n.user_id = x.user_id
    );

-- name: GetBookingToRate :one
-- A booking the user was in (made or joined), with what rating it needs.
SELECT
    b.id,
    b.starts_at,
    b.ends_at,
    b.status,
    b.is_public,
    b.checked_in_at,
    facility_took_checkin(f.id, b.starts_at)::BOOLEAN AS checkin_at_start,
    f.id   AS facility_id,
    f.name AS facility_name,
    f.slug AS facility_slug,
    t.slug AS tenant_slug,
    t.name AS tenant_name
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
WHERE
    b.id = sqlc.arg('booking_id')
    AND (
        b.user_id = sqlc.arg('user_id')
        OR EXISTS (
            SELECT 1 FROM booking_participant p
            WHERE p.booking_id = b.id AND p.user_id = sqlc.arg('user_id')
        )
    );

-- name: ListPeopleToRate :many
-- The others in a booking who have an account, the owner first, with the stars
-- this user gave each (0: none yet). Guests have no account and are left out.
SELECT
    x.user_id,
    COALESCE(u.full_name, '')::TEXT AS name,
    x.is_owner::BOOLEAN AS is_owner,
    COALESCE(r.stars, 0)::SMALLINT AS stars
FROM
    (
        SELECT b.user_id, TRUE AS is_owner, b.created_at AS since
        FROM booking b
        WHERE b.id = sqlc.arg('booking_id') AND b.user_id IS NOT NULL
        UNION ALL
        SELECT p.user_id, FALSE, p.joined_at
        FROM booking_participant p
        WHERE p.booking_id = sqlc.arg('booking_id') AND p.user_id IS NOT NULL
    ) x
    JOIN app_user u ON u.id = x.user_id
    LEFT JOIN user_rating r
        ON r.booking_id = sqlc.arg('booking_id')
        AND r.rater_id = sqlc.arg('rater_id')
        AND r.rated_id = x.user_id
WHERE
    x.user_id <> sqlc.arg('rater_id')
ORDER BY
    x.is_owner DESC,
    x.since ASC;

-- name: GetMyFacilityRating :one
SELECT
    stars
FROM
    facility_rating
WHERE
    booking_id = sqlc.arg('booking_id')
    AND user_id = sqlc.arg('user_id');

-- name: UpsertFacilityRating :exec
INSERT INTO facility_rating (
    facility_id,
    booking_id,
    user_id,
    stars
) VALUES (
    sqlc.arg('facility_id'),
    sqlc.arg('booking_id'),
    sqlc.arg('user_id'),
    sqlc.arg('stars')
)
ON CONFLICT (booking_id, user_id) DO UPDATE
SET
    stars      = EXCLUDED.stars,
    updated_at = NOW();

-- name: UpsertUserRating :exec
INSERT INTO user_rating (
    booking_id,
    rater_id,
    rated_id,
    stars
) VALUES (
    sqlc.arg('booking_id'),
    sqlc.arg('rater_id'),
    sqlc.arg('rated_id'),
    sqlc.arg('stars')
)
ON CONFLICT (booking_id, rater_id, rated_id) DO UPDATE
SET
    stars      = EXCLUDED.stars,
    updated_at = NOW();

-- name: GetFacilityRatingSummary :one
-- The facility page: the average and how many, never who.
SELECT
    COALESCE(AVG(stars), 0)::FLOAT8 AS average,
    COUNT(*) AS count
FROM
    facility_rating
WHERE
    facility_id = sqlc.arg('facility_id');

-- name: GetUserRatingSummary :one
-- What others gave a person after public bookings: the average and how many, never who.
SELECT
    COALESCE(AVG(stars), 0)::FLOAT8 AS average,
    COUNT(*) AS count
FROM
    user_rating
WHERE
    rated_id = sqlc.arg('user_id');

-- name: ExportFacilityRatingsByUser :many
-- The data export: the stars the person gave facilities.
SELECT
    f.name AS facility_name,
    r.booking_id,
    r.stars,
    r.updated_at
FROM
    facility_rating r
    JOIN facility f ON f.id = r.facility_id
WHERE
    r.user_id = sqlc.arg('user_id')
ORDER BY
    r.updated_at DESC;

-- name: ExportPeopleRatingsByUser :many
-- The data export: the stars the person gave others after public bookings;
-- like the rest of the export, without the other people's details.
SELECT
    r.booking_id,
    r.stars,
    r.updated_at
FROM
    user_rating r
WHERE
    r.rater_id = sqlc.arg('user_id')
ORDER BY
    r.updated_at DESC;
