-- +goose Up
-- active_alarm_count 冗余列：由告警模块实时维护，避免 GIS 地图 33K 设备 COUNT(*) 子查询性能瓶颈。

ALTER TABLE device_info
    ADD COLUMN IF NOT EXISTS active_alarm_count integer NOT NULL DEFAULT 0;

-- 幂等回填：先全量置 0，再按 alarms_active 聚合写入正确值。
UPDATE device_info SET active_alarm_count = 0;

UPDATE device_info di
SET active_alarm_count = sub.cnt
FROM (
    SELECT aa.device_id, COUNT(*) AS cnt
    FROM alarms_active aa
    GROUP BY aa.device_id
) AS sub
WHERE di.device_id = sub.device_id;

-- 局部索引：只对有告警的设备建索引（GIS 卡片 "告警 > 0" 过滤场景）
CREATE INDEX IF NOT EXISTS idx_device_info_active_alarm_count
    ON device_info (active_alarm_count)
    WHERE active_alarm_count > 0;

-- +goose Down
DROP INDEX IF EXISTS idx_device_info_active_alarm_count;
ALTER TABLE device_info DROP COLUMN IF EXISTS active_alarm_count;
