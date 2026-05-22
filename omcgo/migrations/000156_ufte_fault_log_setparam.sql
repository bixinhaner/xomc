-- +goose Up
-- ============================================================
-- 000152_ufte_fault_log_setparam.sql
-- FAULT_LOG_COLLECT 切换到 SetParameterValues 触发链路
-- ============================================================
-- 厂商私有协议：故障日志采集不走 TR-069 Upload RPC，而是通过 SetParameterValues
-- 把 ACS 端 FileUploadService 的完整 URL 写入设备私有参数
-- Device.DeviceInfo.X_COM_Log.FaultLogURL，CPE 拿到 URL 后异步 HTTP PUT 日志。
--
--   - rpc_type:        UPLOAD                                    → SET_PARAM_VALUES
--   - step_chain:      ...WAIT_TRANSFER_COMPLETE                 → ...WAIT_FILE_UPLOAD
--   - url_template:    （空）                                    → SPV 参数路径
--   - transport_path:  Upload RPC URL (fileType=LOG&taskId32...) → vendor URL (fileType=RL&id&sn&fileName)
--
-- 完成路径调整：CPE PUT 文件到 ACS → upload handler 落 MinIO → 发 backup.file.received
-- → SoftwareService.HandleFileLandedForCollect 把 sub_task 标 Completed。

UPDATE ufte_task_types
SET
    rpc_type       = 'SET_PARAM_VALUES',
    step_chain     = '["CHECK_PERMISSION","CHECK_ONLINE","PRE_VALIDATE","SEND_RPC","WAIT_RPC_RESPONSE","WAIT_FILE_UPLOAD"]'::jsonb,
    url_template   = 'Device.DeviceInfo.X_COM_Log.FaultLogURL',
    transport_path = '/smallcell/FileUploadService?fileType=RL&id={id}&sn={sn}&fileName=',
    description    = '通过 SetParameterValues 写设备私有参数 Device.DeviceInfo.X_COM_Log.FaultLogURL 触发 CPE 主动上传故障日志，落 MinIO 后由 backup.file.received 推进任务完成。',
    last_editor    = 'system',
    updated_at     = now()
WHERE type_code = 'FAULT_LOG_COLLECT';

-- +goose Down
-- 回退到原 Upload RPC 链路。
UPDATE ufte_task_types
SET
    rpc_type       = 'UPLOAD',
    step_chain     = '["CHECK_PERMISSION","CHECK_ONLINE","PRE_VALIDATE","SEND_RPC","WAIT_RPC_RESPONSE","WAIT_TRANSFER_COMPLETE"]'::jsonb,
    url_template   = '',
    transport_path = '/smallcell/FileUploadService?fileType=LOG&sn={sn}&taskId={taskId32}&filename=',
    description    = '复用现网日志采集 Upload 链路，提供 UFTE 内置的故障日志收集模板。',
    last_editor    = 'system',
    updated_at     = now()
WHERE type_code = 'FAULT_LOG_COLLECT';
