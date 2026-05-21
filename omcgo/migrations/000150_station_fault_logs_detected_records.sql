-- +goose Up
-- +goose StatementBegin

-- T-0158: 让 station_fault_logs 同时承载"识别即落库"（HaltReason 触发，无文件）和
-- "文件上传后补全"两种生命周期。原表强制 file_name/object_path/bucket NOT NULL，
-- 仅支持后者；本迁移放开这三列并新增设备快照 + 收集状态字段。

ALTER TABLE station_fault_logs
    ALTER COLUMN file_name   DROP NOT NULL,
    ALTER COLUMN object_path DROP NOT NULL,
    ALTER COLUMN bucket      DROP NOT NULL;

ALTER TABLE station_fault_logs
    ADD COLUMN IF NOT EXISTS device_name              TEXT,
    ADD COLUMN IF NOT EXISTS device_type              TEXT,
    ADD COLUMN IF NOT EXISTS is_gnb                   BOOLEAN     NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS operate_ip               TEXT,
    ADD COLUMN IF NOT EXISTS software_version         TEXT,
    ADD COLUMN IF NOT EXISTS runtime_before_reboot    BIGINT,
    ADD COLUMN IF NOT EXISTS record_status            TEXT        NOT NULL DEFAULT 'detected',
    ADD COLUMN IF NOT EXISTS collection_fail_reason   TEXT,
    ADD COLUMN IF NOT EXISTS manual_collection_status TEXT        NOT NULL DEFAULT '0';

-- record_status 域：detected | file_received | collection_failed
ALTER TABLE station_fault_logs
    DROP CONSTRAINT IF EXISTS station_fault_logs_record_status_check;
ALTER TABLE station_fault_logs
    ADD CONSTRAINT station_fault_logs_record_status_check
    CHECK (record_status IN ('detected', 'file_received', 'collection_failed'));

-- manual_collection_status 域：0 未收集/已完成 | 1 收集中 | 2 失败
ALTER TABLE station_fault_logs
    DROP CONSTRAINT IF EXISTS station_fault_logs_manual_collection_status_check;
ALTER TABLE station_fault_logs
    ADD CONSTRAINT station_fault_logs_manual_collection_status_check
    CHECK (manual_collection_status IN ('0', '1', '2'));

CREATE INDEX IF NOT EXISTS idx_station_fault_logs_status
    ON station_fault_logs(record_status, collected_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_station_fault_logs_status;

ALTER TABLE station_fault_logs
    DROP CONSTRAINT IF EXISTS station_fault_logs_record_status_check;
ALTER TABLE station_fault_logs
    DROP CONSTRAINT IF EXISTS station_fault_logs_manual_collection_status_check;

ALTER TABLE station_fault_logs
    DROP COLUMN IF EXISTS manual_collection_status,
    DROP COLUMN IF EXISTS collection_fail_reason,
    DROP COLUMN IF EXISTS record_status,
    DROP COLUMN IF EXISTS runtime_before_reboot,
    DROP COLUMN IF EXISTS software_version,
    DROP COLUMN IF EXISTS operate_ip,
    DROP COLUMN IF EXISTS is_gnb,
    DROP COLUMN IF EXISTS device_type,
    DROP COLUMN IF EXISTS device_name;

-- Down 不强制把 file_name/object_path/bucket 改回 NOT NULL：在已生成 detected
-- 占位记录的数据库上，这三列会有真实 NULL 行，强加 NOT NULL 会失败。
-- 如需完全回退到 000126 的形态，需先 DELETE 这些占位记录再手工 ALTER。

-- +goose StatementEnd
