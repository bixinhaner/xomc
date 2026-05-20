-- +goose Up
-- 把 4 个 rpc_type=UPLOAD 任务类型的 transport_path 末尾 `filename={targetFileName}`
-- 改成 `filename=`（空）。
--
-- 背景：厂商真实抓包样本里 Upload SOAP 的 URL 末尾 filename 字段是空的，
-- 由设备自己决定上传时的命名；ACS 端在 filename 为空时按 (fileType, taskId, sn)
-- 服务端兜底生成（与 target_file_name_template 一致），命名链路保持完整。
--
-- 涉及 4 个 UPLOAD 类型（CONFIG_BACKUP_NV / CONFIG_BACKUP_XML / RUNTIME_LOG_COLLECT /
-- FAULT_LOG_COLLECT）。日志类型顺带补齐 sn / taskId query 参数——空 filename 路径
-- 下 ACS 端要靠它们生成名字。
UPDATE ufte_task_types
SET transport_path = '/smallcell/FileUploadService?fileType=CONFIGBACKUP_NV&sn={sn}&taskId={taskId}&filename=',
    updated_at = NOW()
WHERE type_code = 'CONFIG_BACKUP_NV';

UPDATE ufte_task_types
SET transport_path = '/smallcell/FileUploadService?fileType=CONFIGBACKUP_XML&sn={sn}&taskId={taskId}&filename=',
    updated_at = NOW()
WHERE type_code = 'CONFIG_BACKUP_XML';

UPDATE ufte_task_types
SET transport_path = '/smallcell/FileUploadService?fileType={fileType}&sn={sn}&taskId={taskId}&filename=',
    updated_at = NOW()
WHERE type_code = 'RUNTIME_LOG_COLLECT';

UPDATE ufte_task_types
SET transport_path = '/smallcell/FileUploadService?fileType={fileType}&sn={sn}&taskId={taskId}&filename=',
    updated_at = NOW()
WHERE type_code = 'FAULT_LOG_COLLECT';


-- +goose Down
UPDATE ufte_task_types
SET transport_path = '/smallcell/FileUploadService?fileType=CONFIGBACKUP_NV&sn={sn}&taskId={taskId}&filename={targetFileName}',
    updated_at = NOW()
WHERE type_code = 'CONFIG_BACKUP_NV';

UPDATE ufte_task_types
SET transport_path = '/smallcell/FileUploadService?fileType=CONFIGBACKUP_XML&sn={sn}&taskId={taskId}&filename={targetFileName}',
    updated_at = NOW()
WHERE type_code = 'CONFIG_BACKUP_XML';

UPDATE ufte_task_types
SET transport_path = '/smallcell/FileUploadService?fileType={fileType}&filename={targetFileName}',
    updated_at = NOW()
WHERE type_code = 'RUNTIME_LOG_COLLECT';

UPDATE ufte_task_types
SET transport_path = '/smallcell/FileUploadService?fileType={fileType}&filename={targetFileName}',
    updated_at = NOW()
WHERE type_code = 'FAULT_LOG_COLLECT';
