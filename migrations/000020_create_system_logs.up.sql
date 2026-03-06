-- System logs table for application-level logging
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

-- NE (Network Element) message logs table for device communication logging
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
