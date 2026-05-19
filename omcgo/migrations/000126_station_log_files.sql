-- +goose Up
-- +goose StatementBegin

-- 运行日志采集文件表（file_type="6"）
-- 存储设备主动上传的运行日志元数据，与故障日志分表存储，便于独立扩展和清理。
CREATE TABLE station_running_logs (
    id           UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    device_id    UUID,                                   -- 逻辑引用 devices(id)，分区表不支持 FK
    device_sn    TEXT        NOT NULL,
    file_name    TEXT        NOT NULL,          -- MinIO 对象基础名，如 runtime-abc12345-SN001.tar.gz
    object_path  TEXT        NOT NULL,          -- MinIO 对象全路径，如 running/2026/05/18/runtime-...tar.gz
    bucket       TEXT        NOT NULL,          -- MinIO bucket 名称
    file_size    BIGINT      NOT NULL DEFAULT 0,
    task_id      UUID        REFERENCES upgrade_tasks(id) ON DELETE SET NULL,
    is_deleted   BOOLEAN     NOT NULL DEFAULT FALSE,  -- 文件已从 MinIO 删除，记录保留
    collected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_station_running_logs_device_id  ON station_running_logs(device_id);
CREATE INDEX idx_station_running_logs_device_sn  ON station_running_logs(device_sn);
CREATE INDEX idx_station_running_logs_active     ON station_running_logs(collected_at DESC)
    WHERE is_deleted = FALSE;

-- 故障日志采集文件表（file_type="8"/"RL"，即异常重启日志）
-- 数据量相对较大（每次异常重启均触发上传），独立建表便于单独配额管理和清理。
-- 全局最多保留 20 条（FaultLogMaxCount），超出时自动删除最旧文件。
CREATE TABLE station_fault_logs (
    id           UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    device_id    UUID,                                   -- 逻辑引用 devices(id)，分区表不支持 FK
    device_sn    TEXT        NOT NULL,
    file_name    TEXT        NOT NULL,
    object_path  TEXT        NOT NULL,
    bucket       TEXT        NOT NULL,
    file_size    BIGINT      NOT NULL DEFAULT 0,
    fault_reason TEXT,                          -- HaltReason.MainReason（如 halt_reboot）
    fault_detail TEXT,                          -- HaltReason.DetailReason
    task_id      UUID        REFERENCES upgrade_tasks(id) ON DELETE SET NULL,
    is_deleted   BOOLEAN     NOT NULL DEFAULT FALSE,
    collected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_station_fault_logs_device_id  ON station_fault_logs(device_id);
CREATE INDEX idx_station_fault_logs_device_sn  ON station_fault_logs(device_sn);
CREATE INDEX idx_station_fault_logs_active     ON station_fault_logs(collected_at DESC)
    WHERE is_deleted = FALSE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS station_fault_logs;
DROP TABLE IF EXISTS station_running_logs;
-- +goose StatementEnd
