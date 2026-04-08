-- Remove recycle bin metadata from devices table.

DROP INDEX IF EXISTS idx_devices_deleted_at_not_null;
ALTER TABLE devices DROP COLUMN IF EXISTS deleted_by;
