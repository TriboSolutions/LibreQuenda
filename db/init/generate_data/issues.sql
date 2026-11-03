-- A few problems residents reported, in different states, so the councils'
-- maintenance lists show something locally.
INSERT INTO facility_issue (facility_id, reporter_id, category, description, urgent, share_contact, status, staff_note, resolved_at, created_at)
SELECT f.id, u.id, v.category::issue_category, v.description, v.urgent, v.share_contact, v.status::issue_status, v.note,
       CASE WHEN v.status IN ('solved', 'rejected') THEN NOW() - make_interval(days => v.days_ago - 1) END,
       NOW() - make_interval(days => v.days_ago)
FROM (VALUES
    ('lalin', 'pista-de-tenis-1', 'seed-lucia', 'equipment', 'A rede está rota polo medio e non se pode xogar ben.', FALSE, TRUE, 'new', NULL, 1),
    ('lalin', 'pavillon-municipal-de-deportes', 'seed-brais', 'lighting', 'Dous focos da pista non acenden pola tarde.', TRUE, FALSE, 'in_progress', 'Avisado o servizo eléctrico, pasan o xoves.', 3),
    ('lalin', 'piscina-rua-3', 'seed-uxia', 'changing_rooms', 'Non hai auga quente nas duchas de mulleres.', FALSE, FALSE, 'solved', 'Arranxada a caldeira.', 9),
    ('lugo', 'piscina-as-pedreiras-rua-2', 'seed-noa', 'cleaning', 'O vestiario estaba moi sucio ás 8 da mañá.', FALSE, FALSE, 'new', NULL, 2),
    ('ourense', 'pavillon-dos-remedios', 'seed-martin', 'other', 'Pido que poñan máis bancos na grada.', FALSE, FALSE, 'rejected', 'Non hai orzamento este ano; anotámolo para o seguinte.', 12)
) AS v (tenant, facility, reporter, category, description, urgent, share_contact, status, note, days_ago)
JOIN tenant t ON t.slug = v.tenant
JOIN facility f ON f.tenant_id = t.id AND f.slug = v.facility
LEFT JOIN app_user u ON u.idp_subject = v.reporter;

-- …and {EXTRA_ISSUES} more over the last four months, by people of each council:
-- the older ones mostly dealt with, the recent ones mostly waiting.
SELECT setseed(0.31);
INSERT INTO facility_issue (facility_id, reporter_id, category, description, urgent, share_contact, status, staff_note, resolved_at, created_at)
SELECT f.id, reporter.user_id, v.category::issue_category, v.description, x.urgent_roll < 0.12, x.contact_roll < 0.35,
       st.status::issue_status,
       CASE st.status WHEN 'solved' THEN 'Arranxado. Grazas polo aviso.' WHEN 'rejected' THEN 'Revisámolo e está ben; se volve pasar, avísanos.'
                      WHEN 'in_progress' THEN 'Xa o sabe o servizo de mantemento.' END,
       CASE WHEN st.status IN ('solved', 'rejected') THEN x.at + make_interval(hours => 6 + floor(x.fix_roll * 200)::INT) END,
       x.at
FROM (
    SELECT g, random() AS facility_roll, random() AS kind_roll, random() AS who_roll, random() AS urgent_roll,
           random() AS contact_roll, random() AS status_roll, random() AS fix_roll,
           NOW() - make_interval(hours => floor(random() * 120 * 24)::INT) AS at
    FROM generate_series(1, {EXTRA_ISSUES}) AS g
) x
CROSS JOIN LATERAL (SELECT id, tenant_id FROM facility ORDER BY id OFFSET floor(x.facility_roll * (SELECT count(*) FROM facility))::INT LIMIT 1) f
CROSS JOIN LATERAL (
    SELECT * FROM (VALUES
        ('equipment', 'A rede ten un burato e caen as bolas.'),
        ('equipment', 'Unha canastra está frouxa e move ao tirar.'),
        ('equipment', 'Faltan corchos nunha das cordas da piscina.'),
        ('lighting', 'Un foco da esquina non acende.'),
        ('lighting', 'As luces apáganse soas antes de acabar a quenda.'),
        ('cleaning', 'O chan da pista estaba esvaradío.'),
        ('cleaning', 'As papeleiras do vestiario estaban cheas.'),
        ('changing_rooms', 'Unha ducha non ten auga quente.'),
        ('changing_rooms', 'O pechadoiro dunha taquilla está roto.'),
        ('accessibility', 'A rampla da entrada ten unha baldosa levantada.'),
        ('safety', 'Hai un cristal rachado na porta de emerxencia.'),
        ('other', 'Estaría ben ter unha fonte de auga máis preto da pista.')
    ) AS d (category, description)
    OFFSET floor(x.kind_roll * 12)::INT LIMIT 1
) v
LEFT JOIN seed_homes h ON h.tenant_id = f.tenant_id
CROSS JOIN LATERAL (SELECT h.ids[1 + floor(x.who_roll * h.n)::INT] AS user_id) reporter
CROSS JOIN LATERAL (
    SELECT CASE
        WHEN x.at > NOW() - INTERVAL '3 days' THEN CASE WHEN x.status_roll < 0.8 THEN 'new' ELSE 'in_progress' END
        WHEN x.at > NOW() - INTERVAL '14 days' THEN CASE WHEN x.status_roll < 0.3 THEN 'new' WHEN x.status_roll < 0.7 THEN 'in_progress' ELSE 'solved' END
        ELSE CASE WHEN x.status_roll < 0.8 THEN 'solved' WHEN x.status_roll < 0.92 THEN 'rejected' ELSE 'in_progress' END
    END AS status
) st;
