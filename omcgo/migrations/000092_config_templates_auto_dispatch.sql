-- +goose Up
-- T-0120-b
-- 给 config_templates 增加 auto_dispatch 列。
--
-- 用途：opt-in 标记某条模板是否允许在 device.registered 事件触发时
-- 自动走 Path A 下发（无需 UI 手动调 POST /:id/dispatch）。
--
-- 写入方：用户通过 PUT /api/v1/templates/:id 设置；默认 FALSE。
-- 消费方：provision/engine.go HandleBootstrap 选 Path A 条件加 OR tmpl.AutoDispatch。
-- 默认 FALSE：既有 T-0120 e0718c15 测试模板与生产模板不会被意外自动触发，
-- 保留 §5.4 path-b sync 稳态；要启用需逐条 opt-in。

ALTER TABLE config_templates
    ADD COLUMN IF NOT EXISTS auto_dispatch BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN config_templates.auto_dispatch IS 'T-0120-b opt-in 自动下发：device.registered 事件触发时若模板匹配且此列 true → 强制走 Path A，默认 FALSE 保留手动 dispatch 语义';

-- +goose Down
ALTER TABLE config_templates DROP COLUMN IF EXISTS auto_dispatch;
