import { lazy, Suspense, type ComponentType, type ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import {
  AlertTriangle, ArrowLeftRight, Boxes, Cpu, FolderOpen, Globe, KeySquare, LayoutDashboard, LineChart, Package, Radio, ScrollText, Settings, Sliders, Terminal, Wrench,
} from 'lucide-react'

// ---------------------------------------------------------------------------
// 皮肤路由清单（webcode-v3 / STARFORGE HUD）
//
// 唯一标准 = v1（webcode）。本皮肤声明的可路由 path 集合、非 hidden（可见菜单）集合
// 必须与 v1 完全一致，仅外观不同。守卫：node omcmb/scripts/skin-parity.mjs。
//
//   - 路径方案对齐 v1：/device、/alarm、/log、/report、/file（v3 原 /fleet /alarms
//     /logs /reports /files /bridge 已并入 v1 方案）。
//   - 组件复用 v3 现有页面（fleet/* 承担 device/* 职责；bridge 承担 dashboard；
//     拆分的库页 alarm-library/kpi-library/param-model/standard-params/orphan-devices/
//     products 并回 v1 的 /product/* 路径）。
//   - 可见性：path 在 v1 可见集合内 → 不带 hidden；其余 routable 但 hidden:true。
// ---------------------------------------------------------------------------

// Dashboard（v3 bridge 指挥页承担 v1 dashboard 职责）
const DashboardPage = lazy(() => import('@/pages/bridge').then((m) => ({ default: m.BridgePage })))

// Device（v3 fleet/* 承担 v1 device/* 职责）
const DeviceList = lazy(() => import('@/pages/fleet').then((m) => ({ default: m.FleetPage })))
const C_pages_fleet_AbnormalReboot = lazy(() => import('@/pages/fleet/AbnormalReboot'))
const C_pages_fleet_Commissioning = lazy(() => import('@/pages/fleet/Commissioning'))
const C_pages_fleet_DeviceDetail = lazy(() => import('@/pages/fleet/DeviceDetail'))
const C_pages_fleet_DeviceGroup = lazy(() => import('@/pages/fleet/DeviceGroup'))
const C_pages_fleet_DeviceRegister = lazy(() => import('@/pages/fleet/DeviceRegister'))
const C_pages_fleet_HandoverManagement = lazy(() => import('@/pages/fleet/HandoverManagement'))
const C_pages_fleet_ImportExport = lazy(() => import('@/pages/fleet/ImportExport'))
const C_pages_fleet_NEManagement = lazy(() => import('@/pages/fleet/NEManagement'))
const C_pages_fleet_OnlineMonitor = lazy(() => import('@/pages/fleet/OnlineMonitor'))
const C_pages_fleet_PlugAndPlay = lazy(() => import('@/pages/fleet/PlugAndPlay'))
const C_pages_fleet_PlugAndPlayPolicy = lazy(() => import('@/pages/fleet/PlugAndPlayPolicy'))
const C_pages_fleet_RecycleBin = lazy(() => import('@/pages/fleet/RecycleBin'))
const C_pages_fleet_ResourceStats = lazy(() => import('@/pages/fleet/ResourceStats'))
const C_pages_fleet_UeDetail = lazy(() => import('@/pages/fleet/UeDetail'))

// Alarm
const C_pages_alarms_CurrentAlarms = lazy(() => import('@/pages/alarms').then((m) => ({ default: m.AlarmsPage })))
const C_pages_alarms_AlarmRules = lazy(() => import('@/pages/alarms/AlarmRules'))
const C_pages_alarms_AlarmStatistics = lazy(() => import('@/pages/alarms/AlarmStatistics'))
const C_pages_alarms_AlarmSync = lazy(() => import('@/pages/alarms/AlarmSync'))
const C_pages_alarms_CustomAlarmStats = lazy(() => import('@/pages/alarms/CustomAlarmStats'))
const C_pages_alarms_HistoricalAlarms = lazy(() => import('@/pages/alarms/HistoricalAlarms'))

// Config
const C_pages_config_AutoProvisioning = lazy(() => import('@/pages/config/AutoProvisioning'))
const C_pages_config_BaselineManagement = lazy(() => import('@/pages/config/BaselineManagement'))
const C_pages_config_BatchParamClass = lazy(() => import('@/pages/config/BatchParamClass'))
const C_pages_config_BatchParamTemplate = lazy(() => import('@/pages/config/BatchParamTemplate'))
const C_pages_config_CellManagement = lazy(() => import('@/pages/config/CellManagement'))
const C_pages_config_CommandMode = lazy(() => import('@/pages/config/CommandMode'))
const C_pages_config_CommonConfig = lazy(() => import('@/pages/config/CommonConfig'))
const C_pages_config_InteropTesting = lazy(() => import('@/pages/config/InteropTesting'))
const C_pages_config_LiveParamConfig = lazy(() => import('@/pages/config/LiveParamConfig'))
const C_pages_config_NeighborParams = lazy(() => import('@/pages/config/NeighborParams'))
const C_pages_config_NorthboundManagement = lazy(() => import('@/pages/config/NorthboundManagement'))
const C_pages_config_ParamList = lazy(() => import('@/pages/config/ParamList'))
const C_pages_config_ParamSync = lazy(() => import('@/pages/config/ParamSync'))

// Performance
const PerformancePage = lazy(() => import('@/pages/performance').then((m) => ({ default: m.PerformancePage })))
const C_pages_performance_DeviceView = lazy(() => import('@/pages/performance/DeviceView'))
const C_pages_performance_KpiStandard = lazy(() => import('@/pages/performance/KpiStandard'))
const C_pages_performance_KpiStandardDetail = lazy(() => import('@/pages/performance/KpiStandardDetail'))
const C_pages_performance_KpiStation = lazy(() => import('@/pages/performance/KpiStation'))
const C_pages_performance_PerformanceCharts = lazy(() => import('@/pages/performance/PerformanceCharts'))
const C_pages_performance_PerformanceFiles = lazy(() => import('@/pages/performance/PerformanceFiles'))
const C_pages_performance_PerformanceQuery = lazy(() => import('@/pages/performance/PerformanceQuery'))
const C_pages_performance_PerformanceTaskConfig = lazy(() => import('@/pages/performance/PerformanceTaskConfig'))
const C_pages_performance_PmAdhoc = lazy(() => import('@/pages/performance/PmAdhoc'))
const C_pages_performance_PmAdhocWizard = lazy(() => import('@/pages/performance/PmAdhocWizard'))
const C_pages_performance_ThresholdConfig = lazy(() => import('@/pages/performance/ThresholdConfig'))

// MML
const C_pages_mml_admin_catalog_index = lazy(() => import('@/pages/mml/admin-catalog/index'))
const C_pages_mml_commands_index = lazy(() => import('@/pages/mml/commands/index'))
const C_pages_mml_console_v2_index = lazy(() => import('@/pages/mml/console-v2/index'))
const C_pages_mml_private_command_index = lazy(() => import('@/pages/mml/private-command/index'))
const C_pages_mml_script_index = lazy(() => import('@/pages/mml/script/index'))
const C_pages_mml_task_records_index = lazy(() => import('@/pages/mml/task-records/index'))

// Topology
const C_pages_topology_DomainManagement = lazy(() => import('@/pages/topology/DomainManagement'))
const C_pages_topology_GISMapView = lazy(() => import('@/pages/topology/GISMapView'))
const C_pages_topology_LegendSystem = lazy(() => import('@/pages/topology/LegendSystem'))
const C_pages_topology_SiteManagement = lazy(() => import('@/pages/topology/SiteManagement'))
const C_pages_topology_TopologyCanvas = lazy(() => import('@/pages/topology/TopologyCanvas'))
const C_pages_topology_TopologySettings = lazy(() => import('@/pages/topology/TopologySettings'))

// Transfer
const C_pages_transfer_Center = lazy(() => import('@/pages/transfer/Center'))
const C_pages_transfer_FileManagement = lazy(() => import('@/pages/transfer/FileManagement'))
const C_pages_transfer_TemplateManagement = lazy(() => import('@/pages/transfer/TemplateManagement'))

// Backup
const C_pages_backup_ConfigSnapshots = lazy(() => import('@/pages/backup/ConfigSnapshots'))
const C_pages_backup_Ftp = lazy(() => import('@/pages/backup/Ftp'))
const C_pages_backup_Policy = lazy(() => import('@/pages/backup/Policy'))
const C_pages_backup_Restore = lazy(() => import('@/pages/backup/Restore'))
const C_pages_backup_Schedule = lazy(() => import('@/pages/backup/Schedule'))
const C_pages_backup_Tasks = lazy(() => import('@/pages/backup/Tasks'))

// Software
const C_pages_software_Activation = lazy(() => import('@/pages/software/Activation'))
const C_pages_software_Firmware = lazy(() => import('@/pages/software/Firmware'))
const C_pages_software_Rollback = lazy(() => import('@/pages/software/Rollback'))
const C_pages_software_UpgradePlan = lazy(() => import('@/pages/software/UpgradePlan'))
const C_pages_software_Version = lazy(() => import('@/pages/software/Version'))

// File（v3 files/* 承担 v1 file/* 职责）
const C_pages_files_ConfigDistribution = lazy(() => import('@/pages/files/ConfigDistribution'))
const C_pages_files_ConfigRetrieval = lazy(() => import('@/pages/files/ConfigRetrieval'))
const C_pages_files_DeviceFiles = lazy(() => import('@/pages/files/DeviceFiles'))
const C_pages_files_LogRetrieval = lazy(() => import('@/pages/files/LogRetrieval'))
const C_pages_files_MRRetrieval = lazy(() => import('@/pages/files/MRRetrieval'))
const C_pages_files_PerfRetrieval = lazy(() => import('@/pages/files/PerfRetrieval'))
const C_pages_files_UserFiles = lazy(() => import('@/pages/files/UserFiles'))

// Log（v3 logs/* 承担 v1 log/* 职责）
const C_pages_logs_DeviceLog = lazy(() => import('@/pages/logs/DeviceLog'))
const C_pages_logs_EventLog = lazy(() => import('@/pages/logs/EventLog'))
const C_pages_logs_ExceptionLog = lazy(() => import('@/pages/logs/ExceptionLog'))
const C_pages_logs_LogConfig = lazy(() => import('@/pages/logs/LogConfig'))
const C_pages_logs_OperationLog = lazy(() => import('@/pages/logs/OperationLog'))
const C_pages_logs_SystemLog = lazy(() => import('@/pages/logs/SystemLog'))

// MR
const C_pages_mr_DeviceMapping = lazy(() => import('@/pages/mr/DeviceMapping'))
const C_pages_mr_Files = lazy(() => import('@/pages/mr/Files'))
const C_pages_mr_Indicators = lazy(() => import('@/pages/mr/Indicators'))
const C_pages_mr_Reports = lazy(() => import('@/pages/mr/Reports'))
const C_pages_mr_Variables = lazy(() => import('@/pages/mr/Variables'))

// Report（v3 reports/* 承担 v1 report/* 职责）
const C_pages_reports_HistoricalKpi = lazy(() => import('@/pages/reports/HistoricalKpi'))
const C_pages_reports_LteStandard = lazy(() => import('@/pages/reports/LteStandard'))
const C_pages_reports_PollStatistics = lazy(() => import('@/pages/reports/PollStatistics'))
const C_pages_reports_Station = lazy(() => import('@/pages/reports/Station'))

// Ops
const C_pages_ops_AggregationTrigger = lazy(() => import('@/pages/ops/AggregationTrigger'))
const C_pages_ops_CommandManagement = lazy(() => import('@/pages/ops/CommandManagement'))
const C_pages_ops_Downloads = lazy(() => import('@/pages/ops/Downloads'))
const C_pages_ops_MessageTrace = lazy(() => import('@/pages/ops/MessageTrace'))
const C_pages_ops_NetworkDiagnosis = lazy(() => import('@/pages/ops/NetworkDiagnosis'))
const C_pages_ops_TaskManagement = lazy(() => import('@/pages/ops/TaskManagement'))
const C_pages_ops_Templates = lazy(() => import('@/pages/ops/Templates'))

// Product（v3 拆分的库页并回 v1 /product/* 路径）
const C_pages_alarm_library_index = lazy(() => import('@/pages/alarm-library/index'))
const C_pages_kpi_library_index = lazy(() => import('@/pages/kpi-library/index'))
const C_pages_orphan_devices_index = lazy(() => import('@/pages/orphan-devices/index'))
const C_pages_param_model_index = lazy(() => import('@/pages/param-model/index'))
const C_pages_products_index = lazy(() => import('@/pages/products/index'))
const C_pages_standard_params_index = lazy(() => import('@/pages/standard-params/index'))

// License
const LicensePage = lazy(() => import('@/pages/license').then((m) => ({ default: m.LicensePage })))
const C_pages_license_History = lazy(() => import('@/pages/license/History'))

// Notifications
const C_pages_notifications_NotificationCenter = lazy(() => import('@/pages/notifications/NotificationCenter'))

// System
const C_pages_system_ApiManagement = lazy(() => import('@/pages/system/ApiManagement'))
const C_pages_system_DataDictionary = lazy(() => import('@/pages/system/DataDictionary'))
const C_pages_system_DeviceClassification = lazy(() => import('@/pages/system/DeviceClassification'))
const C_pages_system_DictLoader = lazy(() => import('@/pages/system/DictLoader'))
const C_pages_system_GroupManagement = lazy(() => import('@/pages/system/GroupManagement'))
const C_pages_system_KpiConfig = lazy(() => import('@/pages/system/KpiConfig'))
const C_pages_system_MenuManagement = lazy(() => import('@/pages/system/MenuManagement'))
const C_pages_system_OperationLog = lazy(() => import('@/pages/system/OperationLog'))
const C_pages_system_RolePermission = lazy(() => import('@/pages/system/RolePermission'))
const C_pages_system_SystemConfig = lazy(() => import('@/pages/system/SystemConfig'))
const C_pages_system_SystemDashboard = lazy(() => import('@/pages/system/SystemDashboard'))
const C_pages_system_UICustomization = lazy(() => import('@/pages/system/UICustomization'))
const C_pages_system_UserManagement = lazy(() => import('@/pages/system/UserManagement'))

const s = (C: ComponentType): ReactNode => (
  <Suspense fallback={<div className="p-8 font-mono text-xs text-cyan-300/60">LOADING…</div>}>
    <C />
  </Suspense>
)

export type RouteDef = { path: string; element: ReactNode; label: string; hidden?: boolean }
export type ModuleDef = { key: string; label: string; icon: ReactNode; section: string; routes: RouteDef[] }

// 数据驱动路由+导航唯一事实源（path 集合与可见集合对齐 v1，守卫 skin-parity.mjs）。
export const MODULES: ModuleDef[] = [
  { key: 'dashboard', label: 'DASH 仪表盘', icon: <LayoutDashboard />, section: '监控', routes: [
    { path: '/dashboard', element: s(DashboardPage), label: "DASH 仪表盘" },
  ] },
  { key: 'device', label: 'FLEET 设备', icon: <Cpu />, section: '监控', routes: [
    { path: '/device/list', element: s(DeviceList), label: "设备列表" },
    { path: '/device/group', element: s(C_pages_fleet_DeviceGroup), label: "设备分组" },
    { path: '/device/plug-and-play', element: s(C_pages_fleet_PlugAndPlay), label: "即插即用" },
    { path: '/device/recycle', element: s(C_pages_fleet_RecycleBin), label: "回收站" },
    { path: '/device/abnormal-reboot', element: s(C_pages_fleet_AbnormalReboot), label: "异常重启" },
    { path: '/device/register', element: s(C_pages_fleet_DeviceRegister), label: "设备注册", hidden: true },
    { path: '/device/ne', element: s(C_pages_fleet_NEManagement), label: "网元管理", hidden: true },
    { path: '/device/monitor', element: s(C_pages_fleet_OnlineMonitor), label: "在线监控", hidden: true },
    { path: '/device/commission', element: s(C_pages_fleet_Commissioning), label: "开通调测", hidden: true },
    { path: '/device/handover', element: s(C_pages_fleet_HandoverManagement), label: "割接管理", hidden: true },
    { path: '/device/stats', element: s(C_pages_fleet_ResourceStats), label: "资源统计", hidden: true },
    { path: '/device/import', element: s(C_pages_fleet_ImportExport), label: "导入导出", hidden: true },
    { path: '/device/detail/:sn', element: s(C_pages_fleet_DeviceDetail), label: "设备详情", hidden: true },
    { path: '/device/ue-detail/:sn', element: s(C_pages_fleet_UeDetail), label: "UE 详情", hidden: true },
    { path: '/device/plug-and-play/add', element: s(C_pages_fleet_PlugAndPlayPolicy), label: "新建策略", hidden: true },
    { path: '/device/plug-and-play/edit/:id', element: s(C_pages_fleet_PlugAndPlayPolicy), label: "编辑策略", hidden: true },
    { path: '/device/plug-and-play/view/:id', element: s(C_pages_fleet_PlugAndPlayPolicy), label: "查看策略", hidden: true },
  ] },
  { key: 'alarm', label: 'ALARM 告警', icon: <AlertTriangle />, section: '监控', routes: [
    { path: '/alarm/current', element: s(C_pages_alarms_CurrentAlarms), label: "当前告警" },
    { path: '/alarm/history', element: s(C_pages_alarms_HistoricalAlarms), label: "历史告警" },
    { path: '/alarm/rules', element: s(C_pages_alarms_AlarmRules), label: "告警规则" },
    { path: '/alarm/statistics', element: s(C_pages_alarms_AlarmStatistics), label: "告警统计", hidden: true },
    { path: '/alarm/sync', element: s(C_pages_alarms_AlarmSync), label: "告警同步", hidden: true },
    { path: '/alarm/custom-stats', element: s(C_pages_alarms_CustomAlarmStats), label: "自定义告警统计", hidden: true },
  ] },
  { key: 'notifications', label: 'NOTIFY 通知', icon: <Globe />, section: '监控', routes: [
    { path: '/notifications', element: s(C_pages_notifications_NotificationCenter), label: "通知中心", hidden: true },
  ] },
  { key: 'topology', label: 'TOPO 拓扑', icon: <Globe />, section: '监控', routes: [
    { path: '/topology/gis-map', element: s(C_pages_topology_GISMapView), label: "GIS 地图" },
    { path: '/topology/settings', element: s(C_pages_topology_TopologySettings), label: "拓扑设置" },
    { path: '/topology/legend', element: s(C_pages_topology_LegendSystem), label: "图例系统" },
    { path: '/topology/canvas', element: s(C_pages_topology_TopologyCanvas), label: "拓扑画布", hidden: true },
    { path: '/topology/domain', element: s(C_pages_topology_DomainManagement), label: "设备域管理", hidden: true },
    { path: '/topology/site', element: s(C_pages_topology_SiteManagement), label: "站点管理", hidden: true },
  ] },
  { key: 'config', label: 'CONF 配置', icon: <Sliders />, section: '运维', routes: [
    { path: '/config/param-sync', element: s(C_pages_config_ParamSync), label: "参数同步", hidden: true },
    { path: '/config/live-param', element: s(C_pages_config_LiveParamConfig), label: "在线参数配置", hidden: true },
    { path: '/config/batch-class', element: s(C_pages_config_BatchParamClass), label: "按参数类批量", hidden: true },
    { path: '/config/batch-template', element: s(C_pages_config_BatchParamTemplate), label: "按模板批量", hidden: true },
    { path: '/config/param-list', element: s(C_pages_config_ParamList), label: "参数列表", hidden: true },
    { path: '/config/command-mode', element: s(C_pages_config_CommandMode), label: "命令行模式", hidden: true },
    { path: '/config/cell', element: s(C_pages_config_CellManagement), label: "小区管理", hidden: true },
    { path: '/config/baseline', element: s(C_pages_config_BaselineManagement), label: "基线管理", hidden: true },
    { path: '/config/common', element: s(C_pages_config_CommonConfig), label: "公共配置", hidden: true },
    { path: '/config/neighbor', element: s(C_pages_config_NeighborParams), label: "邻区参数", hidden: true },
    { path: '/config/northbound', element: s(C_pages_config_NorthboundManagement), label: "北向接口", hidden: true },
    { path: '/config/auto-provision', element: s(C_pages_config_AutoProvisioning), label: "自动开站", hidden: true },
    { path: '/config/interop-test', element: s(C_pages_config_InteropTesting), label: "互操作测试", hidden: true },
  ] },
  { key: 'mml', label: 'MML', icon: <Terminal />, section: '运维', routes: [
    { path: '/mml/script', element: s(C_pages_mml_script_index), label: "脚本库" },
    { path: '/mml/task-records', element: s(C_pages_mml_task_records_index), label: "任务记录" },
    { path: '/mml/console-v2', element: s(C_pages_mml_console_v2_index), label: "MML控制台", hidden: true },
    { path: '/mml/commands', element: s(C_pages_mml_commands_index), label: "命令字典", hidden: true },
    { path: '/mml/private-command', element: s(C_pages_mml_private_command_index), label: "私有命令", hidden: true },
    { path: '/mml/admin/catalog', element: s(C_pages_mml_admin_catalog_index), label: "命令字典管理", hidden: true },
  ] },
  { key: 'backup', label: 'BACK 备份', icon: <Package />, section: '运维', routes: [
    { path: '/backup/tasks', element: s(C_pages_backup_Tasks), label: "备份任务", hidden: true },
    { path: '/backup/schedule', element: s(C_pages_backup_Schedule), label: "定时备份", hidden: true },
    { path: '/backup/ftp', element: s(C_pages_backup_Ftp), label: "FTP 通道", hidden: true },
    { path: '/backup/restore', element: s(C_pages_backup_Restore), label: "数据恢复", hidden: true },
    { path: '/backup/policy', element: s(C_pages_backup_Policy), label: "保留策略", hidden: true },
    { path: '/backup/config-snapshots', element: s(C_pages_backup_ConfigSnapshots), label: "配置快照库", hidden: true },
  ] },
  { key: 'software', label: 'SOFT 软件', icon: <Package />, section: '运维', routes: [
    { path: '/software/version', element: s(C_pages_software_Version), label: "版本查询", hidden: true },
    { path: '/software/upgrade-plan', element: s(C_pages_software_UpgradePlan), label: "升级计划", hidden: true },
    { path: '/software/activation', element: s(C_pages_software_Activation), label: "激活计划", hidden: true },
    { path: '/software/firmware', element: s(C_pages_software_Firmware), label: "固件库", hidden: true },
    { path: '/software/rollback', element: s(C_pages_software_Rollback), label: "版本回退", hidden: true },
  ] },
  { key: 'ops', label: 'OPS 运维', icon: <Wrench />, section: '运维', routes: [
    { path: '/ops/templates', element: s(C_pages_ops_Templates), label: "动作模板", hidden: true },
    { path: '/ops/commands', element: s(C_pages_ops_CommandManagement), label: "指令流水", hidden: true },
    { path: '/ops/tasks', element: s(C_pages_ops_TaskManagement), label: "编队任务", hidden: true },
    { path: '/ops/network-diagnosis', element: s(C_pages_ops_NetworkDiagnosis), label: "网络诊断", hidden: true },
    { path: '/ops/downloads', element: s(C_pages_ops_Downloads), label: "运维下载", hidden: true },
    { path: '/ops/message-trace', element: s(C_pages_ops_MessageTrace), label: "报文跟踪", hidden: true },
    { path: '/ops/aggregation-trigger', element: s(C_pages_ops_AggregationTrigger), label: "聚合触发", hidden: true },
  ] },
  { key: 'transfer', label: 'XFER 传输', icon: <ArrowLeftRight />, section: '运维', routes: [
    { path: '/transfer/center', element: s(C_pages_transfer_Center), label: "文件传输中心" },
    { path: '/transfer/file-management', element: s(C_pages_transfer_FileManagement), label: "文件管理" },
    { path: '/transfer/template-management', element: s(C_pages_transfer_TemplateManagement), label: "模板配置" },
  ] },
  { key: 'performance', label: 'PERF 性能', icon: <LineChart />, section: '数据', routes: [
    { path: '/performance', element: s(PerformancePage), label: "性能仪表盘" },
    { path: '/performance/pm-adhoc', element: s(C_pages_performance_PmAdhoc), label: "自定义聚合" },
    { path: '/performance/device-view', element: s(C_pages_performance_DeviceView), label: "设备性能查看" },
    { path: '/performance/query', element: s(C_pages_performance_PerformanceQuery), label: "实时取数" },
    { path: '/performance/kpi-standard', element: s(C_pages_performance_KpiStandard), label: "标准指标库", hidden: true },
    { path: '/performance/kpi-standard/detail/:deviceType/:indicatorId', element: s(C_pages_performance_KpiStandardDetail), label: "指标详情", hidden: true },
    { path: '/performance/kpi-station', element: s(C_pages_performance_KpiStation), label: "基站测量", hidden: true },
    { path: '/performance/charts', element: s(C_pages_performance_PerformanceCharts), label: "性能图表", hidden: true },
    { path: '/performance/threshold', element: s(C_pages_performance_ThresholdConfig), label: "门限配置", hidden: true },
    { path: '/performance/files', element: s(C_pages_performance_PerformanceFiles), label: "性能文件", hidden: true },
    { path: '/performance/task-config', element: s(C_pages_performance_PerformanceTaskConfig), label: "采集任务", hidden: true },
    { path: '/performance/pm-adhoc/new', element: s(C_pages_performance_PmAdhocWizard), label: "新建聚合任务", hidden: true },
    { path: '/performance/pm-adhoc/:id/edit', element: s(C_pages_performance_PmAdhocWizard), label: "编辑聚合任务", hidden: true },
    // 兼容重定向（对齐 v1 routes.tsx：旧 pm-dashboard 路径 → /performance）
    { path: '/performance/pm-dashboard', element: <Navigate to="/performance" replace />, label: "PM 看板（重定向）", hidden: true },
    { path: '/performance/pm-dashboard/:id', element: <Navigate to="/performance" replace />, label: "PM 看板（重定向）", hidden: true },
  ] },
  { key: 'mr', label: 'MR 测量', icon: <Radio />, section: '数据', routes: [
    { path: '/mr/indicators', element: s(C_pages_mr_Indicators), label: "测量指标库", hidden: true },
    { path: '/mr/device-mapping', element: s(C_pages_mr_DeviceMapping), label: "设备小区映射", hidden: true },
    { path: '/mr/variables', element: s(C_pages_mr_Variables), label: "采集变量字典", hidden: true },
    { path: '/mr/reports', element: s(C_pages_mr_Reports), label: "分析报表", hidden: true },
    { path: '/mr/files', element: s(C_pages_mr_Files), label: "测量文件", hidden: true },
  ] },
  { key: 'report', label: 'RPT 报表', icon: <ScrollText />, section: '数据', routes: [
    { path: '/report/lte-standard', element: s(C_pages_reports_LteStandard), label: "LTE 标准报表", hidden: true },
    { path: '/report/station', element: s(C_pages_reports_Station), label: "基站报表", hidden: true },
    { path: '/report/historical-kpi', element: s(C_pages_reports_HistoricalKpi), label: "历史 KPI", hidden: true },
    { path: '/report/poll-stats', element: s(C_pages_reports_PollStatistics), label: "轮询统计", hidden: true },
  ] },
  { key: 'file', label: 'FILE 文件', icon: <FolderOpen />, section: '数据', routes: [
    { path: '/file/config-retrieval', element: s(C_pages_files_ConfigRetrieval), label: "配置文件获取", hidden: true },
    { path: '/file/config-distribution', element: s(C_pages_files_ConfigDistribution), label: "配置文件下发", hidden: true },
    { path: '/file/log-retrieval', element: s(C_pages_files_LogRetrieval), label: "日志获取", hidden: true },
    { path: '/file/perf-retrieval', element: s(C_pages_files_PerfRetrieval), label: "性能获取", hidden: true },
    { path: '/file/mr-retrieval', element: s(C_pages_files_MRRetrieval), label: "测量报告获取", hidden: true },
    { path: '/file/user-files', element: s(C_pages_files_UserFiles), label: "用户文件", hidden: true },
    { path: '/file/device-files', element: s(C_pages_files_DeviceFiles), label: "网元文件", hidden: true },
  ] },
  { key: 'product', label: 'PROD 产品', icon: <Boxes />, section: '数据', routes: [
    { path: '/product/standard-params', element: s(C_pages_standard_params_index), label: "标准参数树" },
    { path: '/product/param-model', element: s(C_pages_param_model_index), label: "参数模型库" },
    { path: '/product/kpi-library', element: s(C_pages_kpi_library_index), label: "KPI 指标库" },
    { path: '/product/alarm-library', element: s(C_pages_alarm_library_index), label: "告警库" },
    { path: '/product/products', element: s(C_pages_products_index), label: "产品装配件" },
    { path: '/product/orphan-devices', element: s(C_pages_orphan_devices_index), label: "孤儿设备" },
  ] },
  { key: 'log', label: 'LOG 日志', icon: <ScrollText />, section: '系统', routes: [
    { path: '/log/device', element: s(C_pages_logs_DeviceLog), label: "基站日志", hidden: true },
    { path: '/log/exception', element: s(C_pages_logs_ExceptionLog), label: "异常重启", hidden: true },
    { path: '/log/event', element: s(C_pages_logs_EventLog), label: "设备事件", hidden: true },
    { path: '/log/operation', element: s(C_pages_logs_OperationLog), label: "操作审计", hidden: true },
    { path: '/log/system', element: s(C_pages_logs_SystemLog), label: "系统日志", hidden: true },
    { path: '/log/config', element: s(C_pages_logs_LogConfig), label: "日志配置", hidden: true },
  ] },
  { key: 'license', label: 'LIC 许可', icon: <KeySquare />, section: '系统', routes: [
    { path: '/license', element: s(LicensePage), label: "系统许可" },
    { path: '/license/history', element: s(C_pages_license_History), label: "授权历史", hidden: true },
  ] },
  { key: 'system', label: 'SYS 系统', icon: <Settings />, section: '系统', routes: [
    { path: '/system/users', element: s(C_pages_system_UserManagement), label: "用户管理" },
    { path: '/system/roles', element: s(C_pages_system_RolePermission), label: "角色权限" },
    { path: '/system/menus', element: s(C_pages_system_MenuManagement), label: "菜单管理" },
    { path: '/system/operation-log', element: s(C_pages_system_OperationLog), label: "操作日志" },
    { path: '/system/config', element: s(C_pages_system_SystemConfig), label: "系统参数" },
    { path: '/system/ui-custom', element: s(C_pages_system_UICustomization), label: "界面定制" },
    { path: '/system/api-management', element: s(C_pages_system_ApiManagement), label: "接口管理" },
    { path: '/system/data-dictionary', element: s(C_pages_system_DataDictionary), label: "数据字典" },
    { path: '/system/kpi-config', element: s(C_pages_system_KpiConfig), label: "KPI 指标配置" },
    { path: '/system/device-class', element: s(C_pages_system_DeviceClassification), label: "设备分类", hidden: true },
    { path: '/system/groups', element: s(C_pages_system_GroupManagement), label: "用户组", hidden: true },
    { path: '/system/dashboard', element: s(C_pages_system_SystemDashboard), label: "系统态势", hidden: true },
    { path: '/system/dict-loader', element: s(C_pages_system_DictLoader), label: "字典热重载", hidden: true },
  ] },
]

export const SECTIONS = ['监控', '运维', '数据', '系统']
export const ALL_ROUTES: RouteDef[] = MODULES.flatMap((m) => m.routes)
