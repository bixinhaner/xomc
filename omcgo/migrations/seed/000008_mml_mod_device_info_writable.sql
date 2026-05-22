-- +goose Up
-- ============================================================
-- 000008_mml_mod_device_info_writable.sql —— 已下线 noop
--
-- 原内容：UPDATE mml_params SET is_writable=TRUE WHERE tr069_path IN (...)
-- 修正 MOD DEVICE_INFO 的 7 个可写参数。migration 000154 DROP mml_params。
--
-- 新 schema 下，参数可写性由 standard_params.access_type ('RW') 表达，
-- 由 seed/000152 / standard_params seed 统一维护。
-- ============================================================

SELECT 1;

-- +goose Down
SELECT 1;
