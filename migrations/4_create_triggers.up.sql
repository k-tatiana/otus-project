-- 4_create_triggers.up.sql
-- Create triggers for loyalty system

-- Trigger to update loyalty level when points change
CREATE OR REPLACE FUNCTION update_loyalty_level()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.balance > 5000 THEN
        NEW.level_id := 4; -- Platinum
        RETURN NEW;
    ELSIF NEW.balance > 1000 THEN
        NEW.level_id := 3; -- Gold
        RETURN NEW;
    ELSIF NEW.balance > 500 THEN
        NEW.level_id := 2; -- Silver
        RETURN NEW;
    ELSE
        NEW.level_id := 1; -- Bronze
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_loyalty_level
    AFTER UPDATE OF balance ON loyalty
    FOR EACH ROW
    WHEN (OLD.balance IS DISTINCT FROM NEW.balance)
    EXECUTE FUNCTION update_loyalty_level();
