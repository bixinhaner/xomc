-- Rollback G28: Remove 4 quick-query columns from device_info
DROP INDEX IF EXISTS idx_device_info_license_status;
DROP INDEX IF EXISTS idx_device_info_alarm_severity;
DROP INDEX IF EXISTS idx_device_info_gps_status;

ALTER TABLE device_info
    DROP COLUMN IF EXISTS license_status,
    DROP COLUMN IF EXISTS alarm_severity,
    DROP COLUMN IF EXISTS gps_status,
    DROP COLUMN IF EXISTS num_of_cells;
