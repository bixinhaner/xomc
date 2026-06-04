-- +goose Up
-- 修复 baseline（seed/000001）合并 277 迁移时丢失的"文件传输"任务类型厂商对齐。
-- 丢失的原迁移：000139（filename 置空）/ 000142（运行日志 file_type + fileType=LOG）/
-- 000143（补 sn 查询参数）/ 000156（故障日志改 SPV）/ 000157（故障日志参数路径）。
-- 本迁移把效果重新合入，存量库升级即生效；全新部署因 000001 已修正，这里是幂等 no-op。
-- 详见排障记录：运行日志收集卡 45% → 设备收到 <FileType>6</FileType> 不认，应为 "4 Vendor Log File 1,2,3,4"。

-- 运行日志：file_type 改厂商标准串；URL 用字面 fileType=LOG（ACS 上传处理器识别用）+ taskId32 + 空 filename
UPDATE ufte_task_types
SET file_type      = '4 Vendor Log File 1,2,3,4',
    file_type_label = '4 Vendor Log File 1,2,3,4',
    transport_path = '/smallcell/FileUploadService?fileType=LOG&sn={sn}&taskId={taskId32}&filename=',
    updated_at     = NOW()
WHERE type_code = 'RUNTIME_LOG_COLLECT';

-- 故障日志：改 SPV（SetParameterValues）模式
UPDATE ufte_task_types
SET rpc_type       = 'SET_PARAM_VALUES',
    step_chain     = '["CHECK_PERMISSION", "CHECK_ONLINE", "PRE_VALIDATE", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_FILE_UPLOAD"]'::jsonb,
    url_template   = 'Device.DeviceInfo.FaultLogURL',
    transport_path = '/smallcell/FileUploadService?fileType=RL&id={id}&sn={sn}&fileName=',
    description    = '通过 SetParameterValues 写设备私有参数 Device.DeviceInfo.FaultLogURL 触发 CPE 主动上传故障日志，落 MinIO 后由 backup.file.received 推进任务完成。',
    updated_at     = NOW()
WHERE type_code = 'FAULT_LOG_COLLECT';

-- 配置备份（XML/NV）：URL 末尾 filename 置空（设备自行命名 / ACS 端按 sn+taskId 派生）
UPDATE ufte_task_types
SET transport_path = '/smallcell/FileUploadService?fileType=CONFIGBACKUP_XML&sn={sn}&taskId={taskId}&filename=',
    updated_at     = NOW()
WHERE type_code = 'CONFIG_BACKUP_XML';

UPDATE ufte_task_types
SET transport_path = '/smallcell/FileUploadService?fileType=CONFIGBACKUP_NV&sn={sn}&taskId={taskId}&filename=',
    updated_at     = NOW()
WHERE type_code = 'CONFIG_BACKUP_NV';

-- 模板顺序（原 000147）：恢复文件传输任务类型在 UI 的排序（baseline 里全退回默认 100）
UPDATE ufte_task_types SET sort_order = 10, updated_at = NOW() WHERE type_code IN ('ENB_IMG_UPGRADE', 'GNB_IMG_UPGRADE');
UPDATE ufte_task_types SET sort_order = 15, updated_at = NOW() WHERE type_code = 'ENB_PATCH_UPGRADE';
UPDATE ufte_task_types SET sort_order = 18, updated_at = NOW() WHERE type_code = 'ENB_FPGA_UPGRADE';
UPDATE ufte_task_types SET sort_order = 20, updated_at = NOW() WHERE type_code = 'VERSION_ROLLBACK';
UPDATE ufte_task_types SET sort_order = 30, updated_at = NOW() WHERE type_code = 'RUNTIME_LOG_COLLECT';
UPDATE ufte_task_types SET sort_order = 35, updated_at = NOW() WHERE type_code = 'FAULT_LOG_COLLECT';
UPDATE ufte_task_types SET sort_order = 40, updated_at = NOW() WHERE type_code = 'CONFIG_BACKUP_XML';
UPDATE ufte_task_types SET sort_order = 45, updated_at = NOW() WHERE type_code = 'CONFIG_BACKUP_NV';
UPDATE ufte_task_types SET sort_order = 50, updated_at = NOW() WHERE type_code = 'CONFIG_RESTORE';
UPDATE ufte_task_types SET sort_order = 55, updated_at = NOW() WHERE type_code = 'LICENSE_UPGRADE';


-- +goose Down
-- 回退到 baseline 合并后的（错误）原值。
UPDATE ufte_task_types
SET file_type      = '6',
    file_type_label = '运行日志',
    transport_path = '/smallcell/FileUploadService?fileType={fileType}&filename={targetFileName}',
    updated_at     = NOW()
WHERE type_code = 'RUNTIME_LOG_COLLECT';

UPDATE ufte_task_types
SET rpc_type       = 'UPLOAD',
    step_chain     = '["CHECK_PERMISSION", "CHECK_ONLINE", "PRE_VALIDATE", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_TRANSFER_COMPLETE"]'::jsonb,
    url_template   = '',
    transport_path = '/smallcell/FileUploadService?fileType={fileType}&filename={targetFileName}',
    description    = '复用现网日志采集 Upload 链路，提供 UFTE 内置的故障日志收集模板。',
    updated_at     = NOW()
WHERE type_code = 'FAULT_LOG_COLLECT';

UPDATE ufte_task_types
SET transport_path = '/smallcell/FileUploadService?fileType=CONFIGBACKUP_XML&sn={sn}&taskId={taskId}&filename={targetFileName}',
    updated_at     = NOW()
WHERE type_code = 'CONFIG_BACKUP_XML';

UPDATE ufte_task_types
SET transport_path = '/smallcell/FileUploadService?fileType=CONFIGBACKUP_NV&sn={sn}&taskId={taskId}&filename={targetFileName}',
    updated_at     = NOW()
WHERE type_code = 'CONFIG_BACKUP_NV';

UPDATE ufte_task_types SET sort_order = 100, updated_at = NOW()
WHERE type_code IN ('ENB_IMG_UPGRADE', 'GNB_IMG_UPGRADE', 'ENB_PATCH_UPGRADE', 'ENB_FPGA_UPGRADE', 'VERSION_ROLLBACK', 'RUNTIME_LOG_COLLECT', 'FAULT_LOG_COLLECT', 'CONFIG_BACKUP_XML', 'CONFIG_BACKUP_NV', 'CONFIG_RESTORE', 'LICENSE_UPGRADE');
