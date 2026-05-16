-- +goose Up
-- ============================================================
-- 000113_mml_sub_fields_use_standard_params.sql
-- mml_command_sub_fields 改挂 standard_params（T-0098 系统级标准 path 字典）
--
-- 用户决策（2026-05-16）：
--   - mml 模块不再持有独立 path 字典（mml_params 字典层不再被 sub_fields 引用，
--     表本身暂留供 admin UI 兼容；后续清理在另一 PR）
--   - 标准 path 唯一来源 = standard_params（由 product/param-model 前端管理）
--   - 暂不考虑私有 path 翻译；ACS 命令渲染直接用 standard_path
--
-- 改动总览：
--   1. 清空 mml_command_sub_fields 旧数据（FK 切换前置条件）
--   2. DROP 老 FK + DROP COLUMN param_id（脱离 mml_params 引用）
--   3. ADD COLUMN standard_path_id UUID NOT NULL REFERENCES standard_params(id)
--      ON DELETE RESTRICT（防止字典误删导致命令失效）
--   4. ADD UNIQUE (command_id, standard_path_id) — 替代老 (command_id, param_id)
--   5. 重写 refresh_mml_command_target_paths 函数 JOIN standard_params
--      .standard_path 取值（原 JOIN mml_params.tr069_path）
-- ============================================================

-- 1. 清空旧 sub_fields（FK 切换前置条件，避免 NOT NULL ADD COLUMN 失败）
--    也顺带清空 mml_param_groups / mml_commands 的 source='standard' 数据，
--    因为它们是基于老 mml_params FK 设计的，seed/000111 重新生成时一并重建。
TRUNCATE mml_command_sub_fields;
DELETE FROM mml_commands WHERE source = 'standard';
DELETE FROM mml_param_groups WHERE source = 'standard';

-- 2. DROP 老的 FK 与 UNIQUE 约束
ALTER TABLE mml_command_sub_fields
    DROP CONSTRAINT IF EXISTS mml_command_sub_fields_param_id_fkey,
    DROP CONSTRAINT IF EXISTS uq_command_param;

DROP INDEX IF EXISTS idx_mml_command_sub_fields_param;

-- 3. DROP param_id 列
ALTER TABLE mml_command_sub_fields DROP COLUMN IF EXISTS param_id;

-- 4. ADD 新 standard_path_id 列 + FK + UNIQUE
ALTER TABLE mml_command_sub_fields
    ADD COLUMN IF NOT EXISTS standard_path_id UUID NOT NULL
        REFERENCES standard_params(id) ON DELETE RESTRICT;

ALTER TABLE mml_command_sub_fields
    ADD CONSTRAINT uq_command_standard_path UNIQUE (command_id, standard_path_id);

CREATE INDEX IF NOT EXISTS idx_mml_command_sub_fields_std_path
    ON mml_command_sub_fields(standard_path_id);

COMMENT ON COLUMN mml_command_sub_fields.standard_path_id IS
    'FK → standard_params(id)。MML 命令的子字段直接引用系统级标准 path 字典，'
    '由 product/param-model 前端统一管理。运行时 ACS 通过 ParamModel Translator '
    '把 standard_path 翻译为厂商私有 path 下发（暂未启用，当前直接用 standard_path）。';

-- 5. 重写 target_paths 派生函数：JOIN standard_params 取 standard_path
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION refresh_mml_command_target_paths(p_command_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE mml_commands c
    SET target_paths = COALESCE((
            SELECT jsonb_agg(sp.standard_path ORDER BY csf.sort_order)
            FROM mml_command_sub_fields csf
            JOIN standard_params sp ON sp.id = csf.standard_path_id
            WHERE csf.command_id = p_command_id
        ), '[]'::jsonb),
        updated_at = NOW()
    WHERE c.id = p_command_id;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd


-- +goose Down
-- ============================================================
-- 反向：恢复 param_id FK 引用 mml_params。
-- 注意 seed/000111 在 Down 顺序里也要回退（DELETE source='standard' 行）。
-- ============================================================

TRUNCATE mml_command_sub_fields;
DELETE FROM mml_commands WHERE source = 'standard';
DELETE FROM mml_param_groups WHERE source = 'standard';

-- DROP 新约束 + 列
ALTER TABLE mml_command_sub_fields
    DROP CONSTRAINT IF EXISTS uq_command_standard_path;
DROP INDEX IF EXISTS idx_mml_command_sub_fields_std_path;
ALTER TABLE mml_command_sub_fields DROP COLUMN IF EXISTS standard_path_id;

-- 恢复 param_id 列 + FK + UNIQUE
ALTER TABLE mml_command_sub_fields
    ADD COLUMN IF NOT EXISTS param_id UUID NOT NULL
        REFERENCES mml_params(id) ON DELETE RESTRICT;
ALTER TABLE mml_command_sub_fields
    ADD CONSTRAINT uq_command_param UNIQUE (command_id, param_id);
CREATE INDEX IF NOT EXISTS idx_mml_command_sub_fields_param
    ON mml_command_sub_fields(param_id);

-- 恢复 target_paths 派生函数（JOIN mml_params）
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION refresh_mml_command_target_paths(p_command_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE mml_commands c
    SET target_paths = COALESCE((
            SELECT jsonb_agg(p.tr069_path ORDER BY csf.sort_order)
            FROM mml_command_sub_fields csf
            JOIN mml_params p ON p.id = csf.param_id
            WHERE csf.command_id = p_command_id
        ), '[]'::jsonb),
        updated_at = NOW()
    WHERE c.id = p_command_id;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
