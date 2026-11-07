-- name: CreateBooking :one
-- Overlaps are rejected by the exclusion constraint on booking.
INSERT INTO booking (
    facility_id,
    user_id,
    starts_at,
    ends_at,
    rules_accepted_at
) VALUES (
    sqlc.arg('facility_id'),
    sqlc.arg('user_id'),
    sqlc.arg('starts_at'),
    sqlc.arg('ends_at'),
    NOW()
)
RETURNING
    *;

-- name: CancelBooking :one
UPDATE booking
SET
    status       = 'cancelled',
    cancelled_at = NOW(),
    cancelled_by = 'user'
WHERE
    id = sqlc.arg('id')
    AND user_id = sqlc.arg('user_id')
    AND status = 'confirmed'
RETURNING
    *;

-- name: ChangeBookingTime :one
-- Moves the owner's booking to another turn of the same facility; overlaps are
-- rejected by the exclusion constraint on booking.
UPDATE booking
SET
    starts_at  = sqlc.arg('starts_at'),
    ends_at    = sqlc.arg('ends_at'),
    updated_at = NOW()
WHERE
    id = sqlc.arg('id')
    AND user_id = sqlc.arg('user_id')
    AND status = 'confirmed'
RETURNING
    *;

-- name: LockOwnedBooking :one
-- Inside the change transaction: joins to this booking wait (they lock it too),
-- so nobody joins while the people coming are checked against the new time.
SELECT
    b.id,
    b.starts_at,
    b.status,
    f.min_cancel_hours
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
WHERE
    b.id = sqlc.arg('id')
    AND b.user_id = sqlc.arg('user_id')
FOR UPDATE OF b;

-- name: ListBookingsByUser :many
-- checkin_at_start: the facility took check-ins when the turn began, so a
-- missing check-in means the person did not show up.
SELECT
    b.id,
    b.starts_at,
    b.ends_at,
    b.status,
    b.checkin_token,
    b.checked_in_at,
    (b.kind = 'public')::BOOLEAN AS is_public,
    b.public_description,
    facility_took_checkin(f.id, b.starts_at)::BOOLEAN AS checkin_at_start,
    -- rated: this user saved their stars for it already, which are final
    (EXISTS (SELECT 1 FROM facility_rating fr WHERE fr.booking_id = b.id AND fr.user_id = sqlc.arg('user_id'))
     OR EXISTS (SELECT 1 FROM user_rating ur WHERE ur.booking_id = b.id AND ur.rater_id = sqlc.arg('user_id')))::BOOLEAN AS rated,
    -- the ways in the facility offers now, for the check-in button
    (EXISTS (SELECT 1 FROM checkin_device d WHERE d.facility_id = f.id) AND COALESCE(c.qr_reader, TRUE))::BOOLEAN AS has_reader,
    (COALESCE(c.printed_qr, FALSE) AND f.latitude IS NOT NULL)::BOOLEAN AS printed_qr,
    (COALESCE(c.entry_code, FALSE) AND f.latitude IS NOT NULL)::BOOLEAN AS entry_code,
    (COALESCE(c.nfc, FALSE) AND f.latitude IS NOT NULL)::BOOLEAN AS nfc,
    COALESCE(c.staff_confirm, FALSE)::BOOLEAN AS staff_confirm,
    COALESCE(c.staff_contacts, '[]'::JSONB) AS staff_contacts,
    COALESCE(c.radius_m, 100)::INT AS checkin_radius_m, -- how close the phone must be for the code or the door QR
    t.contact_phone AS tenant_phone, -- who to ask without staff contacts: the facility's, else the council's
    t.contact_email AS tenant_email,
    f.id   AS facility_id,
    f.name AS facility_name,
    f.slug AS facility_slug,
    f.min_cancel_hours,
    t.slug AS tenant_slug,
    t.name AS tenant_name,
    facility_sport(f.id)::TEXT AS facility_sport,
    sqlc.embed(f) -- what the booking details show: image, address, rules
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
    LEFT JOIN facility_checkin c ON c.facility_id = f.id
WHERE
    b.user_id = sqlc.arg('user_id')
    -- the upcoming ones (all of them) and the page of the history asked for (ListHistoryPageByUser)
    AND ((b.status = 'confirmed' AND b.ends_at > NOW()) OR b.id = ANY(sqlc.arg('history_ids')::UUID[]))
ORDER BY
    b.starts_at DESC;

-- name: ListHistoryPageByUser :many
-- A page of the person's history, most recent first: the bookings they made or
-- joined that ended or were cancelled, with how many there are in all.
WITH mine AS (
    SELECT b.id, b.starts_at
    FROM booking b
    WHERE b.user_id = sqlc.arg('user_id') AND (b.status <> 'confirmed' OR b.ends_at <= NOW())
    UNION
    SELECT b.id, b.starts_at
    FROM booking_participant p JOIN booking b ON b.id = p.booking_id
    WHERE p.user_id = sqlc.arg('user_id') AND (b.status <> 'confirmed' OR b.ends_at <= NOW())
)
SELECT
    id,
    COUNT(*) OVER ()::BIGINT AS total
FROM
    mine
ORDER BY
    starts_at DESC,
    id
LIMIT sqlc.arg('max_rows')
OFFSET sqlc.arg('skip');

-- name: ListConfirmedBookingsInRange :many
-- Used to compute availability.
SELECT
    starts_at,
    ends_at
FROM
    booking
WHERE
    facility_id = sqlc.arg('facility_id')
    AND status = 'confirmed'
    AND starts_at < sqlc.arg('range_end')
    AND ends_at   > sqlc.arg('range_start')
ORDER BY
    starts_at ASC;

-- name: CountActiveBookingsByUserFacility :one
SELECT
    COUNT(*)
FROM
    booking
WHERE
    user_id = sqlc.arg('user_id')
    AND facility_id = sqlc.arg('facility_id')
    AND status = 'confirmed'
    AND ends_at > NOW();

-- name: GetBookingForUser :one
SELECT
    b.id,
    b.starts_at,
    b.ends_at,
    b.status,
    b.facility_id,
    f.min_cancel_hours
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
WHERE
    b.id = sqlc.arg('id')
    AND b.user_id = sqlc.arg('user_id');

-- name: LockUserBookings :exec
-- Inside a transaction: one person's bookings and joins wait for each other,
-- so two requests at once cannot both pass the overlap check.
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg('user_id')::UUID::TEXT, 0));

-- name: LockFacilityBookings :exec
-- Inside a transaction, just before writing a booking's time: bookings of one
-- facility are written one after another. The exclusion constraint alone keeps
-- them from overlapping, but two at once can deadlock on it (40P01) instead of the
-- second hearing it is taken; queued, the second gets the conflict at once.
-- Always taken after the people's locks (LockUserBookings), never before.
SELECT pg_advisory_xact_lock(hashtextextended('facility:' || sqlc.arg('facility_id')::UUID::TEXT, 0));

-- name: UserHasOverlappingBooking :one
-- A person is in one place at a time: their own confirmed bookings and the
-- ones they joined through an invite, in any facility and any council.
-- except_booking_id: the booking being moved, which does not count against itself.
SELECT EXISTS (
    SELECT 1
    FROM booking b
    WHERE
        b.status = 'confirmed'
        AND b.id IS DISTINCT FROM sqlc.narg('except_booking_id')
        AND b.starts_at < sqlc.arg('ends_at')
        AND b.ends_at   > sqlc.arg('starts_at')
        AND (
            b.user_id = sqlc.arg('user_id')
            OR EXISTS (
                SELECT 1 FROM booking_participant p
                WHERE p.booking_id = b.id AND p.user_id = sqlc.arg('user_id')
            )
        )
);

-- name: LockParticipantBookings :exec
-- Inside the change transaction: the people who joined wait like the owner
-- (LockUserBookings), in a fixed order.
SELECT pg_advisory_xact_lock(hashtextextended(p.user_id::TEXT, 0))
FROM booking_participant p
WHERE
    p.booking_id = sqlc.arg('booking_id')
    AND p.user_id IS NOT NULL
ORDER BY
    p.user_id;

-- name: ParticipantsHaveOverlappingBooking :one
-- Someone who joined with an account is somewhere else in the new time,
-- as owner or invited (the same rule as UserHasOverlappingBooking).
SELECT EXISTS (
    SELECT 1
    FROM booking_participant p
    JOIN booking b ON b.id <> p.booking_id
    WHERE
        p.booking_id = sqlc.arg('booking_id')
        AND p.user_id IS NOT NULL
        AND b.status = 'confirmed'
        AND b.starts_at < sqlc.arg('ends_at')
        AND b.ends_at   > sqlc.arg('starts_at')
        AND (
            b.user_id = p.user_id
            OR EXISTS (
                SELECT 1 FROM booking_participant o
                WHERE o.booking_id = b.id AND o.user_id = p.user_id
            )
        )
);

-- name: ListCalendarBookings :many
-- The .ics feed: the user's own bookings and the ones they joined, from a month
-- back. Cancelled ones stay in, so calendar apps that already have them drop them.
SELECT
    b.id,
    b.starts_at,
    b.ends_at,
    b.status,
    b.updated_at,
    f.name AS facility_name,
    f.address,
    f.latitude,
    f.longitude,
    t.name AS tenant_name
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
WHERE
    (
        b.user_id = sqlc.arg('user_id')
        OR b.id IN (SELECT p.booking_id FROM booking_participant p WHERE p.user_id = sqlc.arg('user_id'))
    )
    AND b.ends_at > NOW() - INTERVAL '30 days'
ORDER BY
    b.starts_at ASC;

-- name: ListUserBookingPatterns :many
-- For the home page's recommendations: what the person booked or joined since
-- `since` (not cancelled), by facility, local weekday (ISO: 1 = Monday) and
-- starting time, in the facility's council time zone.
SELECT
    b.facility_id,
    EXTRACT(ISODOW FROM b.starts_at AT TIME ZONE t.timezone)::INT AS weekday,
    to_char(b.starts_at AT TIME ZONE t.timezone, 'HH24:MI')::TEXT AS start_time,
    COUNT(*) AS bookings
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
WHERE
    (b.user_id = sqlc.arg('user_id')
        OR EXISTS (SELECT 1 FROM booking_participant p WHERE p.booking_id = b.id AND p.user_id = sqlc.arg('user_id')))
    AND b.status = 'confirmed'
    AND b.starts_at >= sqlc.arg('since')
    AND f.is_active = TRUE
GROUP BY
    1, 2, 3;

-- name: CountUserBookingsInRange :one
-- The person's confirmed bookings here starting in [range_start, range_end), but
-- `exclude` (the one being changed): the booking window's limit per day.
SELECT
    COUNT(*)
FROM
    booking
WHERE
    user_id = sqlc.arg('user_id')
    AND facility_id = sqlc.arg('facility_id')
    AND status = 'confirmed'
    AND starts_at >= sqlc.arg('range_start')
    AND starts_at < sqlc.arg('range_end')
    AND (sqlc.narg('exclude')::UUID IS NULL OR id <> sqlc.narg('exclude'));

-- name: ListUserBookingsNear :many
-- The person's confirmed bookings here touching [range_start, range_end), but
-- `exclude`: to count the turns they would have in a row.
SELECT
    starts_at,
    ends_at
FROM
    booking
WHERE
    user_id = sqlc.arg('user_id')
    AND facility_id = sqlc.arg('facility_id')
    AND status = 'confirmed'
    AND ends_at >= sqlc.arg('range_start')
    AND starts_at <= sqlc.arg('range_end')
    AND (sqlc.narg('exclude')::UUID IS NULL OR id <> sqlc.narg('exclude'))
ORDER BY
    starts_at;

-- name: ListUserNoShows :many
-- The person's no-shows here since `since`, latest first: confirmed bookings
-- over without a check-in, while the facility took check-ins.
SELECT
    ends_at
FROM
    booking
WHERE
    user_id = sqlc.arg('user_id')
    AND facility_id = sqlc.arg('facility_id')
    AND status = 'confirmed'
    AND checked_in_at IS NULL
    AND ends_at <= NOW()
    AND ends_at >= sqlc.arg('since')
    AND facility_took_checkin(facility_id, starts_at)
ORDER BY
    ends_at DESC;

-- name: GetCouncilBookingPolicy :one
-- The rules beyond one facility: whether a person books in one council at a
-- time (platform setting, on without a row) and the council's cap on their
-- upcoming bookings across its facilities (0 = none).
SELECT
    COALESCE((SELECT a.one_council_per_account FROM app_settings a WHERE a.id), TRUE)::BOOLEAN AS one_council_per_account,
    COALESCE((SELECT r.max_active_bookings_council FROM tenant_booking_rules r WHERE r.tenant_id = sqlc.arg('tenant_id')), 0)::INT AS council_max_active;

-- name: GetUserBookingInOtherCouncil :one
-- The council of the person's next upcoming booking of their own outside
-- `tenant_id`, if any. Games they joined are someone else's booking: they do not count.
SELECT
    t.name
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
WHERE
    b.status = 'confirmed'
    AND b.ends_at > NOW()
    AND f.tenant_id <> sqlc.arg('tenant_id')
    AND b.user_id = sqlc.arg('user_id')
ORDER BY
    b.starts_at
LIMIT 1;

-- name: CountUserUpcomingInCouncil :one
-- The person's own upcoming confirmed bookings across the council's facilities.
SELECT
    COUNT(*)
FROM
    booking b
    JOIN facility f ON f.id = b.facility_id
WHERE
    b.user_id = sqlc.arg('user_id')
    AND b.status = 'confirmed'
    AND b.ends_at > NOW()
    AND f.tenant_id = sqlc.arg('tenant_id');
