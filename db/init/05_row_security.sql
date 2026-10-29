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

-- facility_tag: the catalogue (tag, tag_category) belongs to the platform; which tags a facility carries, to its council
ALTER TABLE facility_tag ENABLE ROW LEVEL SECURITY;
CREATE POLICY facility_tag_read   ON facility_tag FOR SELECT USING (TRUE);
CREATE POLICY facility_tag_insert ON facility_tag FOR INSERT WITH CHECK (app_owns_facility(facility_id));
CREATE POLICY facility_tag_delete ON facility_tag FOR DELETE USING (app_owns_facility(facility_id));

-- tenant_smtp: read by the email job for every council; the row is sealed, and the API never sends the password back
ALTER TABLE tenant_smtp ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_smtp_read   ON tenant_smtp FOR SELECT USING (TRUE);
CREATE POLICY tenant_smtp_insert ON tenant_smtp FOR INSERT WITH CHECK (tenant_id = app_tenant_id());
CREATE POLICY tenant_smtp_update ON tenant_smtp FOR UPDATE USING (tenant_id = app_tenant_id()) WITH CHECK (tenant_id = app_tenant_id());
CREATE POLICY tenant_smtp_delete ON tenant_smtp FOR DELETE USING (tenant_id = app_tenant_id());
