-- Participation, so every council's pages, its residents' «For you» and its
-- Participation tab show something: announcements (current, about a facility,
-- and expired), polls (open, residents-only, and closed with its results) with
-- votes by the council's people, and a few problems fixed lately, so each
-- facility page has a «Fixed lately» line and an average to show.
SELECT setseed(0.53);

CREATE TEMP TABLE seed_admin AS
SELECT t.id AS tenant_id, (SELECT m.user_id FROM tenant_membership m WHERE m.tenant_id = t.id AND m.role = 'admin' ORDER BY m.created_at LIMIT 1) AS admin_id
FROM tenant t;

-- a facility of each council to talk about: its busiest one
CREATE TEMP TABLE seed_spot AS
SELECT DISTINCT ON (f.tenant_id) f.tenant_id, f.id AS facility_id, f.name
FROM facility f WHERE f.is_active
ORDER BY f.tenant_id, (SELECT count(*) FROM booking b WHERE b.facility_id = f.id) DESC, f.id;

-- announcements: one for the whole council, one about a facility, one already expired
INSERT INTO council_notice (tenant_id, facility_id, title, body, expires_at, notify, created_by, created_at)
SELECT t.id, NULL, 'Horario de inverno nas instalacións municipais',
       'Desde o 1 de decembro ata o 6 de xaneiro as instalacións abren de 9:00 a 21:00 de luns a sábado, e os domingos e festivos de 10:00 a 14:00.' || E'\n' || 'As reservas xa feitas mantéñense.',
       date_trunc('day', NOW() AT TIME ZONE '{TIMEZONE}') AT TIME ZONE '{TIMEZONE}' + INTERVAL '21 days', TRUE, a.admin_id, NOW() - INTERVAL '2 days'
FROM tenant t JOIN seed_admin a ON a.tenant_id = t.id;

INSERT INTO council_notice (tenant_id, facility_id, title, body, expires_at, notify, created_by, created_at)
SELECT s.tenant_id, s.facility_id, 'Mantemento o vindeiro luns pola mañá',
       'O luns pola mañá o persoal de mantemento revisa a iluminación e o chan. Ata as 14:00 pode haber ruído e zonas pechadas; pola tarde todo funciona con normalidade.',
       date_trunc('day', NOW() AT TIME ZONE '{TIMEZONE}') AT TIME ZONE '{TIMEZONE}' + INTERVAL '6 days', TRUE, a.admin_id, NOW() - INTERVAL '5 hours'
FROM seed_spot s JOIN seed_admin a ON a.tenant_id = s.tenant_id;

INSERT INTO council_notice (tenant_id, facility_id, title, body, expires_at, notify, created_by, created_at)
SELECT s.tenant_id, s.facility_id, 'Redes novas nas pistas',
       'Xa están postas as redes novas. Grazas a quen avisou de que as vellas estaban rotas.',
       NOW() - INTERVAL '3 days', FALSE, a.admin_id, NOW() - INTERVAL '12 days'
FROM seed_spot s JOIN seed_admin a ON a.tenant_id = s.tenant_id;

-- who each announcement reached, as the API would have counted them (only the
-- tester's bell gets the notices, so the seed stays light)
UPDATE council_notice n SET recipients = (
    SELECT count(DISTINCT p.user_id) FROM (
        SELECT b.user_id FROM booking b JOIN facility f ON f.id = b.facility_id
        WHERE f.tenant_id = n.tenant_id AND b.status = 'confirmed' AND b.user_id IS NOT NULL
          AND (n.facility_id IS NULL OR b.facility_id = n.facility_id)
          AND b.ends_at > n.created_at - CASE WHEN n.facility_id IS NULL THEN INTERVAL '90 days' ELSE INTERVAL '30 days' END
        UNION ALL
        SELECT r.user_id FROM tenant_resident r WHERE n.facility_id IS NULL AND r.tenant_id = n.tenant_id
    ) p
) WHERE n.notify;

INSERT INTO notification (user_id, notice_id, type, payload, created_at, read_at, emailed_at)
SELECT u.id, n.id, 'council_notice',
       jsonb_build_object('title', n.title, 'tenant_name', t.name, 'tenant_slug', t.slug,
                          'facility_name', COALESCE(f.name, ''), 'facility_slug', COALESCE(f.slug, '')),
       n.created_at, NULL, n.created_at
FROM council_notice n
JOIN tenant t ON t.id = n.tenant_id
LEFT JOIN facility f ON f.id = n.facility_id
JOIN seed_user_home h ON h.tenant_id = n.tenant_id
JOIN app_user u ON u.id = h.user_id AND u.idp_subject = 'seed-tester'
WHERE n.notify AND n.expires_at > NOW();

-- polls: an open one for everybody, an open one for verified residents only
-- (about the facility), and one closed three days ago with its results
CREATE TEMP TABLE seed_poll (poll_id UUID, tenant_id UUID, kind TEXT, closed BOOLEAN);

WITH made AS (
    INSERT INTO poll (tenant_id, facility_id, question, residents_only, closes_at, closed_at, created_by, created_at)
    SELECT t.id, v.facility_id, v.question, v.residents_only, v.closes_at, v.closed_at, a.admin_id, v.created_at
    FROM tenant t
    JOIN seed_admin a ON a.tenant_id = t.id
    JOIN seed_spot s ON s.tenant_id = t.id
    CROSS JOIN LATERAL (VALUES
        ('hours', NULL::UUID, 'Que horario preferides para as instalacións os sábados?', FALSE,
         NOW() + INTERVAL '9 days', NULL::TIMESTAMPTZ, NOW() - INTERVAL '3 days'),
        ('facility', s.facility_id, 'Que mellorariades primeiro en ' || s.name || '?', TRUE,
         NOW() + INTERVAL '20 days', NULL::TIMESTAMPTZ, NOW() - INTERVAL '1 day'),
        ('sport', NULL::UUID, 'Que actividade botades en falta no concello?', FALSE,
         NOW() - INTERVAL '3 days', NULL::TIMESTAMPTZ, NOW() - INTERVAL '24 days')
    ) AS v (kind, facility_id, question, residents_only, closes_at, closed_at, created_at)
    RETURNING id, tenant_id, question
)
INSERT INTO seed_poll (poll_id, tenant_id, kind, closed)
SELECT id, tenant_id,
       CASE WHEN question LIKE 'Que horario%' THEN 'hours' WHEN question LIKE 'Que mellorariades%' THEN 'facility' ELSE 'sport' END,
       question LIKE 'Que actividade%'
FROM made;

INSERT INTO poll_option (poll_id, position, label)
SELECT p.poll_id, o.position, o.label
FROM seed_poll p
JOIN (VALUES
    ('hours', 0, 'De 8:00 a 14:00'), ('hours', 1, 'De 10:00 a 20:00'), ('hours', 2, 'De 9:00 a 22:00, todo seguido'),
    ('facility', 0, 'Os vestiarios e as duchas'), ('facility', 1, 'A iluminación'), ('facility', 2, 'O material (redes, canastras, porterías)'), ('facility', 3, 'A accesibilidade'),
    ('sport', 0, 'Pilates e ioga'), ('sport', 1, 'Escalada'), ('sport', 2, 'Atletismo para persoas maiores'), ('sport', 3, 'Natación para bebés')
) AS o (kind, position, label) ON o.kind = p.kind;

-- a third of each council's people voted (the tester only in the closed one,
-- so they can still vote in the open ones), the answers leaning one way
INSERT INTO poll_vote (poll_id, option_id, user_id, created_at, updated_at)
SELECT p.poll_id, o.id, h.user_id, v.at, v.at
FROM seed_poll p
JOIN seed_user_home h ON h.tenant_id = p.tenant_id
JOIN app_user u ON u.id = h.user_id
-- correlated with the person and the poll, so each row rolls its own dice
CROSS JOIN LATERAL (SELECT random() AS turnout, random() AS pick, NOW() - make_interval(hours => 2 + floor(random() * 60)::INT) AS at WHERE h.user_id IS NOT NULL AND p.poll_id IS NOT NULL) v
CROSS JOIN LATERAL (SELECT count(*)::INT AS n FROM poll_option WHERE poll_id = p.poll_id) c
JOIN poll_option o ON o.poll_id = p.poll_id AND o.position = LEAST(c.n - 1, floor(power(v.pick, 1.6) * c.n)::INT)
WHERE (v.turnout < 0.34 OR (u.idp_subject = 'seed-tester' AND p.closed))
  AND (u.idp_subject IS DISTINCT FROM 'seed-tester' OR p.closed)
  AND (p.kind <> 'facility' OR EXISTS (SELECT 1 FROM tenant_resident r WHERE r.tenant_id = p.tenant_id AND r.user_id = h.user_id));

-- who each poll asked, as the API would have counted them; the closed one already
-- told them its results (results_sent_at), so loading the seed notifies nobody.
-- Only the tester's bell gets the notices, so the seed stays light.
UPDATE poll p SET
    recipients = (
        SELECT count(DISTINCT x.user_id) FROM (
            SELECT r.user_id FROM tenant_resident r WHERE r.tenant_id = p.tenant_id
            UNION ALL
            SELECT h.user_id FROM seed_user_home h WHERE NOT p.residents_only AND h.tenant_id = p.tenant_id
        ) x
    ),
    results_sent_at = CASE WHEN s.closed THEN NOW() - INTERVAL '3 days' END
FROM seed_poll s
WHERE s.poll_id = p.id;

INSERT INTO notification (user_id, poll_id, type, payload, created_at, read_at, emailed_at)
SELECT u.id, p.id, CASE WHEN s.closed THEN 'poll_closed' ELSE 'poll_opened' END::notification_type,
       jsonb_build_object('title', p.question, 'tenant_name', t.name, 'tenant_slug', t.slug,
                          'facility_name', COALESCE(f.name, ''), 'facility_slug', COALESCE(f.slug, '')),
       CASE WHEN s.closed THEN NOW() - INTERVAL '3 days' ELSE p.created_at END, NULL, NOW()
FROM seed_poll s
JOIN poll p ON p.id = s.poll_id
JOIN tenant t ON t.id = p.tenant_id
LEFT JOIN facility f ON f.id = p.facility_id
JOIN seed_user_home h ON h.tenant_id = p.tenant_id
JOIN app_user u ON u.id = h.user_id AND u.idp_subject = 'seed-tester'
WHERE NOT p.residents_only OR EXISTS (SELECT 1 FROM tenant_resident r WHERE r.tenant_id = p.tenant_id AND r.user_id = u.id);

-- a few problems fixed in the last days at each council's facility, for «Fixed lately»
INSERT INTO facility_issue (facility_id, reporter_id, category, description, urgent, share_contact, status, staff_note, resolved_at, created_at)
SELECT s.facility_id, NULL, v.category::issue_category, v.description, FALSE, FALSE, 'solved',
       'Arranxado. Grazas polo aviso.', NOW() - make_interval(days => v.solved_days_ago), NOW() - make_interval(days => v.reported_days_ago)
FROM seed_spot s
CROSS JOIN (VALUES
    ('equipment', 'A rede estaba rota polo medio.', 2, 5),
    ('cleaning', 'O vestiario estaba sucio pola mañá.', 6, 7)
) AS v (category, description, solved_days_ago, reported_days_ago);

DROP TABLE seed_poll;
DROP TABLE seed_spot;
DROP TABLE seed_admin;
