-- +goose Up
-- ============================================================
-- 000096_mml_standard_params_import.sql —— 已下线 noop（原 2K+ 行）
--
-- 原内容：由 omcctl mml import-standard-xml 生成的 mml_params 批量 INSERT
-- （param_version='STANDARD'），2001 条。migration 000154 DROP mml_params 后失效。
--
-- 标准参数树现走 standard_params 表（migration 000113 引入），由 seed/000152
-- 等业务 catalog seed 注入；mml_param_versions.STANDARD 行也由 000152 维护。
-- ============================================================

SELECT 1;

-- +goose Down
SELECT 1;
