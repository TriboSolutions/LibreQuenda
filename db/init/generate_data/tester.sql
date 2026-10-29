-- The person testing locally (the Zitadel admin in .env), as a resident with a
-- birth date and a booking at Lalín's pavilion in the first free turn from the
-- next full hour, where every way in is on. Their first sign-in takes this row
-- over (api/core/me: AdoptSeedTester), so the booking is waiting for them.
INSERT INTO app_user (idp_subject, email, full_name, birth_date)
VALUES ('seed-tester', '{TESTER_EMAIL}', 'Persoa de probas', DATE '1990-05-17');

INSERT INTO facility_checkin (facility_id, printed_qr, entry_code, staff_confirm)
SELECT f.id, TRUE, TRUE, TRUE
FROM facility f JOIN tenant t ON t.id = f.tenant_id
WHERE t.slug = 'lalin' AND f.slug = 'pavillon-municipal-de-deportes'
ON CONFLICT (facility_id) DO UPDATE SET printed_qr = TRUE, entry_code = TRUE, staff_confirm = TRUE;

INSERT INTO booking (facility_id, user_id, starts_at, ends_at, rules_accepted_at)
SELECT f.id, u.id, slot.starts_at, slot.starts_at + make_interval(mins => f.slot_minutes), NOW()
FROM facility f
    JOIN tenant t ON t.id = f.tenant_id
    JOIN app_user u ON u.idp_subject = 'seed-tester'
    CROSS JOIN LATERAL (
        SELECT date_trunc('hour', NOW()) + make_interval(hours => h) AS starts_at
        FROM generate_series(1, 12) AS h
        WHERE NOT EXISTS (
            SELECT 1 FROM booking b
            WHERE b.facility_id = f.id AND b.status = 'confirmed'
              AND b.starts_at < date_trunc('hour', NOW()) + make_interval(hours => h, mins => f.slot_minutes)
              AND b.ends_at > date_trunc('hour', NOW()) + make_interval(hours => h)
        )
        ORDER BY h
        LIMIT 1
    ) AS slot
WHERE t.slug = 'lalin' AND f.slug = 'pavillon-municipal-de-deportes';
