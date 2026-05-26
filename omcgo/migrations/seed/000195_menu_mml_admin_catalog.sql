-- +goose Up
-- ============================================================
-- 000195_menu_mml_admin_catalog.sql
--
-- T-Mml-Admin：在「系统管理」目录下新增「MML 配置」二级菜单，并把
--   1) 新菜单绑定到 admin 角色（普通 super_admin 通过 builtIn 旁路也可访问，
--      绑 admin 角色让"具备 admin role 但非 builtIn"的运维账号也能进）
--   2) 同步把所有 /api/v1/mml/admin/* 端点绑定到 admin 角色的
--      role_api_permissions（含本次新增的 6 个 GET/batch 端点，由 app
--      启动期 syncApiEndpoints 自动写入 api_endpoints 表后再绑定）
--
-- 注意：本 seed 必须在 app 启动期 syncApiEndpoints 跑完后才能完成端点绑定
-- （依赖 api_endpoints 表先有目标行）。部署 ordering：
--   1. migrate-up 应用 schema（如有；本 seed 无 schema 变更）
--   2. app 启动 → syncApiEndpoints 把新 GET/batch 端点 UPSERT 到 api_endpoints
--   3. migrate-seed 第二次运行 → 本 seed 真正写入 role_api_permissions
--
-- 菜单 ID 规划（与 000057_refresh_menu_seed.sql 风格对齐）：
--   - 系统管理 directory (旧)  : 11111111-1111-1111-1111-111111111108
--   - MML 配置 menu  (本次新增) : aaaa0008-1000-0000-0000-000000000002
--
-- 前端路由：复用已存在的 /mml/admin/catalog（withAdminRole 守卫已挂）。
-- 菜单展示位置：系统管理 → MML 配置。
-- ============================================================

-- 1) 新增「MML 配置」二级菜单（系统管理 directory 下）
INSERT INTO menus (
    id, name, name_i18n, type, permission_key, parent_id, sort_order,
    route_path, component_path, icon, status, show_status
)
VALUES (
    'aaaa0008-1000-0000-0000-000000000002'::uuid,
    'MML 配置',
    '{"zh-CN":"MML 配置","en-US":"MML Catalog"}'::jsonb,
    'menu',
    'mml:admin:catalog',
    '11111111-1111-1111-1111-111111111108'::uuid,
    10,
    '/mml/admin/catalog',
    'mml/admin/catalog',
    'DatabaseOutlined',
    'normal',
    'show'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    name_i18n = EXCLUDED.name_i18n,
    type = EXCLUDED.type,
    permission_key = EXCLUDED.permission_key,
    parent_id = EXCLUDED.parent_id,
    sort_order = EXCLUDED.sort_order,
    route_path = EXCLUDED.route_path,
    component_path = EXCLUDED.component_path,
    icon = EXCLUDED.icon,
    status = EXCLUDED.status,
    show_status = EXCLUDED.show_status,
    updated_at = NOW();

-- 2) 菜单绑定到 admin 角色（10000000-...-0001）
INSERT INTO role_menus (role_id, menu_id)
VALUES
    ('10000000-0000-0000-0000-000000000001'::uuid,
     'aaaa0008-1000-0000-0000-000000000002'::uuid)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- 3) 把所有 /api/v1/mml/admin/* 端点绑定到 admin 角色
--    （含 T-Mml-Admin 本次新增 6 个 GET/batch 端点）
--    JOIN api_endpoints 按 path 模糊匹配，ON CONFLICT 兜底幂等。
INSERT INTO role_api_permissions (role_id, endpoint_id)
SELECT r.id, ae.id
FROM roles r
JOIN api_endpoints ae ON ae.path LIKE '/api/v1/mml/admin/%'
WHERE r.name = 'admin'
ON CONFLICT (role_id, endpoint_id) DO NOTHING;


-- +goose Down
-- ============================================================
-- 回滚：删菜单 + 撤角色绑定 + 撤端点绑定
--   - 端点绑定回滚仅清 admin × mml/admin 端点 ID，不动 api_endpoints
--     自注册结果（保留 syncApiEndpoints 的真值）
-- ============================================================

DELETE FROM role_api_permissions
WHERE role_id IN (SELECT id FROM roles WHERE name = 'admin')
  AND endpoint_id IN (
      SELECT id FROM api_endpoints WHERE path LIKE '/api/v1/mml/admin/%'
  );

DELETE FROM role_menus
WHERE menu_id = 'aaaa0008-1000-0000-0000-000000000002'::uuid;

DELETE FROM menus
WHERE id = 'aaaa0008-1000-0000-0000-000000000002'::uuid;
