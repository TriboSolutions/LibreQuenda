-- ============================================================
--  ROW-LEVEL SECURITY: tables the councils manage
--  A council's facilities, opening hours, closures, check-in readers and SMTP relay
--  can only be written inside a transaction that names that council:
--      SELECT set_config('app.tenant_id', '<tenant uuid>', true);
--  (db.WithTenant in the API). A superadministrator editing a council goes
--  through the same path, so the same rule applies.
--  Reads stay open: facilities, hours and closures are public, and the API
--  filters every endpoint.
--  The table owner (POSTGRES_USER) is exempt: init scripts, seed data and
--  the ON DELETE CASCADE from tenant run as the owner.
-- ============================================================

-- The council named by the current transaction, or NULL
CREATE FUNCTION app_tenant_id() RETURNS UUID
LANGUAGE sql STABLE AS $$
    SELECT NULLIF(current_setting('app.tenant_id', true), '')::UUID
$$;

-- Whether a facility belongs to the council named by the current transaction
CREATE FUNCTION app_owns_facility(facility_id UUID) RETURNS BOOLEAN
LANGUAGE sql STABLE AS $$
    SELECT EXISTS (
        SELECT 1 FROM facility f WHERE f.id = facility_id AND f.tenant_id = app_tenant_id()
    )
$$;


-- facility
ALTER TABLE facility ENABLE ROW LEVEL SECURITY;
CREATE POLICY facility_read   ON facility FOR SELECT USING (TRUE);
CREATE POLICY facility_insert ON facility FOR INSERT WITH CHECK (tenant_id = app_tenant_id());
CREATE POLICY facility_update ON facility FOR UPDATE USING (tenant_id = app_tenant_id()) WITH CHECK (tenant_id = app_tenant_id());
CREATE POLICY facility_delete ON facility FOR DELETE USING (tenant_id = app_tenant_id());

-- opening_hours
ALTER TABLE opening_hours ENABLE ROW LEVEL SECURITY;
CREATE POLICY opening_hours_read   ON opening_hours FOR SELECT USING (TRUE);
CREATE POLICY opening_hours_insert ON opening_hours FOR INSERT WITH CHECK (app_owns_facility(facility_id));
CREATE POLICY opening_hours_update ON opening_hours FOR UPDATE USING (app_owns_facility(facility_id)) WITH CHECK (app_owns_facility(facility_id));
CREATE POLICY opening_hours_delete ON opening_hours FOR DELETE USING (app_owns_facility(facility_id));

-- facility_issue: residents report through the facility's council (db.WithTenant
-- with the facility's council, after the API checked the person), staff update them
ALTER TABLE facility_issue ENABLE ROW LEVEL SECURITY;
CREATE POLICY facility_issue_read   ON facility_issue FOR SELECT USING (TRUE);
CREATE POLICY facility_issue_insert ON facility_issue FOR INSERT WITH CHECK (app_owns_facility(facility_id));
CREATE POLICY facility_issue_update ON facility_issue FOR UPDATE USING (app_owns_facility(facility_id)) WITH CHECK (app_owns_facility(facility_id));

-- closure
ALTER TABLE closure ENABLE ROW LEVEL SECURITY;
CREATE POLICY closure_read   ON closure FOR SELECT USING (TRUE);
CREATE POLICY closure_insert ON closure FOR INSERT WITH CHECK (app_owns_facility(facility_id));
CREATE POLICY closure_update ON closure FOR UPDATE USING (app_owns_facility(facility_id)) WITH CHECK (app_owns_facility(facility_id));
CREATE POLICY closure_delete ON closure FOR DELETE USING (app_owns_facility(facility_id));

-- checkin_device: a reader signing in acts for its own council
ALTER TABLE checkin_device ENABLE ROW LEVEL SECURITY;
CREATE POLICY checkin_device_read   ON checkin_device FOR SELECT USING (TRUE);
CREATE POLICY checkin_device_insert ON checkin_device FOR INSERT WITH CHECK (app_owns_facility(facility_id));
CREATE POLICY checkin_device_update ON checkin_device FOR UPDATE USING (app_owns_facility(facility_id)) WITH CHECK (app_owns_facility(facility_id));
CREATE POLICY checkin_device_delete ON checkin_device FOR DELETE USING (app_owns_facility(facility_id));

-- facility_checkin_period: read by the indicators; written only by refresh_checkin_period (03A_triggers.sql)
ALTER TABLE facility_checkin_period ENABLE ROW LEVEL SECURITY;
CREATE POLICY facility_checkin_period_read ON facility_checkin_period FOR SELECT USING (TRUE);

-- facility_checkin: read when a resident checks in, written by the council
ALTER TABLE facility_checkin ENABLE ROW LEVEL SECURITY;
CREATE POLICY facility_checkin_read   ON facility_checkin FOR SELECT USING (TRUE);
CREATE POLICY facility_checkin_insert ON facility_checkin FOR INSERT WITH CHECK (app_owns_facility(facility_id));
CREATE POLICY facility_checkin_update ON facility_checkin FOR UPDATE USING (app_owns_facility(facility_id)) WITH CHECK (app_owns_facility(facility_id));

-- facility_tag: the catalogue (tag, tag_category) belongs to the platform; which tags a facility carries, to its council
ALTER TABLE facility_tag ENABLE ROW LEVEL SECURITY;
CREATE POLICY facility_tag_read   ON facility_tag FOR SELECT USING (TRUE);
CREATE POLICY facility_tag_insert ON facility_tag FOR INSERT WITH CHECK (app_owns_facility(facility_id));
CREATE POLICY facility_tag_delete ON facility_tag FOR DELETE USING (app_owns_facility(facility_id));

-- tenant_booking_rules: the council's booking window, read when booking, written by the council
ALTER TABLE tenant_booking_rules ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_booking_rules_read   ON tenant_booking_rules FOR SELECT USING (TRUE);
CREATE POLICY tenant_booking_rules_insert ON tenant_booking_rules FOR INSERT WITH CHECK (tenant_id = app_tenant_id());
CREATE POLICY tenant_booking_rules_update ON tenant_booking_rules FOR UPDATE USING (tenant_id = app_tenant_id()) WITH CHECK (tenant_id = app_tenant_id());

-- tenant_smtp: read by the email job for every council; the row is sealed, and the API never sends the password back
ALTER TABLE tenant_smtp ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_smtp_read   ON tenant_smtp FOR SELECT USING (TRUE);
CREATE POLICY tenant_smtp_insert ON tenant_smtp FOR INSERT WITH CHECK (tenant_id = app_tenant_id());
CREATE POLICY tenant_smtp_update ON tenant_smtp FOR UPDATE USING (tenant_id = app_tenant_id()) WITH CHECK (tenant_id = app_tenant_id());
CREATE POLICY tenant_smtp_delete ON tenant_smtp FOR DELETE USING (tenant_id = app_tenant_id());

-- council_notice, poll, poll_option: the council's announcements and questions,
-- read on its public pages, written by the council. Votes (poll_vote) are the
-- residents' own, written after the API checked the person.
ALTER TABLE council_notice ENABLE ROW LEVEL SECURITY;
CREATE POLICY council_notice_read   ON council_notice FOR SELECT USING (TRUE);
CREATE POLICY council_notice_insert ON council_notice FOR INSERT WITH CHECK (tenant_id = app_tenant_id());
CREATE POLICY council_notice_update ON council_notice FOR UPDATE USING (tenant_id = app_tenant_id()) WITH CHECK (tenant_id = app_tenant_id());
CREATE POLICY council_notice_delete ON council_notice FOR DELETE USING (tenant_id = app_tenant_id());

ALTER TABLE poll ENABLE ROW LEVEL SECURITY;
CREATE POLICY poll_read   ON poll FOR SELECT USING (TRUE);
CREATE POLICY poll_insert ON poll FOR INSERT WITH CHECK (tenant_id = app_tenant_id());
CREATE POLICY poll_update ON poll FOR UPDATE USING (tenant_id = app_tenant_id()) WITH CHECK (tenant_id = app_tenant_id());
CREATE POLICY poll_delete ON poll FOR DELETE USING (tenant_id = app_tenant_id());

ALTER TABLE poll_option ENABLE ROW LEVEL SECURITY;
CREATE POLICY poll_option_read   ON poll_option FOR SELECT USING (TRUE);
CREATE POLICY poll_option_insert ON poll_option FOR INSERT WITH CHECK (EXISTS (SELECT 1 FROM poll p WHERE p.id = poll_id AND p.tenant_id = app_tenant_id()));

-- audit_log: append-only for the API. A council's administrators read it
-- inside db.WithTenant and see only their council; superadministrators read
-- it without a council and see everything. Old entries go only through
-- purge_audit_log(), which runs as the owner.
ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
CREATE POLICY audit_log_read   ON audit_log FOR SELECT USING (app_tenant_id() IS NULL OR tenant_id = app_tenant_id());
CREATE POLICY audit_log_insert ON audit_log FOR INSERT WITH CHECK (app_tenant_id() IS NULL OR tenant_id = app_tenant_id());
REVOKE UPDATE, DELETE, TRUNCATE ON audit_log FROM librequenda_app;

-- The retention purge: the only way entries leave the log.
CREATE FUNCTION purge_audit_log(older_than INTERVAL) RETURNS BIGINT
LANGUAGE sql SECURITY DEFINER SET search_path = public AS $$
    WITH gone AS (DELETE FROM audit_log WHERE at < NOW() - older_than RETURNING 1)
    SELECT COUNT(*) FROM gone
$$;
REVOKE ALL ON FUNCTION purge_audit_log(INTERVAL) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION purge_audit_log(INTERVAL) TO librequenda_app;
