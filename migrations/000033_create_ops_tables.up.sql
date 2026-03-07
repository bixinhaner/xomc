-- Ops templates
CREATE TABLE ops_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_name VARCHAR(200) NOT NULL,
    description TEXT,
    category VARCHAR(50),
    target_device_types JSONB NOT NULL DEFAULT '[]',
    steps JSONB NOT NULL DEFAULT '[]',
    estimated_duration INTEGER NOT NULL DEFAULT 0,
    creator VARCHAR(100),
    use_count INTEGER NOT NULL DEFAULT 0,
    tags JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ops_templates_category ON ops_templates(category);
CREATE INDEX idx_ops_templates_created ON ops_templates(created_at DESC);
CREATE TRIGGER trigger_ops_templates_updated_at
    BEFORE UPDATE ON ops_templates FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Ops tasks
CREATE TABLE ops_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_name VARCHAR(200) NOT NULL,
    template_id UUID REFERENCES ops_templates(id) ON DELETE SET NULL,
    device_sns JSONB NOT NULL DEFAULT '[]',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    current_step INTEGER NOT NULL DEFAULT 0,
    total_steps INTEGER NOT NULL DEFAULT 0,
    progress INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    fail_count INTEGER NOT NULL DEFAULT 0,
    total_count INTEGER NOT NULL DEFAULT 0,
    creator VARCHAR(100),
    message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ops_tasks_status ON ops_tasks(status);
CREATE INDEX idx_ops_tasks_template ON ops_tasks(template_id);
CREATE INDEX idx_ops_tasks_created ON ops_tasks(created_at DESC);
CREATE TRIGGER trigger_ops_tasks_updated_at
    BEFORE UPDATE ON ops_tasks FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Ops command records
CREATE TABLE ops_command_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    command_text TEXT NOT NULL,
    device_sn VARCHAR(64) NOT NULL,
    device_name VARCHAR(200),
    operator VARCHAR(100),
    execute_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    duration INTEGER NOT NULL DEFAULT 0,
    success BOOLEAN NOT NULL DEFAULT false,
    output TEXT,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ops_cmd_records_device ON ops_command_records(device_sn);
CREATE INDEX idx_ops_cmd_records_time ON ops_command_records(execute_time DESC);
CREATE INDEX idx_ops_cmd_records_operator ON ops_command_records(operator);
