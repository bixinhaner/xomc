-- +goose Up
-- ============================================================
-- 000025_mml_sub_commands.sql
-- 新增子命令表和命令-子命令关联表，支持命令与子命令 N:M 关系
-- ============================================================

-- 子命令定义表
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

-- 命令-子命令关联表（N:M）
CREATE TABLE IF NOT EXISTS mml_command_subcommand_rel (
    command_id    UUID NOT NULL REFERENCES mml_commands(id) ON DELETE CASCADE,
    subcommand_id UUID NOT NULL REFERENCES mml_sub_commands(id) ON DELETE CASCADE,
    sort_order    INT NOT NULL DEFAULT 0,
    PRIMARY KEY (command_id, subcommand_id)
);

CREATE INDEX IF NOT EXISTS idx_mml_cmd_subcmd_command ON mml_command_subcommand_rel(command_id);

-- +goose Down
DROP TABLE IF EXISTS mml_command_subcommand_rel;
DROP TABLE IF EXISTS mml_sub_commands;
