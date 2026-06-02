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
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- 为管理员角色添加告警统计菜单权限
INSERT INTO role_menus (role_id, menu_id, created_at)
SELECT
    r.id,
    '11111111-1111-1111-1111-111111111120',
    NOW()
FROM roles r
WHERE r.code = 'admin'
ON CONFLICT DO NOTHING;

-- +goose Down
-- 删除告警统计菜单项
DELETE FROM menus WHERE permission_key = 'alarm:statistics';

-- 删除角色菜单中的告警统计权限
DELETE FROM role_menus WHERE menu_id = '11111111-1111-1111-1111-111111111120';
