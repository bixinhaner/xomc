-- +goose Up
-- 迁移 000013: 告警管理增强
-- 创建告警库、告警库国际化、告警过滤规则表
-- 增强活动告警和历史告警表

-- 1. 创建告警库表
CREATE TABLE alarm_libraries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alarm_code      VARCHAR(128) NOT NULL UNIQUE,
    alarm_source    VARCHAR(64) NOT NULL,
    event_type      VARCHAR(64) NOT NULL,
    severity        SMALLINT NOT NULL,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    probable_cause  TEXT NOT NULL,
    explanation     TEXT,
    additional_info JSONB DEFAULT '{}',
    carrier         VARCHAR(4),
    technology      VARCHAR(16),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_alarm_libraries_alarm_source ON alarm_libraries(alarm_source);
CREATE INDEX IF NOT EXISTS idx_alarm_libraries_alarm_code ON alarm_libraries(alarm_code);
CREATE INDEX IF NOT EXISTS idx_alarm_libraries_severity ON alarm_libraries(severity);
CREATE INDEX IF NOT EXISTS idx_alarm_libraries_carrier ON alarm_libraries(carrier) WHERE carrier IS NOT NULL;

-- 2. 创建告警库国际化表
CREATE TABLE alarm_library_i18n (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    library_id      UUID NOT NULL REFERENCES alarm_libraries(id) ON DELETE CASCADE,
    locale          VARCHAR(16) NOT NULL,
    probable_cause  TEXT NOT NULL,
    explanation     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(library_id, locale)
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_alarm_library_i18n_library_id ON alarm_library_i18n(library_id);
CREATE INDEX IF NOT EXISTS idx_alarm_library_i18n_locale ON alarm_library_i18n(locale);

-- 3. 活动告警表增强
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS device_name VARCHAR(128);
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS technology VARCHAR(16);
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS alarm_source VARCHAR(64);
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS event_type VARCHAR(64);
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS alarm_type VARCHAR(32);
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS probable_cause TEXT NOT NULL DEFAULT '';
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS network_location TEXT;
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS explicit_cause TEXT;
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS is_read BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS ack_count INT NOT NULL DEFAULT 0;
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS first_raised_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- 索引
CREATE INDEX IF NOT EXISTS idx_alarms_active_device_name ON alarms_active(device_name);
CREATE INDEX IF NOT EXISTS idx_alarms_active_is_read ON alarms_active(is_read);
CREATE INDEX IF NOT EXISTS idx_alarms_active_alarm_type ON alarms_active(alarm_type);

-- 4. 历史告警表增强
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS device_name VARCHAR(128);
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS technology VARCHAR(16);
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS alarm_source VARCHAR(64);
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS event_type VARCHAR(64);
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS alarm_type VARCHAR(32);
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS network_location TEXT;
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS explicit_cause TEXT;
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS ack_count INT NOT NULL DEFAULT 0;
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS acknowledged_by VARCHAR(128);
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- 索引
CREATE INDEX IF NOT EXISTS idx_alarms_history_alarm_type ON alarms_history(alarm_type);

-- 5. 创建告警过滤规则表
CREATE TABLE alarm_filters (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(255) NOT NULL,
    filter_type         VARCHAR(32) NOT NULL,
    alarm_sources       VARCHAR(64)[] DEFAULT '{}',
    alarm_codes         VARCHAR(128)[] DEFAULT '{}',
    device_ids          UUID[] DEFAULT '{}',
    device_group_ids    UUID[] DEFAULT '{}',
    action              VARCHAR(32) NOT NULL,
    acknowledge_desc    TEXT,
    priority            INT NOT NULL DEFAULT 0,
    enabled             BOOLEAN NOT NULL DEFAULT TRUE,
    created_by          VARCHAR(128),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          VARCHAR(128),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_alarm_filters_enabled_priority ON alarm_filters(enabled, priority);
CREATE INDEX IF NOT EXISTS idx_alarm_filters_action ON alarm_filters(action);
CREATE INDEX IF NOT EXISTS idx_alarm_filters_alarm_sources ON alarm_filters USING GIN(alarm_sources);
CREATE INDEX IF NOT EXISTS idx_alarm_filters_alarm_codes ON alarm_filters USING GIN(alarm_codes);
CREATE INDEX IF NOT EXISTS idx_alarm_filters_device_ids ON alarm_filters USING GIN(device_ids);
CREATE INDEX IF NOT EXISTS idx_alarm_filters_device_group_ids ON alarm_filters USING GIN(device_group_ids);

-- 6. 种子数据 - 告警库记录（中文为主）
INSERT INTO alarm_libraries (
    alarm_code, alarm_source, event_type, severity, enabled, probable_cause, explanation, additional_info
) VALUES
('DEVICE_OFFLINE', 'Device', 'communications', 2, true,
 '设备离线', '设备与网管系统之间的连接中断',
 '{"category": "connectivity", "impact": "high"}'),
('DEVICE_RESTART', 'Device', 'equipment', 3, true,
 '设备重启', '设备已重新启动',
 '{"category": "maintenance", "impact": "medium"}'),
('LINK_FAILURE', 'Device', 'communications', 2, true,
 '链路中断', '基站回传链路异常中断',
 '{"category": "connectivity", "impact": "high"}'),
('S1_INTERFACE_ERROR', 'TR069', 'communications', 2, true,
 'S1接口异常', 'TR069 ACS与CPE的S1接口通信异常',
 '{"category": "protocol", "impact": "medium"}'),
('CPU_OVERLOAD', 'Device', 'processing', 3, true,
 'CPU过载', '设备CPU使用率超过阈值',
 '{"category": "performance", "impact": "medium", "threshold": 80}'),
('MEMORY_OVERLOAD', 'Device', 'processing', 3, true,
 '内存过载', '设备内存使用率超过阈值',
 '{"category": "performance", "impact": "medium", "threshold": 90}'),
('TEMP_HIGH', 'Device', 'environment', 3, true,
 '温度过高', '设备温度超过告警阈值',
 '{"category": "environment", "impact": "medium", "threshold": 70}'),
('POWER_FAILURE', 'Device', 'power', 1, true,
 '电源故障', '设备供电中断或电源模块故障',
 '{"category": "power", "impact": "high"}'),
('FIRMWARE_UPGRADE_FAILED', 'Device', 'software', 2, true,
 '固件升级失败', '设备固件升级过程中发生错误',
 '{"category": "software", "impact": "medium"}'),
('GPS_LOSS', 'Device', 'environment', 4, true,
 'GPS信号丢失', '设备GPS模块无法获取卫星信号',
 '{"category": "positioning", "impact": "low"}'),
('VSWR_MISMATCH', 'Device', 'quality', 3, true,
 'VSWR不匹配', 'VSWR值超出正常范围',
 '{"category": "rf", "impact": "medium", "threshold": 1.5}'),
('RSRP_LOW', 'Device', 'quality', 4, true,
 'RSRP偏低', '参考信号接收功率低于阈值',
 '{"category": "rf", "impact": "low", "threshold": -105}'),
('SINR_LOW', 'Device', 'quality', 4, true,
 'SINR偏低', '信号与干扰噪声比低于阈值',
 '{"category": "rf", "impact": "low", "threshold": -3}'),
('PCI_CONFLICT', 'Device', 'quality', 3, true,
 'PCI冲突', '物理小区标识冲突',
 '{"category": "rf", "impact": "medium"}'),
('NEIGHBOR_CELL', 'Device', 'configuration', 4, true,
 '邻区检测', '检测到未配置的邻区关系',
 '{"category": "rf", "impact": "low"}')
ON CONFLICT (alarm_code) DO NOTHING;

-- 7. 插入英文国际化数据
INSERT INTO alarm_library_i18n (library_id, locale, probable_cause, explanation)
SELECT
    al.id,
    'en-US',
    CASE al.alarm_code
        WHEN 'DEVICE_OFFLINE' THEN 'Device Offline'
        WHEN 'DEVICE_RESTART' THEN 'Device Restart'
        WHEN 'LINK_FAILURE' THEN 'Link Failure'
        WHEN 'S1_INTERFACE_ERROR' THEN 'S1 Interface Error'
        WHEN 'CPU_OVERLOAD' THEN 'CPU Overload'
        WHEN 'MEMORY_OVERLOAD' THEN 'Memory Overload'
        WHEN 'TEMP_HIGH' THEN 'High Temperature'
        WHEN 'POWER_FAILURE' THEN 'Power Failure'
        WHEN 'FIRMWARE_UPGRADE_FAILED' THEN 'Firmware Upgrade Failed'
        WHEN 'GPS_LOSS' THEN 'GPS Signal Loss'
        WHEN 'VSWR_MISMATCH' THEN 'VSWR Mismatch'
        WHEN 'RSRP_LOW' THEN 'Low RSRP'
        WHEN 'SINR_LOW' THEN 'Low SINR'
        WHEN 'PCI_CONFLICT' THEN 'PCI Conflict'
        WHEN 'NEIGHBOR_CELL' THEN 'Neighbor Cell Detected'
    END,
    CASE al.alarm_code
        WHEN 'DEVICE_OFFLINE' THEN 'Connection between device and NMS is interrupted'
        WHEN 'DEVICE_RESTART' THEN 'Device has been restarted'
        WHEN 'LINK_FAILURE' THEN 'Backhaul link is abnormally interrupted'
        WHEN 'S1_INTERFACE_ERROR' THEN 'TR069 S1 interface communication error between ACS and CPE'
        WHEN 'CPU_OVERLOAD' THEN 'Device CPU usage exceeds threshold'
        WHEN 'MEMORY_OVERLOAD' THEN 'Device memory usage exceeds threshold'
        WHEN 'TEMP_HIGH' THEN 'Device temperature exceeds alarm threshold'
        WHEN 'POWER_FAILURE' THEN 'Device power supply interrupted or power module failure'
        WHEN 'FIRMWARE_UPGRADE_FAILED' THEN 'Error occurred during device firmware upgrade'
        WHEN 'GPS_LOSS' THEN 'GPS module cannot acquire satellite signal'
        WHEN 'VSWR_MISMATCH' THEN 'VSWR value is out of normal range'
        WHEN 'RSRP_LOW' THEN 'Reference Signal Received Power is below threshold'
        WHEN 'SINR_LOW' THEN 'Signal to Interference plus Noise Ratio is below threshold'
        WHEN 'PCI_CONFLICT' THEN 'Physical Cell Identifier conflict detected'
        WHEN 'NEIGHBOR_CELL' THEN 'Unconfigured neighbor cell relation detected'
    END
FROM alarm_libraries al
ON CONFLICT (library_id, locale) DO NOTHING;

-- +goose Down
-- 迁移 000013: 告警管理增强 - 回滚
-- 删除新创建的表和列

-- 1. 删除告警过滤规则表
DROP TABLE IF EXISTS alarm_filters CASCADE;

-- 2. 删除活动告警表新增列
ALTER TABLE alarms_active
    DROP COLUMN IF EXISTS device_name,
    DROP COLUMN IF EXISTS technology,
    DROP COLUMN IF EXISTS alarm_source,
    DROP COLUMN IF EXISTS event_type,
    DROP COLUMN IF EXISTS alarm_type,
    DROP COLUMN IF EXISTS probable_cause,
    DROP COLUMN IF EXISTS network_location,
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS explicit_cause,
    DROP COLUMN IF EXISTS is_read,
    DROP COLUMN IF EXISTS ack_count,
    DROP COLUMN IF EXISTS first_raised_at,
    DROP COLUMN IF EXISTS last_updated_at;

-- 3. 删除历史告警表新增列
ALTER TABLE alarms_history
    DROP COLUMN IF EXISTS device_name,
    DROP COLUMN IF EXISTS technology,
    DROP COLUMN IF EXISTS alarm_source,
    DROP COLUMN IF EXISTS event_type,
    DROP COLUMN IF EXISTS alarm_type,
    DROP COLUMN IF EXISTS network_location,
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS explicit_cause,
    DROP COLUMN IF EXISTS ack_count;

-- 4. 删除告警库国际化表
DROP TABLE IF EXISTS alarm_library_i18n CASCADE;

-- 5. 删除告警库表
DROP TABLE IF EXISTS alarm_libraries CASCADE;
