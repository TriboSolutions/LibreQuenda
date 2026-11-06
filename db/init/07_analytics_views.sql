-- ============================================================
--  ANALYTICS VIEWS: what the read-only analytics user (08_analytics_role.sh)
--  can query, for the councils' and the superadministrators' questions about
--  their data (the assistant in the indicators, and the MCP's analytics tools).
--
--  One view per table, in the analytics schema, without:
--    - personal data: names, emails, phones, DNI, addresses, birth dates (only
--      the year), avatars;
--    - secrets and tokens: QR, NFC and calendar tokens, entry-code secrets,
--      reader keys, SMTP settings;
--    - free text people write (descriptions, reasons, notes, rules, titles,
--      notification payloads): it could carry instructions to the model.
--  People are their ids only, for counting and grouping.
--
--  Which councils a query sees is the transaction's analytics.tenants (an array
--  of council ids) or analytics.superadmin = 'on' (a superadministrator), set by the
--  API with SET LOCAL. The analytics user cannot change them: set_config() is
--  revoked from it, and the API runs one SELECT at a time. PostgreSQL has no
--  row-level security on views: each view filters with analytics.sees(), and is
--  a security barrier so no function sees rows before that filter.
--  The views run as their owner (POSTGRES_USER), so the analytics user needs no
--  right on the tables themselves.
-- ============================================================

CREATE SCHEMA analytics;
COMMENT ON SCHEMA analytics IS 'Read-only views for analytics: no personal data, no secrets, no free text';

-- whether the current transaction may see this council's rows
CREATE FUNCTION analytics.sees(council UUID) RETURNS BOOLEAN
LANGUAGE sql STABLE AS $$
    SELECT COALESCE(current_setting('analytics.superadmin', TRUE), '') = 'on'
        OR council = ANY (COALESCE(NULLIF(current_setting('analytics.tenants', TRUE), ''), '{}')::UUID[])
$$;

-- whether it sees everything (a superadministrator): for what belongs to no council
CREATE FUNCTION analytics.sees_all() RETURNS BOOLEAN
LANGUAGE sql STABLE AS $$
    SELECT COALESCE(current_setting('analytics.superadmin', TRUE), '') = 'on'
$$;

-- --- councils and their facilities ------------------------------------------

CREATE VIEW analytics.tenant WITH (security_barrier = TRUE) AS
SELECT id, slug, name, timezone, municipality_code, is_active, created_at
FROM public.tenant
WHERE analytics.sees(id);
COMMENT ON VIEW analytics.tenant IS 'Councils (concellos). municipality_code: INE code';

CREATE VIEW analytics.tenant_booking_rules WITH (security_barrier = TRUE) AS
SELECT tenant_id, max_advance_hours, min_advance_minutes, min_cancel_hours, max_active_bookings_per_user,
       max_bookings_per_day, max_turns_in_a_row, max_bookings_per_week, max_active_bookings_council,
       weekly_opening, opens_weekday, opens_minute, no_show_penalty, no_show_limit, no_show_window_days,
       no_show_block_days, updated_at
FROM public.tenant_booking_rules
WHERE analytics.sees(tenant_id);
COMMENT ON VIEW analytics.tenant_booking_rules IS 'A council''s default booking window, inherited by its facilities';

CREATE VIEW analytics.facility WITH (security_barrier = TRUE) AS
SELECT id, tenant_id, slug, name, latitude, longitude, capacity, show_weather, is_active, slot_minutes, custom_rules,
       max_advance_hours, min_advance_minutes, min_cancel_hours, max_active_bookings_per_user, max_bookings_per_day,
       max_turns_in_a_row, max_bookings_per_week, weekly_opening, opens_weekday, opens_minute, no_show_penalty,
       no_show_limit, no_show_window_days, no_show_block_days, min_age, residents_only, created_at
FROM public.facility
WHERE analytics.sees(tenant_id);
COMMENT ON VIEW analytics.facility IS 'Facilities (pistas, pavillóns, piscinas…). capacity: people per booking, owner included; slot_minutes: the length of a turn';

CREATE VIEW analytics.opening_hours WITH (security_barrier = TRUE) AS
SELECT h.id, h.facility_id, h.weekday, h.opens_at, h.closes_at
FROM public.opening_hours h JOIN public.facility f ON f.id = h.facility_id
WHERE analytics.sees(f.tenant_id);
COMMENT ON VIEW analytics.opening_hours IS 'Weekly opening windows; weekday ISO (1 = Monday), times in the council''s time zone';

CREATE VIEW analytics.closure WITH (security_barrier = TRUE) AS
SELECT c.id, c.facility_id, c.starts_at, c.ends_at, c.created_at
FROM public.closure c JOIN public.facility f ON f.id = c.facility_id
WHERE analytics.sees(f.tenant_id);
COMMENT ON VIEW analytics.closure IS 'Closures (holidays, maintenance): no booking in [starts_at, ends_at)';

CREATE VIEW analytics.facility_checkin WITH (security_barrier = TRUE) AS
SELECT c.facility_id, c.printed_qr, c.entry_code, c.radius_m, c.flash_booking, c.staff_confirm, c.qr_reader, c.nfc, c.updated_at
FROM public.facility_checkin c JOIN public.facility f ON f.id = c.facility_id
WHERE analytics.sees(f.tenant_id);
COMMENT ON VIEW analytics.facility_checkin IS 'The ways a facility takes check-ins (the booleans), and the distance allowed for the phone ones';

CREATE VIEW analytics.facility_checkin_period WITH (security_barrier = TRUE) AS
SELECT p.id, p.facility_id, p.starts_at, p.ends_at
FROM public.facility_checkin_period p JOIN public.facility f ON f.id = p.facility_id
WHERE analytics.sees(f.tenant_id);
COMMENT ON VIEW analytics.facility_checkin_period IS 'When a facility could take check-ins (ends_at NULL: still can): no-shows only count inside these';

CREATE VIEW analytics.checkin_device WITH (security_barrier = TRUE) AS
SELECT d.id, d.facility_id, d.last_seen_at, d.created_at
FROM public.checkin_device d JOIN public.facility f ON f.id = d.facility_id
WHERE analytics.sees(f.tenant_id);
COMMENT ON VIEW analytics.checkin_device IS 'QR readers at facilities'' doors';

CREATE VIEW analytics.tag_category AS
SELECT id, name, position, created_at FROM public.tag_category;
COMMENT ON VIEW analytics.tag_category IS 'Tag categories (Deportes, Instalación, Servizos…), shared by every council';

CREATE VIEW analytics.tag AS
SELECT id, category_id, slug, name, created_at FROM public.tag;
COMMENT ON VIEW analytics.tag IS 'Tags (padel, piscina, cuberta…), shared by every council';

CREATE VIEW analytics.facility_tag WITH (security_barrier = TRUE) AS
SELECT t.facility_id, t.tag_id
FROM public.facility_tag t JOIN public.facility f ON f.id = t.facility_id
WHERE analytics.sees(f.tenant_id);
COMMENT ON VIEW analytics.facility_tag IS 'Which tags each facility has (its sports among them)';

-- --- bookings -------------------------------------------------------------------

CREATE VIEW analytics.booking WITH (security_barrier = TRUE) AS
SELECT b.id, b.facility_id, b.user_id, b.starts_at, b.ends_at, b.status, b.kind, b.channel,
       (b.booked_by IS NOT NULL) AS booked_by_staff,
       (b.user_id IS NULL AND b.holder_name IS NOT NULL) AS holder_without_account,
       (b.rules_accepted_at IS NOT NULL) AS rules_accepted,
       b.cancelled_at, b.cancelled_by, b.checked_in_at, b.checked_in_method, b.event_capacity,
       b.created_at, b.updated_at
FROM public.booking b JOIN public.facility f ON f.id = b.facility_id
WHERE analytics.sees(f.tenant_id);
COMMENT ON VIEW analytics.booking IS 'Bookings. status: confirmed or cancelled; kind: private, public (others can join) or event (the council''s, no owner); channel: online or staff; cancelled_by: user or council; checked_in_method: reader, entry_code, printed_qr, nfc or staff. Times are UTC timestamps: convert with AT TIME ZONE ''Europe/Madrid''';

CREATE VIEW analytics.booking_participant WITH (security_barrier = TRUE) AS
SELECT p.id, p.booking_id, p.user_id, (p.user_id IS NULL) AS is_guest, p.guest_age, p.joined_at
FROM public.booking_participant p JOIN public.booking b ON b.id = p.booking_id JOIN public.facility f ON f.id = b.facility_id
WHERE analytics.sees(f.tenant_id);
COMMENT ON VIEW analytics.booking_participant IS 'People who joined a booking (public, invited, or an event), not counting its owner; a guest has no account, only an age';

CREATE VIEW analytics.booking_removed WITH (security_barrier = TRUE) AS
SELECT r.booking_id, r.user_id, r.removed_at
FROM public.booking_removed r JOIN public.booking b ON b.id = r.booking_id JOIN public.facility f ON f.id = b.facility_id
WHERE analytics.sees(f.tenant_id);
COMMENT ON VIEW analytics.booking_removed IS 'People an owner took out of their public booking';

CREATE VIEW analytics.facility_rating WITH (security_barrier = TRUE) AS
SELECT r.id, r.facility_id, r.booking_id, r.user_id, r.stars, r.created_at
FROM public.facility_rating r JOIN public.facility f ON f.id = r.facility_id
WHERE analytics.sees(f.tenant_id);
COMMENT ON VIEW analytics.facility_rating IS 'Residents'' stars (1 to 5) for a facility, after a booking there';

CREATE VIEW analytics.user_rating WITH (security_barrier = TRUE) AS
SELECT r.id, r.booking_id, r.rater_id, r.rated_id, r.stars, r.created_at
FROM public.user_rating r JOIN public.booking b ON b.id = r.booking_id JOIN public.facility f ON f.id = b.facility_id
WHERE analytics.sees(f.tenant_id);
COMMENT ON VIEW analytics.user_rating IS 'Stars (1 to 5) players gave each other after a public booking';

CREATE VIEW analytics.facility_issue WITH (security_barrier = TRUE) AS
SELECT i.id, i.facility_id, i.reporter_id, i.booking_id, i.category, i.status, i.urgent, i.share_contact,
       (i.photo_ref IS NOT NULL) AS has_photo, i.resolved_by, i.resolved_at, i.created_at, i.updated_at
FROM public.facility_issue i JOIN public.facility f ON f.id = i.facility_id
WHERE analytics.sees(f.tenant_id);
COMMENT ON VIEW analytics.facility_issue IS 'Problems residents reported at facilities. category: equipment, lighting, cleaning, changing_rooms, accessibility, safety, other; status: new, in_progress, solved, rejected';

CREATE VIEW analytics.notification WITH (security_barrier = TRUE) AS
SELECT n.id, n.user_id, n.booking_id, n.issue_id, n.type, n.created_at, n.read_at, n.emailed_at
FROM public.notification n
LEFT JOIN public.booking b ON b.id = n.booking_id
LEFT JOIN public.facility_issue i ON i.id = n.issue_id
LEFT JOIN public.facility f ON f.id = COALESCE(b.facility_id, i.facility_id)
WHERE (f.id IS NOT NULL AND analytics.sees(f.tenant_id)) OR analytics.sees_all();
COMMENT ON VIEW analytics.notification IS 'Notices people got (bell, app and email): their type and when they were read or emailed';

-- --- people (ids only) -------------------------------------------------------------

CREATE VIEW analytics.app_user WITH (security_barrier = TRUE) AS
SELECT u.id, u.email_verified, EXTRACT(YEAR FROM u.birth_date)::INT AS birth_year, u.preferred_language,
       u.email_reminders, u.email_notifications, u.show_recommendations, u.created_at
FROM public.app_user u
WHERE analytics.sees_all()
   OR EXISTS (SELECT 1 FROM public.booking b JOIN public.facility f ON f.id = b.facility_id WHERE b.user_id = u.id AND analytics.sees(f.tenant_id))
   OR EXISTS (SELECT 1 FROM public.booking_participant p JOIN public.booking b ON b.id = p.booking_id JOIN public.facility f ON f.id = b.facility_id WHERE p.user_id = u.id AND analytics.sees(f.tenant_id))
   OR EXISTS (SELECT 1 FROM public.tenant_resident r WHERE r.user_id = u.id AND analytics.sees(r.tenant_id))
   OR EXISTS (SELECT 1 FROM public.tenant_membership m WHERE m.user_id = u.id AND analytics.sees(m.tenant_id));
COMMENT ON VIEW analytics.app_user IS 'Accounts, by id only (no name, email or phone): a council sees the people who booked or joined there, its residents and its staff. birth_year: for age groups';

CREATE VIEW analytics.tenant_membership WITH (security_barrier = TRUE) AS
SELECT user_id, tenant_id, role, created_at FROM public.tenant_membership WHERE analytics.sees(tenant_id);
COMMENT ON VIEW analytics.tenant_membership IS 'Council staff roles (admin)';

CREATE VIEW analytics.tenant_resident WITH (security_barrier = TRUE) AS
SELECT tenant_id, user_id, verified_by, verified_at FROM public.tenant_resident WHERE analytics.sees(tenant_id);
COMMENT ON VIEW analytics.tenant_resident IS 'People a council verified as its residents (one council per person)';

CREATE VIEW analytics.resident_request WITH (security_barrier = TRUE) AS
SELECT id, tenant_id, user_id, created_at FROM public.resident_request WHERE analytics.sees(tenant_id);
COMMENT ON VIEW analytics.resident_request IS 'Requests to be verified as a resident, still undecided (the identity data they carry is not here)';

CREATE VIEW analytics.tenant_invitation WITH (security_barrier = TRUE) AS
SELECT id, tenant_id, role, invited_by, created_at, accepted_at FROM public.tenant_invitation WHERE analytics.sees(tenant_id);
COMMENT ON VIEW analytics.tenant_invitation IS 'Invitations to administer a council';

CREATE VIEW analytics.platform_admin WITH (security_barrier = TRUE) AS
SELECT user_id, created_at FROM public.platform_admin WHERE analytics.sees_all();
COMMENT ON VIEW analytics.platform_admin IS 'Superadministrators (only for a superadministrator)';

-- --- the installation --------------------------------------------------------------

CREATE VIEW analytics.audit_log WITH (security_barrier = TRUE) AS
SELECT id, at, actor_id, actor_role, tenant_id, action, target_type, target_id, target_user_id
FROM public.audit_log
WHERE (tenant_id IS NOT NULL AND analytics.sees(tenant_id)) OR analytics.sees_all();
COMMENT ON VIEW analytics.audit_log IS 'Administrative changes: who (actor_id, actor_role), what (action, e.g. facility.updated) and on what';

CREATE VIEW analytics.app_settings WITH (security_barrier = TRUE) AS
SELECT one_council_per_account, updated_at FROM public.app_settings WHERE analytics.sees_all();
COMMENT ON VIEW analytics.app_settings IS 'Installation-wide settings (only for a superadministrator)';

-- Not here at all: tenant_smtp (relay settings and password) and user_qr_token (personal QR tokens).
