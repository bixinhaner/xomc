-- +goose Up
-- ============================================================
-- 000006_alarms_mr_firmware.up.sql
-- Alarms (active + history), alarm rules, MR files/records/indicators/device_mappings,
-- firmware versions, upgrade tasks, backup tasks/schedules, managed files, licenses
-- ============================================================

-- ============================================================
-- 1. alarms_active — current active alarms
-- ============================================================
CREATE TABLE IF NOT EXISTS alarms_active (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       UUID NOT NULL,
    device_sn       VARCHAR(64) NOT NULL,
    carrier         VARCHAR(4) NOT NULL,
    severity        SMALLINT NOT NULL,
    alarm_type      VARCHAR(64) NOT NULL DEFAULT '',
    alarm_code      VARCHAR(32) NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    status          VARCHAR(16) NOT NULL DEFAULT 'active',
    raised_at       TIMESTAMPTZ NOT NULL,
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by VARCHAR(128),
    additional_info JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alarms_active_device ON alarms_active (device_id);
CREATE INDEX IF NOT EXISTS idx_alarms_active_device_sn_code ON alarms_active (device_sn, alarm_code);
CREATE INDEX IF NOT EXISTS idx_alarms_active_severity ON alarms_active (severity);
CREATE INDEX IF NOT EXISTS idx_alarms_active_status ON alarms_active (status);

-- ============================================================
-- 2. alarms_history — historical alarms (TimescaleDB hypertable)
-- ============================================================
CREATE TABLE IF NOT EXISTS alarms_history (
    time            TIMESTAMPTZ NOT NULL,
    alarm_id        UUID NOT NULL,
    device_id       UUID NOT NULL,
    device_sn       VARCHAR(64) NOT NULL,
    carrier         VARCHAR(4) NOT NULL,
    severity        SMALLINT NOT NULL,
    alarm_type      VARCHAR(64),
    alarm_code      VARCHAR(32) NOT NULL,
    description     TEXT,
    status          VARCHAR(16) NOT NULL,
    raised_at       TIMESTAMPTZ NOT NULL,
    acknowledged_at TIMESTAMPTZ,
    cleared_at      TIMESTAMPTZ
);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        PERFORM create_hypertable('alarms_history', 'time', chunk_time_interval => INTERVAL '7 days', if_not_exists => TRUE);
        PERFORM add_retention_policy('alarms_history', INTERVAL '365 days', if_not_exists => TRUE);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_alarms_history_device ON alarms_history (device_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_alarms_history_alarm ON alarms_history (alarm_id, time DESC);

-- ============================================================
-- 3. alarm_rules — configurable alarm rule management
-- ============================================================
CREATE TABLE IF NOT EXISTS alarm_rules (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name             VARCHAR(255) NOT NULL,
    description      TEXT,
    alarm_code       VARCHAR(128),
    severity         INTEGER NOT NULL DEFAULT 4,
    condition_type   VARCHAR(64) NOT NULL,
    condition_config JSONB NOT NULL DEFAULT '{}',
    action_type      VARCHAR(64) NOT NULL,
    action_config    JSONB DEFAULT '{}',
    carrier          VARCHAR(16),
    technology       VARCHAR(16),
    enabled          BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alarm_rules_carrier ON alarm_rules (carrier);
CREATE INDEX IF NOT EXISTS idx_alarm_rules_enabled ON alarm_rules (enabled);
CREATE INDEX IF NOT EXISTS idx_alarm_rules_alarm_code ON alarm_rules (alarm_code);

CREATE TRIGGER trigger_alarm_rules_updated_at
    BEFORE UPDATE ON alarm_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 4. mr_files — measurement report file metadata
-- ============================================================
CREATE TABLE IF NOT EXISTS mr_files (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id    UUID NOT NULL,
    device_sn    VARCHAR(64) NOT NULL,
    carrier      VARCHAR(4) NOT NULL,
    mr_type      VARCHAR(8) NOT NULL,
    file_name    VARCHAR(256) NOT NULL,
    file_size    BIGINT NOT NULL DEFAULT 0,
    collect_time TIMESTAMPTZ NOT NULL,
    minio_path   VARCHAR(512) NOT NULL,
    parsed       BOOLEAN NOT NULL DEFAULT false,
    parsed_at    TIMESTAMPTZ,
    record_count INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mr_files_device ON mr_files (device_id, collect_time DESC);
CREATE INDEX IF NOT EXISTS idx_mr_files_type ON mr_files (mr_type);
CREATE INDEX IF NOT EXISTS idx_mr_files_carrier ON mr_files (carrier);

-- ============================================================
-- 5. mr_records — parsed MR records (TimescaleDB hypertable)
-- ============================================================
CREATE TABLE IF NOT EXISTS mr_records (
    time             TIMESTAMPTZ NOT NULL,
    file_id          UUID NOT NULL,
    device_id        UUID NOT NULL,
    cell_id          VARCHAR(32) NOT NULL DEFAULT '',
    mr_type          VARCHAR(8) NOT NULL,
    measurement_data JSONB NOT NULL DEFAULT '{}'
);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        PERFORM create_hypertable('mr_records', 'time', chunk_time_interval => INTERVAL '1 day', if_not_exists => TRUE);
        ALTER TABLE mr_records SET (
            timescaledb.compress,
            timescaledb.compress_segmentby = 'device_id, file_id, mr_type'
        );
        PERFORM add_compression_policy('mr_records', INTERVAL '7 days', if_not_exists => TRUE);
        PERFORM add_retention_policy('mr_records', INTERVAL '90 days', if_not_exists => TRUE);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_mr_records_device ON mr_records (device_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_mr_records_file ON mr_records (file_id);
CREATE INDEX IF NOT EXISTS idx_mr_records_type ON mr_records (mr_type, time DESC);

-- ============================================================
-- 6. mr_indicators — MR indicator definitions
-- ============================================================
CREATE TABLE mr_indicators (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    indicator_name VARCHAR(200) NOT NULL,
    indicator_code VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    unit VARCHAR(50),
    category VARCHAR(50),
    value_range_min DOUBLE PRECISION,
    value_range_max DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mr_indicators_code ON mr_indicators(indicator_code);
CREATE INDEX idx_mr_indicators_category ON mr_indicators(category);

-- ============================================================
-- 7. mr_device_mappings — device-to-cell MR sampling mappings
-- ============================================================
CREATE TABLE mr_device_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_sn VARCHAR(64) NOT NULL,
    device_name VARCHAR(200),
    cell_id VARCHAR(64) NOT NULL,
    cell_name VARCHAR(200),
    enabled BOOLEAN NOT NULL DEFAULT true,
    sampling_interval INTEGER NOT NULL DEFAULT 15,
    last_collect_time TIMESTAMPTZ,
    total_records BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mr_mappings_device ON mr_device_mappings(device_sn);
CREATE INDEX idx_mr_mappings_enabled ON mr_device_mappings(enabled);

CREATE TRIGGER trigger_mr_device_mappings_updated_at
    BEFORE UPDATE ON mr_device_mappings FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 8. firmware_versions — firmware version registry
-- ============================================================
CREATE TABLE firmware_versions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    carrier        VARCHAR(4) NOT NULL,
    product_class  VARCHAR(64),
    version        VARCHAR(64) NOT NULL,
    file_name      VARCHAR(256) NOT NULL,
    file_size      BIGINT,
    minio_path     VARCHAR(512) NOT NULL,
    compatible_oui JSONB DEFAULT '[]'::JSONB,
    release_notes  TEXT,
    status         VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_firmware_carrier ON firmware_versions (carrier);
CREATE INDEX idx_firmware_carrier_product ON firmware_versions (carrier, product_class);
CREATE UNIQUE INDEX idx_firmware_unique_version ON firmware_versions (carrier, product_class, version);

CREATE TRIGGER trigger_firmware_versions_updated_at
    BEFORE UPDATE ON firmware_versions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 9. upgrade_tasks — firmware upgrade task tracking
-- ============================================================
CREATE TABLE upgrade_tasks (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id     UUID NOT NULL,
    firmware_id   UUID NOT NULL REFERENCES firmware_versions(id),
    batch_id      UUID,
    status        VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    retry_count   INTEGER NOT NULL DEFAULT 0,
    max_retries   INTEGER NOT NULL DEFAULT 3,
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_upgrade_tasks_device ON upgrade_tasks (device_id);
CREATE INDEX idx_upgrade_tasks_firmware ON upgrade_tasks (firmware_id);
CREATE INDEX idx_upgrade_tasks_batch ON upgrade_tasks (batch_id) WHERE batch_id IS NOT NULL;
CREATE INDEX idx_upgrade_tasks_status ON upgrade_tasks (status);

CREATE TRIGGER trigger_upgrade_tasks_updated_at
    BEFORE UPDATE ON upgrade_tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 10. backup_tasks — backup task execution tracking
-- ============================================================
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

-- ============================================================
-- 11. backup_schedules — scheduled backup configurations
-- ============================================================
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

-- ============================================================
-- 12. managed_files — general file management registry
-- ============================================================
CREATE TABLE managed_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_name VARCHAR(500) NOT NULL,
    file_type VARCHAR(20) NOT NULL DEFAULT 'other',
    file_size BIGINT NOT NULL DEFAULT 0,
    minio_path TEXT NOT NULL,
    content_type VARCHAR(200) DEFAULT 'application/octet-stream',
    uploader VARCHAR(100),
    device_sn VARCHAR(64),
    status VARCHAR(20) NOT NULL DEFAULT 'uploaded',
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_managed_files_type ON managed_files(file_type);
CREATE INDEX idx_managed_files_status ON managed_files(status);
CREATE INDEX idx_managed_files_device ON managed_files(device_sn);
CREATE INDEX idx_managed_files_created ON managed_files(created_at DESC);

CREATE TRIGGER trigger_managed_files_updated_at
    BEFORE UPDATE ON managed_files
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 13. licenses — license management
-- ============================================================
CREATE TABLE licenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_name VARCHAR(200) NOT NULL,
    license_code VARCHAR(100) NOT NULL UNIQUE,
    product_name VARCHAR(200) NOT NULL,
    license_type VARCHAR(20) NOT NULL DEFAULT 'subscription',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    max_devices INTEGER NOT NULL DEFAULT 0,
    used_devices INTEGER NOT NULL DEFAULT 0,
    features JSONB NOT NULL DEFAULT '[]',
    issue_date TIMESTAMPTZ NOT NULL,
    expiry_date TIMESTAMPTZ,
    licensor VARCHAR(200),
    device_type VARCHAR(50),
    region VARCHAR(100),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_licenses_status ON licenses(status);
CREATE INDEX idx_licenses_code ON licenses(license_code);
CREATE INDEX idx_licenses_type ON licenses(license_type);
CREATE INDEX idx_licenses_device_type ON licenses(device_type);
CREATE INDEX idx_licenses_expiry ON licenses(expiry_date);

CREATE TRIGGER trigger_licenses_updated_at
    BEFORE UPDATE ON licenses FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- Seed data: MR indicators
-- ============================================================
INSERT INTO mr_indicators (indicator_name, indicator_code, description, unit, category, value_range_min, value_range_max) VALUES
    ('RSRP', 'RSRP', '参考信号接收功率', 'dBm', 'coverage', -140, -44),
    ('RSRQ', 'RSRQ', '参考信号接收质量', 'dB', 'coverage', -20, -3),
    ('SINR', 'SINR', '信干噪比', 'dB', 'quality', -10, 30),
    ('TA', 'TA', '时间提前量', '', 'timing', 0, 1282),
    ('PHR', 'PHR', '功率余量', 'dB', 'power', -23, 40);

-- +goose Down
DROP TABLE IF EXISTS licenses CASCADE;
DROP TABLE IF EXISTS managed_files CASCADE;
DROP TABLE IF EXISTS backup_schedules CASCADE;
DROP TABLE IF EXISTS backup_tasks CASCADE;
DROP TABLE IF EXISTS upgrade_tasks CASCADE;
DROP TABLE IF EXISTS firmware_versions CASCADE;
DROP TABLE IF EXISTS mr_device_mappings CASCADE;
DROP TABLE IF EXISTS mr_indicators CASCADE;
DROP TABLE IF EXISTS mr_records CASCADE;
DROP TABLE IF EXISTS mr_files CASCADE;
DROP TABLE IF EXISTS alarm_rules CASCADE;
DROP TABLE IF EXISTS alarms_history CASCADE;
DROP TABLE IF EXISTS alarms_active CASCADE;
