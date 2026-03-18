-- MR file metadata table
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

-- MR parsed records table
CREATE TABLE IF NOT EXISTS mr_records (
    time             TIMESTAMPTZ NOT NULL,
    file_id          UUID NOT NULL,
    device_id        UUID NOT NULL,
    cell_id          VARCHAR(32) NOT NULL DEFAULT '',
    mr_type          VARCHAR(8) NOT NULL,
    measurement_data JSONB NOT NULL DEFAULT '{}'
);

-- TimescaleDB hypertable (skip if extension not available)
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
