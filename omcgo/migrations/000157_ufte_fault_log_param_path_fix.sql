-- +goose Up
-- ============================================================
-- 000153_ufte_fault_log_param_path_fix.sql
-- 修正 FAULT_LOG_COLLECT 的 SPV 参数路径
-- ============================================================
-- 000152 把 url_template 设成了 "Device.DeviceInfo.X_COM_Log.FaultLogURL"
-- （按口述字面值写的），CPE 回 SOAP Fault "Empty parameter list"：该路径在
-- CPE 数据模型里不存在。项目自己的 standard_params 字典里（migrations/seed/000126
-- 与 000096）登记的标准路径是 "Device.DeviceInfo.FaultLogURL"（无 .X_COM_Log
-- 中间层），STANDARD 类型 READ_WRITE string，对齐之。

UPDATE ufte_task_types
SET
    url_template = 'Device.DeviceInfo.FaultLogURL',
    description  = '通过 SetParameterValues 写设备私有参数 Device.DeviceInfo.FaultLogURL 触发 CPE 主动上传故障日志，落 MinIO 后由 backup.file.received 推进任务完成。',
    last_editor  = 'system',
    updated_at   = now()
WHERE type_code = 'FAULT_LOG_COLLECT';

-- +goose Down
UPDATE ufte_task_types
SET
    url_template = 'Device.DeviceInfo.X_COM_Log.FaultLogURL',
    description  = '通过 SetParameterValues 写设备私有参数 Device.DeviceInfo.X_COM_Log.FaultLogURL 触发 CPE 主动上传故障日志，落 MinIO 后由 backup.file.received 推进任务完成。',
    last_editor  = 'system',
    updated_at   = now()
WHERE type_code = 'FAULT_LOG_COLLECT';
