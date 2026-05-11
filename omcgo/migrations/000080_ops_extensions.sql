-- F06 运维管理 schema 扩展
-- 来源 PRD: docs/project/prd/F06-ops-management.md §7.1
-- 推进计划: docs/project/F06-ops-management-implementation-plan.md T-0112-a

-- +goose Up
ALTER TABLE ops_templates
    ADD COLUMN IF NOT EXISTS risk_level VARCHAR(16) NOT NULL DEFAULT 'safe'
        CHECK (risk_level IN ('safe', 'cautious', 'dangerous')),
    ADD COLUMN IF NOT EXISTS version INT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS change_log JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS rollback_steps JSONB,
    ADD COLUMN IF NOT EXISTS target_carriers JSONB NOT NULL DEFAULT '[]';

ALTER TABLE ops_tasks
    ADD COLUMN IF NOT EXISTS template_snapshot JSONB,
    ADD COLUMN IF NOT EXISTS risk_level VARCHAR(16) NOT NULL DEFAULT 'safe'
        CHECK (risk_level IN ('safe', 'cautious', 'dangerous')),
    ADD COLUMN IF NOT EXISTS approval_state VARCHAR(16) NOT NULL DEFAULT 'not_required'
        CHECK (approval_state IN ('not_required', 'pending', 'approved', 'rejected')),
    ADD COLUMN IF NOT EXISTS approver_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS approved_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS batch_config JSONB,
    ADD COLUMN IF NOT EXISTS failure_policy VARCHAR(16) NOT NULL DEFAULT 'continue'
        CHECK (failure_policy IN ('continue', 'abort', 'retry', 'rollback'));

CREATE INDEX IF NOT EXISTS idx_ops_templates_risk_level ON ops_templates(risk_level);
CREATE INDEX IF NOT EXISTS idx_ops_tasks_approval_state ON ops_tasks(approval_state) WHERE approval_state = 'pending';

-- +goose Down
DROP INDEX IF EXISTS idx_ops_tasks_approval_state;
DROP INDEX IF EXISTS idx_ops_templates_risk_level;
ALTER TABLE ops_tasks
    DROP COLUMN IF EXISTS failure_policy,
    DROP COLUMN IF EXISTS batch_config,
    DROP COLUMN IF EXISTS approved_at,
    DROP COLUMN IF EXISTS approver_user_id,
    DROP COLUMN IF EXISTS approval_state,
    DROP COLUMN IF EXISTS risk_level,
    DROP COLUMN IF EXISTS template_snapshot;
ALTER TABLE ops_templates
    DROP COLUMN IF EXISTS target_carriers,
    DROP COLUMN IF EXISTS rollback_steps,
    DROP COLUMN IF EXISTS owner_user_id,
    DROP COLUMN IF EXISTS change_log,
    DROP COLUMN IF EXISTS version,
    DROP COLUMN IF EXISTS risk_level;
