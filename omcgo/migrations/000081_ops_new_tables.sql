-- F06 运维管理 6 张新表
-- 来源 PRD: docs/project/prd/F06-ops-management.md §7.2
-- 推进计划: T-0112-b

-- +goose Up

-- 1. 任务执行明细（device × step）
CREATE TABLE IF NOT EXISTS ops_task_executions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id         UUID NOT NULL REFERENCES ops_tasks(id) ON DELETE CASCADE,
    device_sn       VARCHAR(64) NOT NULL,
    step_index      INT NOT NULL,
    step_name       VARCHAR(200) NOT NULL,
    step_type       VARCHAR(32) NOT NULL,
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    duration_ms     INT,
    request         JSONB,
    response        JSONB,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ops_task_exec_task    ON ops_task_executions(task_id);
CREATE INDEX IF NOT EXISTS idx_ops_task_exec_device  ON ops_task_executions(device_sn);
CREATE INDEX IF NOT EXISTS idx_ops_task_exec_status  ON ops_task_executions(status);

-- 2. 网络诊断记录
CREATE TABLE IF NOT EXISTS ops_diagnostics (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_sn       VARCHAR(64),
    diag_type       VARCHAR(32) NOT NULL,
    initiator       VARCHAR(16) NOT NULL DEFAULT 'omc'
        CHECK (initiator IN ('device', 'omc')),
    request         JSONB NOT NULL DEFAULT '{}',
    result          JSONB,
    status          VARCHAR(16) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'running', 'complete', 'failed', 'timeout')),
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    duration_ms     INT,
    operator        VARCHAR(128),
    task_id         UUID REFERENCES ops_tasks(id) ON DELETE SET NULL,
    file_path       VARCHAR(500),
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ops_diag_device   ON ops_diagnostics(device_sn);
CREATE INDEX IF NOT EXISTS idx_ops_diag_type     ON ops_diagnostics(diag_type);
CREATE INDEX IF NOT EXISTS idx_ops_diag_started  ON ops_diagnostics(started_at DESC);

-- 3. 运维下载记录
CREATE TABLE IF NOT EXISTS ops_downloads (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_sn       VARCHAR(64) NOT NULL,
    content_type    VARCHAR(32) NOT NULL,
    file_path       VARCHAR(500) NOT NULL DEFAULT '',
    file_size       BIGINT NOT NULL DEFAULT 0,
    checksum        VARCHAR(64),
    status          VARCHAR(16) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'uploading', 'complete', 'failed', 'expired')),
    operator        VARCHAR(128),
    task_id         UUID REFERENCES ops_tasks(id) ON DELETE SET NULL,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ops_dl_device   ON ops_downloads(device_sn);
CREATE INDEX IF NOT EXISTS idx_ops_dl_type     ON ops_downloads(content_type);
CREATE INDEX IF NOT EXISTS idx_ops_dl_created  ON ops_downloads(created_at DESC);

-- 4. 运维审计日志（独立高频写）
CREATE TABLE IF NOT EXISTS ops_audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    op_type         VARCHAR(32) NOT NULL,
    target_type     VARCHAR(16) NOT NULL,
    target_id       VARCHAR(128) NOT NULL,
    operator_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    operator_name   VARCHAR(128) NOT NULL DEFAULT '',
    risk_level      VARCHAR(16) NOT NULL DEFAULT 'safe',
    input           JSONB,
    output_summary  TEXT,
    result          VARCHAR(16) NOT NULL DEFAULT 'success',
    approver_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    break_glass     BOOLEAN NOT NULL DEFAULT FALSE,
    client_ip       INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ops_audit_operator ON ops_audit_logs(operator_user_id);
CREATE INDEX IF NOT EXISTS idx_ops_audit_target   ON ops_audit_logs(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_ops_audit_created  ON ops_audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ops_audit_bg       ON ops_audit_logs(break_glass) WHERE break_glass = TRUE;

-- 5. 维护窗口
CREATE TABLE IF NOT EXISTS ops_maintenance_windows (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(200) NOT NULL,
    scope_type          VARCHAR(16) NOT NULL DEFAULT 'device'
        CHECK (scope_type IN ('device', 'group', 'all')),
    scope_ids           JSONB NOT NULL DEFAULT '[]',
    start_at            TIMESTAMPTZ NOT NULL,
    end_at              TIMESTAMPTZ NOT NULL,
    suppress_alarms     BOOLEAN NOT NULL DEFAULT TRUE,
    pause_provision     BOOLEAN NOT NULL DEFAULT TRUE,
    allow_dangerous     BOOLEAN NOT NULL DEFAULT TRUE,
    reason              TEXT,
    creator_user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    approver_user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    status              VARCHAR(16) NOT NULL DEFAULT 'planned'
        CHECK (status IN ('planned', 'approved', 'active', 'ended', 'cancelled')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ops_mw_status ON ops_maintenance_windows(status);
CREATE INDEX IF NOT EXISTS idx_ops_mw_active ON ops_maintenance_windows(start_at, end_at) WHERE status = 'active';
-- DROP + CREATE 兜底幂等：PG 14 之前 CREATE TRIGGER 无 IF NOT EXISTS 语法，
-- 半失败重跑或 docker volume 残留场景下裸 CREATE 会报 SQLSTATE 42710
DROP TRIGGER IF EXISTS trigger_ops_mw_updated_at ON ops_maintenance_windows;
CREATE TRIGGER trigger_ops_mw_updated_at BEFORE UPDATE ON ops_maintenance_windows
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 6. 运维知识库 Playbook
CREATE TABLE IF NOT EXISTS ops_playbooks (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alarm_pattern         JSONB NOT NULL DEFAULT '{}',
    recommended_templates JSONB NOT NULL DEFAULT '[]',
    title                 VARCHAR(200) NOT NULL,
    docs                  TEXT,
    tags                  JSONB NOT NULL DEFAULT '[]',
    success_rate          NUMERIC(5,2),
    use_count             INT NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ops_pb_use_count ON ops_playbooks(use_count DESC);
DROP TRIGGER IF EXISTS trigger_ops_pb_updated_at ON ops_playbooks;
CREATE TRIGGER trigger_ops_pb_updated_at BEFORE UPDATE ON ops_playbooks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
DROP TRIGGER IF EXISTS trigger_ops_pb_updated_at ON ops_playbooks;
DROP TABLE IF EXISTS ops_playbooks CASCADE;
DROP TRIGGER IF EXISTS trigger_ops_mw_updated_at ON ops_maintenance_windows;
DROP TABLE IF EXISTS ops_maintenance_windows CASCADE;
DROP TABLE IF EXISTS ops_audit_logs CASCADE;
DROP TABLE IF EXISTS ops_downloads CASCADE;
DROP TABLE IF EXISTS ops_diagnostics CASCADE;
DROP TABLE IF EXISTS ops_task_executions CASCADE;
