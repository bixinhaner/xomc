-- +goose Up
-- 删除 UI 定制化的产品名称配置项 — 与「系统配置 > 基本设置」的 OMC 名称重复。
-- 产品名称统一由 sys_configs (category='basic', key='mrOMCName') 维护。
DELETE FROM sys_configs WHERE category = 'ui_custom' AND key = 'ui_omc_name';

-- +goose Down
INSERT INTO sys_configs (category, key, value, value_type, description, is_public)
VALUES ('ui_custom', 'ui_omc_name', 'BaiOMC', 'string', '产品名称（顶部标题/浏览器 tab）', TRUE)
ON CONFLICT (category, key) DO NOTHING;
