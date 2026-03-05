-- Phase 2: Device Groups table
-- Supports tree-structured topology management for base stations

CREATE TABLE device_groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(128) NOT NULL,
    parent_id   UUID REFERENCES device_groups(id) ON DELETE CASCADE,
    carrier     VARCHAR(4),
    description TEXT,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dg_parent ON device_groups (parent_id);
CREATE INDEX idx_dg_carrier ON device_groups (carrier) WHERE carrier IS NOT NULL;
CREATE INDEX idx_dg_name ON device_groups (name);

CREATE TRIGGER trigger_dg_updated_at
    BEFORE UPDATE ON device_groups
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
