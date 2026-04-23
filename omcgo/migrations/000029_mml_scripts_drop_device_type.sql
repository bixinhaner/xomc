-- +goose Up
-- ============================================================
-- 需求 to-do-list #9：脚本任务不需要定义产品类型
-- 对应前端 ScriptTaskDrawer 删除"产品类型"字段，并同步清理后端
-- API、Repository 与此列。
-- ============================================================
ALTER TABLE mml_scripts DROP COLUMN IF EXISTS device_type;

-- +goose Down
-- 回滚：重新加回列（保持与 000007 原始定义一致）。
ALTER TABLE mml_scripts ADD COLUMN IF NOT EXISTS device_type VARCHAR(50);
