-- Phase 2: Data Model Definitions table
-- Stores TR069 parameter tree definitions per carrier/technology/vendor/product combination

CREATE TABLE data_model_definitions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    carrier          VARCHAR(4) NOT NULL,
    technology       VARCHAR(3) NOT NULL,
    version          VARCHAR(16) NOT NULL,
    oui              VARCHAR(6),
    product_class    VARCHAR(64),
    scope            VARCHAR(16) NOT NULL
                     CHECK (scope IN ('product', 'oui', 'carrier_default')),
    status           VARCHAR(12) NOT NULL DEFAULT 'draft'
                     CHECK (status IN ('draft', 'active', 'deprecated')),
    is_active        BOOLEAN NOT NULL DEFAULT false,
    root_object      VARCHAR(64) NOT NULL DEFAULT 'Device.',
    parameter_tree   JSONB NOT NULL,
    source           VARCHAR(32),
    imported_by      VARCHAR(128),
    spec_document_ref VARCHAR(256),
    description      TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_scope_fields CHECK (
        (scope = 'carrier_default' AND oui IS NULL AND product_class IS NULL) OR
        (scope = 'oui' AND oui IS NOT NULL AND product_class IS NULL) OR
        (scope = 'product' AND oui IS NOT NULL AND product_class IS NOT NULL)
    )
);

-- Partial unique indexes: only one active model per classification
CREATE UNIQUE INDEX idx_dm_active_product
    ON data_model_definitions (carrier, technology, oui, product_class)
    WHERE is_active = true AND scope = 'product';

CREATE UNIQUE INDEX idx_dm_active_oui
    ON data_model_definitions (carrier, technology, oui)
    WHERE is_active = true AND scope = 'oui' AND product_class IS NULL;

CREATE UNIQUE INDEX idx_dm_active_carrier_default
    ON data_model_definitions (carrier, technology)
    WHERE is_active = true AND scope = 'carrier_default' AND oui IS NULL;

-- Query indexes
CREATE INDEX idx_dm_carrier_tech ON data_model_definitions (carrier, technology);
CREATE INDEX idx_dm_oui ON data_model_definitions (oui) WHERE oui IS NOT NULL;
CREATE INDEX idx_dm_active_lookup
    ON data_model_definitions (carrier, technology, oui, product_class, scope)
    WHERE is_active = true;
CREATE INDEX idx_dm_status ON data_model_definitions (status);

-- Reuse the update_updated_at_column function from migration 000001
CREATE TRIGGER trigger_dm_updated_at
    BEFORE UPDATE ON data_model_definitions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
