-- +goose Up
-- ============================================================
-- 000106_trace_message.sql
-- T-0137-M1 TR069 报文跟踪（Message Trace）— MVP schema
-- 设计：docs/design/TR069报文跟踪-设计.md §4-6
-- 决策：D6 独立 internal/trace/ 模块 / D8 retention 3 天 / D2 只存 sn 不存 device_id
-- ============================================================

-- ============================================================
-- 1. trace_tasks — 抓包任务表（单 SN 在 running 期最多一条记录，靠 partial unique index）
-- ============================================================
CREATE TABLE trace_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_sn       VARCHAR(64) NOT NULL,
    operator_code   VARCHAR(16) NOT NULL DEFAULT 'default',
    status          VARCHAR(16) NOT NULL DEFAULT 'running'
                    CHECK (status IN ('running', 'stopped', 'purged')),
    start_time      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    stopped_at      TIMESTAMPTZ,
    purged_at       TIMESTAMPTZ,
    created_by      VARCHAR(64) NOT NULL DEFAULT 'system',
    message_count   INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trace_tasks_sn_status ON trace_tasks (device_sn, status);
CREATE INDEX idx_trace_tasks_status_expires ON trace_tasks (status, expires_at)
    WHERE status = 'running';
CREATE INDEX idx_trace_tasks_created ON trace_tasks (created_at DESC);

-- 同一 SN 同一时刻最多一个 running 任务（M1 新建若冲突由 service 显式覆盖旧任务）
CREATE UNIQUE INDEX uniq_trace_tasks_running_sn
    ON trace_tasks (device_sn) WHERE status = 'running';

CREATE TRIGGER trigger_trace_tasks_updated_at
    BEFORE UPDATE ON trace_tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 2. trace_messages — 报文表（TimescaleDB hypertable，按 captured_at 分块，3 天 retention）
-- ============================================================
CREATE TABLE trace_messages (
    captured_at         TIMESTAMPTZ NOT NULL,
    id                  UUID NOT NULL DEFAULT gen_random_uuid(),
    task_id             UUID NOT NULL,
    device_sn           VARCHAR(64) NOT NULL,
    direction           VARCHAR(8) NOT NULL
                        CHECK (direction IN ('in', 'out')),
    rpc_method          VARCHAR(64),
    cwmp_id             VARCHAR(64),
    session_id          VARCHAR(64),
    http_status         SMALLINT,
    payload_size_bytes  INTEGER NOT NULL DEFAULT 0,
    payload_inline      TEXT,
    payload_object_key  VARCHAR(512)
);

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        PERFORM create_hypertable('trace_messages', 'captured_at',
            chunk_time_interval => INTERVAL '1 day', if_not_exists => TRUE);
        PERFORM add_retention_policy('trace_messages', INTERVAL '3 days', if_not_exists => TRUE);
    END IF;
END $$;
-- +goose StatementEnd

CREATE INDEX idx_trace_messages_task_time ON trace_messages (task_id, captured_at DESC);
CREATE INDEX idx_trace_messages_sn_time ON trace_messages (device_sn, captured_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_trace_messages_sn_time;
DROP INDEX IF EXISTS idx_trace_messages_task_time;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        PERFORM remove_retention_policy('trace_messages', if_exists => TRUE);
    END IF;
END $$;
-- +goose StatementEnd

DROP TABLE IF EXISTS trace_messages CASCADE;

DROP TRIGGER IF EXISTS trigger_trace_tasks_updated_at ON trace_tasks;
DROP INDEX IF EXISTS uniq_trace_tasks_running_sn;
DROP INDEX IF EXISTS idx_trace_tasks_created;
DROP INDEX IF EXISTS idx_trace_tasks_status_expires;
DROP INDEX IF EXISTS idx_trace_tasks_sn_status;
DROP TABLE IF EXISTS trace_tasks CASCADE;
