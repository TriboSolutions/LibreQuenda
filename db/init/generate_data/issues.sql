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
