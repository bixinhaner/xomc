-- 文件传输模板管理菜单 seed
--
-- 目标：把模板定义/治理从“文件传输中心”拆出为独立页面，菜单默认不绑定任何角色，
-- 由管理员在角色菜单配置中手工勾选后显示。

-- +goose Up

INSERT INTO menus (
    id, name, name_i18n, type, permission_key, parent_id, sort_order,
    route_path, component_path, icon, status, show_status
)
VALUES (
    'aaaa000b-1000-0000-0000-000000000002'::uuid,
    '传输模板管理',
    '{"zh-CN":"传输模板管理","en-US":"Transfer Template Management"}'::jsonb,
    'menu',
    'transfer:template-management',
    'aaaa000b-0000-0000-0000-000000000001'::uuid,
    2,
    '/transfer/template-management',
    'transfer/TemplateDefinitionManagement',
    'AppstoreAddOutlined',
    'normal',
    'show'
)
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM role_menus
WHERE menu_id = 'aaaa000b-1000-0000-0000-000000000002'::uuid;

DELETE FROM menus
WHERE id = 'aaaa000b-1000-0000-0000-000000000002'::uuid;