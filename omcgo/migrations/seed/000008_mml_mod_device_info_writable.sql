-- +goose Up
-- ============================================================
-- 000008_mml_mod_device_info_writable.sql
-- 目的：修正 MOD DEVICE_INFO 绑定的 7 个参数为"可写"。
-- 背景：to-do-list 要求在 MML 控制台选择 MOD DEVICE_INFO 时，7 个可修改字段
-- 能出现可编辑输入框。前端根据 mml_params.is_writable 决定 input 是否置灰。
-- 原 seed 000005 将这些参数误设为 is_writable=false（仅 is_modifiable=true），
-- 导致前端全部禁用。这里一次性修正。
-- ============================================================
UPDATE mml_params
SET is_writable = true
WHERE tr069_path IN (
    'DeviceGSM.Mcc',
    'DeviceGSM.Mnc',
    'DeviceGSM.BtsNum',
    'DeviceGSM.Encryption',
    'DeviceGSM.TimerNetT3212',
    'DeviceGSM.NriBitLen',
    'DeviceGSM.NriNullAdd'
);

-- +goose Down
UPDATE mml_params
SET is_writable = false
WHERE tr069_path IN (
    'DeviceGSM.Mcc',
    'DeviceGSM.Mnc',
    'DeviceGSM.BtsNum',
    'DeviceGSM.Encryption',
    'DeviceGSM.TimerNetT3212',
    'DeviceGSM.NriBitLen',
    'DeviceGSM.NriNullAdd'
);
