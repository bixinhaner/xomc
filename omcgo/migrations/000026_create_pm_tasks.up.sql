CREATE TABLE pm_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_name VARCHAR(200) NOT NULL,
    task_type VARCHAR(30) NOT NULL DEFAULT 'extraction',
    device_sns JSONB NOT NULL DEFAULT '[]',
    kpi_codes JSONB NOT NULL DEFAULT '[]',
    granularity VARCHAR(10) NOT NULL DEFAULT '15min',
    time_range JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    progress INTEGER NOT NULL DEFAULT 0,
    creator VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_pm_tasks_status ON pm_tasks(status);
CREATE INDEX idx_pm_tasks_created ON pm_tasks(created_at DESC);

CREATE TRIGGER trigger_pm_tasks_updated_at
    BEFORE UPDATE ON pm_tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
