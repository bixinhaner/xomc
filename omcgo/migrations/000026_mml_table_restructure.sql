-- +goose Up
-- ============================================================
-- 000026_mml_table_restructure.sql
-- MML 表结构重组：
--   1. mml_scripts 新增 status/start_time/end_time/type/progress/result
--   2. mml_templates → mml_custom_command（用户自定义命令）
--   3. mml_sub_commands + mml_command_subcommand_rel → mml_command_params_rel
--      （命令直接关联 mml_params，消除冗余子命令表）
-- ============================================================

-- ============================================================
-- 1. mml_scripts: 新增执行状态相关字段
-- ============================================================
ALTER TABLE mml_scripts ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active';
ALTER TABLE mml_scripts ADD COLUMN IF NOT EXISTS start_time TIMESTAMPTZ;
ALTER TABLE mml_scripts ADD COLUMN IF NOT EXISTS end_time TIMESTAMPTZ;
ALTER TABLE mml_scripts ADD COLUMN IF NOT EXISTS type VARCHAR(20) NOT NULL DEFAULT 'manual';
ALTER TABLE mml_scripts ADD COLUMN IF NOT EXISTS progress NUMERIC(5,2) NOT NULL DEFAULT 0;
ALTER TABLE mml_scripts ADD COLUMN IF NOT EXISTS result JSONB DEFAULT '{}';

-- 新增脚本状态、类型排序索引
CREATE INDEX IF NOT EXISTS idx_mml_scripts_status ON mml_scripts(status);
CREATE INDEX IF NOT EXISTS idx_mml_scripts_type ON mml_scripts(type);

-- ============================================================
-- 2. mml_templates → mml_custom_command
-- ============================================================
ALTER TABLE mml_templates RENAME TO mml_custom_command;

-- 列重命名
ALTER TABLE mml_custom_command RENAME COLUMN template_name TO command_name;
ALTER TABLE mml_custom_command RENAME COLUMN template_scope TO command_scope;

-- 约束重命名（先删旧再建新）
ALTER TABLE mml_custom_command DROP CONSTRAINT IF EXISTS chk_mml_templates_scope;
ALTER TABLE mml_custom_command DROP CONSTRAINT IF EXISTS chk_mml_templates_op;
ALTER TABLE mml_custom_command ADD CONSTRAINT chk_mml_custom_command_scope
    CHECK (command_scope IN ('private', 'public'));
ALTER TABLE mml_custom_command ADD CONSTRAINT chk_mml_custom_command_op
    CHECK (operation_type IN ('LST', 'MOD', 'ADD', 'RMV'));

-- 索引重命名
ALTER INDEX IF EXISTS idx_mml_templates_command_code RENAME TO idx_mml_custom_command_command_code;
ALTER INDEX IF EXISTS idx_mml_templates_scope_creator RENAME TO idx_mml_custom_command_scope_creator;
ALTER INDEX IF EXISTS idx_mml_templates_product_types_gin RENAME TO idx_mml_custom_command_product_types_gin;
ALTER INDEX IF EXISTS idx_mml_templates_parameters_gin RENAME TO idx_mml_custom_command_parameters_gin;

-- 触发器重建
DROP TRIGGER IF EXISTS trigger_mml_templates_updated_at ON mml_custom_command;
CREATE TRIGGER trigger_mml_custom_command_updated_at
    BEFORE UPDATE ON mml_custom_command
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 3. mml_sub_commands → mml_command_params_rel
--    命令直接关联 mml_params，消除 mml_sub_commands 冗余表
-- ============================================================

-- 步骤 3a: 创建临时映射表（mml_sub_commands → mml_params，通过 tr069_path 匹配）
-- +goose StatementBegin
DO $$
BEGIN
    CREATE TEMP TABLE tmp_subcmd_param_map AS
    SELECT DISTINCT ON (sc.tr069_path)
        sc.id AS subcommand_id,
        p.id AS param_id
    FROM mml_sub_commands sc
    JOIN mml_params p ON p.tr069_path = sc.tr069_path
    ORDER BY sc.tr069_path, p.created_at;
EXCEPTION WHEN undefined_table THEN
    -- mml_sub_commands 或 mml_params 不存在，创建空映射
    CREATE TEMP TABLE tmp_subcmd_param_map (subcommand_id UUID, param_id UUID);
END $$;
-- +goose StatementEnd

-- 步骤 3b: 创建新的命令-参数关联表
CREATE TABLE IF NOT EXISTS mml_command_params_rel (
    command_id UUID NOT NULL REFERENCES mml_commands(id) ON DELETE CASCADE,
    param_id   UUID NOT NULL REFERENCES mml_params(id) ON DELETE CASCADE,
    sort_order INT NOT NULL DEFAULT 0,
    PRIMARY KEY (command_id, param_id)
);

CREATE INDEX IF NOT EXISTS idx_mml_cmd_param_command ON mml_command_params_rel(command_id);
CREATE INDEX IF NOT EXISTS idx_mml_cmd_param_param ON mml_command_params_rel(param_id);

-- 步骤 3c: 迁移旧关联数据（在删除旧表之前执行）
-- +goose StatementBegin
DO $$
BEGIN
    INSERT INTO mml_command_params_rel (command_id, param_id, sort_order)
    SELECT DISTINCT r.command_id, m.param_id, r.sort_order
    FROM tmp_subcmd_param_map m
    JOIN mml_command_subcommand_rel r ON r.subcommand_id = m.subcommand_id
    ON CONFLICT DO NOTHING;
EXCEPTION WHEN undefined_table THEN
    NULL;
END $$;
-- +goose StatementEnd

-- 步骤 3d: 删除旧关联表和子命令表
DROP TABLE IF EXISTS mml_command_subcommand_rel;
DROP TABLE IF EXISTS mml_sub_commands;

-- 清理临时表
DROP TABLE IF EXISTS tmp_subcmd_param_map;


-- +goose Down
-- ============================================================
-- 回滚：恢复旧表结构
-- ============================================================

-- 3. 恢复 mml_sub_commands + mml_command_subcommand_rel
DROP TABLE IF EXISTS mml_command_params_rel;

CREATE TABLE IF NOT EXISTS mml_sub_commands (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(200) NOT NULL,
    code        VARCHAR(100) NOT NULL,
    tr069_path  VARCHAR(500) NOT NULL,
    description TEXT,
    value_type  VARCHAR(20) NOT NULL DEFAULT 'string',
    is_writable BOOLEAN NOT NULL DEFAULT false,
    options     JSONB NOT NULL DEFAULT '[]',
    unit        VARCHAR(20),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_mml_sub_commands_code ON mml_sub_commands(code);

CREATE TABLE IF NOT EXISTS mml_command_subcommand_rel (
    command_id    UUID NOT NULL REFERENCES mml_commands(id) ON DELETE CASCADE,
    subcommand_id UUID NOT NULL REFERENCES mml_sub_commands(id) ON DELETE CASCADE,
    sort_order    INT NOT NULL DEFAULT 0,
    PRIMARY KEY (command_id, subcommand_id)
);

CREATE INDEX IF NOT EXISTS idx_mml_cmd_subcmd_command ON mml_command_subcommand_rel(command_id);

-- 2. 恢复 mml_templates
DROP TRIGGER IF EXISTS trigger_mml_custom_command_updated_at ON mml_custom_command;

ALTER TABLE mml_custom_command RENAME TO mml_templates;
ALTER TABLE mml_templates RENAME COLUMN command_name TO template_name;
ALTER TABLE mml_templates RENAME COLUMN command_scope TO template_scope;

ALTER TABLE mml_templates DROP CONSTRAINT IF EXISTS chk_mml_custom_command_scope;
ALTER TABLE mml_templates DROP CONSTRAINT IF EXISTS chk_mml_custom_command_op;
ALTER TABLE mml_templates ADD CONSTRAINT chk_mml_templates_scope
    CHECK (template_scope IN ('private', 'public'));
ALTER TABLE mml_templates ADD CONSTRAINT chk_mml_templates_op
    CHECK (operation_type IN ('LST', 'MOD', 'ADD', 'RMV'));

ALTER INDEX IF EXISTS idx_mml_custom_command_command_code RENAME TO idx_mml_templates_command_code;
ALTER INDEX IF EXISTS idx_mml_custom_command_scope_creator RENAME TO idx_mml_templates_scope_creator;
ALTER INDEX IF EXISTS idx_mml_custom_command_product_types_gin RENAME TO idx_mml_templates_product_types_gin;
ALTER INDEX IF EXISTS idx_mml_custom_command_parameters_gin RENAME TO idx_mml_templates_parameters_gin;

CREATE TRIGGER trigger_mml_templates_updated_at
    BEFORE UPDATE ON mml_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 1. 恢复 mml_scripts
DROP INDEX IF EXISTS idx_mml_scripts_type;
DROP INDEX IF EXISTS idx_mml_scripts_status;
ALTER TABLE mml_scripts DROP COLUMN IF EXISTS result;
ALTER TABLE mml_scripts DROP COLUMN IF EXISTS progress;
ALTER TABLE mml_scripts DROP COLUMN IF EXISTS type;
ALTER TABLE mml_scripts DROP COLUMN IF EXISTS end_time;
ALTER TABLE mml_scripts DROP COLUMN IF EXISTS start_time;
ALTER TABLE mml_scripts DROP COLUMN IF EXISTS status;
