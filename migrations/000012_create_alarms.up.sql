-- Active alarms table (regular PostgreSQL table)
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

-- Historical alarms hypertable (TimescaleDB)
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

SELECT create_hypertable('alarms_history', 'time', chunk_time_interval => INTERVAL '7 days', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_alarms_history_device ON alarms_history (device_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_alarms_history_alarm ON alarms_history (alarm_id, time DESC);

SELECT add_retention_policy('alarms_history', INTERVAL '365 days', if_not_exists => TRUE);
