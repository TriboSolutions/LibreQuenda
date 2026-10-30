-- Ways to check in for a few facilities, so the Entrada tab, the check-in dialog
-- and the flash booking have something to show: staff confirmation with the
-- people residents can ask on site (not accounts), and the entry code.
INSERT INTO facility_checkin (facility_id, entry_code, staff_confirm, staff_contacts, checks_in_since)
SELECT f.id, v.entry_code, TRUE, v.contacts::JSONB, NOW()
FROM (VALUES
    ('lalin', 'pavillon-municipal-de-deportes', TRUE,
     '[{"name":"Marta Rey","phone":"986 787 061","place":"Conserxería, á entrada do pavillón"},{"name":"Xosé Lois Fraga","phone":"","place":"Almacén de material, xunto á pista"}]'),
    ('lugo', 'piscina-as-pedreiras-rua-2', FALSE,
     '[{"name":"Socorrista de quenda","phone":"982 297 200","place":"Cadeira de socorrismo, beira da piscina"}]'),
    ('ourense', 'pavillon-dos-remedios', TRUE,
     '[{"name":"Conserxería dos Remedios","phone":"988 388 100","place":"Mostrador da entrada principal"}]')
) AS v (tenant, facility, entry_code, contacts)
JOIN tenant t ON t.slug = v.tenant
JOIN facility f ON f.tenant_id = t.id AND f.slug = v.facility
ON CONFLICT (facility_id) DO NOTHING;
