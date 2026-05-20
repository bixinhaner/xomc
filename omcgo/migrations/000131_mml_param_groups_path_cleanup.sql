-- +goose Up
-- ============================================================
-- 000131 — 清理 mml_param_groups 中 path 为 NULL 的脏数据，并加 NOT NULL 约束
--
-- 背景：
--   GET /api/v1/mml/group-tree 接口报错：
--     "scan row: can't scan into dest[5] (col: path_text):
--      cannot scan NULL into *string"
--   根因：mml_param_groups.path 字段定义为 LTREE 且无 NOT NULL 约束，
--   存在 path IS NULL 的脏数据行，pgx 无法将 NULL 装入非指针 string 字段。
--
-- 处理策略：
--   1. 硬删 path 为 NULL 的行 —— 这类行无法挂到 LTREE 树结构里，对树查询
--      无业务价值。mml_commands.group_id FK 是 ON DELETE SET NULL
--      （见 000090_mml_schema_rebuild.sql），不会级联破坏命令数据。
--   2. 加 NOT NULL 约束，防止未来再次写入 NULL path。
--
-- 注：仓库层 group_tree_repository.go 已加 WHERE g.path IS NOT NULL 兜底，
-- 迁移失败也不影响接口可用性。
-- ============================================================

-- 1) 记录将被清理的脏数据数量（goose 日志可见，便于审计）
-- +goose StatementBegin
DO $$
DECLARE
    null_path_count INT;
BEGIN
    SELECT COUNT(*) INTO null_path_count
      FROM mml_param_groups
     WHERE path IS NULL;

    IF null_path_count > 0 THEN
        RAISE NOTICE '将清理 mml_param_groups 中 % 行 path IS NULL 的脏数据', null_path_count;
    END IF;
END $$;
-- +goose StatementEnd

-- 2) 硬删脏数据（FK ON DELETE SET NULL，关联 mml_commands.group_id 会被置空）
DELETE FROM mml_param_groups
 WHERE path IS NULL;

-- 3) 加 NOT NULL 约束，防止后续写入 NULL path
ALTER TABLE mml_param_groups
    ALTER COLUMN path SET NOT NULL;

-- +goose Down
-- 回退：仅移除 NOT NULL 约束；已硬删的脏数据无法恢复（这些行本就是无效数据）
ALTER TABLE mml_param_groups
    ALTER COLUMN path DROP NOT NULL;
