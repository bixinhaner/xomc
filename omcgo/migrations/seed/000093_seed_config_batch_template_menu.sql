-- T-0120-b §5.3 Path A 收尾 — 给 BatchParamTemplate 页（/config/batch-template）补菜单入口。
--
-- 现状：routes.tsx 已注册路由 `/config/batch-template` 指向 BatchParamTemplate 页
-- （193 行实现 + T-0120-b 加的"下发"按钮 + Dispatch Modal），但**没有任何 menus 表
-- 记录指向该路径** —— VITE_DYNAMIC_MENU=true 模式下 PrivateRoute 路径守卫读 user
-- menus 集合，不在集合内即跳 /403。导致页面在浏览器侧完全不可达，T-0120-b 加的下发
-- UI 无法被用户实际点击触发。
--
-- 本迁移补全：
--   1. menus 一级目录 `/config`（"配置管理"，directory，sort_order=4，紧跟 性能管理 / 告警管理）
--   2. menus 二级 page `/config/batch-template`（"批量参数模板"，menu）
--   3. role_menus 默认绑定 admin / super_admin（10000000-...001）2 节点
--
-- 设计取舍：
--   - 仅绑 admin，不绑 operator/viewer：批量下发模板是高权限操作（一次可影响多设备），
--     与产品中心 super_admin-only 语义一致
--   - 单独建 "配置管理" 一级目录而非挂到 "系统管理"：systems:config 已被
--     /system/config 页（系统级配置项）占用，避开语义冲突；后续若 /config/* 有更多页
--     （baseline / neighbors 等）可继续挂到此目录下
--
-- 幂等：menus PK + role_menus UNIQUE + ON CONFLICT 兜底，可重复执行。

-- +goose Up
-- ============================================================
-- 1. 一级目录 `/config` (配置管理)
-- ============================================================
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status, show_status)
VALUES (
    'aaaa0120-0000-0000-0000-000000000001'::uuid,
    '配置管理',
    'directory',
    'config',
    NULL,
    4,                                   -- 排在 dashboard / device / alarm / pm 之后
    '',
    'SettingOutlined',
    'normal',
    'show'
)
ON CONFLICT (id) DO UPDATE SET
    name=EXCLUDED.name, type=EXCLUDED.type, permission_key=EXCLUDED.permission_key,
    parent_id=EXCLUDED.parent_id, sort_order=EXCLUDED.sort_order, icon=EXCLUDED.icon,
    status=EXCLUDED.status, show_status=EXCLUDED.show_status, updated_at=NOW();

-- ============================================================
-- 2. 二级 page `批量参数模板`
-- ============================================================
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status, show_status)
VALUES (
    'aaaa0120-1000-0000-0000-000000000001'::uuid,
    '批量参数模板',
    'menu',
    'config:batch-template',
    'aaaa0120-0000-0000-0000-000000000001'::uuid,
    1,
    '/config/batch-template',
    'ProfileOutlined',
    'normal',
    'show'
)
ON CONFLICT (id) DO UPDATE SET
    name=EXCLUDED.name, permission_key=EXCLUDED.permission_key, parent_id=EXCLUDED.parent_id,
    sort_order=EXCLUDED.sort_order, route_path=EXCLUDED.route_path, icon=EXCLUDED.icon,
    status=EXCLUDED.status, show_status=EXCLUDED.show_status, updated_at=NOW();

-- ============================================================
-- 3. role_menus 默认绑定 —— 仅 admin（super_admin 语义）
-- ============================================================
INSERT INTO role_menus (role_id, menu_id)
SELECT '10000000-0000-0000-0000-000000000001'::uuid, menu_id
FROM (VALUES
    ('aaaa0120-0000-0000-0000-000000000001'::uuid),  -- directory
    ('aaaa0120-1000-0000-0000-000000000001'::uuid)   -- 批量参数模板
) AS m(menu_id)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- +goose Down
DELETE FROM role_menus
WHERE menu_id IN (
    'aaaa0120-0000-0000-0000-000000000001'::uuid,
    'aaaa0120-1000-0000-0000-000000000001'::uuid
);

DELETE FROM menus
WHERE id IN (
    'aaaa0120-1000-0000-0000-000000000001'::uuid,
    'aaaa0120-0000-0000-0000-000000000001'::uuid
);
