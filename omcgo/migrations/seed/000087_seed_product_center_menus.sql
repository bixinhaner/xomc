-- T-0098-P4 产品中心菜单补全 + role_menus 默认绑定（仅 admin / super_admin 可见）
--
-- 关闭 P4 wave 遗留：T-0098-P4-02..P4-07 完成了前端 5 个治理页 +
-- routes.tsx withSuperAdmin 守卫 + NAV_CONFIG.requireSuperAdmin 过滤，
-- 但**没有**把对应菜单条目写进数据库 menus 表 seed —— 动态菜单分支
-- （VITE_DYNAMIC_MENU=true，当前生效）从后端 /api/v1/auth/menus 读取
-- 菜单树，结果产品中心 5 项在侧边栏完全缺失。
--
-- 本迁移补全：
--   1. menus 一级目录 `/product` (directory, sort_order=12，排在 ops 后)
--   2. menus 二级 page menus (5)：产品管理 / 参数模型 / KPI 指标库 / 告警库 / 孤儿设备
--   3. role_menus 默认绑定：
--      - admin (10000000-...001)：directory + 5 page（共 6 节点）
--      - operator / viewer：**不绑定** —— 等价于"仅超管可见"的字典治理语义
--
-- 设计取舍：
--   - menus 表无 super_admin 字段；超管旁路在 service 层（user.source='builtIn'
--     直接 GetAllActive 全量菜单），普通角色按 role_menus 过滤。因此**只要不绑
--     定到 operator/viewer**，就实现"仅超管可见"的效果，与前端 NAV_CONFIG.
--     requireSuperAdmin=true 行为一致（参 components/Layout/Sidebar/navConfig.ts）
--   - 不创建 button 子节点：产品中心 5 个治理页内部权限粒度暂走前端 withSuperAdmin
--     守卫整页拦截，无需 RBAC 按钮级控制（与 license 模块不同）
--   - i18n：name 用中文兜底，i18n_key 在后续迁移 000084 backfill 链路里走
--     name_i18n JSON（如需多语言再补，本迁移不强制）
--
-- 命名约定：
--   - UUID 沿用 T-0098 namespace（aaaa0098-0000=dir / aaaa0098-1000-N=page）
--     与 license (aaaa0009) / ops (aaaa000a) namespace 区分
--   - permission_key 两段式：directory='product' / menu='product:<slug>'
--
-- 幂等：menus PK + role_menus UNIQUE(role_id, menu_id) + ON CONFLICT 兜底，
-- 可重复执行。

-- +goose Up
-- ============================================================
-- 1. 一级目录 `/product`
-- ============================================================
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status, show_status)
VALUES (
    'aaaa0098-0000-0000-0000-000000000001'::uuid,
    '产品中心',
    'directory',
    'product',
    NULL,
    12,                              -- 排在 ops (sort=11) 之后
    '',
    'AppstoreAddOutlined',           -- 与前端 navConfig.ts 一致
    'normal',
    'show'
)
ON CONFLICT (id) DO UPDATE SET
    name=EXCLUDED.name, type=EXCLUDED.type, permission_key=EXCLUDED.permission_key,
    parent_id=EXCLUDED.parent_id, sort_order=EXCLUDED.sort_order, icon=EXCLUDED.icon,
    status=EXCLUDED.status, show_status=EXCLUDED.show_status, updated_at=NOW();

-- ============================================================
-- 2. 二级 page menus (5)
-- ============================================================
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status, show_status)
VALUES
    ('aaaa0098-1000-0000-0000-000000000001'::uuid, '产品管理',   'menu', 'product:products',       'aaaa0098-0000-0000-0000-000000000001'::uuid, 1, '/product/products',       'AppstoreOutlined',    'normal', 'show'),
    ('aaaa0098-1000-0000-0000-000000000002'::uuid, '参数模型',   'menu', 'product:param-model',    'aaaa0098-0000-0000-0000-000000000001'::uuid, 2, '/product/param-model',    'ProjectOutlined',     'normal', 'show'),
    ('aaaa0098-1000-0000-0000-000000000003'::uuid, 'KPI 指标库', 'menu', 'product:kpi-library',    'aaaa0098-0000-0000-0000-000000000001'::uuid, 3, '/product/kpi-library',    'BarChartOutlined',    'normal', 'show'),
    ('aaaa0098-1000-0000-0000-000000000004'::uuid, '告警库',     'menu', 'product:alarm-library',  'aaaa0098-0000-0000-0000-000000000001'::uuid, 4, '/product/alarm-library',  'BookOutlined',        'normal', 'show'),
    ('aaaa0098-1000-0000-0000-000000000005'::uuid, '孤儿设备',   'menu', 'product:orphan-devices', 'aaaa0098-0000-0000-0000-000000000001'::uuid, 5, '/product/orphan-devices', 'DisconnectOutlined',  'normal', 'show')
ON CONFLICT (id) DO UPDATE SET
    name=EXCLUDED.name, permission_key=EXCLUDED.permission_key, parent_id=EXCLUDED.parent_id,
    sort_order=EXCLUDED.sort_order, route_path=EXCLUDED.route_path, icon=EXCLUDED.icon,
    status=EXCLUDED.status, show_status=EXCLUDED.show_status, updated_at=NOW();

-- ============================================================
-- 3. role_menus 默认绑定 —— 仅 admin（super_admin 语义）
-- ============================================================
-- admin (10000000-...001)：directory + 5 page（共 6 节点）
-- operator/viewer 不绑定 = 不可见，与前端 requireSuperAdmin 语义对齐
INSERT INTO role_menus (role_id, menu_id)
SELECT '10000000-0000-0000-0000-000000000001'::uuid, menu_id
FROM (VALUES
    ('aaaa0098-0000-0000-0000-000000000001'::uuid),  -- directory
    ('aaaa0098-1000-0000-0000-000000000001'::uuid),  -- 产品管理
    ('aaaa0098-1000-0000-0000-000000000002'::uuid),  -- 参数模型
    ('aaaa0098-1000-0000-0000-000000000003'::uuid),  -- KPI 指标库
    ('aaaa0098-1000-0000-0000-000000000004'::uuid),  -- 告警库
    ('aaaa0098-1000-0000-0000-000000000005'::uuid)   -- 孤儿设备
) AS m(menu_id)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- +goose Down
-- 清理 role_menus（仅 admin 绑定的 6 条；ON CASCADE 也会跟 menus DROP 一起带走，
-- 但显式删避免 down 中间状态有残留）
DELETE FROM role_menus
WHERE menu_id IN (
    'aaaa0098-0000-0000-0000-000000000001'::uuid,
    'aaaa0098-1000-0000-0000-000000000001'::uuid,
    'aaaa0098-1000-0000-0000-000000000002'::uuid,
    'aaaa0098-1000-0000-0000-000000000003'::uuid,
    'aaaa0098-1000-0000-0000-000000000004'::uuid,
    'aaaa0098-1000-0000-0000-000000000005'::uuid
);

-- 删除本迁移新增的 1 directory + 5 page（共 6 节点）
DELETE FROM menus
WHERE id IN (
    'aaaa0098-1000-0000-0000-000000000001'::uuid,
    'aaaa0098-1000-0000-0000-000000000002'::uuid,
    'aaaa0098-1000-0000-0000-000000000003'::uuid,
    'aaaa0098-1000-0000-0000-000000000004'::uuid,
    'aaaa0098-1000-0000-0000-000000000005'::uuid,
    'aaaa0098-0000-0000-0000-000000000001'::uuid
);
