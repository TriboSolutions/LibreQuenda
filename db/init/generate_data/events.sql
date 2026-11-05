-- The councils' events, before the upcoming bookings so those fill around them:
-- at each council's three roomiest facilities, over the coming days (one
-- tomorrow evening, one on the next Saturday, one midweek) and one past the
-- upcoming days. They belong to the council: no owner, booked by staff (its
-- first administrator). Who joined them comes in public.sql.
SELECT setseed(0.31);

CREATE TEMP TABLE seed_event_place AS
SELECT f.id AS facility_id, f.tenant_id, row_number() OVER (PARTITION BY f.tenant_id ORDER BY f.capacity DESC, f.id) AS rank,
       (SELECT m.user_id FROM tenant_membership m WHERE m.tenant_id = f.tenant_id AND m.role = 'admin' ORDER BY m.user_id LIMIT 1) AS booked_by
FROM facility f
WHERE f.is_active AND f.capacity > 1;
DELETE FROM seed_event_place WHERE rank > 3;

-- the next Saturday after today
CREATE TEMP TABLE seed_event_plan (rank INT, day_offset INT, starts TIME, ends TIME, title TEXT, description TEXT, capacity INT);
INSERT INTO seed_event_plan VALUES
    (1, 1, '18:00', '20:00', 'Clase aberta de iniciación',
        'Unha primeira toma de contacto guiada polo persoal do concello. Prazas limitadas: apúntate axiña.', 12),
    (2, 1 + ((13 - EXTRACT(ISODOW FROM {LOCAL_DAY} + 1)::INT) % 7), '10:00', '13:00', 'Torneo aberto',
        'Equipos mixtos que se fan alí mesmo, con partidos curtos e premios para todos. O concello pon o material.', 24),
    (3, 4, '17:30', '19:30', 'Xornada de portas abertas',
        'Ven coñecer a instalación e probar as actividades da tempada. Para todas as idades.', NULL),
    (1, {UPCOMING_DAYS} + 2, '10:00', '14:00', 'Festa do deporte',
        'Actividade aberta organizada polo concello. Ven coa túa familia e amizades; non fai falta experiencia.', 30);

INSERT INTO booking (facility_id, starts_at, ends_at, channel, booked_by, kind, title, public_description, event_capacity, created_at)
SELECT p.facility_id,
       ({LOCAL_DAY} + e.day_offset + e.starts) AT TIME ZONE '{TIMEZONE}',
       ({LOCAL_DAY} + e.day_offset + e.ends) AT TIME ZONE '{TIMEZONE}',
       'staff', p.booked_by, 'event', e.title, e.description, e.capacity,
       NOW() - make_interval(days => 2 + floor(random() * 5)::INT)
FROM seed_event_place p
JOIN seed_event_plan e ON e.rank = p.rank
-- not over a closure, nor over the local tester's booking
WHERE NOT EXISTS (
    SELECT 1 FROM closure c WHERE c.facility_id = p.facility_id
      AND c.starts_at < ({LOCAL_DAY} + e.day_offset + e.ends) AT TIME ZONE '{TIMEZONE}'
      AND c.ends_at > ({LOCAL_DAY} + e.day_offset + e.starts) AT TIME ZONE '{TIMEZONE}')
  AND NOT EXISTS (
    SELECT 1 FROM booking b WHERE b.facility_id = p.facility_id AND b.status = 'confirmed'
      AND b.starts_at < ({LOCAL_DAY} + e.day_offset + e.ends) AT TIME ZONE '{TIMEZONE}'
      AND b.ends_at > ({LOCAL_DAY} + e.day_offset + e.starts) AT TIME ZONE '{TIMEZONE}');

DROP TABLE seed_event_plan;
DROP TABLE seed_event_place;
