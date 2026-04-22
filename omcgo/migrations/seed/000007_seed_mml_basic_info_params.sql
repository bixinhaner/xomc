-- +goose Up
-- ============================================================
-- 000007_seed_mml_basic_info_params.sql
-- 绑定基本信息命令 (LST/MOD DEVICE_INFO) 与 mml_params 的关联
-- ============================================================

-- LST DEVICE_INFO: 关联全部 14 个参数（7 只读 + 7 可写）
INSERT INTO mml_command_params_rel (command_id, param_id, sort_order)
SELECT c.id, p.id, ROW_NUMBER() OVER (ORDER BY p.param_code)
FROM mml_commands c
CROSS JOIN mml_params p
WHERE c.command_code = 'LST DEVICE_INFO'
  AND p.tr069_path IN (
    'Device.DeviceInfo.X_COM_MODULE_TYPE',
    'Device.DeviceInfo.X_COM_STATION_RUN_Time',
    'Device.IP.Interface.{i}.IPv4Address.{i}.IPAddress',
    'Device.DeviceInfo.X_COM_MACAddress',
    'Device.DeviceInfo.SoftwareVersion',
    'Device.DeviceInfo.HardwareVersion',
    'Device.DeviceInfo.X_COM_MME_Status',
    'DeviceGSM.Mcc',
    'DeviceGSM.Mnc',
    'DeviceGSM.BtsNum',
    'DeviceGSM.Encryption',
    'DeviceGSM.TimerNetT3212',
    'DeviceGSM.NriBitLen',
    'DeviceGSM.NriNullAdd'
  )
ON CONFLICT DO NOTHING;

-- MOD DEVICE_INFO: 关联 7 个可写参数
INSERT INTO mml_command_params_rel (command_id, param_id, sort_order)
SELECT c.id, p.id, ROW_NUMBER() OVER (ORDER BY p.param_code)
FROM mml_commands c
CROSS JOIN mml_params p
WHERE c.command_code = 'MOD DEVICE_INFO'
  AND p.tr069_path IN (
    'DeviceGSM.Mcc',
    'DeviceGSM.Mnc',
    'DeviceGSM.BtsNum',
    'DeviceGSM.Encryption',
    'DeviceGSM.TimerNetT3212',
    'DeviceGSM.NriBitLen',
    'DeviceGSM.NriNullAdd'
  )
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM mml_command_params_rel
WHERE command_id IN (
    SELECT id FROM mml_commands WHERE command_code IN ('LST DEVICE_INFO', 'MOD DEVICE_INFO')
);
