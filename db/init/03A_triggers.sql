-- ============================================================
--  AUTO-UPDATE updated_at
--  One shared function + one BEFORE UPDATE trigger per table.
-- ============================================================

CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- tenant
CREATE TRIGGER set_updated_at BEFORE UPDATE ON tenant
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- app_user
CREATE TRIGGER set_updated_at BEFORE UPDATE ON app_user
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- facility
CREATE TRIGGER set_updated_at BEFORE UPDATE ON facility
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- booking
CREATE TRIGGER set_updated_at BEFORE UPDATE ON booking
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- ============================================================
--  CHECK-IN HISTORY (facility_checkin_period)
--  Opens a stretch when a facility starts being able to take check-ins and
--  closes it when it stops, after any change to its readers or its check-in
--  settings. SECURITY DEFINER: the table has no write policy, so only this
--  writes it, whatever council the change runs as.
-- ============================================================

CREATE FUNCTION refresh_checkin_period(target UUID) RETURNS VOID
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public AS $$
DECLARE
    open_id UUID;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM facility WHERE id = target) THEN
        RETURN; -- the facility is being deleted: its history goes with it
    END IF;
    SELECT id INTO open_id FROM facility_checkin_period WHERE facility_id = target AND ends_at IS NULL;
    IF facility_can_check_in(target) THEN
        IF open_id IS NULL THEN
            INSERT INTO facility_checkin_period (facility_id, starts_at) VALUES (target, NOW());
        END IF;
    ELSIF open_id IS NOT NULL THEN
        UPDATE facility_checkin_period SET ends_at = GREATEST(NOW(), starts_at) WHERE id = open_id;
    END IF;
END;
$$;

CREATE FUNCTION trigger_refresh_checkin_period()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM refresh_checkin_period(CASE WHEN TG_OP = 'DELETE' THEN OLD.facility_id ELSE NEW.facility_id END);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER refresh_checkin_period AFTER INSERT OR DELETE ON checkin_device
    FOR EACH ROW EXECUTE FUNCTION trigger_refresh_checkin_period();

CREATE TRIGGER refresh_checkin_period AFTER INSERT OR UPDATE ON facility_checkin
    FOR EACH ROW EXECUTE FUNCTION trigger_refresh_checkin_period();
