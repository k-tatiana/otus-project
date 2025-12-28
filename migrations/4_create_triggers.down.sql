-- 4_create_triggers.down.sql
-- Drop triggers for loyalty system

DROP TRIGGER IF EXISTS trigger_update_loyalty_level ON loyalty;
DROP FUNCTION IF EXISTS update_loyalty_level();