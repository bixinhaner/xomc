-- 菜单动态加载 P2：回填 menus.component_path 字段。
--
-- 背景：commit cca8bc59 (B3-Phase1) / fd507e5d (B3-Phase2-A) 已让权限模型迁移到端点级；
-- 前端 P2 (commit 待提交) 引入 componentRegistry，菜单动态路由需要 menus.component_path 一致。
-- 旧 seed (000057_refresh_menu_seed.sql) 写菜单时 component_path 留空。
--
-- 映射来源：omcmb/webcode/src/router/componentRegistry.ts。键命名约定 '<feature>/<ComponentName>'。
-- 不在本次回填范围：菜单 seed 缺失但 routes.tsx 有路由的页面（如 /performance/charts、
--   /report/*、/mr/*、/license/*、/notifications、/ops/*、/file/*）—— 这些页面后续菜单 seed
--   补齐时由那次 seed 直接写入 component_path 字段，不需要事后 UPDATE。
--
-- 关联 PRD：docs/prd/system/menu-dynamic-loading.md §4.2.5 / §4.3.4。

-- +goose Up
UPDATE menus SET component_path = 'dashboard/Dashboard'              WHERE route_path = '/dashboard';

-- 设备管理（5）
UPDATE menus SET component_path = 'device/DeviceList'                WHERE route_path = '/device/list';
UPDATE menus SET component_path = 'device/DeviceGrouping'            WHERE route_path = '/device/group';
UPDATE menus SET component_path = 'device/PlugAndPlay'               WHERE route_path = '/device/plug-and-play';
UPDATE menus SET component_path = 'device/DeviceRules'               WHERE route_path = '/device/rules';
UPDATE menus SET component_path = 'device/RecycleBin'                WHERE route_path = '/device/recycle';

-- 告警管理（5）
UPDATE menus SET component_path = 'alarm/CurrentAlarms'              WHERE route_path = '/alarm/current';
UPDATE menus SET component_path = 'alarm/HistoricalAlarms'           WHERE route_path = '/alarm/history';
UPDATE menus SET component_path = 'alarm/AlarmRules'                 WHERE route_path = '/alarm/rules';
UPDATE menus SET component_path = 'alarm/AlarmSupportLibrary'        WHERE route_path = '/alarm/library';
UPDATE menus SET component_path = 'alarm/CustomAlarmStats'           WHERE route_path = '/alarm/custom-stats';

-- 性能管理（3）
UPDATE menus SET component_path = 'performance/KPIStandardReport'    WHERE route_path = '/performance/kpi-standard';
UPDATE menus SET component_path = 'performance/KPIStationReport'     WHERE route_path = '/performance/kpi-station';
UPDATE menus SET component_path = 'performance/KPIQuery'             WHERE route_path = '/performance/query';

-- MML 管理（3）
UPDATE menus SET component_path = 'mml/Console'                      WHERE route_path = '/mml/console';
UPDATE menus SET component_path = 'mml/ScriptTask'                   WHERE route_path = '/mml/script';
UPDATE menus SET component_path = 'mml/TaskRecord'                   WHERE route_path = '/mml/task-records';

-- 拓扑管理（1）
UPDATE menus SET component_path = 'topology/TopologyCanvas'          WHERE route_path = '/topology/canvas';

-- 备份管理（3）
UPDATE menus SET component_path = 'backup/BackupTasks'               WHERE route_path = '/backup/tasks';
UPDATE menus SET component_path = 'backup/BackupSchedule'            WHERE route_path = '/backup/schedule';
UPDATE menus SET component_path = 'backup/RestoreData'               WHERE route_path = '/backup/restore';

-- 软件管理（3）
UPDATE menus SET component_path = 'software/UpgradePlan'             WHERE route_path = '/software/upgrade-plan';
UPDATE menus SET component_path = 'software/FirmwareUpload'          WHERE route_path = '/software/firmware';
UPDATE menus SET component_path = 'software/VersionRollback'         WHERE route_path = '/software/rollback';

-- 日志管理（3）
UPDATE menus SET component_path = 'log/DeviceLog'                    WHERE route_path = '/log/device';
UPDATE menus SET component_path = 'log/ExceptionLog'                 WHERE route_path = '/log/exception';
UPDATE menus SET component_path = 'log/EventLog'                     WHERE route_path = '/log/event';

-- 系统管理（8）
UPDATE menus SET component_path = 'system/UserManagement'            WHERE route_path = '/system/users';
UPDATE menus SET component_path = 'system/RolePermission'            WHERE route_path = '/system/roles';
UPDATE menus SET component_path = 'system/MenuManagement'            WHERE route_path = '/system/menus';
UPDATE menus SET component_path = 'system/OperationLog'              WHERE route_path = '/system/operation-log';
UPDATE menus SET component_path = 'system/SystemConfig'              WHERE route_path = '/system/config';
UPDATE menus SET component_path = 'system/UICustomization'           WHERE route_path = '/system/ui-custom';
UPDATE menus SET component_path = 'system/ApiManagement'             WHERE route_path = '/system/api-management';
UPDATE menus SET component_path = 'system/DataDictionary'            WHERE route_path = '/system/data-dictionary';

-- +goose Down
-- 回滚：把上面回填过的 menus.component_path 全部清空。
-- 与 Up 段路径列表一致；新增/缩减时需同步两段。
UPDATE menus
SET component_path = ''
WHERE route_path IN (
  '/dashboard',
  '/device/list', '/device/group', '/device/plug-and-play', '/device/rules', '/device/recycle',
  '/alarm/current', '/alarm/history', '/alarm/rules', '/alarm/library', '/alarm/custom-stats',
  '/performance/kpi-standard', '/performance/kpi-station', '/performance/query',
  '/mml/console', '/mml/script', '/mml/task-records',
  '/topology/canvas',
  '/backup/tasks', '/backup/schedule', '/backup/restore',
  '/software/upgrade-plan', '/software/firmware', '/software/rollback',
  '/log/device', '/log/exception', '/log/event',
  '/system/users', '/system/roles', '/system/menus', '/system/operation-log',
  '/system/config', '/system/ui-custom', '/system/api-management', '/system/data-dictionary'
);
