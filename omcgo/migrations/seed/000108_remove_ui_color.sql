-- +goose Up
-- 删除 UI 定制化的主题色配置项 — 全局只支持 Header 右上角两主题切换（tech ↔ fresh）。
-- 前端 UI 定制化页面已同步移除 ColorPicker；保留此行只会成为孤儿数据。
DELETE FROM sys_configs WHERE category = 'ui_custom' AND key = 'ui_color';

-- +goose Down
INSERT INTO sys_configs (category, key, value, value_type, description, is_public)
VALUES ('ui_custom', 'ui_color', '#FF4614', 'string', '主题色（HEX）', TRUE)
ON CONFLICT (category, key) DO NOTHING;
