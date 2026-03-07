DROP TRIGGER IF EXISTS trigger_devices_updated_at ON devices;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS devices CASCADE;
