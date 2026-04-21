-- +goose Up
-- ============================================================
-- 000006_seed_mml_basic_info.sql
-- 新增"基本信息"分类、DEVICE_INFO 命令、14 个子命令及关联数据
-- ============================================================

-- Step 1: 新增字典项 mml_command_category value='8' label='基本信息'
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT '基本信息', '8', 80, id
FROM sys_dictionaries WHERE type = 'mml_command_category'
ON CONFLICT DO NOTHING;

-- Step 2: 新增 LST DEVICE_INFO 命令
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

-- Step 4: 新增 14 个子命令
INSERT INTO mml_sub_commands (id, name, code, tr069_path, description, value_type, is_writable, options) VALUES
-- 只读子命令 (7 条)
('b0000025-0000-0000-0001-000000000001', '设备类型',    'LTE_GSM_MODEL_NAME',  'Device.DeviceInfo.X_COM_MODULE_TYPE',             '设备型号/模块类型',  'string',  false, '[]'::jsonb),
('b0000025-0000-0000-0001-000000000002', '运行时长',    'LTE_GSM_SYS_TIME',    'Device.DeviceInfo.X_COM_STATION_RUN_Time',        '设备运行时长',        'string',  false, '[]'::jsonb),
('b0000025-0000-0000-0001-000000000003', 'IP地址',      'LTE_GSM_IP',          'Device.IP.Interface.{i}.IPv4Address.{i}.IPAddress', '设备IP地址',        'string',  false, '[]'::jsonb),
('b0000025-0000-0000-0001-000000000004', 'MAC地址',     'LTE_GSM_MAC',         'Device.DeviceInfo.X_COM_MACAddress',              '设备MAC地址',         'string',  false, '[]'::jsonb),
('b0000025-0000-0000-0001-000000000005', '软件版本',    'LTE_GSM_SOFTWARE',    'Device.DeviceInfo.SoftwareVersion',               '软件版本号',          'string',  false, '[]'::jsonb),
('b0000025-0000-0000-0001-000000000006', '硬件版本',    'LTE_GSM_HARDWARE',    'Device.DeviceInfo.HardwareVersion',               '硬件版本号',          'string',  false, '[]'::jsonb),
('b0000025-0000-0000-0001-000000000007', 'MME状态',     'LTE_GSM_MME_STATUS',  'Device.DeviceInfo.X_COM_MME_Status',              'MME连接状态',         'string',  false, '[]'::jsonb),
-- 可写子命令 (7 条，同时属于 LST 和 MOD)
('b0000025-0000-0000-0001-000000000008', 'MCC',         'DEVICEGSM_MCC',       'DeviceGSM.Mcc',                                   '移动国家代码',        'string',  true,  '[]'::jsonb),
('b0000025-0000-0000-0001-000000000009', 'MNC',         'DEVICEGSM_MNC',       'DeviceGSM.Mnc',                                   '移动网络代码',        'string',  true,  '[]'::jsonb),
('b0000025-0000-0000-0001-000000000010', 'BtsNum',      'LTE_BTSNUM',          'DeviceGSM.BtsNum',                                '基站数量',            'number',  true,  '[]'::jsonb),
('b0000025-0000-0000-0001-000000000011', 'Encryption',  'BSC_ENCRYPTION',      'DeviceGSM.Encryption',                            '加密方式',            'enum',    true,  '[{"label":"不加密","value":0},{"label":"A5/1","value":1},{"label":"A5/3","value":3}]'::jsonb),
('b0000025-0000-0000-0001-000000000012', 'TimerNetT3212','DEVICEGSM_TIMERNETT3212','DeviceGSM.TimerNetT3212',                       'T3212定时器(秒)',     'number',  true,  '[]'::jsonb),
('b0000025-0000-0000-0001-000000000013', 'NriBitLen',   'DEVICEGSM_NRIBITLEN', 'DeviceGSM.NriBitLen',                             'NRI比特长度',         'number',  true,  '[]'::jsonb),
('b0000025-0000-0000-0001-000000000014', 'NriNullAdd',  'DEVICEGSM_NRINULLADD','DeviceGSM.NriNullAdd',                            'NRI空地址',           'number',  true,  '[]'::jsonb)
ON CONFLICT DO NOTHING;

-- Step 5: LST DEVICE_INFO 关联全部 14 个子命令
INSERT INTO mml_command_subcommand_rel (command_id, subcommand_id, sort_order)
SELECT c.id, s.id, row_number() OVER (ORDER BY s.code)
FROM mml_commands c
CROSS JOIN mml_sub_commands s
WHERE c.command_code = 'LST DEVICE_INFO'
ON CONFLICT DO NOTHING;

-- Step 6: MOD DEVICE_INFO 关联 7 个可写子命令
INSERT INTO mml_command_subcommand_rel (command_id, subcommand_id, sort_order)
SELECT c.id, s.id, row_number() OVER (ORDER BY s.code)
FROM mml_commands c
CROSS JOIN mml_sub_commands s
WHERE c.command_code = 'MOD DEVICE_INFO'
  AND s.is_writable = true
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM mml_command_subcommand_rel WHERE command_id IN (
    SELECT id FROM mml_commands WHERE command_code IN ('LST DEVICE_INFO', 'MOD DEVICE_INFO')
);
DELETE FROM mml_sub_commands WHERE code IN (
    'LTE_GSM_MODEL_NAME','LTE_GSM_SYS_TIME','LTE_GSM_IP','LTE_GSM_MAC',
    'LTE_GSM_SOFTWARE','LTE_GSM_HARDWARE','LTE_GSM_MME_STATUS',
    'DEVICEGSM_MCC','DEVICEGSM_MNC','LTE_BTSNUM','BSC_ENCRYPTION',
    'DEVICEGSM_TIMERNETT3212','DEVICEGSM_NRIBITLEN','DEVICEGSM_NRINULLADD'
);
DELETE FROM mml_commands WHERE command_code IN ('LST DEVICE_INFO', 'MOD DEVICE_INFO');
DELETE FROM sys_dictionary_details WHERE value = '8' AND sys_dictionary_id IN (
    SELECT id FROM sys_dictionaries WHERE type = 'mml_command_category'
);
