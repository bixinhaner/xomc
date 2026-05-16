-- +goose Up
-- ============================================================
-- 000107_trace_export_jobs.sql
-- T-0137-M2 异步下载支持 — 用户触发 POST /tasks/{id}/export 时创建一行；
-- worker 消费 trace.export.requested 后生成 XML 写 MinIO exchange，回写 status + object_key。
-- 完成后前端轮询 GET /exports/{job_id} 拿预签名 URL 下载。
-- ============================================================

CREATE TABLE trace_export_jobs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id       UUID NOT NULL REFERENCES trace_tasks(id) ON DELETE CASCADE,
    requested_by  VARCHAR(64) NOT NULL DEFAULT 'system',
    status        VARCHAR(16) NOT NULL DEFAULT 'queued'
                  CHECK (status IN ('queued', 'running', 'done', 'failed')),
    object_key    VARCHAR(512),
    object_bucket VARCHAR(64),
    message_count INTEGER NOT NULL DEFAULT 0,
    size_bytes    BIGINT  NOT NULL DEFAULT 0,
    error_message TEXT,
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trace_export_jobs_task ON trace_export_jobs (task_id);
CREATE INDEX idx_trace_export_jobs_status ON trace_export_jobs (status, created_at DESC);

CREATE TRIGGER trigger_trace_export_jobs_updated_at
    BEFORE UPDATE ON trace_export_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
DROP TRIGGER IF EXISTS trigger_trace_export_jobs_updated_at ON trace_export_jobs;
DROP INDEX IF EXISTS idx_trace_export_jobs_status;
DROP INDEX IF EXISTS idx_trace_export_jobs_task;
DROP TABLE IF EXISTS trace_export_jobs CASCADE;
