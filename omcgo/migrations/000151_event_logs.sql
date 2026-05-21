-- +goose Up
-- +goose StatementBegin

-- 事件日志（设备活动审计流水）
--
-- 与 station_fault_logs 语义分开：
--   - station_fault_logs：异常重启专表（1 BOOT + HaltReason 非空），含文件管理
--   - event_logs：通用设备事件审计（本期只接 1 BOOT 普通重启 = 无 HaltReason）
--
-- 字段对应前端 EventLog 页面列：
--   device_sn ↔ neCode / operate_ip ↔ neIpAddress / event_type ↔ eventName /
--   event_reason ↔ eventReason / occurred_at ↔ time / event_level ↔ eventLevel
CREATE TABLE IF NOT EXISTS event_logs (
    id              UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    device_id       UUID,                                    -- 逻辑引用 devices(id)，分区表不支持 FK
    device_sn       TEXT        NOT NULL,
    device_name     TEXT,                                    -- freeze 当时设备名（与 station_fault_logs 对齐）
    device_type     TEXT,                                    -- eNB / gNB
    is_gnb          BOOLEAN     NOT NULL DEFAULT FALSE,
    operate_ip      TEXT,
    software_version TEXT,
    event_type      TEXT        NOT NULL,                    -- 'boot' / 后续 'connection_lost' / 'config_change' / ...
    event_reason    TEXT,                                    -- 人类可读原因（如 "设备重启完成"）
    event_level     TEXT        NOT NULL DEFAULT 'info'      -- info / warning / error / success
        CHECK (event_level IN ('info', 'warning', 'error', 'success')),
    event_data      JSONB,                                   -- 扩展信息（如 boot_count / events 列表）
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_event_logs_device_sn_time
    ON event_logs(device_sn, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_event_logs_event_type_time
    ON event_logs(event_type, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_event_logs_occurred_at
    ON event_logs(occurred_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS event_logs;
-- +goose StatementEnd
