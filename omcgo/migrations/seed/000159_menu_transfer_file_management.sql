-- 菜单调整（与 navConfig.ts 同步）
--
-- 目标：
--   1. 「任务创建」改名为「任务管理」（id aaaa000b-1000-0000-0000-000000000001）
--   2. 文件传输下新增「文件管理」菜单（id aaaa000b-1000-0000-0000-000000000003），
--      路由 /transfer/file-management，页面内 2 tab（版本文件 / 配置文件）
--   3. 「备份恢复」目录整棵下线（配置快照能力已搬到「文件传输 / 文件管理」）
--
-- 前端对照：
--   - omcmb/webcode/src/components/Layout/Sidebar/navConfig.ts
--   - omcmb/webcode/src/router/routes.tsx （/transfer/file-management 路由已就位）
--   - omcmb/webcode/src/pages/transfer/FileManagement/index.tsx
--
-- 幂等：menus PK + role_menus UNIQUE(role_id, menu_id) + ON CONFLICT。

-- +goose Up

-- 1. 「任务创建」→「任务管理」
UPDATE menus
SET name = '任务管理',
    name_i18n = '{"zh-CN":"任务管理","en-US":"Task Management"}'::jsonb,
    updated_at = NOW()
WHERE id = 'aaaa000b-1000-0000-0000-000000000001'::uuid;

-- 2. 新增「文件管理」二级菜单
INSERT INTO menus (
    id, name, name_i18n, type, permission_key, parent_id, sort_order,
    route_path, component_path, icon, status, show_status
)
VALUES (
    'aaaa000b-1000-0000-0000-000000000003'::uuid,
    '文件管理',
    '{"zh-CN":"文件管理","en-US":"File Management"}'::jsonb,
    'menu',
    'transfer:file-management',
    'aaaa000b-0000-0000-0000-000000000001'::uuid,
    2,
    '/transfer/file-management',
    'transfer/FileManagement',
    'FolderOpenOutlined',
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

-- 3. 「文件管理」绑定到 admin / operator / viewer 三个默认角色
INSERT INTO role_menus (role_id, menu_id)
SELECT role_id, menu_id FROM (
    VALUES
        ('10000000-0000-0000-0000-000000000001'::uuid, 'aaaa000b-1000-0000-0000-000000000003'::uuid),
        ('10000000-0000-0000-0000-000000000002'::uuid, 'aaaa000b-1000-0000-0000-000000000003'::uuid),
        ('10000000-0000-0000-0000-000000000003'::uuid, 'aaaa000b-1000-0000-0000-000000000003'::uuid)
) AS bindings(role_id, menu_id)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- 4. 「备份恢复」目录整棵下线
--    涉及节点（参考 000057_refresh_menu_seed.sql）：
--      - aaaa0005-0000-0000-0000-000000000001  「备份恢复」directory
--      - aaaa0005-1000-0000-0000-000000000001  「备份任务」  → /backup/tasks
--      - aaaa0005-1000-0000-0000-000000000002  「配置文件」  → /backup/schedule
--      - aaaa0005-1000-0000-0000-000000000003  「数据恢复」  → /backup/restore
--    DELETE 父 directory，FK CASCADE 自动清子菜单 + role_menus 绑定。
DELETE FROM menus WHERE id = 'aaaa0005-0000-0000-0000-000000000001'::uuid;


-- +goose Down

-- 回滚 4：恢复「备份恢复」目录 + 3 个子菜单（按 000057 原值，按钮/角色绑定不还原）
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status) VALUES
  ('aaaa0005-0000-0000-0000-000000000001', '备份恢复', 'directory', 'backup',         NULL,                                   6, NULL,                'SaveOutlined',     'normal'),
  ('aaaa0005-1000-0000-0000-000000000001', '备份任务', 'menu',      'backup:tasks',   'aaaa0005-0000-0000-0000-000000000001', 1, '/backup/tasks',    'FileDoneOutlined', 'normal'),
  ('aaaa0005-1000-0000-0000-000000000002', '配置文件', 'menu',      'backup:schedule','aaaa0005-0000-0000-0000-000000000001', 2, '/backup/schedule', 'CalendarOutlined', 'normal'),
  ('aaaa0005-1000-0000-0000-000000000003', '数据恢复', 'menu',      'backup:restore', 'aaaa0005-0000-0000-0000-000000000001', 3, '/backup/restore',  'RollbackOutlined', 'normal')
ON CONFLICT (id) DO NOTHING;

-- 回滚 3：撤销「文件管理」角色绑定
DELETE FROM role_menus
WHERE menu_id = 'aaaa000b-1000-0000-0000-000000000003'::uuid;

-- 回滚 2：删除「文件管理」菜单
DELETE FROM menus WHERE id = 'aaaa000b-1000-0000-0000-000000000003'::uuid;

-- 回滚 1：「任务管理」→「任务创建」
UPDATE menus
SET name = '任务创建',
    name_i18n = '{"zh-CN":"任务创建","en-US":"Task Creation"}'::jsonb,
    updated_at = NOW()
WHERE id = 'aaaa000b-1000-0000-0000-000000000001'::uuid;
