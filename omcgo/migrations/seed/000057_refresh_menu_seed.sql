-- +goose Up
-- ============================================================
-- 000057_refresh_menu_seed.sql
-- 把 menus 表刷成与前端 omcmb/webcode/src/components/Layout/Sidebar/navConfig.ts
-- 一致的菜单全集，幂等 upsert (ON CONFLICT (id) DO UPDATE)。
--
-- 旧 seed (000009) 仅覆盖 设备/告警/系统 三个目录的部分菜单，导致 system/menus
-- 列表显示数据不全（约 22 条）。本迁移补齐：
--   一级目录 10 个：dashboard / device / alarm / performance / mml / topology /
--                  backup / software / log / system
--   二级菜单 32 个 + 三级按钮 25 个 (覆盖主要 CRUD 页面)
--
-- UUID 命名约定：
--   * 沿用旧 seed 的 11111111-... 前缀（原 device / alarm / system 三组）
--   * 新增菜单使用 aaaa<NN>-... 前缀，避免与旧数据冲突
-- ============================================================

-- ------------------------------------------------------------
-- 一级目录（10）
-- ------------------------------------------------------------
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, icon, status) VALUES
  ('aaaa0001-0000-0000-0000-000000000001', '仪表板',   'directory', 'dashboard',   NULL, 0, 'DashboardOutlined',  'normal'),
  ('11111111-1111-1111-1111-111111111101', '设备管理', 'directory', 'device',      NULL, 1, 'ClusterOutlined',    'normal'),
  ('11111111-1111-1111-1111-111111111105', '告警管理', 'directory', 'alarm',       NULL, 2, 'AlertOutlined',      'normal'),
  ('aaaa0002-0000-0000-0000-000000000001', '性能管理', 'directory', 'performance', NULL, 3, 'LineChartOutlined',  'normal'),
  ('aaaa0003-0000-0000-0000-000000000001', 'MML管理',  'directory', 'mml',         NULL, 4, 'CodeOutlined',       'normal'),
  ('aaaa0004-0000-0000-0000-000000000001', '拓扑管理', 'directory', 'topology',    NULL, 5, 'GlobalOutlined',     'normal'),
  ('aaaa0005-0000-0000-0000-000000000001', '备份恢复', 'directory', 'backup',      NULL, 6, 'SaveOutlined',       'normal'),
  ('aaaa0006-0000-0000-0000-000000000001', '软件管理', 'directory', 'software',    NULL, 7, 'CloudUploadOutlined','normal'),
  ('aaaa0007-0000-0000-0000-000000000001', '日志管理', 'directory', 'log',         NULL, 8, 'FileTextOutlined',   'normal'),
  ('11111111-1111-1111-1111-111111111108', '系统管理', 'directory', 'system',      NULL, 9, 'ToolOutlined',       'normal')
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, type=EXCLUDED.type, permission_key=EXCLUDED.permission_key,
  parent_id=EXCLUDED.parent_id, sort_order=EXCLUDED.sort_order, icon=EXCLUDED.icon,
  status=EXCLUDED.status, updated_at=NOW();

-- ------------------------------------------------------------
-- 仪表板（1）
-- ------------------------------------------------------------
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status) VALUES
  ('aaaa0001-1000-0000-0000-000000000001', '仪表板', 'menu', 'dashboard:home', 'aaaa0001-0000-0000-0000-000000000001', 1, '/dashboard', 'DashboardOutlined', 'normal')
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, permission_key=EXCLUDED.permission_key, parent_id=EXCLUDED.parent_id,
  sort_order=EXCLUDED.sort_order, route_path=EXCLUDED.route_path, icon=EXCLUDED.icon,
  status=EXCLUDED.status, updated_at=NOW();

-- ------------------------------------------------------------
-- 设备管理（5 menus）
-- ------------------------------------------------------------
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status) VALUES
  ('11111111-1111-1111-1111-111111111102', '设备列表', 'menu', 'device:list',          '11111111-1111-1111-1111-111111111101', 1, '/device/list',          'UnorderedListOutlined', 'normal'),
  ('11111111-1111-1111-1111-111111111103', '设备分组', 'menu', 'device:group',         '11111111-1111-1111-1111-111111111101', 2, '/device/group',         'ApartmentOutlined',     'normal'),
  ('aaaa0010-1000-0000-0000-000000000001', '即插即用', 'menu', 'device:plug-and-play', '11111111-1111-1111-1111-111111111101', 3, '/device/plug-and-play', 'ThunderboltOutlined',   'normal'),
  ('aaaa0010-1000-0000-0000-000000000002', '设备规则', 'menu', 'device:rules',         '11111111-1111-1111-1111-111111111101', 4, '/device/rules',         'ControlOutlined',       'normal'),
  ('aaaa0010-1000-0000-0000-000000000003', '回收站',   'menu', 'device:recycle',       '11111111-1111-1111-1111-111111111101', 5, '/device/recycle',       'DeleteOutlined',        'normal')
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, permission_key=EXCLUDED.permission_key, parent_id=EXCLUDED.parent_id,
  sort_order=EXCLUDED.sort_order, route_path=EXCLUDED.route_path, icon=EXCLUDED.icon,
  status=EXCLUDED.status, updated_at=NOW();

-- ------------------------------------------------------------
-- 告警管理（5 menus）
-- ------------------------------------------------------------
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status) VALUES
  ('11111111-1111-1111-1111-111111111106', '当前告警',     'menu', 'alarm:current',     '11111111-1111-1111-1111-111111111105', 1, '/alarm/current',       'BellOutlined',          'normal'),
  ('11111111-1111-1111-1111-111111111107', '历史告警',     'menu', 'alarm:history',     '11111111-1111-1111-1111-111111111105', 2, '/alarm/history',       'HistoryOutlined',       'normal'),
  ('aaaa0011-1000-0000-0000-000000000001', '告警规则',     'menu', 'alarm:rules',       '11111111-1111-1111-1111-111111111105', 3, '/alarm/rules',         'FieldNumberOutlined',   'normal'),
  ('aaaa0011-1000-0000-0000-000000000002', '告警库',       'menu', 'alarm:library',     '11111111-1111-1111-1111-111111111105', 4, '/alarm/library',       'BookOutlined',          'normal'),
  ('aaaa0011-1000-0000-0000-000000000003', '自定义告警',   'menu', 'alarm:custom-stats','11111111-1111-1111-1111-111111111105', 5, '/alarm/custom-stats',  'BarChartOutlined',      'normal')
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, permission_key=EXCLUDED.permission_key, parent_id=EXCLUDED.parent_id,
  sort_order=EXCLUDED.sort_order, route_path=EXCLUDED.route_path, icon=EXCLUDED.icon,
  status=EXCLUDED.status, updated_at=NOW();

-- ------------------------------------------------------------
-- 性能管理（3 menus）
-- ------------------------------------------------------------
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status) VALUES
  ('aaaa0002-1000-0000-0000-000000000001', '性能查询', 'menu', 'performance:query',        'aaaa0002-0000-0000-0000-000000000001', 1, '/performance/query',        'SearchOutlined',     'normal'),
  ('aaaa0002-1000-0000-0000-000000000002', '基站KPI',  'menu', 'performance:kpi-station',  'aaaa0002-0000-0000-0000-000000000001', 2, '/performance/kpi-station',  'NodeIndexOutlined',  'normal'),
  ('aaaa0002-1000-0000-0000-000000000003', '标准KPI',  'menu', 'performance:kpi-standard', 'aaaa0002-0000-0000-0000-000000000001', 3, '/performance/kpi-standard', 'AreaChartOutlined',  'normal')
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, permission_key=EXCLUDED.permission_key, parent_id=EXCLUDED.parent_id,
  sort_order=EXCLUDED.sort_order, route_path=EXCLUDED.route_path, icon=EXCLUDED.icon,
  status=EXCLUDED.status, updated_at=NOW();

-- ------------------------------------------------------------
-- MML 管理（3 menus）
-- ------------------------------------------------------------
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status) VALUES
  ('aaaa0003-1000-0000-0000-000000000001', 'MML控制台', 'menu', 'mml:console',      'aaaa0003-0000-0000-0000-000000000001', 1, '/mml/console',      'ConsoleSqlOutlined', 'normal'),
  ('aaaa0003-1000-0000-0000-000000000002', '脚本任务',  'menu', 'mml:script',       'aaaa0003-0000-0000-0000-000000000001', 2, '/mml/script',       'FileTextOutlined',   'normal'),
  ('aaaa0003-1000-0000-0000-000000000003', '任务记录',  'menu', 'mml:task-records', 'aaaa0003-0000-0000-0000-000000000001', 3, '/mml/task-records', 'ScheduleOutlined',   'normal')
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, permission_key=EXCLUDED.permission_key, parent_id=EXCLUDED.parent_id,
  sort_order=EXCLUDED.sort_order, route_path=EXCLUDED.route_path, icon=EXCLUDED.icon,
  status=EXCLUDED.status, updated_at=NOW();

-- ------------------------------------------------------------
-- 拓扑管理（1 menu）
-- ------------------------------------------------------------
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status) VALUES
  ('aaaa0004-1000-0000-0000-000000000001', '拓扑图', 'menu', 'topology:canvas', 'aaaa0004-0000-0000-0000-000000000001', 1, '/topology/canvas', 'PartitionOutlined', 'normal')
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, permission_key=EXCLUDED.permission_key, parent_id=EXCLUDED.parent_id,
  sort_order=EXCLUDED.sort_order, route_path=EXCLUDED.route_path, icon=EXCLUDED.icon,
  status=EXCLUDED.status, updated_at=NOW();

-- ------------------------------------------------------------
-- 备份恢复（3 menus）
-- ------------------------------------------------------------
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status) VALUES
  ('aaaa0005-1000-0000-0000-000000000001', '备份任务', 'menu', 'backup:tasks',    'aaaa0005-0000-0000-0000-000000000001', 1, '/backup/tasks',    'FileDoneOutlined',   'normal'),
  ('aaaa0005-1000-0000-0000-000000000002', '配置文件', 'menu', 'backup:schedule', 'aaaa0005-0000-0000-0000-000000000001', 2, '/backup/schedule', 'CalendarOutlined',   'normal'),
  ('aaaa0005-1000-0000-0000-000000000003', '数据恢复', 'menu', 'backup:restore',  'aaaa0005-0000-0000-0000-000000000001', 3, '/backup/restore',  'RollbackOutlined',   'normal')
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, permission_key=EXCLUDED.permission_key, parent_id=EXCLUDED.parent_id,
  sort_order=EXCLUDED.sort_order, route_path=EXCLUDED.route_path, icon=EXCLUDED.icon,
  status=EXCLUDED.status, updated_at=NOW();

-- ------------------------------------------------------------
-- 软件管理（3 menus）
-- ------------------------------------------------------------
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status) VALUES
  ('aaaa0006-1000-0000-0000-000000000001', '版本升级', 'menu', 'software:upgrade-plan', 'aaaa0006-0000-0000-0000-000000000001', 1, '/software/upgrade-plan', 'UpCircleOutlined',   'normal'),
  ('aaaa0006-1000-0000-0000-000000000002', '升级文件', 'menu', 'software:firmware',     'aaaa0006-0000-0000-0000-000000000001', 2, '/software/firmware',     'FileZipOutlined',    'normal'),
  ('aaaa0006-1000-0000-0000-000000000003', '版本回退', 'menu', 'software:rollback',     'aaaa0006-0000-0000-0000-000000000001', 3, '/software/rollback',     'UndoOutlined',       'normal')
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, permission_key=EXCLUDED.permission_key, parent_id=EXCLUDED.parent_id,
  sort_order=EXCLUDED.sort_order, route_path=EXCLUDED.route_path, icon=EXCLUDED.icon,
  status=EXCLUDED.status, updated_at=NOW();

-- ------------------------------------------------------------
-- 日志管理（3 menus）
-- ------------------------------------------------------------
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status) VALUES
  ('aaaa0007-1000-0000-0000-000000000001', '设备上报日志', 'menu', 'log:device',    'aaaa0007-0000-0000-0000-000000000001', 1, '/log/device',    'FileSearchOutlined',   'normal'),
  ('aaaa0007-1000-0000-0000-000000000002', '设备异常日志', 'menu', 'log:exception', 'aaaa0007-0000-0000-0000-000000000001', 2, '/log/exception', 'WarningOutlined',      'normal'),
  ('aaaa0007-1000-0000-0000-000000000003', '事件日志',     'menu', 'log:event',     'aaaa0007-0000-0000-0000-000000000001', 3, '/log/event',     'ProfileOutlined',      'normal')
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, permission_key=EXCLUDED.permission_key, parent_id=EXCLUDED.parent_id,
  sort_order=EXCLUDED.sort_order, route_path=EXCLUDED.route_path, icon=EXCLUDED.icon,
  status=EXCLUDED.status, updated_at=NOW();

-- ------------------------------------------------------------
-- 系统管理（8 menus）—— 修正 4 条旧菜单的 route_path（旧 /system/user 改 /system/users 等）
-- ------------------------------------------------------------
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status) VALUES
  ('11111111-1111-1111-1111-111111111109', '用户管理', 'menu', 'system:user',           '11111111-1111-1111-1111-111111111108', 1, '/system/users',           'UserOutlined',       'normal'),
  ('11111111-1111-1111-1111-111111111110', '角色管理', 'menu', 'system:role',           '11111111-1111-1111-1111-111111111108', 2, '/system/roles',           'TeamOutlined',       'normal'),
  ('11111111-1111-1111-1111-111111111111', '菜单管理', 'menu', 'system:menu',           '11111111-1111-1111-1111-111111111108', 3, '/system/menus',           'MenuOutlined',       'normal'),
  ('11111111-1111-1111-1111-111111111112', '操作日志', 'menu', 'system:operation-log',  '11111111-1111-1111-1111-111111111108', 4, '/system/operation-log',   'FileTextOutlined',   'normal'),
  ('aaaa0008-1000-0000-0000-000000000001', '系统配置', 'menu', 'system:config',         '11111111-1111-1111-1111-111111111108', 5, '/system/config',          'SettingOutlined',    'normal'),
  ('aaaa0008-1000-0000-0000-000000000002', 'UI定制化', 'menu', 'system:ui-custom',      '11111111-1111-1111-1111-111111111108', 6, '/system/ui-custom',       'BgColorsOutlined',   'normal'),
  ('aaaa0008-1000-0000-0000-000000000003', 'API管理',  'menu', 'system:api-management', '11111111-1111-1111-1111-111111111108', 7, '/system/api-management',  'ApiOutlined',        'normal'),
  ('aaaa0008-1000-0000-0000-000000000004', '字典管理', 'menu', 'system:data-dict',      '11111111-1111-1111-1111-111111111108', 8, '/system/data-dictionary', 'BookOutlined',       'normal')
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, permission_key=EXCLUDED.permission_key, parent_id=EXCLUDED.parent_id,
  sort_order=EXCLUDED.sort_order, route_path=EXCLUDED.route_path, icon=EXCLUDED.icon,
  status=EXCLUDED.status, updated_at=NOW();

-- ------------------------------------------------------------
-- 三级按钮（CRUD 权限点）—— 主要 CRUD 页面
-- 旧 seed 已有：设备列表 6 个按钮 + 用户管理 5 个按钮（保留不动）
-- 新增：角色管理 / 菜单管理 / API管理 / 字典管理 / 告警规则
-- ------------------------------------------------------------

-- 角色管理 buttons
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, status) VALUES
  ('aaaa0008-2110-0000-0000-000000000001', '查询', 'button', 'system:role:query',  '11111111-1111-1111-1111-111111111110', 1, 'normal'),
  ('aaaa0008-2110-0000-0000-000000000002', '添加', 'button', 'system:role:add',    '11111111-1111-1111-1111-111111111110', 2, 'normal'),
  ('aaaa0008-2110-0000-0000-000000000003', '修改', 'button', 'system:role:edit',   '11111111-1111-1111-1111-111111111110', 3, 'normal'),
  ('aaaa0008-2110-0000-0000-000000000004', '删除', 'button', 'system:role:delete', '11111111-1111-1111-1111-111111111110', 4, 'normal')
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, permission_key=EXCLUDED.permission_key, parent_id=EXCLUDED.parent_id,
  sort_order=EXCLUDED.sort_order, status=EXCLUDED.status, updated_at=NOW();

-- 菜单管理 buttons
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, status) VALUES
  ('aaaa0008-2111-0000-0000-000000000001', '查询', 'button', 'system:menu:query',  '11111111-1111-1111-1111-111111111111', 1, 'normal'),
  ('aaaa0008-2111-0000-0000-000000000002', '添加', 'button', 'system:menu:add',    '11111111-1111-1111-1111-111111111111', 2, 'normal'),
  ('aaaa0008-2111-0000-0000-000000000003', '修改', 'button', 'system:menu:edit',   '11111111-1111-1111-1111-111111111111', 3, 'normal'),
  ('aaaa0008-2111-0000-0000-000000000004', '删除', 'button', 'system:menu:delete', '11111111-1111-1111-1111-111111111111', 4, 'normal')
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, permission_key=EXCLUDED.permission_key, parent_id=EXCLUDED.parent_id,
  sort_order=EXCLUDED.sort_order, status=EXCLUDED.status, updated_at=NOW();

-- API管理 buttons
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, status) VALUES
  ('aaaa0008-2003-0000-0000-000000000001', '查询', 'button', 'system:api-management:query',  'aaaa0008-1000-0000-0000-000000000003', 1, 'normal'),
  ('aaaa0008-2003-0000-0000-000000000002', '添加', 'button', 'system:api-management:add',    'aaaa0008-1000-0000-0000-000000000003', 2, 'normal'),
  ('aaaa0008-2003-0000-0000-000000000003', '修改', 'button', 'system:api-management:edit',   'aaaa0008-1000-0000-0000-000000000003', 3, 'normal'),
  ('aaaa0008-2003-0000-0000-000000000004', '删除', 'button', 'system:api-management:delete', 'aaaa0008-1000-0000-0000-000000000003', 4, 'normal'),
  ('aaaa0008-2003-0000-0000-000000000005', '同步', 'button', 'system:api-management:sync',   'aaaa0008-1000-0000-0000-000000000003', 5, 'normal')
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, permission_key=EXCLUDED.permission_key, parent_id=EXCLUDED.parent_id,
  sort_order=EXCLUDED.sort_order, status=EXCLUDED.status, updated_at=NOW();

-- 字典管理 buttons
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, status) VALUES
  ('aaaa0008-2004-0000-0000-000000000001', '查询', 'button', 'system:data-dict:query',  'aaaa0008-1000-0000-0000-000000000004', 1, 'normal'),
  ('aaaa0008-2004-0000-0000-000000000002', '添加', 'button', 'system:data-dict:add',    'aaaa0008-1000-0000-0000-000000000004', 2, 'normal'),
  ('aaaa0008-2004-0000-0000-000000000003', '修改', 'button', 'system:data-dict:edit',   'aaaa0008-1000-0000-0000-000000000004', 3, 'normal'),
  ('aaaa0008-2004-0000-0000-000000000004', '删除', 'button', 'system:data-dict:delete', 'aaaa0008-1000-0000-0000-000000000004', 4, 'normal')
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, permission_key=EXCLUDED.permission_key, parent_id=EXCLUDED.parent_id,
  sort_order=EXCLUDED.sort_order, status=EXCLUDED.status, updated_at=NOW();

-- 告警规则 buttons
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, status) VALUES
  ('aaaa0011-2000-0000-0000-000000000001', '查询', 'button', 'alarm:rules:query',  'aaaa0011-1000-0000-0000-000000000001', 1, 'normal'),
  ('aaaa0011-2000-0000-0000-000000000002', '添加', 'button', 'alarm:rules:add',    'aaaa0011-1000-0000-0000-000000000001', 2, 'normal'),
  ('aaaa0011-2000-0000-0000-000000000003', '修改', 'button', 'alarm:rules:edit',   'aaaa0011-1000-0000-0000-000000000001', 3, 'normal'),
  ('aaaa0011-2000-0000-0000-000000000004', '删除', 'button', 'alarm:rules:delete', 'aaaa0011-1000-0000-0000-000000000001', 4, 'normal')
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, permission_key=EXCLUDED.permission_key, parent_id=EXCLUDED.parent_id,
  sort_order=EXCLUDED.sort_order, status=EXCLUDED.status, updated_at=NOW();


-- +goose Down
-- ============================================================
-- 仅删除本迁移新增的菜单（aaaa....-... 命名空间）；旧 1111... 行不动。
-- ============================================================
DELETE FROM menus WHERE id::text LIKE 'aaaa____-____-____-____-____________';
