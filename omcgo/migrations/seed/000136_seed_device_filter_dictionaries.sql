-- ============================================================
-- 000136_seed_device_filter_dictionaries.sql
-- 设备列表筛选条件的字典种子数据：在线状态 / 激活状态 / 网络制式。
--
-- 业务来源：docs/design/device-list-and-group-improvements-20260520.md §3 R6c
--
-- 约束：
--   · ON CONFLICT 兜底防重复（已有 type 同名直接保留）
--   · `conn_status` 只放 online(1) / offline(0) — UI 不再暴露"同步中""同步失败"（D4）
--   · `op_state` 用字符串 active/inactive — 与后端 `device_info.op_state` 取值对齐
--   · `network_type` 用 eNB / gNB — 与设备表 network_type 字符串一致
--   · `product_type` 已由 seed/000004_mml_enhance.sql 注入，本处不重复
-- ============================================================

-- +goose Up

-- 1. 上层字典（type 唯一）
INSERT INTO sys_dictionaries (name, type, status, description) VALUES
    ('设备在线状态', 'conn_status', TRUE, '设备列表筛选 — connStatus 取值：1=在线, 0=离线'),
    ('设备激活状态', 'op_state',    TRUE, '设备列表筛选 — opState 取值：active=激活, inactive=未激活'),
    ('设备网络制式', 'network_type',TRUE, '设备列表筛选 — networkType 取值：eNB=LTE, gNB=NR')
ON CONFLICT DO NOTHING;

-- 2. 字典明细 — 在线状态（仅两项，UI 不暴露"同步中""同步失败"）
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT '在线', '1', 1, id FROM sys_dictionaries WHERE type = 'conn_status'
ON CONFLICT DO NOTHING;

INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT '离线', '0', 2, id FROM sys_dictionaries WHERE type = 'conn_status'
ON CONFLICT DO NOTHING;

-- 3. 字典明细 — 激活状态
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT '激活',   'active',   1, id FROM sys_dictionaries WHERE type = 'op_state'
ON CONFLICT DO NOTHING;

INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT '未激活', 'inactive', 2, id FROM sys_dictionaries WHERE type = 'op_state'
ON CONFLICT DO NOTHING;

-- 4. 字典明细 — 网络制式
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT 'eNB (LTE)', 'eNB', 1, id FROM sys_dictionaries WHERE type = 'network_type'
ON CONFLICT DO NOTHING;

INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT 'gNB (NR)',  'gNB', 2, id FROM sys_dictionaries WHERE type = 'network_type'
ON CONFLICT DO NOTHING;


-- +goose Down

-- 先删明细（CASCADE 也行，但显式 DELETE 更清晰）
DELETE FROM sys_dictionary_details
    WHERE sys_dictionary_id IN (
        SELECT id FROM sys_dictionaries
            WHERE type IN ('conn_status', 'op_state', 'network_type')
    );

DELETE FROM sys_dictionaries
    WHERE type IN ('conn_status', 'op_state', 'network_type');
