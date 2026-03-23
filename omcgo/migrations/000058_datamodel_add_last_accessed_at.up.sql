-- 000058: 添加 last_accessed_at 字段和调整唯一约束
-- 支持参数模版过期清理（自动模版 15 天，手动模版 60 天）和二级精确匹配

-- 1. 添加 last_accessed_at 字段
ALTER TABLE data_model_definitions
    ADD COLUMN last_accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

COMMENT ON COLUMN data_model_definitions.last_accessed_at IS
    '最后一次被匹配命中的时间，用于过期清理判断';

-- 初始化：将现有记录的 last_accessed_at 设为 updated_at
UPDATE data_model_definitions SET last_accessed_at = updated_at;

-- 2. 过期清理查询索引
CREATE INDEX idx_dm_expiry_check
    ON data_model_definitions (source_type, last_accessed_at)
    WHERE is_active = true;

-- 3. 调整唯一约束以支持同 OUI+ProductClass 下共存自动/手动模版

-- 移除旧的 product scope 唯一约束
DROP INDEX IF EXISTS idx_dm_active_product;

-- 新约束 1: 带 firmware_version 的模版唯一性
-- 同 carrier+tech+oui+product_class+firmware_version 只能有一条 active 模版
CREATE UNIQUE INDEX idx_dm_active_product_versioned
    ON data_model_definitions (carrier, technology, oui, product_class, firmware_version)
    WHERE is_active = true
      AND scope = 'product'
      AND firmware_version IS NOT NULL
      AND firmware_version != '';

-- 新约束 2: 不带 firmware_version 的模版唯一性（按 source_type 区分）
-- 同 carrier+tech+oui+product_class+source_type 只能有一条 active 无版本模版
CREATE UNIQUE INDEX idx_dm_active_product_unversioned
    ON data_model_definitions (carrier, technology, oui, product_class, source_type)
    WHERE is_active = true
      AND scope = 'product'
      AND (firmware_version IS NULL OR firmware_version = '');
