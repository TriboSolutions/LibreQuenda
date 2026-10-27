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
