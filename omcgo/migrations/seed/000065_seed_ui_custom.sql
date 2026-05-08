-- +goose Up
-- 系统管理 — UI 定制化（PRD docs/prd/system/ui-customization.md §3）
-- 5 个 key 全部 is_public = TRUE：登录页需要在未登录状态读取产品名 / Logo / 背景图。
-- 默认值与前端 defaultUIConfig 保持一致（前端 bundle 自带的相对路径资源）。
-- 用户在 /system/ui-custom 上传后，value 会被替换为 /api/v1/admin/public/ui-assets/<uuid>.<ext>。
INSERT INTO sys_configs (category, key, value, value_type, description, is_public) VALUES
('ui_custom', 'ui_omc_name',          'BaiOMC',                                 'string', '产品名称（顶部标题/浏览器 tab）',      TRUE),
('ui_custom', 'ui_color',             '#FF4614',                                'string', '主题色（HEX）',                        TRUE),
('ui_custom', 'ui_login_background',  './images/login/login_bg.png',            'string', '登录页背景图 URL',                     TRUE),
('ui_custom', 'ui_menu_logo_up',      './images/login/nav_logo_collapse.png',   'string', '菜单收起时的小 Logo URL',              TRUE),
('ui_custom', 'ui_menu_logo_down',    './images/login/logo_big.png',            'string', '菜单展开时的大 Logo URL',              TRUE)
ON CONFLICT (category, key) DO NOTHING;

-- +goose Down
DELETE FROM sys_configs WHERE category = 'ui_custom' AND key IN (
    'ui_omc_name',
    'ui_color',
    'ui_login_background',
    'ui_menu_logo_up',
    'ui_menu_logo_down'
);
