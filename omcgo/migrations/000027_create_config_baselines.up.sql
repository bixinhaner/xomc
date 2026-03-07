-- Baseline config table
CREATE TABLE config_baselines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    baseline_name VARCHAR(200) NOT NULL,
    description TEXT,
    device_type VARCHAR(50),
    version VARCHAR(50),
    params JSONB NOT NULL DEFAULT '[]',
    creator VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_config_baselines_status ON config_baselines(status);
CREATE INDEX idx_config_baselines_device_type ON config_baselines(device_type);
CREATE TRIGGER trigger_config_baselines_updated_at
    BEFORE UPDATE ON config_baselines FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Config task table
CREATE TABLE config_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_name VARCHAR(200) NOT NULL,
    task_type VARCHAR(30) NOT NULL,
    device_sns JSONB NOT NULL DEFAULT '[]',
    template_id UUID,
    baseline_id UUID,
    params JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    progress INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    fail_count INTEGER NOT NULL DEFAULT 0,
    total_count INTEGER NOT NULL DEFAULT 0,
    creator VARCHAR(100),
    message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_config_tasks_status ON config_tasks(status);
CREATE INDEX idx_config_tasks_created ON config_tasks(created_at DESC);
CREATE TRIGGER trigger_config_tasks_updated_at
    BEFORE UPDATE ON config_tasks FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Neighbor param table
CREATE TABLE config_neighbors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_cell_id VARCHAR(64) NOT NULL,
    source_cell_name VARCHAR(200),
    target_cell_id VARCHAR(64) NOT NULL,
    target_cell_name VARCHAR(200),
    neighbor_type VARCHAR(20) NOT NULL DEFAULT 'intra-freq',
    params JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_config_neighbors_source ON config_neighbors(source_cell_id);
CREATE INDEX idx_config_neighbors_target ON config_neighbors(target_cell_id);
CREATE TRIGGER trigger_config_neighbors_updated_at
    BEFORE UPDATE ON config_neighbors FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
