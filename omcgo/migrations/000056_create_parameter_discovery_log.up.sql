-- 000056: 参数发现日志表

CREATE TABLE IF NOT EXISTS parameter_discovery_log (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id        UUID NOT NULL,
    device_sn        TEXT NOT NULL,
    oui              TEXT NOT NULL,
    product_class    TEXT,
    firmware_version TEXT,

    -- 发现结果
    parameter_count  INT NOT NULL DEFAULT 0,
    data_model_id    UUID REFERENCES data_model_definitions(id),

    -- 状态
    status           TEXT NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending', 'discovering', 'syncing', 'completed', 'failed')),
    error_message    TEXT,

    -- 时间
    started_at       TIMESTAMPTZ,
    completed_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pdl_device_id ON parameter_discovery_log (device_id);
CREATE INDEX idx_pdl_status ON parameter_discovery_log (status);
CREATE INDEX idx_pdl_device_sn ON parameter_discovery_log (device_sn);
