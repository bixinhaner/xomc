-- 新增最后上线时间字段，用于记录设备最近一次上线（从离线变在线）的时间
-- 与 first_online_time（首次上线时间）区分：
--   - first_online_time: 设备生命周期内只记录一次
--   - last_online_time: 每次设备从离线变为在线时更新

ALTER TABLE device_info
    ADD COLUMN IF NOT EXISTS last_online_time TIMESTAMPTZ;

COMMENT ON COLUMN device_info.last_online_time IS '最后上线时间（设备从离线变为在线的时间）';

-- 索引（用于按上线时间筛选）
CREATE INDEX IF NOT EXISTS idx_device_info_last_online_time
    ON device_info (last_online_time);

-- 初始化数据：将 first_online_time 复制到 last_online_time（历史数据兼容）
-- 仅对当前在线设备执行此初始化
UPDATE device_info di
SET last_online_time = di.first_online_time
FROM devices d
WHERE di.device_id = d.id
  AND di.last_online_time IS NULL
  AND di.first_online_time IS NOT NULL
  AND d.status = 'active';

-- 离线检测查询优化索引
CREATE INDEX IF NOT EXISTS idx_devices_status_last_inform
    ON devices (status, last_inform_at)
    WHERE deleted_at IS NULL AND status = 'active';
