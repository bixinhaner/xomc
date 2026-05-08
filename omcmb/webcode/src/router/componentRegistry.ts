// componentRegistry — 菜单动态加载 P2 起的组件路径白名单。
//
// 设计依据：docs/prd/system/menu-dynamic-loading.md §4.3.4。
//
// 用途：
//   1. 后端 menus.component_path 字段的值必须命中本注册表的某个 key，否则该菜单
//      在路由层会被忽略 + 控制台告警（避免类型不安全的字符串 import）。
//   2. P3 RolePermission / 调试工具可读 listComponentPaths() 校验数据完整性。
//   3. 未来 buildRouter 全动态路由时直接消费本注册表的 ComponentType。
//
// 维护规则：
//   - 与 routes.tsx 的 lazy import **严格 1:1 对齐**。新增页面时同时改两边。
//   - key 命名约定：'<feature>/<ComponentName>'，对应 @/pages 下的相对路径。
//   - 不允许把字符串拼成 import() —— vite 不支持完全动态字符串 import 且无类型安全。

import { lazy, type ComponentType } from 'react';

/** 入口路径 → 懒加载组件。 */
export const componentRegistry: Record<string, ComponentType> = {
  // Dashboard
  'dashboard/Dashboard': lazy(() => import('@/pages/dashboard')),

  // Device Management
  'device/DeviceList': lazy(() => import('@/pages/device/DeviceList')),
  'device/DeviceRegistration': lazy(() => import('@/pages/device/DeviceRegistration')),
  'device/DeviceGrouping': lazy(() => import('@/pages/device/DeviceGrouping')),
  'device/DeviceDetail': lazy(() => import('@/pages/device/DeviceDetail')),
  'device/NEManagement': lazy(() => import('@/pages/device/NEManagement')),
  'device/OnlineMonitoring': lazy(() => import('@/pages/device/OnlineMonitoring')),
  'device/Commissioning': lazy(() => import('@/pages/device/Commissioning')),
  'device/HandoverManagement': lazy(() => import('@/pages/device/HandoverManagement')),
  'device/ResourceStatistics': lazy(() => import('@/pages/device/ResourceStatistics')),
  'device/ImportExport': lazy(() => import('@/pages/device/ImportExport')),
  'device/DeviceRules': lazy(() => import('@/pages/device/DeviceRules')),
  'device/RecycleBin': lazy(() => import('@/pages/device/RecycleBin')),
  'device/UeDetail': lazy(() => import('@/pages/device/UeDetail')),
  'device/PlugAndPlay': lazy(() => import('@/pages/device/PlugAndPlay')),
  'device/PlugAndPlay/AddPolicyPage': lazy(
    () => import('@/pages/device/PlugAndPlay/AddPolicyPage'),
  ),

  // Alarm Management
  'alarm/CurrentAlarms': lazy(() => import('@/pages/alarm/CurrentAlarms')),
  'alarm/HistoricalAlarms': lazy(() => import('@/pages/alarm/HistoricalAlarms')),
  'alarm/AlarmStatistics': lazy(() => import('@/pages/alarm/AlarmStatistics')),
  'alarm/AlarmRules': lazy(() => import('@/pages/alarm/AlarmRules')),
  'alarm/AlarmSupportLibrary': lazy(() => import('@/pages/alarm/AlarmSupportLibrary')),
  'alarm/AlarmSync': lazy(() => import('@/pages/alarm/AlarmSync')),
  'alarm/CustomAlarmStats': lazy(() => import('@/pages/alarm/CustomAlarmStats')),

  // Configuration Management
  'config/ParamSync': lazy(() => import('@/pages/config/ParamSync')),
  'config/LiveParamConfig': lazy(() => import('@/pages/config/LiveParamConfig')),
  'config/BatchParamClass': lazy(() => import('@/pages/config/BatchParamClass')),
  'config/BatchParamTemplate': lazy(() => import('@/pages/config/BatchParamTemplate')),
  'config/ParamList': lazy(() => import('@/pages/config/ParamList')),
  'config/CommandMode': lazy(() => import('@/pages/config/CommandMode')),
  'config/CellManagement': lazy(() => import('@/pages/config/CellManagement')),
  'config/BaselineManagement': lazy(() => import('@/pages/config/BaselineManagement')),
  'config/CommonConfig': lazy(() => import('@/pages/config/CommonConfig')),
  'config/NeighborParams': lazy(() => import('@/pages/config/NeighborParams')),
  'config/DataModelManagement': lazy(() => import('@/pages/config/DataModelManagement')),
  'config/NorthboundManagement': lazy(() => import('@/pages/config/NorthboundManagement')),
  'config/AutoProvisioning': lazy(() => import('@/pages/config/AutoProvisioning')),
  'config/InteropTesting': lazy(() => import('@/pages/config/InteropTesting')),

  // Performance Management
  'performance/KPIStandardReport': lazy(() => import('@/pages/performance/KPIStandardReport')),
  'performance/KPIStandardReport/IndicatorDetail': lazy(
    () => import('@/pages/performance/KPIStandardReport/IndicatorDetail'),
  ),
  'performance/KPIStationReport': lazy(() => import('@/pages/performance/KPIStationReport')),
  'performance/KPIQuery': lazy(() => import('@/pages/performance/KPIQuery')),
  'performance/PerformanceCharts': lazy(() => import('@/pages/performance/PerformanceCharts')),
  'performance/ThresholdConfig': lazy(() => import('@/pages/performance/ThresholdConfig')),
  'performance/PerformanceFiles': lazy(() => import('@/pages/performance/PerformanceFiles')),
  'performance/PerformanceTaskConfig': lazy(
    () => import('@/pages/performance/PerformanceTaskConfig'),
  ),

  // MML Management
  'mml/Console': lazy(() => import('@/pages/mml/Console')),
  'mml/ScriptTask': lazy(() => import('@/pages/mml/ScriptTask')),
  'mml/TaskRecord': lazy(() => import('@/pages/mml/TaskRecord')),
  'mml/CommandTree': lazy(() => import('@/pages/mml/CommandTree')),

  // Topology Management
  'topology/GISMapView': lazy(() => import('@/pages/topology/GISMapView')),
  'topology/TopologyCanvas': lazy(() => import('@/pages/topology/TopologyCanvas')),
  'topology/DomainManagement': lazy(() => import('@/pages/topology/DomainManagement')),
  'topology/SiteManagement': lazy(() => import('@/pages/topology/SiteManagement')),
  'topology/TopologySettings': lazy(() => import('@/pages/topology/TopologySettings')),
  'topology/LegendSystem': lazy(() => import('@/pages/topology/LegendSystem')),

  // Backup & Restore
  'backup/BackupTasks': lazy(() => import('@/pages/backup/BackupTasks')),
  'backup/BackupSchedule': lazy(() => import('@/pages/backup/BackupSchedule')),
  'backup/FTPConfig': lazy(() => import('@/pages/backup/FTPConfig')),
  'backup/RestoreData': lazy(() => import('@/pages/backup/RestoreData')),
  'backup/BackupPolicy': lazy(() => import('@/pages/backup/BackupPolicy')),

  // Software Management
  'software/VersionQuery': lazy(() => import('@/pages/software/VersionQuery')),
  'software/UpgradePlan': lazy(() => import('@/pages/software/UpgradePlan')),
  'software/ActivationPlan': lazy(() => import('@/pages/software/ActivationPlan')),
  'software/FirmwareUpload': lazy(() => import('@/pages/software/FirmwareUpload')),
  'software/VersionRollback': lazy(() => import('@/pages/software/VersionRollback')),

  // File Management
  'file/ConfigRetrieval': lazy(() => import('@/pages/file/ConfigRetrieval')),
  'file/ConfigDistribution': lazy(() => import('@/pages/file/ConfigDistribution')),
  'file/LogRetrieval': lazy(() => import('@/pages/file/LogRetrieval')),
  'file/PerfRetrieval': lazy(() => import('@/pages/file/PerfRetrieval')),
  'file/MRRetrieval': lazy(() => import('@/pages/file/MRRetrieval')),
  'file/UserFiles': lazy(() => import('@/pages/file/UserFiles')),
  'file/DeviceFiles': lazy(() => import('@/pages/file/DeviceFiles')),

  // Log Management
  'log/DeviceLog': lazy(() => import('@/pages/log/DeviceLog')),
  'log/ExceptionLog': lazy(() => import('@/pages/log/ExceptionLog')),
  'log/EventLog': lazy(() => import('@/pages/log/EventLog')),
  'log/OperationLog': lazy(() => import('@/pages/log/OperationLog')),
  'log/SystemLog': lazy(() => import('@/pages/log/SystemLog')),
  'log/LogConfig': lazy(() => import('@/pages/log/LogConfig')),

  // System Management
  'system/DeviceClassification': lazy(() => import('@/pages/system/DeviceClassification')),
  'system/UserManagement': lazy(() => import('@/pages/system/UserManagement')),
  'system/GroupManagement': lazy(() => import('@/pages/system/GroupManagement')),
  'system/RolePermission': lazy(() => import('@/pages/system/RolePermission')),
  'system/OperationLog': lazy(() => import('@/pages/system/OperationLog')),
  'system/SystemConfig': lazy(() => import('@/pages/system/SystemConfig')),
  'system/UICustomization': lazy(() => import('@/pages/system/UICustomization')),
  'system/MenuManagement': lazy(() => import('@/pages/system/MenuManagement')),
  'system/SystemDashboard': lazy(() => import('@/pages/system/SystemDashboard')),
  'system/ApiManagement': lazy(() => import('@/pages/system/ApiManagement')),
  'system/DataDictionary': lazy(() => import('@/pages/system/DataDictionary')),

  // Report Management
  'report/LTEStandardReport': lazy(() => import('@/pages/report/LTEStandardReport')),
  'report/StationReport': lazy(() => import('@/pages/report/StationReport')),
  'report/HistoricalKPI': lazy(() => import('@/pages/report/HistoricalKPI')),
  'report/PollStatistics': lazy(() => import('@/pages/report/PollStatistics')),

  // MR Management
  'mr/Indicators': lazy(() => import('@/pages/mr/Indicators')),
  'mr/DeviceMapping': lazy(() => import('@/pages/mr/DeviceMapping')),
  'mr/Variables': lazy(() => import('@/pages/mr/Variables')),
  'mr/Reports': lazy(() => import('@/pages/mr/Reports')),
  'mr/Tasks': lazy(() => import('@/pages/mr/Tasks')),
  'mr/Files': lazy(() => import('@/pages/mr/Files')),

  // License Management
  'license/LicenseList': lazy(() => import('@/pages/license/LicenseList')),
  'license/LicenseOperations': lazy(() => import('@/pages/license/LicenseOperations')),
  'license/LicenseLogs': lazy(() => import('@/pages/license/LicenseLogs')),

  // Notifications
  'notifications/Index': lazy(() => import('@/pages/notifications')),

  // Ops Management
  'ops/Templates': lazy(() => import('@/pages/ops/Templates')),
  'ops/CommandManagement': lazy(() => import('@/pages/ops/CommandManagement')),
  'ops/TaskManagement': lazy(() => import('@/pages/ops/TaskManagement')),
  'ops/NetworkDiagnosis': lazy(() => import('@/pages/ops/NetworkDiagnosis')),
  'ops/Downloads': lazy(() => import('@/pages/ops/Downloads')),
};

/** 是否注册了某个 component_path（菜单创建/校验工具用）。 */
export function hasComponentPath(path: string | undefined | null): boolean {
  if (!path) return false;
  return path in componentRegistry;
}

/** 全部已注册 component_path 列表（调试 / 数据校验工具用）。 */
export function listComponentPaths(): string[] {
  return Object.keys(componentRegistry);
}
