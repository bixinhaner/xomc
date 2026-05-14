-- +goose Up
-- ============================================================
-- 000102_seed_ufte_task_types.sql
-- UFTE 内置任务类型种子数据（9 条）
-- 幂等：ON CONFLICT (type_code) DO NOTHING
-- ============================================================

INSERT INTO ufte_task_types (type_code, category, category_label, display_name, description, rpc_type, built_in, enabled, step_chain, post_tc_event_code, permission_code, platform_scope, file_type, file_type_label, file_type_editable, url_template, target_file_name_template, file_name_template, file_size_field, checksum_field, raw_mode, delay_seconds, transport_path, last_editor)
VALUES
-- 4G 升级
('ENB_IMG_UPGRADE',   'enb_upgrade',     '4G升级',       '4G 基站软件升级',
 '复用现网软件升级链路，统一承载 4G 基站镜像升级任务。',
 'DOWNLOAD', true, true,
 '["CHECK_PERMISSION","CHECK_ONLINE","CHECK_CONFLICT","SEND_RPC","WAIT_RPC_RESPONSE","WAIT_FILE_TRANSFER","WAIT_TRANSFER_COMPLETE"]',
 '', 'CODE_ENB_UPGRADE_IMAGE',
 '["4G eNB","QAFA","QAFB"]', '1', 'Firmware Upgrade Image', true,
 'firmware/{minio_path}', '{firmware_name}', '{firmware_name}',
 'firmware.fileSize', 'firmware.md5', 'false', 0,
 '/smallcell/FileDownloadService/firmware/img/{path}', 'system')
ON CONFLICT (type_code) DO NOTHING;

INSERT INTO ufte_task_types (type_code, category, category_label, display_name, description, rpc_type, built_in, enabled, step_chain, post_tc_event_code, permission_code, platform_scope, file_type, file_type_label, file_type_editable, url_template, target_file_name_template, file_name_template, file_size_field, checksum_field, raw_mode, delay_seconds, transport_path, last_editor)
VALUES
('ENB_PATCH_UPGRADE', 'enb_upgrade',     '4G升级',       '4G Patch 增量升级',
 '复用软件管理补丁升级任务链路，统一收口到 UFTE 任务入口。',
 'DOWNLOAD', true, true,
 '["CHECK_PERMISSION","CHECK_ONLINE","CHECK_CONFLICT","SEND_RPC","WAIT_RPC_RESPONSE","WAIT_FILE_TRANSFER","WAIT_TRANSFER_COMPLETE"]',
 '', 'CODE_ENB_UPGRADE_PATCH',
 '["4G eNB","QAFA","QAFB","PATCH"]', '1', 'Patch Package', true,
 'firmware/{patch_path}', '{patch_name}', '{patch_name}',
 'firmware.fileSize', 'firmware.md5', 'true', 0,
 '/smallcell/FileDownloadService/firmware/patch/{path}', 'system')
ON CONFLICT (type_code) DO NOTHING;

-- 5G 升级
INSERT INTO ufte_task_types (type_code, category, category_label, display_name, description, rpc_type, built_in, enabled, step_chain, post_tc_event_code, permission_code, platform_scope, file_type, file_type_label, file_type_editable, url_template, target_file_name_template, file_name_template, file_size_field, checksum_field, raw_mode, delay_seconds, transport_path, last_editor)
VALUES
('GNB_IMG_UPGRADE',   'gnb_upgrade',     '5G升级',       '5G 基站软件升级',
 '复用现网 5G 升级调测通过的下载与升级完成事件链路。',
 'DOWNLOAD', true, true,
 '["CHECK_PERMISSION","CHECK_ONLINE","CHECK_CONFLICT","SEND_RPC","WAIT_RPC_RESPONSE","WAIT_FILE_TRANSFER","WAIT_TRANSFER_COMPLETE","WAIT_INFORM_EVENT"]',
 '102 UPGRADE FINISH', 'CODE_GNB_UPGRADE_IMAGE',
 '["5G gNB","BBU-XSS","BBU-QSS"]', '1', 'Firmware Upgrade Image', true,
 'firmware/{minio_path}', '{firmware_name}', '{firmware_name}',
 'firmware.fileSize', 'firmware.md5', 'false', 0,
 '/smallcell/FileDownloadService/firmware/img/{path}', 'system')
ON CONFLICT (type_code) DO NOTHING;

INSERT INTO ufte_task_types (type_code, category, category_label, display_name, description, rpc_type, built_in, enabled, step_chain, post_tc_event_code, permission_code, platform_scope, file_type, file_type_label, file_type_editable, url_template, target_file_name_template, file_name_template, file_size_field, checksum_field, raw_mode, delay_seconds, transport_path, last_editor)
VALUES
('GNB_FPGA_UPGRADE',  'gnb_upgrade',     '5G升级',       '5G FPGA 升级',
 '复用 5G 侧 FPGA 升级任务链路，统一到 UFTE 任务中心。',
 'DOWNLOAD', true, true,
 '["CHECK_PERMISSION","CHECK_ONLINE","CHECK_CONFLICT","SEND_RPC","WAIT_RPC_RESPONSE","WAIT_FILE_TRANSFER","WAIT_TRANSFER_COMPLETE","WAIT_INFORM_EVENT"]',
 '102 UPGRADE FINISH', 'CODE_GNB_UPGRADE_FPGA',
 '["5G gNB","BBU-XSS","BBU-QSS","FPGA"]', '1', 'FPGA Package', true,
 'firmware/{fpga_path}', '{fpga_name}', '{fpga_name}',
 'firmware.fileSize', 'firmware.md5', 'false', 0,
 '/smallcell/FileDownloadService/firmware/fpga/{path}', 'system')
ON CONFLICT (type_code) DO NOTHING;

-- 版本回退
INSERT INTO ufte_task_types (type_code, category, category_label, display_name, description, rpc_type, built_in, enabled, step_chain, post_tc_event_code, permission_code, platform_scope, file_type, file_type_label, file_type_editable, url_template, target_file_name_template, file_name_template, file_size_field, checksum_field, raw_mode, delay_seconds, transport_path, last_editor)
VALUES
('VERSION_ROLLBACK',  'version_rollback', '基站版本回退', '基站版本回退',
 '复用现网已验证的版本回退链路，实现 UFTE 统一入口下的回退任务创建。',
 'DOWNLOAD', true, true,
 '["CHECK_PERMISSION","CHECK_ONLINE","CHECK_CONFLICT","PRE_VALIDATE","SEND_RPC","WAIT_RPC_RESPONSE","WAIT_TRANSFER_COMPLETE"]',
 '', 'CODE_VERSION_ROLLBACK',
 '["4G eNB","5G gNB","QAFA","QAFB","BBU-XSS","BBU-QSS"]', '1', 'Rollback Image', true,
 'firmware/rollback/{rollback_path}', '{rollback_name}', '{rollback_name}',
 'rollback.fileSize', 'rollback.md5', 'false', 0,
 '/smallcell/FileDownloadService/firmware/rollback/{path}', 'system')
ON CONFLICT (type_code) DO NOTHING;

-- 日志采集
INSERT INTO ufte_task_types (type_code, category, category_label, display_name, description, rpc_type, built_in, enabled, step_chain, post_tc_event_code, permission_code, platform_scope, file_type, file_type_label, file_type_editable, url_template, target_file_name_template, file_name_template, file_size_field, checksum_field, raw_mode, delay_seconds, transport_path, last_editor)
VALUES
('RUNTIME_LOG_COLLECT', 'station_log',   '日志收集',     '运行日志采集',
 '复用现网日志采集 Upload 链路，提供 UFTE 内置的运行日志收集模板。',
 'UPLOAD', true, true,
 '["CHECK_PERMISSION","CHECK_ONLINE","PRE_VALIDATE","SEND_RPC","WAIT_RPC_RESPONSE","WAIT_TRANSFER_COMPLETE"]',
 '', 'CODE_RUNTIME_LOG_COLLECT',
 '["4G eNB","5G gNB","DXDF","BBU-QSS"]', '6', '运行日志', true,
 '', 'runtime-{task_id8}-{sn}.tar.gz', 'runtime-{task_id8}-{sn}.tar.gz',
 '', '', '', 0,
 '/smallcell/FileUploadService?fileType={fileType}&filename={targetFileName}', 'system')
ON CONFLICT (type_code) DO NOTHING;

INSERT INTO ufte_task_types (type_code, category, category_label, display_name, description, rpc_type, built_in, enabled, step_chain, post_tc_event_code, permission_code, platform_scope, file_type, file_type_label, file_type_editable, url_template, target_file_name_template, file_name_template, file_size_field, checksum_field, raw_mode, delay_seconds, transport_path, last_editor)
VALUES
('FAULT_LOG_COLLECT',   'station_log',   '日志收集',     '故障日志采集',
 '复用现网日志采集 Upload 链路，提供 UFTE 内置的故障日志收集模板。',
 'UPLOAD', true, true,
 '["CHECK_PERMISSION","CHECK_ONLINE","PRE_VALIDATE","SEND_RPC","WAIT_RPC_RESPONSE","WAIT_TRANSFER_COMPLETE"]',
 '', 'CODE_FAULT_LOG_COLLECT',
 '["4G eNB","5G gNB","DXDF","BBU-QSS"]', '8', '故障日志', true,
 '', 'fault-{task_id8}-{sn}.tar.gz', 'fault-{task_id8}-{sn}.tar.gz',
 '', '', '', 0,
 '/smallcell/FileUploadService?fileType={fileType}&filename={targetFileName}', 'system')
ON CONFLICT (type_code) DO NOTHING;

-- 配置备份/恢复
INSERT INTO ufte_task_types (type_code, category, category_label, display_name, description, rpc_type, built_in, enabled, step_chain, post_tc_event_code, permission_code, platform_scope, file_type, file_type_label, file_type_editable, url_template, target_file_name_template, file_name_template, file_size_field, checksum_field, raw_mode, delay_seconds, transport_path, last_editor)
VALUES
('CONFIG_BACKUP',     'config_backup',   '配置文件备份', '配置文件备份',
 '复用现网配置备份 Upload 链路，提供 UFTE 内置的配置文件备份模板。',
 'UPLOAD', true, true,
 '["CHECK_PERMISSION","CHECK_ONLINE","CHECK_CONFLICT","PRE_VALIDATE","SEND_RPC","WAIT_RPC_RESPONSE","WAIT_TRANSFER_COMPLETE"]',
 '', 'CODE_CONFIG_BACKUP',
 '["4G eNB","5G gNB","QAFA","QAFB","BBU-XSS","BBU-QSS"]', '3', 'Vendor Configuration File', false,
 '', 'backup-{task_id8}-{sn}.xml', 'backup-{task_id8}-{sn}.xml',
 '', '', '', 0,
 '/smallcell/FileUploadService?fileType={fileType}&filename={targetFileName}', 'system')
ON CONFLICT (type_code) DO NOTHING;

INSERT INTO ufte_task_types (type_code, category, category_label, display_name, description, rpc_type, built_in, enabled, step_chain, post_tc_event_code, permission_code, platform_scope, file_type, file_type_label, file_type_editable, url_template, target_file_name_template, file_name_template, file_size_field, checksum_field, raw_mode, delay_seconds, transport_path, last_editor)
VALUES
('CONFIG_RESTORE',    'config_restore',  '配置文件恢复', '配置文件恢复',
 '复用现网配置恢复 Download 链路，提供 UFTE 内置的配置文件恢复模板。',
 'DOWNLOAD', true, true,
 '["CHECK_PERMISSION","CHECK_ONLINE","CHECK_CONFLICT","PRE_VALIDATE","SEND_RPC","WAIT_RPC_RESPONSE","WAIT_TRANSFER_COMPLETE"]',
 '', 'CODE_CONFIG_RESTORE',
 '["4G eNB","5G gNB","QAFA","QAFB","BBU-XSS","BBU-QSS"]', '3', 'Vendor Configuration File', false,
 'config_backup/{object_path}', '{file_name}', '{file_name}',
 '', '', '', 0,
 '/smallcell/FileDownloadService/config_backup/{object_path}', 'system')
ON CONFLICT (type_code) DO NOTHING;

-- +goose Down
DELETE FROM ufte_task_types WHERE built_in = true;
