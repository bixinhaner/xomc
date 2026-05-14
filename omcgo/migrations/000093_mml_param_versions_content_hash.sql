-- +goose Up
-- ============================================================================
-- 000093: mml_param_versions 加 content_hash 列支持 Loader 增量重载
--
-- mmlstandardloader 启动期会扫 data/param-mappings/standard-model.xml 全量
-- UPSERT 1988 params + ~260 groups + ~840 commands（~0.5s 但全量事务）。
-- 多次启动 / reload 端点频繁调用时浪费。
--
-- 方案：算 XML 文件 sha256 → 与本列存储值对比相同则 skip。
-- 字段允许 NULL 兼容历史行；新 Loader 跑过一遍后填充。
-- ============================================================================

ALTER TABLE mml_param_versions
    ADD COLUMN IF NOT EXISTS content_hash VARCHAR(64);

COMMENT ON COLUMN mml_param_versions.content_hash IS
    'Sprint B 增量重载：标准 XML 文件 sha256 hex。Loader 通过比对此值跳过未变更 reload';

-- +goose Down
ALTER TABLE mml_param_versions DROP COLUMN IF EXISTS content_hash;
