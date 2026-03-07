CREATE TABLE backup_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_type VARCHAR(20) NOT NULL DEFAULT 'full',
    target_type VARCHAR(20) NOT NULL DEFAULT 'device',
    target_ids JSONB NOT NULL DEFAULT '[]',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    progress INTEGER NOT NULL DEFAULT 0,
    file_path TEXT,
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_backup_tasks_status ON backup_tasks(status);
CREATE INDEX idx_backup_tasks_created ON backup_tasks(created_at DESC);
CREATE INDEX idx_backup_tasks_type ON backup_tasks(task_type);

CREATE TRIGGER trigger_backup_tasks_updated_at
    BEFORE UPDATE ON backup_tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE backup_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    cron_expr VARCHAR(100) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    task_type VARCHAR(20) NOT NULL DEFAULT 'full',
    target_type VARCHAR(20),
    target_ids JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_backup_schedules_enabled ON backup_schedules(enabled);

CREATE TRIGGER trigger_backup_schedules_updated_at
    BEFORE UPDATE ON backup_schedules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
