-- PM counters hypertable (TimescaleDB)
CREATE TABLE IF NOT EXISTS pm_counters (
    time           TIMESTAMPTZ NOT NULL,
    device_id      UUID NOT NULL,
    cell_id        VARCHAR(32) NOT NULL DEFAULT '',
    counter_group  VARCHAR(32) NOT NULL,
    counter_name   VARCHAR(64) NOT NULL,
    counter_value  DOUBLE PRECISION NOT NULL,
    granularity    SMALLINT NOT NULL DEFAULT 15
);

SELECT create_hypertable('pm_counters', 'time', chunk_time_interval => INTERVAL '1 day', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_pm_counters_device_time ON pm_counters (device_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_pm_counters_group_name ON pm_counters (counter_group, counter_name, time DESC);

-- Compression policy: compress chunks older than 7 days
SELECT add_compression_policy('pm_counters', INTERVAL '7 days', if_not_exists => TRUE);

-- Retention policy: drop data older than 90 days
SELECT add_retention_policy('pm_counters', INTERVAL '90 days', if_not_exists => TRUE);
