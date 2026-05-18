-- +goose Up
-- ============================================================
-- 000106_align_ufte_download_file_types.sql
-- 删除 GNB_FPGA_UPGRADE + 对齐 Download 模板 FileType
-- ============================================================

-- 删除 5G FPGA 升级（不再作为内置类型）
DELETE FROM ufte_task_types
WHERE type_code = 'GNB_FPGA_UPGRADE';

-- 4G 基站软件升级 → 1 Firmware Upgrade Image
UPDATE ufte_task_types
SET file_type = '1 Firmware Upgrade Image',
    file_type_label = '1 Firmware Upgrade Image'
WHERE type_code = 'ENB_IMG_UPGRADE';

-- 4G Patch 增量升级 → X {OUI} Software Upgrade Patch
UPDATE ufte_task_types
SET file_type = 'X {OUI} Software Upgrade Patch',
    file_type_label = 'X {OUI} Software Upgrade Patch'
WHERE type_code = 'ENB_PATCH_UPGRADE';

-- 5G 基站软件升级 → 1 Firmware Upgrade Image
UPDATE ufte_task_types
SET file_type = '1 Firmware Upgrade Image',
    file_type_label = '1 Firmware Upgrade Image'
WHERE type_code = 'GNB_IMG_UPGRADE';

-- +goose Down
-- 恢复 GNB_FPGA_UPGRADE
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
ON CONFLICT (type_code) DO NOTHING;

-- 恢复旧 file_type 值
UPDATE ufte_task_types
SET file_type = '1',
    file_type_label = CASE type_code
        WHEN 'ENB_PATCH_UPGRADE' THEN 'Patch Package'
        WHEN 'GNB_IMG_UPGRADE' THEN 'Firmware Upgrade Image'
        ELSE file_type_label
    END
WHERE type_code IN ('ENB_IMG_UPGRADE', 'ENB_PATCH_UPGRADE', 'GNB_IMG_UPGRADE');
