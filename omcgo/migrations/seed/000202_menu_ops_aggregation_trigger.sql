-- +goose Up
-- ============================================================
-- 000197_menu_ops_aggregation_trigger.sql
--
-- 阶段 3 (TODO.MD)：在「运维管理」目录下新增「PM 聚合手动触发」二级菜单。
--   1) 菜单挂到 aaaa000a-0000-0000-0000-000000000001（运维管理 directory）
--   2) 绑定到 admin 角色
--   3) 绑定 POST /api/v1/pm/aggregation/recompute 端点到 admin 角色
--
-- 新菜单 UUID：aaaa000a-1000-0000-0000-000000000007（运维管理下 sort=70）
-- 前端路由：/ops/aggregation-trigger（routes.tsx 已注册）
-- ============================================================

-- 1) 新增「PM 聚合手动触发」二级菜单
INSERT INTO menus (
    id, name, name_i18n, type, permission_key, parent_id, sort_order,
    route_path, component_path, icon, status, show_status
)
VALUES (
    'aaaa000a-1000-0000-0000-000000000007'::uuid,
    'PM 聚合手动触发',
    '{"zh-CN":"PM 聚合手动触发","en-US":"PM Aggregation Trigger"}'::jsonb,
    'menu',
    'ops:pm:aggregation:trigger',
    'aaaa000a-0000-0000-0000-000000000001'::uuid,
    70,
    '/ops/aggregation-trigger',
    'ops/AggregationTrigger',
    'ThunderboltOutlined',
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

-- 2) 绑定到 admin 角色
INSERT INTO role_menus (role_id, menu_id)
VALUES
    ('10000000-0000-0000-0000-000000000001'::uuid,
     'aaaa000a-1000-0000-0000-000000000007'::uuid)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- 3) 绑定 POST /api/v1/pm/aggregation/recompute 端点到 admin 角色
--    依赖 syncApiEndpoints 自注册（app 启动期写入 api_endpoints 表）。
INSERT INTO role_api_permissions (role_id, endpoint_id)
SELECT r.id, ae.id
FROM roles r
JOIN api_endpoints ae ON ae.path = '/api/v1/pm/aggregation/recompute'
WHERE r.name = 'admin'
ON CONFLICT (role_id, endpoint_id) DO NOTHING;


-- +goose Down
DELETE FROM role_api_permissions
WHERE role_id IN (SELECT id FROM roles WHERE name = 'admin')
  AND endpoint_id IN (
      SELECT id FROM api_endpoints WHERE path = '/api/v1/pm/aggregation/recompute'
  );

DELETE FROM role_menus
WHERE menu_id = 'aaaa000a-1000-0000-0000-000000000007'::uuid;

DELETE FROM menus
WHERE id = 'aaaa000a-1000-0000-0000-000000000007'::uuid;
