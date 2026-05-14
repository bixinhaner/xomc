-- bugfix(menus): 补全 4 个详情/编辑类二级页菜单，修复 PrivateRoute 路径白名单漏判。
--
-- 现状：routes.tsx 注册了下列详情/编辑类二级路由，但 menus 表未登记：
--   - /device/detail/:sn                                   → DeviceDetail
--   - /device/ue-detail/:sn                                → UeDetail
--   - /device/plug-and-play/edit/:id                       → AddPolicyPage
--   - /performance/kpi-standard/detail/:deviceType/:indId  → KPIIndicatorDetail
--
-- VITE_DYNAMIC_MENU=true 模式下 PrivateRoute.isPathAllowedByMenu 只匹配 routePaths
-- 集合内路径或其前缀子路径。例如 /device/detail/<SN> 仅当 routePaths 含
-- /device/detail 才被允许；缺则一律跳 /403。导致设备列表行点击进详情页全员 403。
--
-- 本迁移补全：
--   1. 4 个 hide 状态 menu（type='menu', show_status='hide'）—— 不进侧边栏，仅参与路由白名单
--   2. role_menus 绑定 admin（10000000-...001）
--
-- 设计取舍：
--   - 选 hide menu 而非改前端 PrivateRoute：菜单系统已有 hide 语义，数据修复零代码改动；
--     PrivateRoute 改通用规则需考虑跨域边界（如 /performance/kpi-standard/detail 不应跟随
--     /performance/kpi-standard 自动放行）风险更大
--   - route_path 填裸路径（无 :sn 占位符），由前端 startsWith 前缀匹配覆盖所有参数实例
--   - 仅绑 admin：详情页跟父菜单权限对齐，operator/viewer 需要时按各自菜单授权
--
-- 幂等：menus PK + role_menus UNIQUE，可重复执行。

-- +goose Up
-- ============================================================
-- 1. menus 插入 4 个隐藏详情页节点
-- ============================================================
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status, show_status)
VALUES
    -- 设备详情（父：设备管理 directory）
    ('aaaa0126-1000-0000-0000-000000000001'::uuid,
     '设备详情', 'menu', 'device:detail',
     '11111111-1111-1111-1111-111111111101'::uuid,
     91, '/device/detail', '', 'normal', 'hide'),
    -- UE 详情（父：设备管理 directory）
    ('aaaa0126-1000-0000-0000-000000000002'::uuid,
     'UE详情', 'menu', 'device:ue-detail',
     '11111111-1111-1111-1111-111111111101'::uuid,
     92, '/device/ue-detail', '', 'normal', 'hide'),
    -- 即插即用策略编辑（父：即插即用 menu）
    ('aaaa0126-1000-0000-0000-000000000003'::uuid,
     '即插即用策略编辑', 'menu', 'device:plug-and-play:edit',
     'aaaa0010-1000-0000-0000-000000000001'::uuid,
     93, '/device/plug-and-play/edit', '', 'normal', 'hide'),
    -- KPI 指标详情（父：标准KPI menu）
    ('aaaa0126-1000-0000-0000-000000000004'::uuid,
     'KPI指标详情', 'menu', 'performance:kpi-standard:detail',
     'aaaa0002-1000-0000-0000-000000000003'::uuid,
     94, '/performance/kpi-standard/detail', '', 'normal', 'hide')
ON CONFLICT (id) DO UPDATE SET
    name=EXCLUDED.name, type=EXCLUDED.type, permission_key=EXCLUDED.permission_key,
    parent_id=EXCLUDED.parent_id, sort_order=EXCLUDED.sort_order,
    route_path=EXCLUDED.route_path, icon=EXCLUDED.icon,
    status=EXCLUDED.status, show_status=EXCLUDED.show_status, updated_at=NOW();

-- ============================================================
-- 2. role_menus 绑定 admin
-- ============================================================
INSERT INTO role_menus (role_id, menu_id)
SELECT '10000000-0000-0000-0000-000000000001'::uuid, menu_id
FROM (VALUES
    ('aaaa0126-1000-0000-0000-000000000001'::uuid),
    ('aaaa0126-1000-0000-0000-000000000002'::uuid),
    ('aaaa0126-1000-0000-0000-000000000003'::uuid),
    ('aaaa0126-1000-0000-0000-000000000004'::uuid)
) AS m(menu_id)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- +goose Down
DELETE FROM role_menus
WHERE menu_id IN (
    'aaaa0126-1000-0000-0000-000000000001'::uuid,
    'aaaa0126-1000-0000-0000-000000000002'::uuid,
    'aaaa0126-1000-0000-0000-000000000003'::uuid,
    'aaaa0126-1000-0000-0000-000000000004'::uuid
);

DELETE FROM menus
WHERE id IN (
    'aaaa0126-1000-0000-0000-000000000001'::uuid,
    'aaaa0126-1000-0000-0000-000000000002'::uuid,
    'aaaa0126-1000-0000-0000-000000000003'::uuid,
    'aaaa0126-1000-0000-0000-000000000004'::uuid
);
