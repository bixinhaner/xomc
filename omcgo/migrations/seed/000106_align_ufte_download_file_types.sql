-- +goose Up
-- ============================================================
-- 000103_align_ufte_download_file_types.sql
-- 对齐 UFTE Download 模板 FileType 为最终下发字符串
-- ============================================================

UPDATE ufte_task_types
SET file_type = '1 Firmware Upgrade Image',
    file_type_label = '1 Firmware Upgrade Image'
WHERE type_code IN ('ENB_IMG_UPGRADE', 'ENB_PATCH_UPGRADE', 'GNB_IMG_UPGRADE', 'VERSION_ROLLBACK');

UPDATE ufte_task_types
SET file_type = '3 Vendor Configuration File',
    file_type_label = '3 Vendor Configuration File'
WHERE type_code = 'CONFIG_RESTORE';

DELETE FROM ufte_task_types
WHERE type_code = 'GNB_FPGA_UPGRADE';

INSERT INTO ufte_task_types (
    type_code, category, category_label, display_name, description,
    rpc_type, built_in, enabled, step_chain, post_tc_event_code,
    permission_code, platform_scope, file_type, file_type_label,
    file_type_editable, url_template, target_file_name_template,
    file_name_template, file_size_field, checksum_field, raw_mode,
    delay_seconds, transport_path, last_editor
)
VALUES (
    'ENB_FPGA_UPGRADE', 'enb_upgrade', '4G升级', '4G FPGA 升级',
    '复用 4G 侧 FPGA 升级任务链路，统一到 UFTE 任务中心。',
    'DOWNLOAD', true, true,
    '["CHECK_PERMISSION","CHECK_ONLINE","CHECK_CONFLICT","SEND_RPC","WAIT_RPC_RESPONSE","WAIT_FILE_TRANSFER","WAIT_TRANSFER_COMPLETE"]'::jsonb,
    '', 'CODE_ENB_UPGRADE_FPGA',
    '["4G eNB","QAFA","QAFB","FPGA"]'::jsonb,
    'Firmware Upgrade Fpga', 'Firmware Upgrade Fpga',
    true, 'firmware/{fpga_path}', '{fpga_name}', '{fpga_name}',
    'firmware.fileSize', 'firmware.md5', 'false', 0,
    '/smallcell/FileDownloadService/firmware/fpga/{path}', 'system'
)
ON CONFLICT (type_code) DO UPDATE SET
    category = EXCLUDED.category,
    category_label = EXCLUDED.category_label,
    display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    rpc_type = EXCLUDED.rpc_type,
    built_in = EXCLUDED.built_in,
    enabled = EXCLUDED.enabled,
    step_chain = EXCLUDED.step_chain,
    post_tc_event_code = EXCLUDED.post_tc_event_code,
    permission_code = EXCLUDED.permission_code,
    platform_scope = EXCLUDED.platform_scope,
    file_type = EXCLUDED.file_type,
    file_type_label = EXCLUDED.file_type_label,
    file_type_editable = EXCLUDED.file_type_editable,
    url_template = EXCLUDED.url_template,
    target_file_name_template = EXCLUDED.target_file_name_template,
    file_name_template = EXCLUDED.file_name_template,
    file_size_field = EXCLUDED.file_size_field,
    checksum_field = EXCLUDED.checksum_field,
    raw_mode = EXCLUDED.raw_mode,
    delay_seconds = EXCLUDED.delay_seconds,
    transport_path = EXCLUDED.transport_path,
    last_editor = EXCLUDED.last_editor,
    updated_at = now();

-- +goose Down
DELETE FROM ufte_task_types
WHERE type_code = 'ENB_FPGA_UPGRADE';

INSERT INTO ufte_task_types (
    type_code, category, category_label, display_name, description,
    rpc_type, built_in, enabled, step_chain, post_tc_event_code,
    permission_code, platform_scope, file_type, file_type_label,
    file_type_editable, url_template, target_file_name_template,
    file_name_template, file_size_field, checksum_field, raw_mode,
    delay_seconds, transport_path, last_editor
)
VALUES (
    'GNB_FPGA_UPGRADE', 'gnb_upgrade', '5G升级', '5G FPGA 升级',
    '复用 5G 侧 FPGA 升级任务链路，统一到 UFTE 任务中心。',
    'DOWNLOAD', true, true,
    '["CHECK_PERMISSION","CHECK_ONLINE","CHECK_CONFLICT","SEND_RPC","WAIT_RPC_RESPONSE","WAIT_FILE_TRANSFER","WAIT_TRANSFER_COMPLETE","WAIT_INFORM_EVENT"]'::jsonb,
    '102 UPGRADE FINISH', 'CODE_GNB_UPGRADE_FPGA',
    '["5G gNB","BBU-XSS","BBU-QSS","FPGA"]'::jsonb,
    '1', 'FPGA Package',
    true, 'firmware/{fpga_path}', '{fpga_name}', '{fpga_name}',
    'firmware.fileSize', 'firmware.md5', 'false', 0,
    '/smallcell/FileDownloadService/firmware/fpga/{path}', 'system'
)
ON CONFLICT (type_code) DO UPDATE SET
    category = EXCLUDED.category,
    category_label = EXCLUDED.category_label,
    display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    rpc_type = EXCLUDED.rpc_type,
    built_in = EXCLUDED.built_in,
    enabled = EXCLUDED.enabled,
    step_chain = EXCLUDED.step_chain,
    post_tc_event_code = EXCLUDED.post_tc_event_code,
    permission_code = EXCLUDED.permission_code,
    platform_scope = EXCLUDED.platform_scope,
    file_type = EXCLUDED.file_type,
    file_type_label = EXCLUDED.file_type_label,
    file_type_editable = EXCLUDED.file_type_editable,
    url_template = EXCLUDED.url_template,
    target_file_name_template = EXCLUDED.target_file_name_template,
    file_name_template = EXCLUDED.file_name_template,
    file_size_field = EXCLUDED.file_size_field,
    checksum_field = EXCLUDED.checksum_field,
    raw_mode = EXCLUDED.raw_mode,
    delay_seconds = EXCLUDED.delay_seconds,
    transport_path = EXCLUDED.transport_path,
    last_editor = EXCLUDED.last_editor,
    updated_at = now();

UPDATE ufte_task_types
SET file_type = '1',
    file_type_label = CASE type_code
        WHEN 'ENB_PATCH_UPGRADE' THEN 'Patch Package'
        WHEN 'VERSION_ROLLBACK' THEN 'Rollback Image'
        ELSE 'Firmware Upgrade Image'
    END
WHERE type_code IN ('ENB_IMG_UPGRADE', 'ENB_PATCH_UPGRADE', 'GNB_IMG_UPGRADE', 'VERSION_ROLLBACK');

UPDATE ufte_task_types
SET file_type = '3',
    file_type_label = 'Vendor Configuration File'
WHERE type_code = 'CONFIG_RESTORE';
