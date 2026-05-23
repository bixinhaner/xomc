-- T-0165 UFTE LICENSE_UPGRADE 任务类型
--
-- 与 CONFIG_RESTORE 同款的 Download 链路，只把目标文件改成 license：
--   - file_type:     "License File"（运行时由 license_dispatch 拼为
--                     "License File"——CPE 拿到的字符串就是这个值，
--                     不需要 OUI 前缀，与配置恢复的 "10 <OUI> Configuration File"
--                     不同）
--   - url_template:  device-licenses/{object_path}（运行时 dispatch 替换）
--   - target_file:   {file_name}（保持设备视角的命名 <SN>_LIC.<ext>）
--
-- 派发由 backup.LicenseService.DispatchLicenseUpgradeBySN 直接构造 device_tasks
-- 的 Params JSON，本表的 file_type 字段仅作 UI 展示。

-- +goose Up

INSERT INTO ufte_task_types (
    type_code, category, category_label, display_name, description,
    rpc_type, built_in, enabled, step_chain,
    post_tc_event_code, permission_code,
    platform_scope, file_type, file_type_label, file_type_editable,
    url_template, target_file_name_template, file_name_template,
    file_size_field, checksum_field, raw_mode, delay_seconds,
    transport_path, last_editor
)
VALUES (
    'LICENSE_UPGRADE', 'license_upgrade', '设备License升级', '设备License升级',
    '从 license 库取目标设备的最新一份 license 文件，通过 TR-069 Download RPC 下发到设备。',
    'DOWNLOAD', true, true,
    '["CHECK_PERMISSION","CHECK_ONLINE","CHECK_CONFLICT","PRE_VALIDATE","SEND_RPC","WAIT_RPC_RESPONSE","WAIT_TRANSFER_COMPLETE"]',
    '', 'CODE_LICENSE_UPGRADE',
    '["4G eNB","5G gNB","QAFA","QAFB","BBU-XSS","BBU-QSS"]',
    'License File', 'License File', false,
    'device-licenses/{object_path}', '{file_name}', '{file_name}',
    '', '', '', 0,
    '/smallcell/FileDownloadService/device-licenses/{object_path}', 'system'
)
ON CONFLICT (type_code) DO UPDATE SET
    category               = EXCLUDED.category,
    category_label         = EXCLUDED.category_label,
    display_name           = EXCLUDED.display_name,
    description            = EXCLUDED.description,
    rpc_type               = EXCLUDED.rpc_type,
    built_in               = EXCLUDED.built_in,
    enabled                = EXCLUDED.enabled,
    step_chain             = EXCLUDED.step_chain,
    post_tc_event_code     = EXCLUDED.post_tc_event_code,
    permission_code        = EXCLUDED.permission_code,
    platform_scope         = EXCLUDED.platform_scope,
    file_type              = EXCLUDED.file_type,
    file_type_label        = EXCLUDED.file_type_label,
    file_type_editable     = EXCLUDED.file_type_editable,
    url_template           = EXCLUDED.url_template,
    target_file_name_template = EXCLUDED.target_file_name_template,
    file_name_template     = EXCLUDED.file_name_template,
    transport_path         = EXCLUDED.transport_path,
    updated_at             = NOW();

-- +goose Down
DELETE FROM ufte_task_types WHERE type_code = 'LICENSE_UPGRADE';
