-- 设备规则应用任务表
-- 用于跟踪规则应用的异步任务状态

CREATE TABLE device_rule_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id         UUID NOT NULL REFERENCES device_rules(id) ON DELETE CASCADE,
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    total_devices   INTEGER DEFAULT 0,
    matched_count   INTEGER DEFAULT 0,
    failed_count    INTEGER DEFAULT 0,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    error_message   TEXT,
    created_by      VARCHAR(64),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_device_rule_tasks_rule ON device_rule_tasks (rule_id);
CREATE INDEX idx_device_rule_tasks_status ON device_rule_tasks (status);
CREATE INDEX idx_device_rule_tasks_created ON device_rule_tasks (created_at DESC);

-- 注释
COMMENT ON TABLE device_rule_tasks IS '设备规则应用任务';
COMMENT ON COLUMN device_rule_tasks.id IS '任务唯一标识';
COMMENT ON COLUMN device_rule_tasks.rule_id IS '关联的规则ID';
COMMENT ON COLUMN device_rule_tasks.status IS '任务状态: pending(待执行), running(执行中), completed(已完成), failed(失败)';
COMMENT ON COLUMN device_rule_tasks.total_devices IS '总设备数';
COMMENT ON COLUMN device_rule_tasks.matched_count IS '匹配成功数';
COMMENT ON COLUMN device_rule_tasks.failed_count IS '失败数';
COMMENT ON COLUMN device_rule_tasks.started_at IS '开始时间';
COMMENT ON COLUMN device_rule_tasks.completed_at IS '完成时间';
