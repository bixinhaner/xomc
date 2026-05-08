-- 北向 OSS 主备服务器编辑按钮 — 菜单 button 节点 + admin/operator role_menus 绑定
--
-- 业务背景：与 PUT /api/v1/northbound/servers/:role 配套，前端 system/config
-- 北向设置页"编辑"按钮通过 usePermission('system:config:northbound:edit') 控制
-- disabled / Tooltip。viewer 等无此权限的角色看到按钮但不能点（PRD
-- docs/prd/system/menu-dynamic-loading.md §4.3.6 "disabled 而非隐藏"决议）。
--
-- 与端点级 RBAC 的关系：本 seed 控制 *前端按钮可见交互态*；后端鉴权由
-- seed/000070 注入的 role_api_permissions 兜底（即使前端绕过按钮直接调 PUT，
-- 后端中间件仍 403）。前端预测 + 后端兜底双层防护。
--
-- 命名约定：`system:config:northbound:edit` — 与既有 system:config:{query,add,
-- edit,delete} button 同前缀，避免散落。

-- +goose Up
-- 1. menus button 节点（挂在 system/config 父目录下）
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, status, show_status)
VALUES (
    'aaaa0008-1100-0000-0000-000000000001'::uuid,
    '北向服务器编辑',
    'button',
    'system:config:northbound:edit',
    'aaaa0008-1000-0000-0000-000000000001'::uuid,  -- 系统配置父节点
    100,                                            -- 排在 query/add/edit/delete (sort 1-4) 之后
    '',                                             -- button 类型无路由
    'active',
    'show'
)
ON CONFLICT (id) DO NOTHING;

-- 2. role_menus 绑定 — admin / operator 默认拥有；viewer 不绑定
INSERT INTO role_menus (role_id, menu_id)
SELECT
    role_id,
    'aaaa0008-1100-0000-0000-000000000001'::uuid
FROM (VALUES
    ('10000000-0000-0000-0000-000000000001'::uuid),  -- admin
    ('10000000-0000-0000-0000-000000000002'::uuid)   -- operator
) AS r(role_id)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- +goose Down
DELETE FROM role_menus
WHERE menu_id = 'aaaa0008-1100-0000-0000-000000000001'::uuid;

DELETE FROM menus
WHERE id = 'aaaa0008-1100-0000-0000-000000000001'::uuid;
