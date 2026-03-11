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
