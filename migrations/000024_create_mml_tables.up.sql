-- MML (Man-Machine Language) console tables for predefined commands, user scripts, and execution tasks.

CREATE TABLE mml_commands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    command_name VARCHAR(200) NOT NULL,
    command_code VARCHAR(100) NOT NULL UNIQUE,
    category VARCHAR(50),
    description TEXT,
    rpc_method VARCHAR(50) NOT NULL,
    param_template JSONB,
    product_types JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE mml_scripts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    script_name VARCHAR(200) NOT NULL,
    description TEXT,
    content TEXT NOT NULL,
    device_type VARCHAR(50),
    creator VARCHAR(100),
    tags JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trigger_mml_scripts_updated_at
    BEFORE UPDATE ON mml_scripts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE mml_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_name VARCHAR(200),
    script_id UUID REFERENCES mml_scripts(id) ON DELETE SET NULL,
    device_sns JSONB NOT NULL,
    commands JSONB NOT NULL DEFAULT '[]',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    results JSONB DEFAULT '[]',
    creator VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mml_tasks_status ON mml_tasks(status);
CREATE INDEX idx_mml_tasks_created ON mml_tasks(created_at DESC);

CREATE TRIGGER trigger_mml_tasks_updated_at
    BEFORE UPDATE ON mml_tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Seed predefined MML commands
INSERT INTO mml_commands (command_name, command_code, category, rpc_method, description, param_template) VALUES
    ('查询设备参数', 'LST_DEVPARAM', 'query', 'GetParameterValues', '查询设备TR069参数树', '{"parameter_path": "Device."}'),
    ('设置设备参数', 'SET_DEVPARAM', 'config', 'SetParameterValues', '设置设备TR069参数', '{"parameters": []}'),
    ('设备重启', 'RST_DEV', 'maintenance', 'Reboot', '远程重启设备', '{}');
