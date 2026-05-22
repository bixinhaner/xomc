-- +goose Up
-- ============================================================
-- 000005_seed_mml_param_library.sql —— 已下线 noop（原 103K+ 行）
--
-- 原内容：mml_params (7226 条) + mml_param_groups → mml_group_param_rel 旧
-- MML 参数库批量 INSERT，数据来自老 OMC small_cell_param.sql 等。
--
-- 下线原因（fresh release deploy 跑 migrations/seed/ 后暴露）：
--   1. migration 000154 DROP TABLE mml_params + mml_group_param_rel
--   2. migration 000090 重构 mml_commands schema (DROP product_types /
--      param_template / param_paths / supported_operations)
--   3. 新 MML catalog 数据（cmcc-tdlte-v2.3 / 18 章节 / 190 命令 / 1054
--      sub_fields）由 seed/000152_cmcc_tdlte_v23_mml_commands.sql 注入
--
-- 保留版本号 000005 占位让 goose_db_version_seed 连续，避免跳号。
-- ============================================================

SELECT 1;

-- +goose Down
SELECT 1;
