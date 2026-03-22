-- 000055: DataModel 增加 firmware_version 匹配维度 + source_type 来源标记

-- 1. 增加 firmware_version 列
ALTER TABLE data_model_definitions
    ADD COLUMN IF NOT EXISTS firmware_version TEXT;

-- 2. 增加来源标记 (manual: 手动导入, auto_discovered: 自动发现)
ALTER TABLE data_model_definitions
    ADD COLUMN IF NOT EXISTS source_type TEXT NOT NULL DEFAULT 'manual';

ALTER TABLE data_model_definitions
    ADD CONSTRAINT chk_dm_source_type CHECK (source_type IN ('manual', 'auto_discovered'));

-- 3. 更新 product scope 唯一约束（加入 firmware_version）
DROP INDEX IF EXISTS idx_dm_active_product;
CREATE UNIQUE INDEX idx_dm_active_product
    ON data_model_definitions (carrier, technology, oui, product_class, COALESCE(firmware_version, ''))
    WHERE scope = 'product' AND is_active = true;

-- 4. firmware_version 查询索引
CREATE INDEX idx_dm_firmware_version
    ON data_model_definitions (firmware_version)
    WHERE firmware_version IS NOT NULL;
