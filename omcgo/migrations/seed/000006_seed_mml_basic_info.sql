-- +goose Up
-- ============================================================
-- 000006_seed_mml_basic_info.sql
-- 新增"基本信息"分类 + LST/MOD DEVICE_INFO 命令
--
-- 历史说明：本文件原本还包含 mml_sub_commands / mml_command_subcommand_rel
-- 的插入逻辑（子命令 14 条 + 2 条命令关联）。迁移 000026_mml_table_restructure
-- 已经：
--   1) 删除 mml_sub_commands 表
--   2) 删除 mml_command_subcommand_rel 表
--   3) 新建 mml_command_params_rel（命令直接关联 mml_params）
--
-- 因此本文件只保留"分类字典"和"命令"两个 INSERT；"命令 ↔ 参数"的绑定统一交给
-- 000007_seed_mml_basic_info_params.sql（它基于 tr069_path 直接 JOIN mml_params
-- 插入 mml_command_params_rel）。
-- ============================================================

-- Step 1: 新增字典项 mml_command_category value='8' label='基本信息'
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT '基本信息', '8', 80, id
FROM sys_dictionaries WHERE type = 'mml_command_category'
ON CONFLICT DO NOTHING;

-- Step 2: 新增 LST DEVICE_INFO 命令
-- 注意：operation_type / param_paths / supported_operations / help_doc / notes
-- 这些列由 seed/000004_mml_enhance.sql 通过 ALTER TABLE 动态加到 mml_commands。
INSERT INTO mml_commands (id, command_name, command_code, category, description, rpc_method, operation_type, param_template, param_paths, supported_operations, product_types, help_doc)
VALUES (
    'a0000025-0000-0000-0000-000000000001',
    '设备信息', 'LST DEVICE_INFO', '8',
    '查询设备基本信息', 'GetParameterValues', 'LST',
    '{}'::jsonb, '[]'::jsonb, '["LST"]'::jsonb, '["eNB", "gNB"]'::jsonb,
    '查询设备基本信息，包括型号、运行时长、IP、MAC、版本等'
) ON CONFLICT (command_code) DO NOTHING;

-- Step 3: 新增 MOD DEVICE_INFO 命令
INSERT INTO mml_commands (id, command_name, command_code, category, description, rpc_method, operation_type, param_template, param_paths, supported_operations, product_types, help_doc)
VALUES (
    'a0000025-0000-0000-0000-000000000002',
    '设备信息', 'MOD DEVICE_INFO', '8',
    '修改设备基本信息', 'SetParameterValues', 'MOD',
    '{}'::jsonb, '[]'::jsonb, '["MOD"]'::jsonb, '["eNB", "gNB"]'::jsonb,
    '修改设备基本信息中的可写参数'
) ON CONFLICT (command_code) DO NOTHING;

-- 命令 ↔ 参数绑定统一由 seed/000007_seed_mml_basic_info_params.sql 负责。

-- +goose Down
-- 回滚：仅删除本文件创建的命令 + 字典项；mml_command_params_rel 的清理由
-- 000007 的 Down 处理。
DELETE FROM mml_commands WHERE command_code IN ('LST DEVICE_INFO', 'MOD DEVICE_INFO');
DELETE FROM sys_dictionary_details
WHERE value = '8'
  AND sys_dictionary_id IN (SELECT id FROM sys_dictionaries WHERE type = 'mml_command_category');
