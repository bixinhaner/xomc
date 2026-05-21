-- +goose Up
-- ============================================================
-- 000138_software_version_seed.sql
-- T-0162 后续：填充 software_version 字典与对应设备数据
--
-- 背景：seed/000137 §8 想从 device_parameters (Device.DeviceInfo.SoftwareVersion)
-- distinct 灌字典，但 dev DB 这张表常态为空（设备 inform 还没真正上来），
-- 结果字典 0 条 → 前端"软件版本"下拉始终为空。
--
-- 策略：
--   1) 选 4 个与现有 firmware 风格匹配的 SW 版本号
--   2) 给前 400 台 active 设备按 ROW_NUMBER mod 4 round-robin 写一条
--      device_parameters 行（每版本 ~100 台），保证字典任意一项都对应
--      至少 ~100 台设备 —— 用户在前端选了一项后端 filter 一定命中
--   3) 把 4 个版本号 INSERT 进 software_version 字典明细
--
-- 全部 ON CONFLICT DO NOTHING，可重复执行
-- ============================================================

-- ─── 1. device_parameters 灌入 ───────────────────────────────────
-- 取活跃设备（commissioned + online，无则放宽到 registered/active），按
-- serial_number 排序，ROW_NUMBER mod 4 round-robin 分配 4 个版本号。
-- 限 400 台，每版本 100 台，避免大量分区写入影响后续 schema 测试。
-- 这里写的 path 与 device_info_pg_repository.go L237-244 的 software_version
-- 子查询路径 'Device.DeviceInfo.SoftwareVersion' 严格一致。
WITH picked AS (
    SELECT id, ROW_NUMBER() OVER (ORDER BY serial_number) AS rn
    FROM devices
    WHERE deleted_at IS NULL
    LIMIT 400
)
INSERT INTO device_parameters (
    device_id, parameter_path, parameter_value,
    parameter_type, writable, last_updated_at, fap_instance, param_group
)
SELECT
    id,
    'Device.DeviceInfo.SoftwareVersion',
    CASE rn % 4
        WHEN 0 THEN 'BaiBNX-V13.0.1.3'
        WHEN 1 THEN 'BaiBNX-V12.3.0.5'
        WHEN 2 THEN 'V100R019C10'
        WHEN 3 THEN 'V4.16.30P4'
    END,
    'string',
    FALSE,
    NOW(),
    0,
    'device-info'
FROM picked
ON CONFLICT (device_id, parameter_path) DO NOTHING;

-- ─── 2. software_version 字典明细 ────────────────────────────────
-- 直接写 4 项；不再依赖 device_parameters distinct 灌入。每项已对应 ~100
-- 台设备（步骤 1 保证），前端任意选一项 backend filter 都能命中。
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT v.label, v.value, v.sort, d.id
FROM sys_dictionaries d, (VALUES
    ('BaiBNX-V13.0.1.3', 'BaiBNX-V13.0.1.3', 1),
    ('BaiBNX-V12.3.0.5', 'BaiBNX-V12.3.0.5', 2),
    ('V100R019C10',      'V100R019C10',      3),
    ('V4.16.30P4',       'V4.16.30P4',       4)
) AS v(label, value, sort)
WHERE d.type = 'software_version'
ON CONFLICT DO NOTHING;


-- +goose Down
-- ============================================================
-- 反向：删字典 4 项 + 删对应 device_parameters 行（仅清掉本 seed 写入的，
-- 不动设备 inform 真实上来后写入的同路径数据）
-- ============================================================

DELETE FROM sys_dictionary_details
WHERE value IN ('BaiBNX-V13.0.1.3', 'BaiBNX-V12.3.0.5', 'V100R019C10', 'V4.16.30P4')
  AND sys_dictionary_id IN (SELECT id FROM sys_dictionaries WHERE type = 'software_version');

DELETE FROM device_parameters
WHERE parameter_path = 'Device.DeviceInfo.SoftwareVersion'
  AND parameter_value IN ('BaiBNX-V13.0.1.3', 'BaiBNX-V12.3.0.5', 'V100R019C10', 'V4.16.30P4')
  AND param_group = 'device-info';
