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
