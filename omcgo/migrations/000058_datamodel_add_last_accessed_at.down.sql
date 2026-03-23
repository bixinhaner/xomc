-- 000058 down: 回滚 last_accessed_at 和唯一约束变更

DROP INDEX IF EXISTS idx_dm_active_product_unversioned;
DROP INDEX IF EXISTS idx_dm_active_product_versioned;
DROP INDEX IF EXISTS idx_dm_expiry_check;

-- 恢复旧的唯一约束
CREATE UNIQUE INDEX idx_dm_active_product
    ON data_model_definitions (carrier, technology, oui, product_class)
    WHERE is_active = true AND scope = 'product';

ALTER TABLE data_model_definitions DROP COLUMN IF EXISTS last_accessed_at;
