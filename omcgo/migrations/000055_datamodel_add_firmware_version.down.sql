-- 000055 down: revert firmware_version + source_type

DROP INDEX IF EXISTS idx_dm_firmware_version;
DROP INDEX IF EXISTS idx_dm_active_product;

-- Restore original unique index without firmware_version
CREATE UNIQUE INDEX idx_dm_active_product
    ON data_model_definitions (carrier, technology, oui, product_class)
    WHERE scope = 'product' AND is_active = true;

ALTER TABLE data_model_definitions DROP CONSTRAINT IF EXISTS chk_dm_source_type;
ALTER TABLE data_model_definitions DROP COLUMN IF EXISTS source_type;
ALTER TABLE data_model_definitions DROP COLUMN IF EXISTS firmware_version;
