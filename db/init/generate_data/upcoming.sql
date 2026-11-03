-- Upcoming bookings: today and the next {UPCOMING_DAYS} - 1 days, cut from each
-- facility's opening hours like the history. Busiest today, quieter further
-- ahead, as people book closer to the day. Each owner lives in the facility's
-- council, holds at most {MAX_UPCOMING_PER_PERSON} upcoming bookings (two per
-- facility, the default limit) and is never in two places at once. Some turns
-- were booked by staff for someone without an account; ~4 % were cancelled.
-- Turns earlier today already happened: most were checked in where there is a reader.
SELECT setseed(0.17);

-- who can book where: everyone with a home council except the local tester (they have their own)
CREATE TEMP TABLE seed_homes AS
SELECT h.tenant_id, array_agg(h.user_id ORDER BY h.user_id) AS ids, count(*)::INT AS n
FROM seed_user_home h JOIN app_user u ON u.id = h.user_id
WHERE u.idp_subject <> 'seed-tester'
GROUP BY h.tenant_id;

CREATE TEMP TABLE seed_upcoming AS
WITH turns AS (
    SELECT
        f.id AS facility_id,
        f.tenant_id,
        f.slug,
        f.slot_minutes,
        day::DATE AS day,
        local_start,
        local_start + make_interval(mins => f.slot_minutes) AS local_end
    FROM
        generate_series({LOCAL_DAY}, {LOCAL_DAY} + {UPCOMING_DAYS} - 1, INTERVAL '1 day') AS day
        JOIN opening_hours oh ON oh.weekday = EXTRACT(ISODOW FROM day)
        JOIN facility f ON f.id = oh.facility_id
        CROSS JOIN LATERAL generate_series(
            day + oh.opens_at,
            day + oh.closes_at - make_interval(mins => f.slot_minutes),
            make_interval(mins => f.slot_minutes)
        ) AS local_start
)
SELECT
    uuidv7() AS booking_id,
    t.*,
    (t.local_start AT TIME ZONE '{TIMEZONE}') AS starts_at,
    (t.local_end AT TIME ZONE '{TIMEZONE}') AS ends_at,
    random() AS demand_roll,
    random() AS staff_roll,
    random() AS cancel_roll,
    random() AS checkin_roll,
    random() AS pick_roll,
    random() AS lead_roll
FROM turns t;

ALTER TABLE seed_upcoming ADD COLUMN demand FLOAT8;
UPDATE seed_upcoming SET demand =
    (0.30 + (abs(hashtext(slug)) % 25) / 100.0)
    * CASE
        WHEN EXTRACT(HOUR FROM local_start) BETWEEN 17 AND 20 THEN 1.7
        WHEN EXTRACT(HOUR FROM local_start) >= 21 THEN 0.9
        WHEN EXTRACT(HOUR FROM local_start) < 12 THEN 0.6
        ELSE 0.45
      END
    * CASE WHEN EXTRACT(ISODOW FROM day) >= 6 THEN 1.3 ELSE 1.0 END
    * {UPCOMING_FILL} * (1.0 - 0.65 * (day - {LOCAL_DAY})::FLOAT8 / {UPCOMING_DAYS});

-- the turns that get booked: not closed, not taken (the tester's), by demand
DELETE FROM seed_upcoming u
WHERE u.demand_roll >= LEAST(u.demand, 0.95)
   OR EXISTS (SELECT 1 FROM closure c WHERE c.facility_id = u.facility_id AND c.starts_at < u.ends_at AND c.ends_at > u.starts_at)
   OR EXISTS (SELECT 1 FROM booking b WHERE b.facility_id = u.facility_id AND b.status = 'confirmed' AND b.starts_at < u.ends_at AND b.ends_at > u.starts_at);

-- owners, round robin over the council's residents in random order: nobody gets more than the cap
ALTER TABLE seed_upcoming ADD COLUMN owner_id UUID;
UPDATE seed_upcoming u SET owner_id = pick.owner_id
FROM (
    SELECT x.booking_id, h.ids[1 + (x.rn % h.n)] AS owner_id, x.rn < h.n * {MAX_UPCOMING_PER_PERSON} AS fits
    FROM (
        SELECT booking_id, tenant_id, row_number() OVER (PARTITION BY tenant_id ORDER BY pick_roll) - 1 AS rn
        FROM seed_upcoming WHERE staff_roll >= {STAFF_SHARE}
    ) x
    JOIN seed_homes h ON h.tenant_id = x.tenant_id
) pick
WHERE u.booking_id = pick.booking_id AND pick.fits;
-- more turns than people allow, and not for staff: stays free
DELETE FROM seed_upcoming WHERE owner_id IS NULL AND staff_roll >= {STAFF_SHARE};
-- one place at a time, and two upcoming per facility
DELETE FROM seed_upcoming a USING seed_upcoming b
WHERE a.owner_id = b.owner_id AND a.booking_id > b.booking_id AND a.starts_at < b.ends_at AND a.ends_at > b.starts_at;
DELETE FROM seed_upcoming WHERE booking_id IN (
    SELECT booking_id FROM (
        SELECT booking_id, row_number() OVER (PARTITION BY owner_id, facility_id ORDER BY starts_at) AS k
        FROM seed_upcoming WHERE owner_id IS NOT NULL
    ) x WHERE k > 2
);

INSERT INTO booking (id, facility_id, user_id, starts_at, ends_at, status, cancelled_by, cancelled_at, created_at, rules_accepted_at,
                     channel, holder_name, checked_in_at, checked_in_device_id, checked_in_method)
SELECT
    u.booking_id,
    u.facility_id,
    u.owner_id,
    u.starts_at,
    u.ends_at,
    CASE WHEN cancelled THEN 'cancelled' ELSE 'confirmed' END::booking_status,
    CASE WHEN cancelled THEN 'user' END::booking_canceller,
    CASE WHEN cancelled THEN LEAST(NOW(), made.at + INTERVAL '3 hours') END,
    made.at,
    made.at,
    CASE WHEN u.owner_id IS NULL THEN 'staff' ELSE 'online' END::booking_channel,
    CASE WHEN u.owner_id IS NULL THEN (ARRAY['Club deportivo do barrio', 'Asociación de veciños', 'Escola de baloncesto', 'Clube de natación Máster',
        'Carmen Losada', 'Manuel Rivas', 'Grupo de maiores do centro social', 'ANPA do colexio', 'Equipo de fútbol sala veterano', 'Xosé Antón Paz'])[1 + floor(u.pick_roll * 10)::INT] END,
    CASE WHEN went THEN u.starts_at - make_interval(mins => (u.checkin_roll * 10)::INT) END,
    CASE WHEN went THEN reader.id END,
    CASE WHEN went THEN 'reader' END::checkin_method
FROM
    seed_upcoming u
    LEFT JOIN LATERAL (SELECT d.id FROM checkin_device d WHERE d.facility_id = u.facility_id LIMIT 1) reader ON TRUE
    -- booked hours or days before; never after now
    CROSS JOIN LATERAL (SELECT LEAST(NOW() - INTERVAL '20 minutes', u.starts_at - make_interval(hours => 2 + floor(u.lead_roll * 96)::INT)) AS at) made
    CROSS JOIN LATERAL (SELECT u.cancel_roll < 0.04 AND u.owner_id IS NOT NULL AS cancelled) c
    CROSS JOIN LATERAL (SELECT NOT cancelled AND reader.id IS NOT NULL AND u.starts_at < NOW() AND u.checkin_roll < 0.88 AS went) w;

DROP TABLE seed_upcoming;
