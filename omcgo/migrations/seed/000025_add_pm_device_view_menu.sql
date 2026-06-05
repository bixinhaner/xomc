-- +goose Up
-- 新增「性能管理 → 设备性能查看」菜单。
--
-- 背景：原「性能仪表盘」页内有 [任务仪表盘 | 设备列表] 双页签，本次把「设备列表」页签
-- 拆为性能管理下的独立子菜单「设备性能查看」（前端组件仍是 DeviceListPane，独立即席查看，
-- 不依赖聚合任务）。性能仪表盘页随之去掉页签外壳，只剩任务仪表盘。
--
-- 位置：性能管理子菜单第 3 位 —— 性能仪表盘(0) → 自定义聚合(1) → 设备性能查看(2) → 指标查询(3)。
-- 故同时把「指标查询」sort_order 由 2 顺延到 3。
--
-- 父菜单：性能管理 aaaa0002-0000-0000-0000-000000000001。

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
    'aaaa0002-1000-0000-0000-000000000012',
    '设备性能查看',
    '{"en-US": "Device Performance View", "zh-CN": "设备性能查看"}'::jsonb,
    'menu',
    'performance:device-view',
    'aaaa0002-0000-0000-0000-000000000001',  -- 父菜单：性能管理
    2,
    '/performance/device-view',
    'performance/PmDashboard/DeviceListPane',
    'BarChartOutlined',
    'show',
    'normal',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- 「指标查询」顺延到第 4 位（sort_order 2 → 3），给设备性能查看让出第 3 位。
UPDATE menus SET sort_order = 3, updated_at = NOW() WHERE route_path = '/performance/query';

-- 凡是已拥有「性能仪表盘」菜单权限的角色，同步授予「设备性能查看」，保证可见性一致。
INSERT INTO role_menus (id, role_id, menu_id, created_at)
SELECT
    gen_random_uuid(),
    rm.role_id,
    'aaaa0002-1000-0000-0000-000000000012',
    NOW()
FROM role_menus rm
WHERE rm.menu_id = 'aaaa0002-1000-0000-0000-000000000010'  -- 性能仪表盘
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- +goose Down
DELETE FROM role_menus WHERE menu_id = 'aaaa0002-1000-0000-0000-000000000012';
DELETE FROM menus WHERE id = 'aaaa0002-1000-0000-0000-000000000012';
-- 还原「指标查询」sort_order 到第 3 位（2）。
UPDATE menus SET sort_order = 2, updated_at = NOW() WHERE route_path = '/performance/query';
