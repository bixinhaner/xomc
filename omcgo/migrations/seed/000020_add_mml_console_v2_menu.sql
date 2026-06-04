-- +goose Up
-- 新增「MML 控制台 V2」菜单（MML 控制台改版预览页，设计见
-- docs/design/mml-console-redesign-20260603.md）。
--
-- 与既有「MML控制台」(aaaa0003-1000-0000-0000-000000000001) 并存，互不影响：
-- V2 路由 /mml/console-v2，前端组件 mml/ConsoleV2，挂在 MML管理 目录下，
-- sort_order=2 使其紧随 V1 控制台显示（脚本任务 show_status=hide 不占可见序）。

INSERT INTO menus (
    id,
    name,
    name_i18n,
    type,
    permission_key,
    parent_id,
    sort_order,
    route_path,
    component_path,
    icon,
    show_status,
    status,
    created_at,
    updated_at
) VALUES (
    'aaaa0003-1000-0000-0000-000000000004',
    'MML控制台V2',
    '{"en-US": "MML Console V2", "zh-CN": "MML控制台V2"}'::jsonb,
    'menu',
    'mml:console-v2',
    'aaaa0003-0000-0000-0000-000000000001',  -- 父菜单：MML管理
    2,
    '/mml/console-v2',
    'mml/ConsoleV2',
    'ExperimentOutlined',
    'show',
    'normal',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- 凡是已拥有 V1「MML控制台」菜单权限的角色，同步授予 V2，保证可见性一致。
INSERT INTO role_menus (id, role_id, menu_id, created_at)
SELECT
    gen_random_uuid(),
    rm.role_id,
    'aaaa0003-1000-0000-0000-000000000004',
    NOW()
FROM role_menus rm
WHERE rm.menu_id = 'aaaa0003-1000-0000-0000-000000000001'
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- +goose Down
DELETE FROM role_menus WHERE menu_id = 'aaaa0003-1000-0000-0000-000000000004';
DELETE FROM menus WHERE id = 'aaaa0003-1000-0000-0000-000000000004';
