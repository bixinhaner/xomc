-- +goose Up
-- 在运行日志/故障日志收集的 transport_path 里加回 sn 参数。
--
-- 背景：000142 对齐厂商 baicells/MMMM 样本时去掉了 sn 参数，结果 ACS upload handler
-- 在 publishBackupFileReceivedEvent 拿不到 device_sn → FilePathRecorder.upsertFileMetadata
-- 因 sn 为空跳过写库 → backup_restore_file 没行 → 前端「目标文件名」列空。
-- 设备 PUT 时会原样保留额外 query 参数（NV/XML 路径已验证），加 sn 不影响 CPE 行为，
-- 但服务端能用 sn 写元数据→前端展示文件名+下载。
UPDATE ufte_task_types
SET transport_path = '/smallcell/FileUploadService?fileType=LOG&sn={sn}&taskId={taskId32}&filename=',
    updated_at = NOW()
WHERE type_code IN ('RUNTIME_LOG_COLLECT','FAULT_LOG_COLLECT');


-- +goose Down
UPDATE ufte_task_types
SET transport_path = '/smallcell/FileUploadService?fileType=LOG&taskId={taskId32}&filename=',
    updated_at = NOW()
WHERE type_code IN ('RUNTIME_LOG_COLLECT','FAULT_LOG_COLLECT');
