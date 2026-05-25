-- T-0164-P6 / G6 收尾：补 PM 性能查看 + 自定义聚合菜单 seed
--
-- 背景：G6 后端 commit ac3f1c54 + 前端 commit e188cb38 落了 PmDashboard / PmAdhoc
-- 三条路由（/performance, /performance/pm-adhoc, /performance/pm-dashboard），但
-- 没在 menus 表 seed。前端 PrivateRoute 动态菜单守卫拦截，登录后访问 /performance
-- 跳 /403。
--
-- 修复：在 performance 父目录（aaaa0002-0000-0000-0000-000000000001）下新增 2 个
-- 子菜单：
--   1. "性能查看" → /performance (PerformanceLayout，新单 tab 主入口)
--   2. "自定义聚合" → /performance/pm-adhoc (PmAdhoc)
-- 兼容老路由 /performance/pm-dashboard 由 PrivateRoute.isPathAllowedByMenu 的
-- startsWith('/performance/') 命中（依赖 /performance 在 routePaths set 内）。
--
-- 幂等：menus.id PK + role_menus UNIQUE(role_id,menu_id) + ON CONFLICT。

-- +goose Up

-- 1. 性能查看（新主入口，sort_order=0 排第一）
INSERT INTO menus (
    id, name, name_i18n, type, permission_key, parent_id, sort_order,
    route_path, component_path, icon, status, show_status
)
VALUES (
    'aaaa0002-1000-0000-0000-000000000010'::uuid,
    '性能查看',
    '{"zh-CN":"性能查看","en-US":"PM Dashboard"}'::jsonb,
    'menu',
    'performance:pm-dashboard',
    'aaaa0002-0000-0000-0000-000000000001'::uuid,
    0,
    '/performance',
    'performance/PmDashboard/PerformanceLayout',
    'DashboardOutlined',
    'normal',
    'show'
)
ON CONFLICT (id) DO NOTHING;

-- 2. 自定义聚合（adhoc 任务）
INSERT INTO menus (
    id, name, name_i18n, type, permission_key, parent_id, sort_order,
    route_path, component_path, icon, status, show_status
)
VALUES (
    'aaaa0002-1000-0000-0000-000000000011'::uuid,
    '自定义聚合',
    '{"zh-CN":"自定义聚合","en-US":"Ad-hoc Aggregation"}'::jsonb,
    'menu',
    'performance:pm-adhoc',
    'aaaa0002-0000-0000-0000-000000000001'::uuid,
    4,
    '/performance/pm-adhoc',
    'performance/PmAdhoc',
    'FundProjectionScreenOutlined',
    'normal',
    'show'
)
ON CONFLICT (id) DO NOTHING;

-- 3. 角色绑定：admin / operator / viewer 全部可见
INSERT INTO role_menus (role_id, menu_id)
SELECT role_id, menu_id
FROM (
    VALUES
        ('10000000-0000-0000-0000-000000000001'::uuid, 'aaaa0002-1000-0000-0000-000000000010'::uuid),
        ('10000000-0000-0000-0000-000000000001'::uuid, 'aaaa0002-1000-0000-0000-000000000011'::uuid),
        ('10000000-0000-0000-0000-000000000002'::uuid, 'aaaa0002-1000-0000-0000-000000000010'::uuid),
        ('10000000-0000-0000-0000-000000000002'::uuid, 'aaaa0002-1000-0000-0000-000000000011'::uuid),
        ('10000000-0000-0000-0000-000000000003'::uuid, 'aaaa0002-1000-0000-0000-000000000010'::uuid),
        ('10000000-0000-0000-0000-000000000003'::uuid, 'aaaa0002-1000-0000-0000-000000000011'::uuid)
) AS bindings(role_id, menu_id)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- +goose Down
DELETE FROM role_menus
WHERE menu_id IN (
    'aaaa0002-1000-0000-0000-000000000010'::uuid,
    'aaaa0002-1000-0000-0000-000000000011'::uuid
);

DELETE FROM menus
WHERE id IN (
    'aaaa0002-1000-0000-0000-000000000010'::uuid,
    'aaaa0002-1000-0000-0000-000000000011'::uuid
);
