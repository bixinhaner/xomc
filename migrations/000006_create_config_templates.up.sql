-- Phase 2: Config Templates table
-- Stores provisioning and configuration templates per carrier/technology

CREATE TABLE config_templates (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           VARCHAR(128) NOT NULL,
    carrier        VARCHAR(4) NOT NULL,
    technology     VARCHAR(3) NOT NULL,
    product_class  VARCHAR(64),
    template_type  VARCHAR(32) NOT NULL
                   CHECK (template_type IN ('provisioning', 'batch_config', 'firmware_upgrade')),
    parameters     JSONB NOT NULL DEFAULT '{}',
    priority       INTEGER NOT NULL DEFAULT 0,
    version        INTEGER NOT NULL DEFAULT 1,
    active         BOOLEAN NOT NULL DEFAULT true,
    description    TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ct_carrier_tech ON config_templates (carrier, technology);
CREATE INDEX idx_ct_type ON config_templates (template_type);
CREATE INDEX idx_ct_active ON config_templates (active) WHERE active = true;
CREATE INDEX idx_ct_match ON config_templates (carrier, technology, product_class, template_type)
    WHERE active = true;

CREATE TRIGGER trigger_ct_updated_at
    BEFORE UPDATE ON config_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
