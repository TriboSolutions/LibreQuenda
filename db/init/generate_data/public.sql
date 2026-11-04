-- Shared upcoming bookings, where others fit: {PUBLIC_SHARE} of them public
-- (in «For you», most with a note), {INVITE_SHARE} shared by invite link only.
-- On those, people from the same council and guests (name and age) joined:
-- some still have room, some are full. The local tester's own stays private.
-- Then each council's events: one at its roomiest facility, past the upcoming
-- days (nothing booked there yet), with some neighbours already in.
SELECT setseed(0.23);

CREATE TEMP TABLE seed_shared AS
SELECT b.id, b.user_id AS owner_id, b.created_at, b.starts_at, b.ends_at, f.tenant_id, f.capacity, f.min_age, random() AS r
FROM booking b
JOIN facility f ON f.id = b.facility_id
JOIN app_user u ON u.id = b.user_id
WHERE b.status = 'confirmed' AND b.ends_at > NOW() AND f.capacity > 1 AND u.idp_subject <> 'seed-tester';
DELETE FROM seed_shared WHERE r >= {PUBLIC_SHARE} + {INVITE_SHARE};

UPDATE booking b
SET kind = CASE WHEN s.r < {PUBLIC_SHARE} THEN 'public' ELSE 'private' END::booking_kind,
    invite_token = gen_random_uuid(),
    public_description = CASE WHEN s.r < {PUBLIC_SHARE} THEN (ARRAY[
        '',
        'Falta xente para completar. Nivel medio, sen présa.',
        'Partido tranquilo, calquera nivel é benvido.',
        'Buscamos un par de persoas máis para facer equipos.',
        'Imos todas as semanas; se che gusta, quédaste.',
        'Adestramento suave. Trae auga!',
        'Somos principiantes, ninguén se vai rir de ti.',
        'Nivel alto: buscamos xente con experiencia.',
        'Despois tomamos algo, se apetece.',
        'Pachanga mixta, ven con quen queiras.'
    ])[1 + floor(random() * 10)::INT] ELSE '' END
FROM seed_shared s
WHERE b.id = s.id;

-- how many joined: from nobody to full, never more than 11 others
CREATE TEMP TABLE seed_join AS
SELECT s.*, g.i, random() AS guest_roll, random() AS who_roll, random() AS age_roll
FROM seed_shared s
CROSS JOIN LATERAL (SELECT LEAST(s.capacity - 1, 11, floor(random() * (s.capacity + 0 * s.r))::INT) AS k) c
CROSS JOIN LATERAL generate_series(1, c.k) AS g(i);

-- people with an account: from the council, free at that time
INSERT INTO booking_participant (booking_id, user_id, joined_at)
SELECT j.id, who.user_id, LEAST(NOW() - INTERVAL '5 minutes', j.created_at + make_interval(mins => 25 * j.i))
FROM seed_join j
JOIN seed_homes h ON h.tenant_id = j.tenant_id
CROSS JOIN LATERAL (SELECT h.ids[1 + floor(j.who_roll * h.n)::INT] AS user_id) who
WHERE j.guest_roll >= 0.3
  AND who.user_id <> j.owner_id
  AND NOT EXISTS (
      SELECT 1 FROM booking o WHERE o.user_id = who.user_id AND o.status = 'confirmed' AND o.starts_at < j.ends_at AND o.ends_at > j.starts_at)
  AND NOT EXISTS (
      SELECT 1 FROM booking_participant p JOIN booking o ON o.id = p.booking_id
      WHERE p.user_id = who.user_id AND o.starts_at < j.ends_at AND o.ends_at > j.starts_at)
  AND (j.min_age IS NULL OR (SELECT date_part('year', age(j.starts_at::DATE, u.birth_date)) >= j.min_age FROM app_user u WHERE u.id = who.user_id))
ON CONFLICT DO NOTHING;

-- guests: a name and an age, old enough for the facility
INSERT INTO booking_participant (booking_id, guest_name, guest_age, joined_at)
SELECT j.id,
       (ARRAY['Lucas', 'Marta', 'Xián', 'Carlota', 'Nico', 'Aroa', 'Mateo', 'Vera', 'Bruno', 'Ainhoa', 'Leo', 'Candela', 'Pepe', 'Tere'])[1 + floor(j.who_roll * 14)::INT],
       GREATEST(COALESCE(j.min_age, 8), 10 + floor(j.age_roll * 50)::INT),
       LEAST(NOW() - INTERVAL '5 minutes', j.created_at + make_interval(mins => 25 * j.i))
FROM seed_join j
WHERE j.guest_roll < 0.3;

DROP TABLE seed_join;
DROP TABLE seed_shared;

-- events: the council's, no owner; 10:00–14:00 a few days after the upcoming bookings end
CREATE TEMP TABLE seed_event AS
SELECT DISTINCT ON (f.tenant_id)
    f.id AS facility_id, f.tenant_id,
    ({LOCAL_DAY} + {UPCOMING_DAYS} + 2 + TIME '10:00') AT TIME ZONE '{TIMEZONE}' AS starts_at,
    ({LOCAL_DAY} + {UPCOMING_DAYS} + 2 + TIME '14:00') AT TIME ZONE '{TIMEZONE}' AS ends_at,
    (ARRAY['Fiesta do deporte', 'Torneo de fútbol 5x5', 'Xornada de portas abertas', 'Maratón de baloncesto 3x3'])[1 + floor(random() * 4)::INT] AS title,
    (SELECT m.user_id FROM tenant_membership m WHERE m.tenant_id = f.tenant_id AND m.role = 'admin' ORDER BY m.user_id LIMIT 1) AS booked_by
FROM facility f
WHERE f.is_active AND f.capacity > 1
ORDER BY f.tenant_id, f.capacity DESC, f.id;

INSERT INTO booking (facility_id, starts_at, ends_at, channel, booked_by, kind, title, public_description, event_capacity, created_at)
SELECT e.facility_id, e.starts_at, e.ends_at, 'staff', e.booked_by, 'event', e.title,
       'Actividade aberta organizada polo concello. Ven coa túa familia e amizades; non fai falta experiencia.',
       30, NOW() - INTERVAL '3 days'
FROM seed_event e
WHERE NOT EXISTS (
    SELECT 1 FROM booking b WHERE b.facility_id = e.facility_id AND b.status = 'confirmed'
      AND b.starts_at < e.ends_at AND b.ends_at > e.starts_at);

-- a handful of the council's residents already joined
INSERT INTO booking_participant (booking_id, user_id, joined_at)
SELECT b.id, h.ids[1 + ((g.i * 7) % h.n)], NOW() - make_interval(hours => g.i)
FROM booking b
JOIN seed_event e ON e.facility_id = b.facility_id AND e.starts_at = b.starts_at
JOIN seed_homes h ON h.tenant_id = e.tenant_id
CROSS JOIN generate_series(1, 8) AS g(i)
WHERE b.kind = 'event'
ON CONFLICT DO NOTHING;

DROP TABLE seed_event;
