-- +goose Up
-- ============================================================
-- 000111_mml_old_catalog_import.sql —— 已下线 noop（原 26K+ 行）
--
-- 原内容：老 OMC MML catalog 合并导入 v3（21 个 param_version 切片合并为
-- STANDARD），写 mml_params + mml_param_groups + mml_commands +
-- mml_command_sub_fields。
--
-- 下线原因：
--   1. migration 000152 RENAME mml_param_groups → mml_command_groups
--   2. migration 000154 DROP mml_params
--   3. 新版统一 catalog 由 seed/000152_cmcc_tdlte_v23_mml_commands.sql
--      （18 章节 / 190 命令 / 1054 sub_fields）取代
-- ============================================================

SELECT 1;

-- +goose Down
SELECT 1;
