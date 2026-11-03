-- Notifications, as the API would have left them: confirmations of the
-- upcoming bookings, who joined them, the council's cancellations of the last
-- weeks and the outcome of reported problems. All marked as already emailed
-- (emailed_at), so loading the seed never sends a thing; older ones read.
SELECT setseed(0.41);

INSERT INTO notification (user_id, booking_id, type, payload, created_at, read_at, emailed_at)
SELECT b.user_id, b.id, 'booking_confirmed',
       jsonb_build_object('facility_name', f.name, 'starts_at', b.starts_at, 'ends_at', b.ends_at),
       b.created_at,
       CASE WHEN b.created_at < NOW() - INTERVAL '1 day' OR random() < 0.5 THEN b.created_at + INTERVAL '10 minutes' END,
       b.created_at
FROM booking b JOIN facility f ON f.id = b.facility_id
WHERE b.channel = 'online' AND b.user_id IS NOT NULL AND b.ends_at > {LOCAL_DAY}::TIMESTAMP AT TIME ZONE '{TIMEZONE}';

INSERT INTO notification (user_id, booking_id, type, payload, created_at, read_at, emailed_at)
SELECT b.user_id, b.id, 'booking_joined',
       jsonb_build_object('facility_name', f.name, 'starts_at', b.starts_at, 'ends_at', b.ends_at,
                          'name', COALESCE(p.guest_name, split_part(u.full_name, ' ', 1)), 'guest', p.user_id IS NULL),
       p.joined_at,
       CASE WHEN p.joined_at < NOW() - INTERVAL '6 hours' THEN p.joined_at + INTERVAL '1 hour' END,
       p.joined_at
FROM booking_participant p
JOIN booking b ON b.id = p.booking_id
JOIN facility f ON f.id = b.facility_id
LEFT JOIN app_user u ON u.id = p.user_id
WHERE b.user_id IS NOT NULL AND b.ends_at > {LOCAL_DAY}::TIMESTAMP AT TIME ZONE '{TIMEZONE}';

INSERT INTO notification (user_id, booking_id, type, payload, created_at, read_at, emailed_at)
SELECT b.user_id, b.id, 'booking_cancelled_by_council',
       jsonb_build_object('facility_id', f.id, 'facility_name', f.name, 'tenant_slug', t.slug, 'starts_at', b.starts_at, 'ends_at', b.ends_at,
                          'reason', c.reason, 'closed_from', c.starts_at, 'closed_until', c.ends_at),
       b.cancelled_at, b.cancelled_at + INTERVAL '2 hours', b.cancelled_at
FROM booking b
JOIN facility f ON f.id = b.facility_id
JOIN tenant t ON t.id = f.tenant_id
JOIN closure c ON c.facility_id = b.facility_id AND c.starts_at < b.ends_at AND c.ends_at > b.starts_at
WHERE b.cancelled_by = 'council' AND b.user_id IS NOT NULL AND b.starts_at > NOW() - INTERVAL '45 days';

INSERT INTO notification (user_id, issue_id, type, payload, created_at, read_at, emailed_at)
SELECT i.reporter_id, i.id, CASE i.status WHEN 'solved' THEN 'issue_solved' ELSE 'issue_rejected' END::notification_type,
       jsonb_build_object('facility_name', f.name, 'facility_slug', f.slug, 'tenant_name', t.name, 'tenant_slug', t.slug, 'reason', COALESCE(i.staff_note, '')),
       i.resolved_at,
       CASE WHEN i.resolved_at < NOW() - INTERVAL '2 days' THEN i.resolved_at + INTERVAL '3 hours' END,
       i.resolved_at
FROM facility_issue i
JOIN facility f ON f.id = i.facility_id
JOIN tenant t ON t.id = f.tenant_id
WHERE i.status IN ('solved', 'rejected') AND i.reporter_id IS NOT NULL AND i.resolved_at IS NOT NULL;

DROP TABLE seed_homes;
