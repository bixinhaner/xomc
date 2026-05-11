-- F06 运维管理（Ops Management）菜单 + 角色绑定 seed
--
-- 来源 PRD: docs/project/prd/F06-ops-management.md §8
--
-- 设计取舍：
--   1. UUID 命名空间 aaaa000a-... 与既有目录互不冲突
--      （aaaa0001~0009 已被占用，aaaa000a 从顶部新启）
--   2. 目录 sort_order=11，排在"许可证管理"(10) 之后；不动既有顺序
--   3. 当前后端 ops handler 已注册在 permGroup("devices")，role_api_permissions
--      已通过现有 devices 组覆盖，本 seed **只控菜单可见性 + 按钮权限点**
--   4. button 权限点命名规范：ops:<sub>:<action>
--      - ops:template:view/create/edit/delete
--      - ops:task:view/create/cancel/control
--      - ops:command:view/execute
--      - ops:diagnostic:view（暂为占位，后续 W3 接 trigger button）
--      - ops:download:view（同上）
--   5. role_menus 默认绑定（PRD §8.1）：
--      - admin (super_admin): 全部
--      - operator: directory + 5 page + 安全按钮（view/create/control/execute）
--      - viewer:   directory + 5 page + 仅 view 按钮
--   6. NetworkDiagnosis / Downloads 后端 endpoint 尚未落地（PRD §12 W3），但菜单
--      先 seed 占位，前端展示"路线图实施中"卡片，避免动态菜单灰度上找不到入口
--
-- 幂等：menus PK + role_menus UNIQUE(role_id, menu_id) + ON CONFLICT 兜底。

-- +goose Up
-- ============================================================
-- 1. 一级目录：运维管理
-- ============================================================
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status, show_status)
VALUES
    ('aaaa000a-0000-0000-0000-000000000001'::uuid, '运维管理', 'directory', 'ops', NULL, 11, '', 'ToolOutlined', 'normal', 'show')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- 2. 二级 page menus（5 个）
-- ============================================================
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, component_path, icon, status, show_status)
VALUES
    ('aaaa000a-1000-0000-0000-000000000001'::uuid, '运维模板', 'menu', 'ops:template', 'aaaa000a-0000-0000-0000-000000000001'::uuid, 1, '/ops/templates',    'ops/Templates',         'FileTextOutlined',  'normal', 'show'),
    ('aaaa000a-1000-0000-0000-000000000002'::uuid, '运维任务', 'menu', 'ops:task',     'aaaa000a-0000-0000-0000-000000000001'::uuid, 2, '/ops/tasks',        'ops/TaskManagement',    'ScheduleOutlined',  'normal', 'show'),
    ('aaaa000a-1000-0000-0000-000000000003'::uuid, '运维命令', 'menu', 'ops:command',  'aaaa000a-0000-0000-0000-000000000001'::uuid, 3, '/ops/commands',     'ops/CommandManagement', 'CodeOutlined',      'normal', 'show'),
    ('aaaa000a-1000-0000-0000-000000000004'::uuid, '网络诊断', 'menu', 'ops:diagnostic','aaaa000a-0000-0000-0000-000000000001'::uuid, 4, '/ops/diagnostics',  'ops/NetworkDiagnosis',  'RadarChartOutlined','normal', 'show'),
    ('aaaa000a-1000-0000-0000-000000000005'::uuid, '运维下载', 'menu', 'ops:download', 'aaaa000a-0000-0000-0000-000000000001'::uuid, 5, '/ops/downloads',    'ops/Downloads',         'DownloadOutlined',  'normal', 'show')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- 3. 三级 buttons —— 模板（4）+ 任务（4）+ 命令（2）+ 诊断（1）+ 下载（1）= 12
-- ============================================================
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, status, show_status)
VALUES
    -- 模板 buttons (parent = aaaa000a-1000-...-001)
    ('aaaa000a-1100-0000-0000-000000000001'::uuid, '查询', 'button', 'ops:template:view',   'aaaa000a-1000-0000-0000-000000000001'::uuid, 1, '', 'normal', 'show'),
    ('aaaa000a-1100-0000-0000-000000000002'::uuid, '新增', 'button', 'ops:template:create', 'aaaa000a-1000-0000-0000-000000000001'::uuid, 2, '', 'normal', 'show'),
    ('aaaa000a-1100-0000-0000-000000000003'::uuid, '编辑', 'button', 'ops:template:edit',   'aaaa000a-1000-0000-0000-000000000001'::uuid, 3, '', 'normal', 'show'),
    ('aaaa000a-1100-0000-0000-000000000004'::uuid, '删除', 'button', 'ops:template:delete', 'aaaa000a-1000-0000-0000-000000000001'::uuid, 4, '', 'normal', 'show'),
    -- 任务 buttons (parent = aaaa000a-1000-...-002)
    ('aaaa000a-1200-0000-0000-000000000001'::uuid, '查询', 'button', 'ops:task:view',    'aaaa000a-1000-0000-0000-000000000002'::uuid, 1, '', 'normal', 'show'),
    ('aaaa000a-1200-0000-0000-000000000002'::uuid, '新建', 'button', 'ops:task:create',  'aaaa000a-1000-0000-0000-000000000002'::uuid, 2, '', 'normal', 'show'),
    ('aaaa000a-1200-0000-0000-000000000003'::uuid, '取消', 'button', 'ops:task:cancel',  'aaaa000a-1000-0000-0000-000000000002'::uuid, 3, '', 'normal', 'show'),
    ('aaaa000a-1200-0000-0000-000000000004'::uuid, '暂停/恢复', 'button', 'ops:task:control', 'aaaa000a-1000-0000-0000-000000000002'::uuid, 4, '', 'normal', 'show'),
    -- 命令 buttons (parent = aaaa000a-1000-...-003)
    ('aaaa000a-1300-0000-0000-000000000001'::uuid, '查询', 'button', 'ops:command:view',    'aaaa000a-1000-0000-0000-000000000003'::uuid, 1, '', 'normal', 'show'),
    ('aaaa000a-1300-0000-0000-000000000002'::uuid, '执行', 'button', 'ops:command:execute', 'aaaa000a-1000-0000-0000-000000000003'::uuid, 2, '', 'normal', 'show'),
    -- 诊断 button (parent = aaaa000a-1000-...-004)
    ('aaaa000a-1400-0000-0000-000000000001'::uuid, '查询', 'button', 'ops:diagnostic:view', 'aaaa000a-1000-0000-0000-000000000004'::uuid, 1, '', 'normal', 'show'),
    -- 下载 button (parent = aaaa000a-1000-...-005)
    ('aaaa000a-1500-0000-0000-000000000001'::uuid, '查询', 'button', 'ops:download:view',   'aaaa000a-1000-0000-0000-000000000005'::uuid, 1, '', 'normal', 'show')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- 4. role_menus 默认绑定
-- ============================================================
-- 4.1 admin (10000000-...001) — 全部菜单 + 全部 button
INSERT INTO role_menus (role_id, menu_id)
SELECT '10000000-0000-0000-0000-000000000001'::uuid, id
FROM menus
WHERE id IN (
    'aaaa000a-0000-0000-0000-000000000001'::uuid,
    'aaaa000a-1000-0000-0000-000000000001'::uuid,
    'aaaa000a-1000-0000-0000-000000000002'::uuid,
    'aaaa000a-1000-0000-0000-000000000003'::uuid,
    'aaaa000a-1000-0000-0000-000000000004'::uuid,
    'aaaa000a-1000-0000-0000-000000000005'::uuid,
    'aaaa000a-1100-0000-0000-000000000001'::uuid,
    'aaaa000a-1100-0000-0000-000000000002'::uuid,
    'aaaa000a-1100-0000-0000-000000000003'::uuid,
    'aaaa000a-1100-0000-0000-000000000004'::uuid,
    'aaaa000a-1200-0000-0000-000000000001'::uuid,
    'aaaa000a-1200-0000-0000-000000000002'::uuid,
    'aaaa000a-1200-0000-0000-000000000003'::uuid,
    'aaaa000a-1200-0000-0000-000000000004'::uuid,
    'aaaa000a-1300-0000-0000-000000000001'::uuid,
    'aaaa000a-1300-0000-0000-000000000002'::uuid,
    'aaaa000a-1400-0000-0000-000000000001'::uuid,
    'aaaa000a-1500-0000-0000-000000000001'::uuid
)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- 4.2 operator (10000000-...002) — directory + 5 page + 安全 buttons（view/create/control/execute；不含 delete）
INSERT INTO role_menus (role_id, menu_id)
SELECT '10000000-0000-0000-0000-000000000002'::uuid, id
FROM menus
WHERE id IN (
    'aaaa000a-0000-0000-0000-000000000001'::uuid,  -- dir
    'aaaa000a-1000-0000-0000-000000000001'::uuid,  -- template page
    'aaaa000a-1000-0000-0000-000000000002'::uuid,  -- task page
    'aaaa000a-1000-0000-0000-000000000003'::uuid,  -- command page
    'aaaa000a-1000-0000-0000-000000000004'::uuid,  -- diag page
    'aaaa000a-1000-0000-0000-000000000005'::uuid,  -- download page
    'aaaa000a-1100-0000-0000-000000000001'::uuid,  -- template:view
    'aaaa000a-1100-0000-0000-000000000002'::uuid,  -- template:create
    'aaaa000a-1100-0000-0000-000000000003'::uuid,  -- template:edit
    'aaaa000a-1200-0000-0000-000000000001'::uuid,  -- task:view
    'aaaa000a-1200-0000-0000-000000000002'::uuid,  -- task:create
    'aaaa000a-1200-0000-0000-000000000003'::uuid,  -- task:cancel
    'aaaa000a-1200-0000-0000-000000000004'::uuid,  -- task:control
    'aaaa000a-1300-0000-0000-000000000001'::uuid,  -- command:view
    'aaaa000a-1300-0000-0000-000000000002'::uuid,  -- command:execute
    'aaaa000a-1400-0000-0000-000000000001'::uuid,  -- diag:view
    'aaaa000a-1500-0000-0000-000000000001'::uuid   -- download:view
)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- 4.3 viewer (10000000-...003) — directory + 5 page + 仅 view 按钮
INSERT INTO role_menus (role_id, menu_id)
SELECT '10000000-0000-0000-0000-000000000003'::uuid, id
FROM menus
WHERE id IN (
    'aaaa000a-0000-0000-0000-000000000001'::uuid,
    'aaaa000a-1000-0000-0000-000000000001'::uuid,
    'aaaa000a-1000-0000-0000-000000000002'::uuid,
    'aaaa000a-1000-0000-0000-000000000003'::uuid,
    'aaaa000a-1000-0000-0000-000000000004'::uuid,
    'aaaa000a-1000-0000-0000-000000000005'::uuid,
    'aaaa000a-1100-0000-0000-000000000001'::uuid,  -- template:view
    'aaaa000a-1200-0000-0000-000000000001'::uuid,  -- task:view
    'aaaa000a-1300-0000-0000-000000000001'::uuid,  -- command:view
    'aaaa000a-1400-0000-0000-000000000001'::uuid,  -- diag:view
    'aaaa000a-1500-0000-0000-000000000001'::uuid   -- download:view
)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- +goose Down
-- 清理本 seed 注入的 role_menus + menus 节点
DELETE FROM role_menus WHERE menu_id IN (
    SELECT id FROM menus WHERE id::text LIKE 'aaaa000a-%'
);
DELETE FROM menus WHERE id::text LIKE 'aaaa000a-%';
