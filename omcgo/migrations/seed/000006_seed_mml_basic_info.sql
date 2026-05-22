-- +goose Up
-- ============================================================
-- 000006_seed_mml_basic_info.sql —— 已下线 noop（原 LST/MOD DEVICE_INFO 注入）
--
-- 原内容：往 mml_commands 注入"基本信息分类 + LST DEVICE_INFO + MOD DEVICE_INFO"，
-- 引用 param_template / param_paths / supported_operations / product_types 等
-- 列；migration 000090 已 DROP 这些列，INSERT 必失败。
--
-- 新 MML 命令树由 seed/000152_cmcc_tdlte_v23_mml_commands.sql 统一管理
-- （18 章节 / 190 命令，包含 LST/MOD DeviceInfo 系列），无需本 seed。
-- 保留版本号占位防止 goose 跳号。
-- ============================================================

SELECT 1;

-- +goose Down
SELECT 1;
