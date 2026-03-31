-- Add soft-delete support to devices table.
ALTER TABLE devices ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Partial index: only non-deleted devices need to be indexed for lookups.
CREATE INDEX IF NOT EXISTS idx_devices_deleted_at ON devices (deleted_at) WHERE deleted_at IS NOT NULL;
