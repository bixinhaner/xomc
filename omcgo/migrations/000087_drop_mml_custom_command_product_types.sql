-- +goose Up
-- T-0090-b: 删除 mml_custom_command.product_types 列 + GIN 索引（破坏性变更）。
--
-- 背景：T-0090-a 已从 UI 删除"适用产品类型"字段；T-0098 ProductRegistry
--   接管产品路由后 product_types 已成 dead column。本迁移彻底下线该列。
--
-- 范围：仅动 mml_custom_command 表。同名 product_types 列在 mml_commands
--   （内建命令树）和 indicator_definitions（KPI 指标表）保持不变。
--
-- 数据：列内现有数据将永久丢失。Down 段仅恢复 schema 默认空数组，
--   不恢复历史数据（per backlog/subtasks/T-0090-mml-ux-rework.md R-NEW-1）。

DROP INDEX IF EXISTS idx_mml_custom_command_product_types_gin;
ALTER TABLE mml_custom_command DROP COLUMN IF EXISTS product_types;

-- +goose Down
-- 警告：Down 仅恢复 schema 至 Up 之前的结构，历史数据无法回填。
ALTER TABLE mml_custom_command ADD COLUMN IF NOT EXISTS product_types JSONB NOT NULL DEFAULT '[]'::jsonb;
CREATE INDEX IF NOT EXISTS idx_mml_custom_command_product_types_gin ON mml_custom_command USING GIN (product_types);
