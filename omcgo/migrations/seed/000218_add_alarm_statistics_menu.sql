-- +goose Up
-- 新增告警统计菜单项
-- 根据 alarm-submenu-design.md 设计方案，在告警管理下新增"告警统计"子菜单

-- 插入告警统计菜单项
-- 排序顺序设为10，位于告警规则之后，告警同步之前
-- 权限键使用 alarm:statistics，与现有告警菜单权限命名模式保持一致
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
    route_params,
    created_at,
    updated_at
) VALUES (
    '11111111-1111-1111-1111-111111111120',
    '告警统计',
    '{"en-US": "Alarm Statistics", "zh-CN": "告警统计"}'::jsonb,
    'menu',
    'alarm:statistics',
    '11111111-1111-1111-1111-111111111105',  -- 父菜单：告警管理
    10,
    '/alarm/statistics',
    'alarm/AlarmStatistics',
    'BarChartOutlined',
    'show',
    'normal',
    NULL,
    NOW(),
    NOW()
) ON CONFLICT (permission_key) DO NOTHING;

-- 为管理员角色添加告警统计权限
-- 获取admin角色的ID并添加权限
INSERT INTO role_permissions (role_id, permission_key, created_at, updated_at)
SELECT
    id,
    'alarm:statistics',
    NOW(),
    NOW()
FROM roles
WHERE name_i18n->>'zh-CN' = '管理员'
  OR name = 'admin'
ON CONFLICT DO NOTHING;

-- +goose Down
-- 删除告警统计菜单项
DELETE FROM menus WHERE permission_key = 'alarm:statistics';

-- 删除角色权限中的告警统计权限
DELETE FROM role_permissions WHERE permission_key = 'alarm:statistics';
