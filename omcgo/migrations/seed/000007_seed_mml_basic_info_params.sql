-- +goose Up
-- ============================================================
-- 000007_seed_mml_basic_info_params.sql
-- 绑定基本信息命令 (LST/MOD DEVICE_INFO) 与 mml_params 的关联
-- ============================================================

-- LST DEVICE_INFO: 关联全部 14 个参数（7 只读 + 7 可写）
-- 同一 tr069_path 在 mml_params 中可能存在多个 param_version 行；
-- 用 DISTINCT ON 确保每条 path 只取一行，避免重复关联（参见 migrations/000036）。
INSERT INTO mml_command_params_rel (command_id, param_id, sort_order)
SELECT c.id, p.id, ROW_NUMBER() OVER (ORDER BY p.tr069_path)
FROM mml_commands c
CROSS JOIN (
    SELECT DISTINCT ON (tr069_path) id, tr069_path
    FROM mml_params
    WHERE tr069_path IN (
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
    ORDER BY tr069_path, param_version
) p
WHERE c.command_code = 'LST DEVICE_INFO'
ON CONFLICT DO NOTHING;

-- MOD DEVICE_INFO: 关联 7 个可写参数（同样按 tr069_path 去重）
INSERT INTO mml_command_params_rel (command_id, param_id, sort_order)
SELECT c.id, p.id, ROW_NUMBER() OVER (ORDER BY p.tr069_path)
FROM mml_commands c
CROSS JOIN (
    SELECT DISTINCT ON (tr069_path) id, tr069_path
    FROM mml_params
    WHERE tr069_path IN (
        'DeviceGSM.Mcc',
        'DeviceGSM.Mnc',
        'DeviceGSM.BtsNum',
        'DeviceGSM.Encryption',
        'DeviceGSM.TimerNetT3212',
        'DeviceGSM.NriBitLen',
        'DeviceGSM.NriNullAdd'
    )
    ORDER BY tr069_path, param_version
) p
WHERE c.command_code = 'MOD DEVICE_INFO'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM mml_command_params_rel
WHERE command_id IN (
    SELECT id FROM mml_commands WHERE command_code IN ('LST DEVICE_INFO', 'MOD DEVICE_INFO')
);
