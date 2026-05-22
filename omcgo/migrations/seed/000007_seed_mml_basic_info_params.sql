-- +goose Up
-- ============================================================
-- 000007_seed_mml_basic_info_params.sql —— 已下线 noop
--
-- 原内容：INSERT INTO mml_command_params_rel 关联 LST/MOD DEVICE_INFO 命令
-- 与 mml_params 14 条参数。migration 000090 DROP mml_command_params_rel，
-- 000154 DROP mml_params 主表 —— 老关联链全部失效。
--
-- 新命令-参数关联走 mml_command_sub_fields（migration 000113 引入），由
-- seed/000152 统一注入。
-- ============================================================

SELECT 1;

-- +goose Down
SELECT 1;
