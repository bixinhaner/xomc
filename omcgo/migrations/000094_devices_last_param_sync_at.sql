-- +goose Up
-- T-0124 F09 参数同步触发链 §2：周期性参数同步兜底字段
--
-- last_param_sync_at 由 HandleSyncResultPathB 在 BatchUpsert 成功后回写
-- （不区分触发源 device_online / periodic / firmware_changed / manual 统一口径）。
-- PeriodicSyncer 按 interval 扫描 last_param_sync_at NULL 或过期的 active 设备触发 Path B 同步。
ALTER TABLE devices ADD COLUMN IF NOT EXISTS last_param_sync_at TIMESTAMPTZ;

-- 部分索引：仅覆盖 active 设备（PeriodicSyncer 查询路径），NULLS FIRST 让从未同步过的设备优先入队。
CREATE INDEX IF NOT EXISTS idx_devices_last_param_sync
    ON devices (last_param_sync_at NULLS FIRST)
    WHERE status = 'active';

-- +goose Down
DROP INDEX IF EXISTS idx_devices_last_param_sync;
ALTER TABLE devices DROP COLUMN IF EXISTS last_param_sync_at;
