-- Add recycle bin metadata to devices table.
-- These fields track who deleted a device and when, for the recycle bin feature.

ALTER TABLE devices ADD COLUMN IF NOT EXISTS deleted_by TEXT;

-- Create index for recycle bin queries (deleted devices)
CREATE INDEX IF NOT EXISTS idx_devices_deleted_at_not_null ON devices (deleted_at) WHERE deleted_at IS NOT NULL;
