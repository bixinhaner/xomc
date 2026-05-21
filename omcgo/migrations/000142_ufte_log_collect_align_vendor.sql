-- +goose Up
-- 把运行日志 / 故障日志收集的 file_type + transport_path 对齐厂商 baicells/MMMM 真实样本：
--
-- 真实样本（运行日志）：
--   FileType: 4 Vendor Log File 1,2,3,4         （TR-069 §A.3.2.9 标准 vendor log file 编号）
--   URL:     ?fileType=LOG&taskId=<32hex>&filename=
--
-- 旧实现的问题：
--   · file_type='6' 是项目自定义编号，跟 TR-069 spec 不对齐，CPE 不识别
--   · transport_path 多了 sn={sn} query 参数（厂商样本只有 fileType+taskId+filename）
--   · taskId 用 UUID 36 字符含连字符（厂商样本用 32 字符纯 hex）—— 模板里换成 {taskId32}
--
-- 故障日志暂保留旧 file_type='8'（厂商样本未提供，等真实样本再调）。
UPDATE ufte_task_types
SET file_type = '4 Vendor Log File 1,2,3,4',
    file_type_label = '4 Vendor Log File 1,2,3,4',
    transport_path = '/smallcell/FileUploadService?fileType=LOG&taskId={taskId32}&filename=',
    updated_at = NOW()
WHERE type_code = 'RUNTIME_LOG_COLLECT';

UPDATE ufte_task_types
SET transport_path = '/smallcell/FileUploadService?fileType=LOG&taskId={taskId32}&filename=',
    updated_at = NOW()
WHERE type_code = 'FAULT_LOG_COLLECT';


-- +goose Down
UPDATE ufte_task_types
SET file_type = '6',
    file_type_label = '运行日志',
    transport_path = '/smallcell/FileUploadService?fileType={fileType}&sn={sn}&taskId={taskId}&filename=',
    updated_at = NOW()
WHERE type_code = 'RUNTIME_LOG_COLLECT';

UPDATE ufte_task_types
SET transport_path = '/smallcell/FileUploadService?fileType={fileType}&sn={sn}&taskId={taskId}&filename=',
    updated_at = NOW()
WHERE type_code = 'FAULT_LOG_COLLECT';
