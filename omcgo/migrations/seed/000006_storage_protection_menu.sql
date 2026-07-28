-- +goose Up
-- 存储与资源保护管理页：仅授予内置 admin 角色，API 仍由 users:admin RBAC 保护。
INSERT INTO public.menus (
    id, name, type, permission_key, parent_id, sort_order, route_path,
    component_path, icon, show_status, status, name_i18n
) VALUES (
    'aaaa0008-1000-0000-0000-000000000010',
    '资源与存储保护',
    'menu',
    'system:storage-protection',
    '11111111-1111-1111-1111-111111111108',
    21,
    '/system/storage-protection',
    'system/StorageProtection',
    'DatabaseOutlined',
    'show',
    'normal',
    '{"en-US":"Resource and Storage Protection","zh-CN":"资源与存储保护"}'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    permission_key = EXCLUDED.permission_key,
    parent_id = EXCLUDED.parent_id,
    sort_order = EXCLUDED.sort_order,
    route_path = EXCLUDED.route_path,
    component_path = EXCLUDED.component_path,
    icon = EXCLUDED.icon,
    show_status = EXCLUDED.show_status,
    status = EXCLUDED.status,
    name_i18n = EXCLUDED.name_i18n,
    updated_at = NOW();

INSERT INTO public.role_menus (role_id, menu_id)
VALUES (
    '10000000-0000-0000-0000-000000000001',
    'aaaa0008-1000-0000-0000-000000000010'
)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- +goose Down
DELETE FROM public.role_menus
 WHERE menu_id = 'aaaa0008-1000-0000-0000-000000000010';
DELETE FROM public.menus
 WHERE id = 'aaaa0008-1000-0000-0000-000000000010';
