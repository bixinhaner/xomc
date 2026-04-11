-- +goose Up
-- ============================================================
-- 000005_config_pm.up.sql
-- Config templates, baselines, tasks, neighbors, provisioning,
-- FTP configs, PM counters/files/tasks, KPI definitions/values/thresholds,
-- hourly continuous aggregate
-- ============================================================

-- ============================================================
-- 1. config_templates — provisioning and configuration templates
-- ============================================================
CREATE TABLE config_templates (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           VARCHAR(128) NOT NULL,
    carrier        VARCHAR(4) NOT NULL,
    technology     VARCHAR(3) NOT NULL,
    product_class  VARCHAR(64),
    template_type  VARCHAR(32) NOT NULL
                   CHECK (template_type IN ('provisioning', 'batch_config', 'firmware_upgrade')),
    parameters     JSONB NOT NULL DEFAULT '{}',
    priority       INTEGER NOT NULL DEFAULT 0,
    version        INTEGER NOT NULL DEFAULT 1,
    active         BOOLEAN NOT NULL DEFAULT true,
    description    TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ct_carrier_tech ON config_templates (carrier, technology);
CREATE INDEX idx_ct_type ON config_templates (template_type);
CREATE INDEX idx_ct_active ON config_templates (active) WHERE active = true;
CREATE INDEX idx_ct_match ON config_templates (carrier, technology, product_class, template_type)
    WHERE active = true;

CREATE TRIGGER trigger_ct_updated_at
    BEFORE UPDATE ON config_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 2. config_baselines — baseline configurations
-- ============================================================
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

-- ============================================================
-- 3. config_tasks — configuration task tracking
-- ============================================================
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

-- ============================================================
-- 4. config_neighbors — neighbor parameter management
-- ============================================================
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

-- ============================================================
-- 5. provisioning_tasks — auto-provisioning workflow state
-- ============================================================
CREATE TABLE provisioning_tasks (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id      UUID NOT NULL,
    template_id    UUID REFERENCES config_templates(id),
    status         VARCHAR(20) NOT NULL DEFAULT 'discovered'
                   CHECK (status IN ('discovered', 'identifying', 'matching', 'configuring', 'verifying', 'completed', 'failed')),
    current_step   INTEGER NOT NULL DEFAULT 0,
    total_steps    INTEGER NOT NULL DEFAULT 0,
    error_message  TEXT,
    retry_count    INTEGER NOT NULL DEFAULT 0,
    max_retries    INTEGER NOT NULL DEFAULT 3,
    started_at     TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pt_device ON provisioning_tasks (device_id);
CREATE INDEX idx_pt_status ON provisioning_tasks (status);
CREATE INDEX idx_pt_device_status ON provisioning_tasks (device_id, status);
CREATE INDEX idx_pt_created ON provisioning_tasks (created_at DESC);

CREATE TRIGGER trigger_pt_updated_at
    BEFORE UPDATE ON provisioning_tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 6. ftp_configs — FTP server configurations for PM file collection
-- ============================================================
CREATE TABLE ftp_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_name VARCHAR(200) NOT NULL,
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL DEFAULT 21,
    username VARCHAR(100) NOT NULL,
    password_encrypted TEXT,
    protocol VARCHAR(10) NOT NULL DEFAULT 'FTP',
    remote_path VARCHAR(512) NOT NULL DEFAULT '/',
    passive BOOLEAN NOT NULL DEFAULT true,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ftp_configs_enabled ON ftp_configs(enabled);

CREATE TRIGGER trigger_ftp_configs_updated_at
    BEFORE UPDATE ON ftp_configs FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 7. pm_counters — performance management counters (TimescaleDB hypertable)
-- ============================================================
CREATE TABLE IF NOT EXISTS pm_counters (
    time           TIMESTAMPTZ NOT NULL,
    device_id      UUID NOT NULL,
    cell_id        VARCHAR(32) NOT NULL DEFAULT '',
    counter_group  VARCHAR(32) NOT NULL,
    counter_name   VARCHAR(64) NOT NULL,
    counter_value  DOUBLE PRECISION NOT NULL,
    granularity    SMALLINT NOT NULL DEFAULT 15
);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        PERFORM create_hypertable('pm_counters', 'time', chunk_time_interval => INTERVAL '1 day', if_not_exists => TRUE);
        ALTER TABLE pm_counters SET (
            timescaledb.compress,
            timescaledb.compress_segmentby = 'device_id, cell_id, counter_group, counter_name'
        );
        PERFORM add_compression_policy('pm_counters', INTERVAL '7 days', if_not_exists => TRUE);
        PERFORM add_retention_policy('pm_counters', INTERVAL '90 days', if_not_exists => TRUE);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_pm_counters_device_time ON pm_counters (device_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_pm_counters_group_name ON pm_counters (counter_group, counter_name, time DESC);

-- ============================================================
-- 8. pm_files — PM file metadata
-- ============================================================
CREATE TABLE IF NOT EXISTS pm_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL,
    device_sn VARCHAR(128) NOT NULL,
    carrier VARCHAR(16) NOT NULL,
    technology VARCHAR(16) NOT NULL,
    file_name VARCHAR(512) NOT NULL,
    file_size BIGINT DEFAULT 0,
    collect_time TIMESTAMPTZ,
    minio_path VARCHAR(1024) NOT NULL,
    parsed BOOLEAN DEFAULT FALSE,
    parsed_at TIMESTAMPTZ,
    counter_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_pm_files_device ON pm_files(device_id);
CREATE INDEX idx_pm_files_created ON pm_files(created_at DESC);

-- ============================================================
-- 9. pm_tasks — PM extraction/analysis tasks
-- ============================================================
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

-- ============================================================
-- 10. kpi_definitions — KPI formula definitions
-- ============================================================
CREATE TABLE IF NOT EXISTS kpi_definitions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(64) NOT NULL UNIQUE,
    display_name VARCHAR(128) NOT NULL,
    formula      TEXT NOT NULL,
    unit         VARCHAR(16) NOT NULL,
    category     VARCHAR(32) NOT NULL,
    carrier      VARCHAR(4),
    technology   VARCHAR(3),
    counters     JSONB NOT NULL DEFAULT '[]',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- 11. kpi_values — KPI calculated values (TimescaleDB hypertable)
-- ============================================================
CREATE TABLE IF NOT EXISTS kpi_values (
    time       TIMESTAMPTZ NOT NULL,
    device_id  UUID NOT NULL,
    cell_id    VARCHAR(32) NOT NULL DEFAULT '',
    kpi_name   VARCHAR(64) NOT NULL,
    kpi_value  DOUBLE PRECISION NOT NULL,
    carrier    VARCHAR(4) NOT NULL,
    technology VARCHAR(3) NOT NULL
);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        PERFORM create_hypertable('kpi_values', 'time', chunk_time_interval => INTERVAL '1 day', if_not_exists => TRUE);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_kpi_values_device_time ON kpi_values (device_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_kpi_values_name ON kpi_values (kpi_name, time DESC);

-- ============================================================
-- 12. kpi_thresholds — configurable KPI threshold management
-- ============================================================
CREATE TABLE IF NOT EXISTS kpi_thresholds (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kpi_name            VARCHAR(128) NOT NULL,
    carrier             VARCHAR(16),
    technology          VARCHAR(16),
    warning_threshold   DOUBLE PRECISION,
    minor_threshold     DOUBLE PRECISION,
    major_threshold     DOUBLE PRECISION,
    critical_threshold  DOUBLE PRECISION,
    comparison          VARCHAR(16) NOT NULL DEFAULT 'gt',
    enabled             BOOLEAN NOT NULL DEFAULT true,
    description         TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_kpi_thresholds_kpi_name ON kpi_thresholds (kpi_name);
CREATE INDEX IF NOT EXISTS idx_kpi_thresholds_carrier ON kpi_thresholds (carrier);
CREATE INDEX IF NOT EXISTS idx_kpi_thresholds_enabled ON kpi_thresholds (enabled);

CREATE TRIGGER trigger_kpi_thresholds_updated_at
    BEFORE UPDATE ON kpi_thresholds
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 13. Continuous aggregate: hourly PM rollup (TimescaleDB only)
-- ============================================================
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        EXECUTE '
            CREATE MATERIALIZED VIEW IF NOT EXISTS pm_counters_hourly
            WITH (timescaledb.continuous) AS
            SELECT time_bucket(''1 hour'', time) AS bucket,
                   device_id, cell_id, counter_group, counter_name,
                   SUM(counter_value) AS sum_value,
                   AVG(counter_value) AS avg_value,
                   MIN(counter_value) AS min_value,
                   MAX(counter_value) AS max_value,
                   COUNT(*) AS sample_count
            FROM pm_counters
            GROUP BY bucket, device_id, cell_id, counter_group, counter_name
            WITH NO DATA';
    END IF;
END $$;

-- +goose Down
DROP MATERIALIZED VIEW IF EXISTS pm_counters_hourly;
DROP TABLE IF EXISTS kpi_thresholds CASCADE;
DROP TABLE IF EXISTS kpi_values CASCADE;
DROP TABLE IF EXISTS kpi_definitions CASCADE;
DROP TABLE IF EXISTS pm_tasks CASCADE;
DROP TABLE IF EXISTS pm_files CASCADE;
DROP TABLE IF EXISTS pm_counters CASCADE;
DROP TABLE IF EXISTS ftp_configs CASCADE;
DROP TABLE IF EXISTS provisioning_tasks CASCADE;
DROP TABLE IF EXISTS config_neighbors CASCADE;
DROP TABLE IF EXISTS config_tasks CASCADE;
DROP TABLE IF EXISTS config_baselines CASCADE;
DROP TABLE IF EXISTS config_templates CASCADE;
