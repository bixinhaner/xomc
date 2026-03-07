-- Firmware versions table
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

-- Upgrade tasks table
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
