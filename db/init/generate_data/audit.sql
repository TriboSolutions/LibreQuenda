-- A few audit log entries, so the "Activity log" tabs show something locally.
-- Written by the system (no account), dated in the last days.
INSERT INTO audit_log (at, actor_role, tenant_id, tenant_label, action, target_type, target_id, target_label, summary)
SELECT NOW() - v.ago, 'system', t.id, t.name, v.action, v.target_type, f.id, f.name, v.summary
FROM (VALUES
    ('lalin', 'pavillon-municipal-de-deportes', INTERVAL '3 days',  'facility.updated',            'facility', 'slot_minutes: 90 → 60'),
    ('lalin', 'pavillon-municipal-de-deportes', INTERVAL '2 days',  'checkin.settings_changed',    'facility', 'entry_code: false → true; staff_contacts: 0 → 2'),
    ('lalin', 'pista-de-padel-1',               INTERVAL '30 hours', 'closure.created',            'facility', 'closed 2026-11-15T00:00 – 2026-11-16T23:59; bookings cancelled: 0'),
    ('lugo',  'piscina-as-pedreiras-rua-2',     INTERVAL '1 day',   'facility.activation_changed', 'facility', 'active: true'),
    ('ourense','pavillon-dos-remedios',         INTERVAL '5 hours', 'reader.added',                'facility', 'reader "Entrada dos Remedios"')
) AS v (tenant, facility, ago, action, target_type, summary)
JOIN tenant t ON t.slug = v.tenant
JOIN facility f ON f.tenant_id = t.id AND f.slug = v.facility;

INSERT INTO audit_log (at, actor_role, tenant_id, tenant_label, action, summary)
SELECT NOW() - INTERVAL '4 days', 'system', t.id, t.name, 'booking_window.council_changed', 'max_bookings_per_day: 0 → 1'
FROM tenant t WHERE t.slug = 'lalin';

INSERT INTO audit_log (at, actor_role, action, target_type, target_label, summary)
VALUES (NOW() - INTERVAL '6 days', 'system', 'tag.created', 'tag', 'Pádel', '"Pádel"');

-- …and {EXTRA_AUDIT} more from the system over the last three months, across the councils
SELECT setseed(0.37);
INSERT INTO audit_log (at, actor_role, tenant_id, tenant_label, action, target_type, target_id, target_label, summary)
SELECT NOW() - make_interval(mins => floor(x.ago_roll * 90 * 24 * 60)::INT), 'system', t.id, t.name, a.action, 'facility', f.id, f.name, a.summary
FROM (
    SELECT g, random() AS facility_roll, random() AS action_roll, random() AS ago_roll FROM generate_series(1, {EXTRA_AUDIT}) AS g
) x
CROSS JOIN LATERAL (SELECT id, tenant_id, name FROM facility ORDER BY id OFFSET floor(x.facility_roll * (SELECT count(*) FROM facility))::INT LIMIT 1) f
JOIN tenant t ON t.id = f.tenant_id
CROSS JOIN LATERAL (
    SELECT * FROM (VALUES
        ('facility.updated', 'usage_rules_text: changed'),
        ('facility.updated', 'capacity: 4 → 6'),
        ('facility.updated', 'description: changed'),
        ('facility.schedule_changed', 'opening hours: 2 windows'),
        ('closure.created', 'closed for maintenance; bookings cancelled: 0'),
        ('closure.deleted', 'closure removed'),
        ('booking_window.facility_changed', 'max_bookings_per_day: 0 → 1'),
        ('checkin.code_renewed', 'entry code renewed'),
        ('checkin.settings_changed', 'entry_code: false → true'),
        ('issue.changed', 'status: new → in_progress'),
        ('issue.changed', 'status: in_progress → solved')
    ) AS d (action, summary)
    OFFSET floor(x.action_roll * 11)::INT LIMIT 1
) a;
