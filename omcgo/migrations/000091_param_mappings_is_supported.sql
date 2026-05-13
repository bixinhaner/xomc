-- +goose Up
-- T-0103
-- 给 param_mappings / discovered_param_mappings 增加 is_supported 列。
--
-- 用途：标注"某些 XML 字典里列出的 path，目标固件实际不支持下发 GPV"，
-- path-b sync extractStorablePrefixes 据此过滤，避免触发 9005 整批 reject。
--
-- 写入方：dictloader 启动期从 XML 的 supported="false" 属性读入。
-- 默认 TRUE：旧字典 + 旧条目无须改动即可保持原行为。

ALTER TABLE param_mappings
    ADD COLUMN IF NOT EXISTS is_supported BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE discovered_param_mappings
    ADD COLUMN IF NOT EXISTS is_supported BOOLEAN NOT NULL DEFAULT TRUE;

COMMENT ON COLUMN param_mappings.is_supported IS 'T-0103 XML supported="false" → false；path-b sync 据此过滤，默认 TRUE';
COMMENT ON COLUMN discovered_param_mappings.is_supported IS 'T-0103 从 param_mappings 继承（intersect 时复制），默认 TRUE';

-- +goose Down
ALTER TABLE discovered_param_mappings DROP COLUMN IF EXISTS is_supported;
ALTER TABLE param_mappings           DROP COLUMN IF EXISTS is_supported;
