-- T-0100-P3: License 操作权限点 seed —— menus button + role_api_permissions
--
-- 来源 PRD: docs/project/prd/F06-license.md §6.5、§15 Q3=C 决议、Gap #6
-- 关闭 Risk: R-110（治理层 license 写操作权限粒度过粗）
--
-- 业务背景：Q3=C 决议明确"不新增 license-admin / auditor 内置角色"，企业自助
-- 在 /system/roles 通过 RolePermission 页面给特定角色勾选 system:license:operate
-- 即可授权。本 seed 只做最小增量：
--   1. menus 表加 button 节点 permission_key='system:license:operate'
--   2. api_endpoints 显式注册 license 写端点（防止启动期 syncApiEndpoints race）
--   3. role_api_permissions 默认仅 grant 给 admin（10000000-...001）
--
-- 与端点级 RBAC 的关系（同 seed/000071 北向编辑模式）：
--   * 前端 usePermission('system:license:operate') 控制 4 Tab 提交按钮
--     disabled + Tooltip（disabled 而非隐藏，PRD §6.5）
--   * 后端中间件按 role_api_permissions 兜底：即使前端绕过按钮直接 POST，
--     无权限角色仍 403
--
-- 命名约定：
--   * menus.permission_key = 'system:license:operate'（与 system:license:view /
--     system:license:audit 同前缀）
--   * UUID aaaa0009-1100-... 前缀，避免与既有 system/config (aaaa0008) 冲突
--
-- 幂等：menus PK + role_api_permissions UNIQUE (role_id, endpoint_id) +
-- ON CONFLICT DO NOTHING 兜底，可重复执行。

-- +goose Up
-- 1. menus button 节点（无 parent_id —— /license/* 一级菜单暂未 seed，先以
--    无父挂载方式只提供 permission_key 语义；T-0100-P4 完整菜单时再补
--    parent_id 关联到 license/operations 父菜单）。
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, status, show_status)
VALUES (
    'aaaa0009-1100-0000-0000-000000000001'::uuid,
    '许可证操作',
    'button',
    'system:license:operate',
    NULL,
    100,
    '',
    'normal',
    'show'
)
ON CONFLICT (id) DO NOTHING;

-- 2. role_menus 绑定 button —— 仅 admin 默认拥有；operator / viewer 不绑定
INSERT INTO role_menus (role_id, menu_id)
SELECT role_id, 'aaaa0009-1100-0000-0000-000000000001'::uuid
FROM (VALUES
    ('10000000-0000-0000-0000-000000000001'::uuid)  -- admin
) AS r(role_id)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- 3. 显式注册 license 写端点到 api_endpoints（避免 startup race）
INSERT INTO api_endpoints (id, path, method, name, description, api_group, is_auto)
VALUES
    ('50000000-0001-0000-0000-000000000001'::uuid, '/api/v1/licenses/import',     'POST', 'POST /api/v1/licenses/import',     '许可证 - 导入（独立操作权限）',  'license', FALSE),
    ('50000000-0001-0000-0000-000000000002'::uuid, '/api/v1/licenses/activate',   'POST', 'POST /api/v1/licenses/activate',   '许可证 - 激活（独立操作权限）',  'license', FALSE),
    ('50000000-0001-0000-0000-000000000003'::uuid, '/api/v1/licenses/:id/revoke', 'POST', 'POST /api/v1/licenses/:id/revoke', '许可证 - 撤销（独立操作权限）',  'license', FALSE)
ON CONFLICT (path, method) DO NOTHING;

-- 4. grant 给 admin
INSERT INTO role_api_permissions (role_id, endpoint_id)
SELECT '10000000-0000-0000-0000-000000000001'::uuid, ae.id
FROM api_endpoints ae
WHERE (ae.path, ae.method) IN (
    ('/api/v1/licenses/import',     'POST'),
    ('/api/v1/licenses/activate',   'POST'),
    ('/api/v1/licenses/:id/revoke', 'POST')
)
ON CONFLICT (role_id, endpoint_id) DO NOTHING;

-- +goose Down
-- 仅清理本 seed 注入的 admin grant + button menu；endpoint 行不删
-- （app 启动期 syncApiEndpoints 会再 sync 进来）。
DELETE FROM role_api_permissions
WHERE endpoint_id IN (
    SELECT id FROM api_endpoints
    WHERE (path, method) IN (
        ('/api/v1/licenses/import',     'POST'),
        ('/api/v1/licenses/activate',   'POST'),
        ('/api/v1/licenses/:id/revoke', 'POST')
    )
)
AND role_id = '10000000-0000-0000-0000-000000000001'::uuid;

DELETE FROM role_menus
WHERE menu_id = 'aaaa0009-1100-0000-0000-000000000001'::uuid;

DELETE FROM menus
WHERE id = 'aaaa0009-1100-0000-0000-000000000001'::uuid;
