-- Booking history for the indicators: HISTORY_DAYS days before the day the
-- database is created, cut from each facility's real opening hours (the weekday
-- of each day is only known then). Seeded random numbers: the same pattern every time.
--   more demand in the evening and at weekends, and growing over the period;
--   ~7 % cancelled by the resident; a closure cancels what it catches (council);
--   ~88 % checked in where there is a QR reader; some bookings take invited people.
SELECT setseed(0.42);

CREATE TEMP TABLE seed_history AS
WITH people AS (
    SELECT array_agg(id ORDER BY id) AS ids, count(*)::INT AS n FROM app_user WHERE idp_subject LIKE 'seed-%'
),
turns AS (
    SELECT
        f.id AS facility_id,
        f.slug,
        f.capacity,
        day::DATE AS day,
        local_start,
        local_start + make_interval(mins => f.slot_minutes) AS local_end
    FROM
        generate_series({LOCAL_DAY} - {HISTORY_DAYS}, {LOCAL_DAY} - 1, INTERVAL '1 day') AS day
        JOIN opening_hours oh ON oh.weekday = EXTRACT(ISODOW FROM day)
        JOIN facility f ON f.id = oh.facility_id
        CROSS JOIN LATERAL generate_series(
            day + oh.opens_at,
            day + oh.closes_at - make_interval(mins => f.slot_minutes),
            make_interval(mins => f.slot_minutes)
        ) AS local_start
    ORDER BY
        f.slug, local_start
)
SELECT
    uuidv7() AS booking_id,
    t.*,
    (t.local_start AT TIME ZONE '{TIMEZONE}') AS starts_at,
    (t.local_end AT TIME ZONE '{TIMEZONE}') AS ends_at,
    random() AS demand_roll,
    random() AS cancel_roll,
    random() AS checkin_roll,
    random() AS guest_roll,
    p.ids AS people,
    p.n AS people_count,
    floor(random() * p.n)::INT AS owner_index
FROM
    turns t, people p;

-- how likely each turn is to be booked
ALTER TABLE seed_history ADD COLUMN demand FLOAT8;
UPDATE seed_history SET demand =
    (0.30 + (abs(hashtext(slug)) % 25) / 100.0)                                       -- some facilities are more popular
    * CASE
        WHEN EXTRACT(HOUR FROM local_start) BETWEEN 17 AND 20 THEN 1.7
        WHEN EXTRACT(HOUR FROM local_start) >= 21 THEN 0.9
        WHEN EXTRACT(HOUR FROM local_start) < 12 THEN 0.6
        ELSE 0.45
      END
    * CASE WHEN EXTRACT(ISODOW FROM day) >= 6 THEN 1.3 ELSE 1.0 END
    * (0.65 + 0.35 * (1 - ({LOCAL_DAY} - day)::FLOAT8 / {HISTORY_DAYS}));                -- more residents use it every week

INSERT INTO booking (id, facility_id, user_id, starts_at, ends_at, status, cancelled_by, cancelled_at, created_at, rules_accepted_at, checked_in_at, checked_in_device_id)
SELECT
    h.booking_id,
    h.facility_id,
    h.people[1 + h.owner_index],
    h.starts_at,
    h.ends_at,
    CASE WHEN closed OR h.cancel_roll < 0.07 THEN 'cancelled' ELSE 'confirmed' END::booking_status,
    CASE WHEN closed THEN 'council' WHEN h.cancel_roll < 0.07 THEN 'user' END::booking_canceller,
    CASE WHEN closed OR h.cancel_roll < 0.07 THEN h.starts_at - INTERVAL '26 hours' END,
    h.starts_at - make_interval(hours => 30 + h.owner_index * 7),
    h.starts_at - make_interval(hours => 30 + h.owner_index * 7),
    CASE WHEN NOT closed AND h.cancel_roll >= 0.07 AND reader.id IS NOT NULL AND h.checkin_roll < 0.88
         THEN h.starts_at - make_interval(mins => (h.checkin_roll * 10)::INT) END,
    CASE WHEN NOT closed AND h.cancel_roll >= 0.07 AND reader.id IS NOT NULL AND h.checkin_roll < 0.88
         THEN reader.id END
FROM
    seed_history h
    LEFT JOIN LATERAL (SELECT d.id FROM checkin_device d WHERE d.facility_id = h.facility_id LIMIT 1) reader ON TRUE
    CROSS JOIN LATERAL (
        SELECT EXISTS (
            SELECT 1 FROM closure c
            WHERE c.facility_id = h.facility_id AND c.starts_at < h.ends_at AND c.ends_at > h.starts_at
        ) AS closed
    ) closure_hit
WHERE
    h.demand_roll < LEAST(h.demand, 0.95)
    -- turns somebody already booked stay as they are
    AND NOT EXISTS (
        SELECT 1 FROM booking x
        WHERE x.facility_id = h.facility_id AND x.status = 'confirmed' AND x.starts_at < h.ends_at AND x.ends_at > h.starts_at
    );

-- invited people on some of the confirmed bookings of shared facilities
INSERT INTO booking_participant (booking_id, user_id, joined_at)
SELECT
    b.id,
    h.people[1 + (h.owner_index + 1 + (h.guest_roll * 1000)::INT % (h.people_count - 1)) % h.people_count],
    b.created_at + INTERVAL '2 hours'
FROM
    booking b
    JOIN seed_history h ON h.booking_id = b.id
WHERE
    b.status = 'confirmed'
    AND h.capacity > 1
    AND h.guest_roll < 0.4;

DROP TABLE seed_history;
