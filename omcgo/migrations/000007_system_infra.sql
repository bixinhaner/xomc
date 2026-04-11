-- +goose Up
-- ============================================================
-- 000007_system_infra.up.sql
-- Audit logs, system logs, NE message logs, sites, topology nodes/edges,
-- report definitions/records, ops templates/tasks/command records,
-- MML commands/scripts/tasks
-- ============================================================

-- ============================================================
-- 1. audit_logs — user action audit trail
-- ============================================================
CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    username    VARCHAR(64) NOT NULL,
    action      VARCHAR(64) NOT NULL,
    resource    VARCHAR(64),
    resource_id VARCHAR(128),
    details     JSONB,
    ip_address  INET,
    user_agent  VARCHAR(256),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user_time ON audit_logs (user_id, created_at DESC);
CREATE INDEX idx_audit_logs_time ON audit_logs (created_at DESC);
CREATE INDEX idx_audit_logs_resource ON audit_logs (resource, resource_id, created_at DESC);

-- ============================================================
-- 2. system_logs — application-level logging
-- ============================================================
CREATE TABLE IF NOT EXISTS system_logs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    level      VARCHAR(16) NOT NULL,
    source     VARCHAR(128),
    message    TEXT NOT NULL,
    details    JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_system_logs_created_at ON system_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_system_logs_level ON system_logs (level);
CREATE INDEX IF NOT EXISTS idx_system_logs_source ON system_logs (source);

-- ============================================================
-- 3. ne_message_logs — network element communication logging
-- ============================================================
CREATE TABLE IF NOT EXISTS ne_message_logs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_sn    VARCHAR(128) NOT NULL,
    device_id    UUID,
    message_type VARCHAR(64) NOT NULL,
    direction    VARCHAR(8) NOT NULL,
    content      TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ne_message_logs_created_at ON ne_message_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ne_message_logs_device_sn ON ne_message_logs (device_sn);
CREATE INDEX IF NOT EXISTS idx_ne_message_logs_device_id ON ne_message_logs (device_id);

-- ============================================================
-- 4. sites — GIS / topology site management
-- ============================================================
CREATE TABLE sites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    domain_id UUID REFERENCES device_groups(id) ON DELETE SET NULL,
    address TEXT,
    longitude DOUBLE PRECISION,
    latitude DOUBLE PRECISION,
    device_count INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sites_domain ON sites(domain_id);
CREATE INDEX idx_sites_status ON sites(status);
CREATE INDEX idx_sites_geo ON sites(longitude, latitude);

CREATE TRIGGER trigger_sites_updated_at
    BEFORE UPDATE ON sites FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 5. topo_nodes — topology graph nodes
-- ============================================================
CREATE TABLE topo_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    label VARCHAR(200) NOT NULL,
    node_type VARCHAR(20) NOT NULL,
    x DOUBLE PRECISION NOT NULL DEFAULT 0,
    y DOUBLE PRECISION NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'online',
    device_sn VARCHAR(64),
    site_id UUID REFERENCES sites(id) ON DELETE SET NULL,
    domain_id UUID REFERENCES device_groups(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_topo_nodes_type ON topo_nodes(node_type);
CREATE INDEX idx_topo_nodes_domain ON topo_nodes(domain_id);
CREATE INDEX idx_topo_nodes_device ON topo_nodes(device_sn);

CREATE TRIGGER trigger_topo_nodes_updated_at
    BEFORE UPDATE ON topo_nodes FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 6. topo_edges — topology graph edges
-- ============================================================
CREATE TABLE topo_edges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES topo_nodes(id) ON DELETE CASCADE,
    target_id UUID NOT NULL REFERENCES topo_nodes(id) ON DELETE CASCADE,
    label VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_topo_edges_source ON topo_edges(source_id);
CREATE INDEX idx_topo_edges_target ON topo_edges(target_id);

-- ============================================================
-- 7. report_definitions — report templates
-- ============================================================
CREATE TABLE report_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_name VARCHAR(200) NOT NULL,
    report_type VARCHAR(20) NOT NULL,
    description TEXT,
    format JSONB NOT NULL DEFAULT '["pdf"]',
    period VARCHAR(20) NOT NULL DEFAULT 'daily',
    kpi_codes JSONB DEFAULT '[]',
    device_groups JSONB DEFAULT '[]',
    auto_generate BOOLEAN NOT NULL DEFAULT false,
    cron_expression VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    creator VARCHAR(100),
    last_gen_time TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_report_defs_type ON report_definitions(report_type);
CREATE INDEX idx_report_defs_status ON report_definitions(status);

CREATE TRIGGER trigger_report_definitions_updated_at
    BEFORE UPDATE ON report_definitions FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 8. report_records — generated report instances
-- ============================================================
CREATE TABLE report_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_definition_id UUID NOT NULL REFERENCES report_definitions(id) ON DELETE CASCADE,
    report_name VARCHAR(200) NOT NULL,
    period VARCHAR(100),
    generate_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    file_size BIGINT NOT NULL DEFAULT 0,
    download_url TEXT,
    format VARCHAR(10) NOT NULL DEFAULT 'pdf',
    status VARCHAR(20) NOT NULL DEFAULT 'generating',
    minio_path TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_report_records_def ON report_records(report_definition_id);
CREATE INDEX idx_report_records_status ON report_records(status);
CREATE INDEX idx_report_records_time ON report_records(generate_time DESC);

-- ============================================================
-- 9. ops_templates — operations templates
-- ============================================================
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

-- ============================================================
-- 10. ops_tasks — operations task execution tracking
-- ============================================================
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

-- ============================================================
-- 11. ops_command_records — CLI/MML command execution history
-- ============================================================
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

-- ============================================================
-- 12. mml_commands — predefined MML command catalog
-- ============================================================
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

-- ============================================================
-- 13. mml_scripts — user-defined MML script library
-- ============================================================
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

-- ============================================================
-- 14. mml_tasks — MML script batch execution tasks
-- ============================================================
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

-- ============================================================
-- Seed data: Predefined MML commands
-- ============================================================
INSERT INTO mml_commands (command_name, command_code, category, rpc_method, description, param_template) VALUES
    ('查询设备参数', 'LST_DEVPARAM', 'query', 'GetParameterValues', '查询设备TR069参数树', '{"parameter_path": "Device."}'),
    ('设置设备参数', 'SET_DEVPARAM', 'config', 'SetParameterValues', '设置设备TR069参数', '{"parameters": []}'),
    ('设备重启', 'RST_DEV', 'maintenance', 'Reboot', '远程重启设备', '{}');

-- +goose Down
DROP TABLE IF EXISTS mml_tasks CASCADE;
DROP TABLE IF EXISTS mml_scripts CASCADE;
DROP TABLE IF EXISTS mml_commands CASCADE;
DROP TABLE IF EXISTS ops_command_records CASCADE;
DROP TABLE IF EXISTS ops_tasks CASCADE;
DROP TABLE IF EXISTS ops_templates CASCADE;
DROP TABLE IF EXISTS report_records CASCADE;
DROP TABLE IF EXISTS report_definitions CASCADE;
DROP TABLE IF EXISTS topo_edges CASCADE;
DROP TABLE IF EXISTS topo_nodes CASCADE;
DROP TABLE IF EXISTS sites CASCADE;
DROP TABLE IF EXISTS ne_message_logs CASCADE;
DROP TABLE IF EXISTS system_logs CASCADE;
DROP TABLE IF EXISTS audit_logs CASCADE;
