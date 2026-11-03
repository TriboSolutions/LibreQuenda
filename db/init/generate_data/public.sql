-- Shared upcoming bookings, where others fit: {PUBLIC_SHARE} of them public
-- (in «For you», most with a note), {INVITE_SHARE} shared by invite link only.
-- On those, people from the same council and guests (name and age) joined:
-- some still have room, some are full. The local tester's own stays private.
SELECT setseed(0.23);

CREATE TEMP TABLE seed_shared AS
SELECT b.id, b.user_id AS owner_id, b.created_at, b.starts_at, b.ends_at, f.tenant_id, f.capacity, f.min_age, random() AS r
FROM booking b
JOIN facility f ON f.id = b.facility_id
JOIN app_user u ON u.id = b.user_id
WHERE b.status = 'confirmed' AND b.ends_at > NOW() AND f.capacity > 1 AND u.idp_subject <> 'seed-tester';
DELETE FROM seed_shared WHERE r >= {PUBLIC_SHARE} + {INVITE_SHARE};

UPDATE booking b
SET is_public = s.r < {PUBLIC_SHARE},
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
