-- +goose Up
-- ============================================================
-- 000004_data_models.up.sql
-- 数据模型与厂商注册
-- ============================================================

CREATE TABLE data_model_definitions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    carrier          VARCHAR(4) NOT NULL,
    technology       VARCHAR(3) NOT NULL,
    version          VARCHAR(16) NOT NULL,
    oui              VARCHAR(6),
    product_class    VARCHAR(64),
    firmware_version TEXT,
    scope            VARCHAR(16) NOT NULL CHECK (scope IN ('product', 'oui', 'carrier_default')),
    status           VARCHAR(12) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'deprecated')),
    is_active        BOOLEAN NOT NULL DEFAULT false,
    root_object      VARCHAR(64) NOT NULL DEFAULT 'Device.',
    parameter_tree   JSONB NOT NULL,
    source           VARCHAR(128),
    source_type      TEXT NOT NULL DEFAULT 'manual' CHECK (source_type IN ('manual', 'auto_discovered', 'cpe_uploaded')),
    imported_by      VARCHAR(128),
    spec_document_ref VARCHAR(256),
    description      TEXT,
    object_tree      JSONB,
    model_metadata   JSONB,
    last_accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_scope_fields CHECK (
        (scope = 'carrier_default' AND oui IS NULL AND product_class IS NULL) OR
        (scope = 'oui' AND oui IS NOT NULL AND product_class IS NULL) OR
        (scope = 'product' AND oui IS NOT NULL AND product_class IS NOT NULL)
    )
);
CREATE UNIQUE INDEX idx_dm_active_product_versioned ON data_model_definitions (carrier, technology, oui, product_class, firmware_version) WHERE is_active = true AND scope = 'product' AND firmware_version IS NOT NULL AND firmware_version != '';
CREATE UNIQUE INDEX idx_dm_active_product_unversioned ON data_model_definitions (carrier, technology, oui, product_class, source_type) WHERE is_active = true AND scope = 'product' AND (firmware_version IS NULL OR firmware_version = '');
CREATE UNIQUE INDEX idx_dm_active_oui ON data_model_definitions (carrier, technology, oui) WHERE is_active = true AND scope = 'oui' AND product_class IS NULL;
CREATE UNIQUE INDEX idx_dm_active_carrier_default ON data_model_definitions (carrier, technology) WHERE is_active = true AND scope = 'carrier_default' AND oui IS NULL;
CREATE INDEX idx_dm_carrier_tech ON data_model_definitions (carrier, technology);
CREATE INDEX idx_dm_oui ON data_model_definitions (oui) WHERE oui IS NOT NULL;
CREATE INDEX idx_dm_active_lookup ON data_model_definitions (carrier, technology, oui, product_class, scope) WHERE is_active = true;
CREATE INDEX idx_dm_status ON data_model_definitions (status);
CREATE INDEX idx_dm_firmware_version ON data_model_definitions (firmware_version) WHERE firmware_version IS NOT NULL;
CREATE INDEX idx_dm_expiry_check ON data_model_definitions (source_type, last_accessed_at) WHERE is_active = true;
CREATE TRIGGER trigger_dm_updated_at BEFORE UPDATE ON data_model_definitions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE oui_registry (
    oui              VARCHAR(6) PRIMARY KEY,
    manufacturer     VARCHAR(128) NOT NULL,
    short_name       VARCHAR(32) NOT NULL,
    country          VARCHAR(64),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_oui_manufacturer ON oui_registry (short_name);

CREATE TABLE data_model_import_log (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    data_model_id    UUID NOT NULL REFERENCES data_model_definitions(id) ON DELETE CASCADE,
    action           VARCHAR(16) NOT NULL CHECK (action IN ('created', 'updated', 'activated', 'deprecated', 'deleted')),
    performed_by     VARCHAR(128) NOT NULL,
    changes_summary  JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_dm_import_log_model ON data_model_import_log (data_model_id, created_at DESC);
CREATE INDEX idx_dm_import_log_action ON data_model_import_log (action);

CREATE TABLE parameter_discovery_log (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id        UUID NOT NULL,
    device_sn        TEXT NOT NULL,
    oui              TEXT NOT NULL,
    product_class    TEXT,
    firmware_version TEXT,
    parameter_count  INT NOT NULL DEFAULT 0,
    data_model_id    UUID REFERENCES data_model_definitions(id),
    status           TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'discovering', 'syncing', 'completed', 'failed')),
    error_message    TEXT,
    started_at       TIMESTAMPTZ,
    completed_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_pdl_device_id ON parameter_discovery_log (device_id);
CREATE INDEX idx_pdl_status ON parameter_discovery_log (status);
CREATE INDEX idx_pdl_device_sn ON parameter_discovery_log (device_sn);

-- +goose Down
DROP TABLE IF EXISTS parameter_discovery_log CASCADE;
DROP TABLE IF EXISTS data_model_import_log CASCADE;
DROP TABLE IF EXISTS oui_registry CASCADE;
DROP TABLE IF EXISTS data_model_definitions CASCADE;
