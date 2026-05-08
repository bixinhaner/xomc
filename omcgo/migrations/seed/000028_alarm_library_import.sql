-- +goose Up
-- ============================================================
-- 000028_alarm_library_import.sql — 占位（已废弃）
--
-- T-0098-P5-06：alarm_libraries / alarm_library_i18n 表已 DROP（migrations/000062）。
-- 新告警字典走 alarm_definitions 表（migrations/000059），由 dictloader 从 XML 重载，不需要 SQL 种子。
--
-- 保留版本号 000028 占位，避免回滚到老 commit 时 goose_db_version 出现空洞。
-- ============================================================

SELECT 1;

-- +goose Down

SELECT 1;
