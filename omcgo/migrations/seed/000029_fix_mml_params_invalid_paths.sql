-- +goose Up
-- ============================================================
-- 000029_fix_mml_params_invalid_paths.sql —— 已下线 noop
--
-- 原内容：UPDATE / DELETE mml_params 修正 tr069_path 不合规（DeviceGSM.* /
-- {i} 未替换 / X_COM_* 非 6 位 OUI）。migration 000154 DROP mml_params。
--
-- 新 schema 走 standard_params，path 在 catalogloader / seed/000152 注入时
-- 已是规范形态，不再需要事后修正。
-- ============================================================

SELECT 1;

-- +goose Down
SELECT 1;
