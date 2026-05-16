-- +goose Up
-- ============================================================
-- 000112_mml_commands_updated_at.sql
-- 补齐 mml_commands.updated_at 列与触发器
--
-- 背景：000095 引入 refresh_mml_command_target_paths() 触发器函数，函数体
-- 执行 ``UPDATE mml_commands SET ..., updated_at = NOW() WHERE id = ...``，
-- 但 mml_commands 自 000007 起从未有 updated_at 列。在 sub_fields 表为空
-- 之前函数不会被触发；seed/000111 一旦导入老 OMC catalog（产生 ~26k 条
-- sub_fields），触发器立即报 ``column updated_at of relation mml_commands
-- does not exist`` (SQLSTATE 42703)。
--
-- 本次补齐 updated_at 列并挂 update_updated_at_column 触发器，符合项目
-- §5.5 "所有表含 created_at/updated_at" 约定。000110（platform_tags）已
-- 部署到 dev 环境，故 updated_at 单独走 000112，不重做 000110。
-- ============================================================

ALTER TABLE mml_commands
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

COMMENT ON COLUMN mml_commands.updated_at IS
    '最后修改时间，由 update_updated_at_column 触发器自动维护；'
    '也由 refresh_mml_command_target_paths()（000095）在 sub_fields 变更时回填。';

-- 自动维护 updated_at（复用 000001 共享函数）
DROP TRIGGER IF EXISTS trg_mml_commands_updated_at ON mml_commands;
CREATE TRIGGER trg_mml_commands_updated_at
    BEFORE UPDATE ON mml_commands
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();


-- +goose Down
DROP TRIGGER IF EXISTS trg_mml_commands_updated_at ON mml_commands;
ALTER TABLE mml_commands DROP COLUMN IF EXISTS updated_at;
