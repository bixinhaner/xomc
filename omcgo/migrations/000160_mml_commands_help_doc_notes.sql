-- +goose Up
-- ============================================================
-- 000160_mml_commands_help_doc_notes.sql
-- 把 mml_commands 的 help_doc / notes 字段正式落到 schema 层
-- 功能域: F06 / MML 控制台
--
-- 背景：
--   help_doc / notes 历史上由 seed/000004_mml_enhance.sql B 段
--   `ALTER TABLE mml_commands ADD COLUMN IF NOT EXISTS ...`（idempotent）补出，
--   实际是 schema 变更被错放在 seed 文件中。2026-05-22 seed/000004 整体下线
--   noop 后，fresh DB 不再带这两列，导致：
--     1. seed/000152_cmcc_tdlte_v23_mml_commands.sql INSERT 引用 help_doc
--        → SQLSTATE 42703 column does not exist
--     2. internal/mml/pg_repository.go commandColumns 包含 help_doc / notes，
--        app 启动后 GetByID SELECT 失败
--
-- 行为：
--   ADD COLUMN IF NOT EXISTS，prod DB (已通过老 seed/000004 加过两列) 走
--   IF NOT EXISTS 短路 no-op，fresh DB 直加。两侧最终一致。
-- ============================================================

ALTER TABLE mml_commands ADD COLUMN IF NOT EXISTS help_doc TEXT DEFAULT '';
ALTER TABLE mml_commands ADD COLUMN IF NOT EXISTS notes    TEXT DEFAULT '';

COMMENT ON COLUMN mml_commands.help_doc IS '命令的帮助文档（前端 tooltip / detail panel 使用）';
COMMENT ON COLUMN mml_commands.notes    IS '命令的备注信息（admin 后台维护）';

-- +goose Down
-- 警告：DROP 会让 internal/mml/pg_repository.go commandColumns SELECT 失败；
-- 仅在确认应用代码已不依赖时才执行。
ALTER TABLE mml_commands DROP COLUMN IF EXISTS notes;
ALTER TABLE mml_commands DROP COLUMN IF EXISTS help_doc;
