-- +goose Up
-- ============================================================
-- 000137_device_filter_dict_v2.sql
-- T-0162: 设备列表筛选字典 v2 — 与 DB 列值 1:1 对齐
--
-- 改动 4 类：
--   1) 删旧 conn_status 字典（被 is_online 替代，前端读 dict_is_online）
--   2) 修 op_state value 'active/inactive' → '1'/'0'，与 device_info.op_state 对齐
--   3) 修 network_type value 'eNB/gNB' → 'lte/nr'，与 devices.technology 对齐
--   4) 新增 5 个字典：lifecycle_state / is_online / device_model /
--      software_version / firmware_version
--      其中 device_model / firmware_version 初始化数据从 devices 表 distinct 灌入；
--      software_version 从 device_parameters 表 (TR-069 路径 Device.DeviceInfo.SoftwareVersion)
--      distinct 灌入（device_info 表无 software_version 列，软件版本走参数树）。
--      保证"用户在前端选了一项后端 filter 一定能命中至少一条设备"。
--
-- 设计文档：docs/design/device-lifecycle-online-status-decouple-20260520.md §4.2
-- ============================================================

-- ─── 1. 删旧 conn_status ─────────────────────────────────────────
DELETE FROM sys_dictionary_details WHERE sys_dictionary_id IN
    (SELECT id FROM sys_dictionaries WHERE type = 'conn_status');
DELETE FROM sys_dictionaries WHERE type = 'conn_status';

-- ─── 2. 修 op_state ──────────────────────────────────────────────
UPDATE sys_dictionary_details
SET value = '1' WHERE value = 'active'
  AND sys_dictionary_id IN (SELECT id FROM sys_dictionaries WHERE type = 'op_state');
UPDATE sys_dictionary_details
SET value = '0' WHERE value = 'inactive'
  AND sys_dictionary_id IN (SELECT id FROM sys_dictionaries WHERE type = 'op_state');

-- ─── 3. 修 network_type ─────────────────────────────────────────
UPDATE sys_dictionary_details
SET value = 'lte' WHERE value = 'eNB'
  AND sys_dictionary_id IN (SELECT id FROM sys_dictionaries WHERE type = 'network_type');
UPDATE sys_dictionary_details
SET value = 'nr' WHERE value = 'gNB'
  AND sys_dictionary_id IN (SELECT id FROM sys_dictionaries WHERE type = 'network_type');

-- ─── 4. 新增 5 个字典 type ──────────────────────────────────────
INSERT INTO sys_dictionaries (name, type, status, description) VALUES
    ('设备生命周期', 'lifecycle_state',  TRUE, 'T-0162: 业务流程进度，6 状态（discovered/registered/provisioning/commissioned/maintenance/decommissioned）'),
    ('设备在线状态', 'is_online',        TRUE, 'T-0162: 实时心跳活跃，true=在线 false=离线'),
    ('设备型号',     'device_model',     TRUE, 'T-0162: 设备硬件型号，对齐 devices.model_name；初始化从设备表 distinct 灌入'),
    ('软件版本',     'software_version', TRUE, 'T-0162: 软件版本号，TR-069 参数 Device.DeviceInfo.SoftwareVersion；初始化从 device_parameters 表 distinct 灌入'),
    ('固件版本',     'firmware_version', TRUE, 'T-0162: 固件版本号，对齐 devices.firmware_version；初始化从设备表 distinct 灌入')
ON CONFLICT DO NOTHING;

-- ─── 5. lifecycle_state 6 项明细 ─────────────────────────────────
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT v.label, v.value, v.sort, d.id
FROM sys_dictionaries d, (VALUES
    ('已发现',     'discovered',     1),
    ('已注册',     'registered',     2),
    ('配置中',     'provisioning',   3),
    ('已入网',     'commissioned',   4),
    ('维护中',     'maintenance',    5),
    ('已退役',     'decommissioned', 6)
) AS v(label, value, sort)
WHERE d.type = 'lifecycle_state'
ON CONFLICT DO NOTHING;

-- ─── 6. is_online 2 项明细 ──────────────────────────────────────
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT v.label, v.value, v.sort, d.id
FROM sys_dictionaries d, (VALUES
    ('在线', 'true',  1),
    ('离线', 'false', 2)
) AS v(label, value, sort)
WHERE d.type = 'is_online'
ON CONFLICT DO NOTHING;

-- ─── 7. device_model：从 devices.model_name distinct 灌入 ─────────
-- 用 ROW_NUMBER 给 sort 编号；TRIM/NULLIF 过滤空白与 NULL；ON CONFLICT
-- 配合 sys_dictionary_details 的 unique 约束（如存在）兜底重复
WITH d AS (
    SELECT DISTINCT NULLIF(TRIM(model_name), '') AS v
    FROM devices
    WHERE model_name IS NOT NULL AND TRIM(model_name) <> ''
)
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT d.v, d.v,
       ROW_NUMBER() OVER (ORDER BY d.v),
       sd.id
FROM d, sys_dictionaries sd
WHERE sd.type = 'device_model'
ON CONFLICT DO NOTHING;

-- ─── 8. software_version：从 device_parameters 表 TR-069 路径 distinct 灌入 ─
-- 设计文档原稿写 device_info.software_version 但该列不存在；软件版本作为 TR-069
-- 标准参数走参数树，路径 Device.DeviceInfo.SoftwareVersion 存在 device_parameters。
WITH s AS (
    SELECT DISTINCT NULLIF(TRIM(parameter_value), '') AS v
    FROM device_parameters
    WHERE parameter_path = 'Device.DeviceInfo.SoftwareVersion'
      AND parameter_value IS NOT NULL AND TRIM(parameter_value) <> ''
)
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT s.v, s.v,
       ROW_NUMBER() OVER (ORDER BY s.v),
       sd.id
FROM s, sys_dictionaries sd
WHERE sd.type = 'software_version'
ON CONFLICT DO NOTHING;

-- ─── 9. firmware_version：从 devices.firmware_version distinct 灌入 ─
WITH f AS (
    SELECT DISTINCT NULLIF(TRIM(firmware_version), '') AS v
    FROM devices
    WHERE firmware_version IS NOT NULL AND TRIM(firmware_version) <> ''
)
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT f.v, f.v,
       ROW_NUMBER() OVER (ORDER BY f.v),
       sd.id
FROM f, sys_dictionaries sd
WHERE sd.type = 'firmware_version'
ON CONFLICT DO NOTHING;


-- +goose Down
-- ============================================================
-- 反向：删 5 个新字典 + 还原 op_state/network_type 旧值 + 重建 conn_status
-- ============================================================

DELETE FROM sys_dictionary_details WHERE sys_dictionary_id IN
    (SELECT id FROM sys_dictionaries WHERE type IN
        ('lifecycle_state','is_online','device_model','software_version','firmware_version'));
DELETE FROM sys_dictionaries WHERE type IN
    ('lifecycle_state','is_online','device_model','software_version','firmware_version');

UPDATE sys_dictionary_details
SET value = 'active' WHERE value = '1'
  AND sys_dictionary_id IN (SELECT id FROM sys_dictionaries WHERE type = 'op_state');
UPDATE sys_dictionary_details
SET value = 'inactive' WHERE value = '0'
  AND sys_dictionary_id IN (SELECT id FROM sys_dictionaries WHERE type = 'op_state');

UPDATE sys_dictionary_details
SET value = 'eNB' WHERE value = 'lte'
  AND sys_dictionary_id IN (SELECT id FROM sys_dictionaries WHERE type = 'network_type');
UPDATE sys_dictionary_details
SET value = 'gNB' WHERE value = 'nr'
  AND sys_dictionary_id IN (SELECT id FROM sys_dictionaries WHERE type = 'network_type');

INSERT INTO sys_dictionaries (name, type, status, description) VALUES
    ('设备在线状态(旧)', 'conn_status', TRUE, 'T-0162 down: 重建供旧前端代码兼容；is_online 是新字典')
ON CONFLICT DO NOTHING;

INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT v.label, v.value, v.sort, d.id
FROM sys_dictionaries d, (VALUES
    ('在线', '1', 1),
    ('离线', '0', 2)
) AS v(label, value, sort)
WHERE d.type = 'conn_status'
ON CONFLICT DO NOTHING;
