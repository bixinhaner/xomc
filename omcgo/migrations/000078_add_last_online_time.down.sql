-- 回滚：删除最后上线时间字段及相关索引

DROP INDEX IF EXISTS idx_devices_status_last_inform;
DROP INDEX IF EXISTS idx_device_info_last_online_time;
ALTER TABLE device_info DROP COLUMN IF EXISTS last_online_time;
