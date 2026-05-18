-- +goose Up
-- 补回 mml_scripts 的单次执行字段：start_time / end_time / type / progress / result。
--
-- 背景（修复代码↔schema 不一致）：
--   · 000026_mml_table_restructure 曾给 mml_scripts 新增这些列；
--   · 000090_mml_schema_rebuild 将其 DROP（设计上拟把单次执行字段移到 mml_tasks）；
--   · 但其后「脚本生命周期 API」(commit a3cc18fb) 的 internal/mml/pg_repository.go
--     仍按这些列 SELECT / UPDATE mml_scripts（scriptColumns / UpdateLifecycle），
--     且 90 之后再无迁移把列加回 —— 导致 list mml_scripts 报
--     `ERROR: column "start_time" does not exist (SQLSTATE 42703)`，app 启动失败。
--
-- 本迁移按 000026 的原始列定义把 5 列加回，使 mml_scripts 与现行 mml 代码一致。
-- 列定义与 000026 完全一致；ADD COLUMN IF NOT EXISTS 保证幂等、可重复执行。
ALTER TABLE mml_scripts ADD COLUMN IF NOT EXISTS start_time TIMESTAMPTZ;
ALTER TABLE mml_scripts ADD COLUMN IF NOT EXISTS end_time   TIMESTAMPTZ;
ALTER TABLE mml_scripts ADD COLUMN IF NOT EXISTS type       VARCHAR(20)  NOT NULL DEFAULT 'manual';
ALTER TABLE mml_scripts ADD COLUMN IF NOT EXISTS progress   NUMERIC(5,2) NOT NULL DEFAULT 0;
ALTER TABLE mml_scripts ADD COLUMN IF NOT EXISTS result     JSONB        DEFAULT '{}';
CREATE INDEX IF NOT EXISTS idx_mml_scripts_type ON mml_scripts(type);

-- +goose Down
DROP INDEX IF EXISTS idx_mml_scripts_type;
ALTER TABLE mml_scripts DROP COLUMN IF EXISTS result;
ALTER TABLE mml_scripts DROP COLUMN IF EXISTS progress;
ALTER TABLE mml_scripts DROP COLUMN IF EXISTS type;
ALTER TABLE mml_scripts DROP COLUMN IF EXISTS end_time;
ALTER TABLE mml_scripts DROP COLUMN IF EXISTS start_time;
