-- 2026-05-28 把"标准参数树"从 param-model 页面 Tab 拆出独立菜单页
--
-- 用户决策(2026-05-28):产品中心拆分 — 增加「标准参数树」一级菜单项,
-- 角色可以通过 menus + role_menus 单独控制访问权限。
--
-- 沿用 T-0098-P4 namespace 命名规则:
--   - directory:aaaa0098-0000-0000-0000-000000000001(产品中心,seed/000087 已建)
--   - page:aaaa0098-1000-...(产品中心子菜单)
--     - 1 产品管理 / 2 参数模型 / 3 KPI 指标库 / 4 告警库 / 5 孤儿设备(均 seed/000087)
--     - 6 标准参数树(本迁移新增)
--
-- 与 navConfig.ts 中 product-standard-params 项 + routes.tsx
-- /product/standard-params 路由 + StandardParamsPage 组件配套(同 commit)。
--
-- RBAC:默认绑定 admin (10000000-...001) 与其它产品中心子菜单一致;
-- operator/viewer 不绑定 = 不可见,与"仅超管可见"语义对齐。

-- +goose Up
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status, show_status)
VALUES (
    'aaaa0098-1000-0000-0000-000000000006'::uuid,
    '标准参数树',
    'menu',
    'product:standard-params',
    'aaaa0098-0000-0000-0000-000000000001'::uuid,
    6,  -- 排在"孤儿设备"(5)之后,可在管理界面手动调整
    '/product/standard-params',
    'BranchesOutlined',
    'normal',
    'show'
)
ON CONFLICT (id) DO UPDATE SET
    name=EXCLUDED.name, permission_key=EXCLUDED.permission_key, parent_id=EXCLUDED.parent_id,
    sort_order=EXCLUDED.sort_order, route_path=EXCLUDED.route_path, icon=EXCLUDED.icon,
    status=EXCLUDED.status, show_status=EXCLUDED.show_status, updated_at=NOW();

-- 多语言名称 — sidebar/Layout 优先读 name_i18n, name 兜底
UPDATE menus SET name_i18n = jsonb_build_object(
    'zh-CN', '标准参数树',
    'en-US', 'Standard Params'
)
WHERE id = 'aaaa0098-1000-0000-0000-000000000006'::uuid;

-- 默认绑定 admin(super_admin)
INSERT INTO role_menus (role_id, menu_id)
VALUES (
    '10000000-0000-0000-0000-000000000001'::uuid,
    'aaaa0098-1000-0000-0000-000000000006'::uuid
)
ON CONFLICT (role_id, menu_id) DO NOTHING;


-- +goose Down
DELETE FROM role_menus
WHERE menu_id = 'aaaa0098-1000-0000-0000-000000000006'::uuid;

DELETE FROM menus
WHERE id = 'aaaa0098-1000-0000-0000-000000000006'::uuid;
