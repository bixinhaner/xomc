-- 000129_split_ufte_config_backup.sql
-- 将旧的单条 CONFIG_BACKUP（file_type='3'）拆分为两条：
--   CONFIG_BACKUP_XML  — 标准平台，TR-069 FileType = 10 {OUI} Configuration File
--   CONFIG_BACKUP_NV   — NV 平台（MLQ/MLN_SC），TR-069 FileType = 12 {OUI} Configuration File
-- {OUI} 是占位符，由 backup.BackupExecutor 在执行时替换为设备实际 OUI。

-- +goose Up

-- 删除旧条目（DO NOTHING 的 000102 种子已写入，此处需显式删除）
DELETE FROM ufte_task_types WHERE type_code = 'CONFIG_BACKUP';

INSERT INTO ufte_task_types (
    type_code, category, category_label, display_name, description,
    rpc_type, built_in, enabled,
    step_chain, post_tc_event_code, permission_code, platform_scope,
    file_type, file_type_label, file_type_editable,
    url_template, target_file_name_template, file_name_template,
    file_size_field, checksum_field, raw_mode, delay_seconds,
    transport_path, last_editor
)
VALUES
(
    'CONFIG_BACKUP_XML', 'config_backup', '配置文件备份', '配置文件备份（XML）',
    '标准平台（BLQ/QLS 等）配置文件备份，TR-069 Upload FileType=10 {OUI} Configuration File。',
    'UPLOAD', true, true,
    '["CHECK_PERMISSION","CHECK_ONLINE","CHECK_CONFLICT","PRE_VALIDATE","SEND_RPC","WAIT_RPC_RESPONSE","WAIT_TRANSFER_COMPLETE"]',
    '', 'CODE_CONFIG_BACKUP',
    '["4G eNB","5G gNB","QAFA","QAFB","BBU-XSS","BBU-QSS"]',
    '10 {OUI} Configuration File', '10 {OUI} Configuration File', false,
    '', 'backup-{task_id8}-{sn}.xml', 'backup-{task_id8}-{sn}.xml',
    '', '', '', 0,
    '/smallcell/FileUploadService?fileType=CONFIGBACKUP_XML&sn={sn}&taskId={taskId}&filename={targetFileName}',
    'system'
)
ON CONFLICT (type_code) DO UPDATE SET
    display_name            = EXCLUDED.display_name,
    description             = EXCLUDED.description,
    platform_scope          = EXCLUDED.platform_scope,
    file_type               = EXCLUDED.file_type,
    file_type_label         = EXCLUDED.file_type_label,
    target_file_name_template = EXCLUDED.target_file_name_template,
    file_name_template      = EXCLUDED.file_name_template,
    transport_path          = EXCLUDED.transport_path,
    enabled                 = EXCLUDED.enabled;

INSERT INTO ufte_task_types (
    type_code, category, category_label, display_name, description,
    rpc_type, built_in, enabled,
    step_chain, post_tc_event_code, permission_code, platform_scope,
    file_type, file_type_label, file_type_editable,
    url_template, target_file_name_template, file_name_template,
    file_size_field, checksum_field, raw_mode, delay_seconds,
    transport_path, last_editor
)
VALUES
(
    'CONFIG_BACKUP_NV', 'config_backup', '配置文件备份', '配置文件备份（NV）',
    'NV 平台（MLQ/MLN_SC 等）配置文件备份，TR-069 Upload FileType=12 {OUI} Configuration File。',
    'UPLOAD', true, true,
    '["CHECK_PERMISSION","CHECK_ONLINE","CHECK_CONFLICT","PRE_VALIDATE","SEND_RPC","WAIT_RPC_RESPONSE","WAIT_TRANSFER_COMPLETE"]',
    '', 'CODE_CONFIG_BACKUP',
    '["MLQ","MLN_SC"]',
    '12 {OUI} Configuration File', '12 {OUI} Configuration File', false,
    '', 'backup-{task_id8}-{sn}.nv', 'backup-{task_id8}-{sn}.nv',
    '', '', '', 0,
    '/smallcell/FileUploadService?fileType=CONFIGBACKUP_NV&sn={sn}&taskId={taskId}&filename={targetFileName}',
    'system'
)
ON CONFLICT (type_code) DO UPDATE SET
    display_name            = EXCLUDED.display_name,
    description             = EXCLUDED.description,
    platform_scope          = EXCLUDED.platform_scope,
    file_type               = EXCLUDED.file_type,
    file_type_label         = EXCLUDED.file_type_label,
    target_file_name_template = EXCLUDED.target_file_name_template,
    file_name_template      = EXCLUDED.file_name_template,
    transport_path          = EXCLUDED.transport_path,
    enabled                 = EXCLUDED.enabled;

-- +goose Down
DELETE FROM ufte_task_types WHERE type_code IN ('CONFIG_BACKUP_XML', 'CONFIG_BACKUP_NV');

INSERT INTO ufte_task_types (
    type_code, category, category_label, display_name, description,
    rpc_type, built_in, enabled,
    step_chain, post_tc_event_code, permission_code, platform_scope,
    file_type, file_type_label, file_type_editable,
    url_template, target_file_name_template, file_name_template,
    file_size_field, checksum_field, raw_mode, delay_seconds,
    transport_path, last_editor
)
VALUES
(
    'CONFIG_BACKUP', 'config_backup', '配置文件备份', '配置文件备份',
    '复用现网配置备份 Upload 链路，提供 UFTE 内置的配置文件备份模板。',
    'UPLOAD', true, true,
    '["CHECK_PERMISSION","CHECK_ONLINE","CHECK_CONFLICT","PRE_VALIDATE","SEND_RPC","WAIT_RPC_RESPONSE","WAIT_TRANSFER_COMPLETE"]',
    '', 'CODE_CONFIG_BACKUP',
    '["4G eNB","5G gNB","QAFA","QAFB","BBU-XSS","BBU-QSS"]',
    '3', 'Vendor Configuration File', false,
    '', 'backup-{task_id8}-{sn}.xml', 'backup-{task_id8}-{sn}.xml',
    '', '', '', 0,
    '/smallcell/FileUploadService?fileType={fileType}&filename={targetFileName}',
    'system'
)
ON CONFLICT (type_code) DO NOTHING;
