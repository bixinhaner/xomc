-- +goose Up
-- ============================================================
-- 000027_devices_boot_tracking.sql
-- 为 devices 表新增重启追踪字段：
--   - last_boot_at：设备最近一次重启完成时间（1 BOOT / M Reboot Inform）
--   - boot_count：设备生命周期内累计重启次数（频繁异常重启告警的计数源）
-- ============================================================

ALTER TABLE devices ADD COLUMN IF NOT EXISTS last_boot_at TIMESTAMPTZ;
ALTER TABLE devices ADD COLUMN IF NOT EXISTS boot_count INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_devices_last_boot_at ON devices (last_boot_at) WHERE last_boot_at IS NOT NULL;

COMMENT ON COLUMN devices.last_boot_at IS '设备最近一次重启完成时间（由 TR-069 Inform 1 BOOT / M Reboot 驱动）';
COMMENT ON COLUMN devices.boot_count IS '设备累计重启次数';

-- +goose Down
DROP INDEX IF EXISTS idx_devices_last_boot_at;
ALTER TABLE devices DROP COLUMN IF EXISTS boot_count;
ALTER TABLE devices DROP COLUMN IF EXISTS last_boot_at;
