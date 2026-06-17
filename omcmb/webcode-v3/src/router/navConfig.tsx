import { lazy, Suspense, type ComponentType, type ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import {
  AlertTriangle, ArrowLeftRight, Bell, Boxes, Cpu, FileBarChart, FileStack, FolderOpen, Globe, KeySquare, LayoutDashboard, LineChart, Package, Radio, ScrollText, Settings, Sliders, Terminal, Wrench,
} from 'lucide-react'

const BridgePage = lazy(() => import('@/pages/bridge').then((m) => ({ default: m.BridgePage })))
const FleetPage = lazy(() => import('@/pages/fleet').then((m) => ({ default: m.FleetPage })))
const AlarmsPage = lazy(() => import('@/pages/alarms').then((m) => ({ default: m.AlarmsPage })))
const TopologyPage = lazy(() => import('@/pages/topology').then((m) => ({ default: m.TopologyPage })))
const ConfigPage = lazy(() => import('@/pages/config').then((m) => ({ default: m.ConfigPage })))
const MMLPage = lazy(() => import('@/pages/mml').then((m) => ({ default: m.MMLPage })))
const SoftwarePage = lazy(() => import('@/pages/software').then((m) => ({ default: m.SoftwarePage })))
const BackupPage = lazy(() => import('@/pages/backup').then((m) => ({ default: m.BackupPage })))
const OpsPage = lazy(() => import('@/pages/ops').then((m) => ({ default: m.OpsPage })))
const PerformancePage = lazy(() => import('@/pages/performance').then((m) => ({ default: m.PerformancePage })))
const MRPage = lazy(() => import('@/pages/mr').then((m) => ({ default: m.MRPage })))
const ReportsPage = lazy(() => import('@/pages/reports').then((m) => ({ default: m.ReportsPage })))
const FilesPage = lazy(() => import('@/pages/files').then((m) => ({ default: m.FilesPage })))
const LogsPage = lazy(() => import('@/pages/logs').then((m) => ({ default: m.LogsPage })))
const LicensePage = lazy(() => import('@/pages/license').then((m) => ({ default: m.LicensePage })))
const SystemPage = lazy(() => import('@/pages/system').then((m) => ({ default: m.SystemPage })))

const C_pages_alarm_library_index = lazy(() => import('@/pages/alarm-library/index'))
const C_pages_alarms_AlarmRuleDetail = lazy(() => import('@/pages/alarms/AlarmRuleDetail'))
const C_pages_alarms_AlarmRules = lazy(() => import('@/pages/alarms/AlarmRules'))
const C_pages_alarms_AlarmStatistics = lazy(() => import('@/pages/alarms/AlarmStatistics'))
const C_pages_alarms_AlarmSync = lazy(() => import('@/pages/alarms/AlarmSync'))
const C_pages_alarms_CustomAlarmStats = lazy(() => import('@/pages/alarms/CustomAlarmStats'))
const C_pages_alarms_HistoricalAlarms = lazy(() => import('@/pages/alarms/HistoricalAlarms'))
const C_pages_backup_ConfigSnapshots = lazy(() => import('@/pages/backup/ConfigSnapshots'))
const C_pages_backup_Ftp = lazy(() => import('@/pages/backup/Ftp'))
const C_pages_backup_Policy = lazy(() => import('@/pages/backup/Policy'))
const C_pages_backup_Restore = lazy(() => import('@/pages/backup/Restore'))
const C_pages_backup_Schedule = lazy(() => import('@/pages/backup/Schedule'))
const C_pages_backup_TaskDetail = lazy(() => import('@/pages/backup/TaskDetail'))
const C_pages_backup_Tasks = lazy(() => import('@/pages/backup/Tasks'))
const C_pages_config_AutoProvisioning = lazy(() => import('@/pages/config/AutoProvisioning'))
const C_pages_config_BaselineDetail = lazy(() => import('@/pages/config/BaselineDetail'))
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
const C_pages_files_ConfigDistribution = lazy(() => import('@/pages/files/ConfigDistribution'))
const C_pages_files_ConfigDistributionDetail = lazy(() => import('@/pages/files/ConfigDistributionDetail'))
const C_pages_files_ConfigRetrieval = lazy(() => import('@/pages/files/ConfigRetrieval'))
const C_pages_files_ConfigRetrievalDetail = lazy(() => import('@/pages/files/ConfigRetrievalDetail'))
const C_pages_files_DeviceFiles = lazy(() => import('@/pages/files/DeviceFiles'))
const C_pages_files_LogRetrieval = lazy(() => import('@/pages/files/LogRetrieval'))
const C_pages_files_LogRetrievalDetail = lazy(() => import('@/pages/files/LogRetrievalDetail'))
const C_pages_files_MRRetrieval = lazy(() => import('@/pages/files/MRRetrieval'))
const C_pages_files_MRRetrievalDetail = lazy(() => import('@/pages/files/MRRetrievalDetail'))
const C_pages_files_PerfRetrieval = lazy(() => import('@/pages/files/PerfRetrieval'))
const C_pages_files_PerfRetrievalDetail = lazy(() => import('@/pages/files/PerfRetrievalDetail'))
const C_pages_files_UserFiles = lazy(() => import('@/pages/files/UserFiles'))
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
const C_pages_kpi_library_index = lazy(() => import('@/pages/kpi-library/index'))
const C_pages_license_History = lazy(() => import('@/pages/license/History'))
const C_pages_logs_DeviceLog = lazy(() => import('@/pages/logs/DeviceLog'))
const C_pages_logs_DeviceLogDetail = lazy(() => import('@/pages/logs/DeviceLogDetail'))
const C_pages_logs_EventLog = lazy(() => import('@/pages/logs/EventLog'))
const C_pages_logs_EventLogDetail = lazy(() => import('@/pages/logs/EventLogDetail'))
const C_pages_logs_ExceptionLog = lazy(() => import('@/pages/logs/ExceptionLog'))
const C_pages_logs_LogConfig = lazy(() => import('@/pages/logs/LogConfig'))
const C_pages_logs_OperationLog = lazy(() => import('@/pages/logs/OperationLog'))
const C_pages_logs_SystemLog = lazy(() => import('@/pages/logs/SystemLog'))
const C_pages_mml_admin_catalog_index = lazy(() => import('@/pages/mml/admin-catalog/index'))
const C_pages_mml_commands_detail = lazy(() => import('@/pages/mml/commands/detail'))
const C_pages_mml_commands_index = lazy(() => import('@/pages/mml/commands/index'))
const C_pages_mml_console_index = lazy(() => import('@/pages/mml/console/index'))
const C_pages_mml_private_command_detail = lazy(() => import('@/pages/mml/private-command/detail'))
const C_pages_mml_private_command_index = lazy(() => import('@/pages/mml/private-command/index'))
const C_pages_mml_script_index = lazy(() => import('@/pages/mml/script/index'))
const C_pages_mml_task_records_index = lazy(() => import('@/pages/mml/task-records/index'))
const C_pages_mr_DeviceMapping = lazy(() => import('@/pages/mr/DeviceMapping'))
const C_pages_mr_Files = lazy(() => import('@/pages/mr/Files'))
const C_pages_mr_FilesDevice = lazy(() => import('@/pages/mr/FilesDevice'))
const C_pages_mr_IndicatorDetail = lazy(() => import('@/pages/mr/IndicatorDetail'))
const C_pages_mr_Indicators = lazy(() => import('@/pages/mr/Indicators'))
const C_pages_mr_MappingDetail = lazy(() => import('@/pages/mr/MappingDetail'))
const C_pages_mr_ReportDetail = lazy(() => import('@/pages/mr/ReportDetail'))
const C_pages_mr_Reports = lazy(() => import('@/pages/mr/Reports'))
const C_pages_mr_Variables = lazy(() => import('@/pages/mr/Variables'))
const C_pages_notifications_HistoryDetail = lazy(() => import('@/pages/notifications/HistoryDetail'))
const C_pages_notifications_HistoryList = lazy(() => import('@/pages/notifications/HistoryList'))
const C_pages_notifications_NotificationCenter = lazy(() => import('@/pages/notifications/NotificationCenter'))
const C_pages_notifications_TemplateDetail = lazy(() => import('@/pages/notifications/TemplateDetail'))
const C_pages_notifications_TemplateList = lazy(() => import('@/pages/notifications/TemplateList'))
const C_pages_ops_AggregationTrigger = lazy(() => import('@/pages/ops/AggregationTrigger'))
const C_pages_ops_CommandManagement = lazy(() => import('@/pages/ops/CommandManagement'))
const C_pages_ops_Downloads = lazy(() => import('@/pages/ops/Downloads'))
const C_pages_ops_MessageTrace = lazy(() => import('@/pages/ops/MessageTrace'))
const C_pages_ops_MessageTraceDetail = lazy(() => import('@/pages/ops/MessageTraceDetail'))
const C_pages_ops_NetworkDiagnosis = lazy(() => import('@/pages/ops/NetworkDiagnosis'))
const C_pages_ops_TaskManagement = lazy(() => import('@/pages/ops/TaskManagement'))
const C_pages_ops_Templates = lazy(() => import('@/pages/ops/Templates'))
const C_pages_orphan_devices_index = lazy(() => import('@/pages/orphan-devices/index'))
const C_pages_param_model_index = lazy(() => import('@/pages/param-model/index'))
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
const C_pages_products_detail = lazy(() => import('@/pages/products/detail'))
const C_pages_products_index = lazy(() => import('@/pages/products/index'))
const C_pages_reports_HistoricalKpi = lazy(() => import('@/pages/reports/HistoricalKpi'))
const C_pages_reports_LteStandard = lazy(() => import('@/pages/reports/LteStandard'))
const C_pages_reports_PollStatistics = lazy(() => import('@/pages/reports/PollStatistics'))
const C_pages_reports_Station = lazy(() => import('@/pages/reports/Station'))
const C_pages_reports_StationDetail = lazy(() => import('@/pages/reports/StationDetail'))
const C_pages_software_Activation = lazy(() => import('@/pages/software/Activation'))
const C_pages_software_Firmware = lazy(() => import('@/pages/software/Firmware'))
const C_pages_software_Rollback = lazy(() => import('@/pages/software/Rollback'))
const C_pages_software_UpgradePlan = lazy(() => import('@/pages/software/UpgradePlan'))
const C_pages_software_UpgradeTaskDetail = lazy(() => import('@/pages/software/UpgradeTaskDetail'))
const C_pages_software_Version = lazy(() => import('@/pages/software/Version'))
const C_pages_software_VersionDetail = lazy(() => import('@/pages/software/VersionDetail'))
const C_pages_standard_params_index = lazy(() => import('@/pages/standard-params/index'))
const C_pages_system_ApiManagement = lazy(() => import('@/pages/system/ApiManagement'))
const C_pages_system_DataDictionary = lazy(() => import('@/pages/system/DataDictionary'))
const C_pages_system_DataDictionaryDetail = lazy(() => import('@/pages/system/DataDictionaryDetail'))
const C_pages_system_DeviceClassification = lazy(() => import('@/pages/system/DeviceClassification'))
const C_pages_system_DictLoader = lazy(() => import('@/pages/system/DictLoader'))
const C_pages_system_GroupDetail = lazy(() => import('@/pages/system/GroupDetail'))
const C_pages_system_GroupManagement = lazy(() => import('@/pages/system/GroupManagement'))
const C_pages_system_KpiConfig = lazy(() => import('@/pages/system/KpiConfig'))
const C_pages_system_MenuManagement = lazy(() => import('@/pages/system/MenuManagement'))
const C_pages_system_OperationLog = lazy(() => import('@/pages/system/OperationLog'))
const C_pages_system_RoleDetail = lazy(() => import('@/pages/system/RoleDetail'))
const C_pages_system_RolePermission = lazy(() => import('@/pages/system/RolePermission'))
const C_pages_system_SystemConfig = lazy(() => import('@/pages/system/SystemConfig'))
const C_pages_system_SystemDashboard = lazy(() => import('@/pages/system/SystemDashboard'))
const C_pages_system_UICustomization = lazy(() => import('@/pages/system/UICustomization'))
const C_pages_system_UserDetail = lazy(() => import('@/pages/system/UserDetail'))
const C_pages_system_UserManagement = lazy(() => import('@/pages/system/UserManagement'))
const C_pages_topology_DomainManagement = lazy(() => import('@/pages/topology/DomainManagement'))
const C_pages_topology_GISMapView = lazy(() => import('@/pages/topology/GISMapView'))
const C_pages_topology_LegendSystem = lazy(() => import('@/pages/topology/LegendSystem'))
const C_pages_topology_SiteDetail = lazy(() => import('@/pages/topology/SiteDetail'))
const C_pages_topology_SiteManagement = lazy(() => import('@/pages/topology/SiteManagement'))
const C_pages_topology_TopologyCanvas = lazy(() => import('@/pages/topology/TopologyCanvas'))
const C_pages_topology_TopologySettings = lazy(() => import('@/pages/topology/TopologySettings'))
const C_pages_transfer_Center = lazy(() => import('@/pages/transfer/Center'))
const C_pages_transfer_FileManagement = lazy(() => import('@/pages/transfer/FileManagement'))
const C_pages_transfer_TaskDetail = lazy(() => import('@/pages/transfer/TaskDetail'))
const C_pages_transfer_TemplateDetail = lazy(() => import('@/pages/transfer/TemplateDetail'))
const C_pages_transfer_TemplateManagement = lazy(() => import('@/pages/transfer/TemplateManagement'))

const s = (C: ComponentType): ReactNode => (
  <Suspense fallback={<div className="p-8 font-mono text-xs text-cyan-300/60">LOADING…</div>}>
    <C />
  </Suspense>
)

export type RouteDef = { path: string; element: ReactNode; label: string; hidden?: boolean }
export type ModuleDef = { key: string; label: string; icon: ReactNode; section: string; routes: RouteDef[] }

// 数据驱动路由+导航唯一事实源（由 metadata 生成；对齐 v1 170 路由）。
export const MODULES: ModuleDef[] = [
  { key: 'bridge', label: 'BRIDGE 指挥', icon: <LayoutDashboard />, section: '监控', routes: [
    { path: '/bridge', element: s(BridgePage), label: "BRIDGE 指挥" },
  ] },
  { key: 'fleet', label: 'FLEET 设备', icon: <Cpu />, section: '监控', routes: [
    { path: '/fleet', element: s(FleetPage), label: "FLEET 设备" },
    { path: '/fleet/detail/:sn', element: s(C_pages_fleet_DeviceDetail), label: "设备详情", hidden: true },
    { path: '/fleet/group', element: s(C_pages_fleet_DeviceGroup), label: "设备分组" },
    { path: '/fleet/register', element: s(C_pages_fleet_DeviceRegister), label: "设备注册" },
    { path: '/fleet/ne', element: s(C_pages_fleet_NEManagement), label: "网元管理" },
    { path: '/fleet/monitor', element: s(C_pages_fleet_OnlineMonitor), label: "在线监控" },
    { path: '/fleet/stats', element: s(C_pages_fleet_ResourceStats), label: "资源统计" },
    { path: '/fleet/commission', element: s(C_pages_fleet_Commissioning), label: "开通调测" },
    { path: '/fleet/handover', element: s(C_pages_fleet_HandoverManagement), label: "割接管理" },
    { path: '/fleet/import', element: s(C_pages_fleet_ImportExport), label: "导入导出" },
    { path: '/fleet/recycle', element: s(C_pages_fleet_RecycleBin), label: "回收站" },
    { path: '/fleet/abnormal-reboot', element: s(C_pages_fleet_AbnormalReboot), label: "异常重启" },
    { path: '/fleet/ue-detail/:sn', element: s(C_pages_fleet_UeDetail), label: "UE 详情", hidden: true },
    { path: '/fleet/plug-and-play', element: s(C_pages_fleet_PlugAndPlay), label: "即插即用" },
    { path: '/fleet/plug-and-play/add', element: s(C_pages_fleet_PlugAndPlayPolicy), label: "新建策略", hidden: true },
    { path: '/fleet/plug-and-play/edit/:id', element: s(C_pages_fleet_PlugAndPlayPolicy), label: "编辑策略", hidden: true },
    { path: '/fleet/plug-and-play/view/:id', element: s(C_pages_fleet_PlugAndPlayPolicy), label: "查看策略", hidden: true },
  ] },
  { key: 'alarms', label: 'ALARM 告警', icon: <AlertTriangle />, section: '监控', routes: [
    { path: '/alarms', element: s(AlarmsPage), label: "ALARM 告警" },
    { path: '/alarms/statistics', element: s(C_pages_alarms_AlarmStatistics), label: "告警统计" },
    { path: '/alarms/history', element: s(C_pages_alarms_HistoricalAlarms), label: "历史告警" },
    { path: '/alarms/rules', element: s(C_pages_alarms_AlarmRules), label: "告警规则" },
    { path: '/alarms/rules/:id', element: s(C_pages_alarms_AlarmRuleDetail), label: "规则详情", hidden: true },
    { path: '/alarms/sync', element: s(C_pages_alarms_AlarmSync), label: "告警同步" },
    { path: '/alarms/custom-stats', element: s(C_pages_alarms_CustomAlarmStats), label: "自定义告警统计" },
  ] },
  { key: 'notifications', label: 'NOTIFY 通知', icon: <Bell />, section: '监控', routes: [
    { path: '/notifications', element: s(C_pages_notifications_NotificationCenter), label: "NOTIFY" },
    { path: '/notifications/templates', element: s(C_pages_notifications_TemplateList), label: "通知模板", hidden: true },
    { path: '/notifications/templates/:id', element: s(C_pages_notifications_TemplateDetail), label: "模板详情", hidden: true },
    { path: '/notifications/history', element: s(C_pages_notifications_HistoryList), label: "发送历史", hidden: true },
    { path: '/notifications/history/:id', element: s(C_pages_notifications_HistoryDetail), label: "发送详情", hidden: true },
  ] },
  { key: 'topology', label: 'TOPO 拓扑', icon: <Globe />, section: '监控', routes: [
    { path: '/topology', element: s(TopologyPage), label: "TOPO 拓扑" },
    { path: '/topology/gis-map', element: s(C_pages_topology_GISMapView), label: "GIS 地图" },
    { path: '/topology/canvas', element: s(C_pages_topology_TopologyCanvas), label: "拓扑画布" },
    { path: '/topology/domain', element: s(C_pages_topology_DomainManagement), label: "设备域管理" },
    { path: '/topology/site', element: s(C_pages_topology_SiteManagement), label: "站点管理" },
    { path: '/topology/site/:id', element: s(C_pages_topology_SiteDetail), label: "站点详情", hidden: true },
    { path: '/topology/settings', element: s(C_pages_topology_TopologySettings), label: "拓扑设置" },
    { path: '/topology/legend', element: s(C_pages_topology_LegendSystem), label: "图例系统" },
  ] },
  { key: 'config', label: 'CONF 配置', icon: <Sliders />, section: '运维', routes: [
    { path: '/config', element: s(ConfigPage), label: "CONF 配置" },
    { path: 'config/param-sync', element: s(C_pages_config_ParamSync), label: "参数同步" },
    { path: 'config/live-param', element: s(C_pages_config_LiveParamConfig), label: "在线参数配置" },
    { path: 'config/batch-class', element: s(C_pages_config_BatchParamClass), label: "按参数类批量" },
    { path: 'config/batch-template', element: s(C_pages_config_BatchParamTemplate), label: "按模板批量" },
    { path: 'config/param-list', element: s(C_pages_config_ParamList), label: "参数列表" },
    { path: 'config/command-mode', element: s(C_pages_config_CommandMode), label: "命令行模式" },
    { path: 'config/cell', element: s(C_pages_config_CellManagement), label: "小区管理" },
    { path: 'config/baseline', element: s(C_pages_config_BaselineManagement), label: "基线管理" },
    { path: 'config/baseline/:id', element: s(C_pages_config_BaselineDetail), label: "基线详情", hidden: true },
    { path: 'config/common', element: s(C_pages_config_CommonConfig), label: "公共配置" },
    { path: 'config/neighbor', element: s(C_pages_config_NeighborParams), label: "邻区参数" },
    { path: 'config/northbound', element: s(C_pages_config_NorthboundManagement), label: "北向接口" },
    { path: 'config/auto-provision', element: s(C_pages_config_AutoProvisioning), label: "自动开站" },
    { path: 'config/interop-test', element: s(C_pages_config_InteropTesting), label: "互操作测试" },
  ] },
  { key: 'mml', label: 'MML', icon: <Terminal />, section: '运维', routes: [
    { path: '/mml', element: s(MMLPage), label: "MML" },
    { path: '/mml/console', element: s(C_pages_mml_console_index), label: "MML控制台" },
    { path: '/mml/script', element: s(C_pages_mml_script_index), label: "脚本库" },
    { path: '/mml/task-records', element: s(C_pages_mml_task_records_index), label: "任务记录" },
    { path: '/mml/commands', element: s(C_pages_mml_commands_index), label: "命令字典" },
    { path: '/mml/commands/:id', element: s(C_pages_mml_commands_detail), label: "命令详情", hidden: true },
    { path: '/mml/private-command', element: s(C_pages_mml_private_command_index), label: "私有命令" },
    { path: '/mml/private-command/:id', element: s(C_pages_mml_private_command_detail), label: "私有命令详情", hidden: true },
    { path: '/mml/admin-catalog', element: s(C_pages_mml_admin_catalog_index), label: "命令字典管理" },
  ] },
  { key: 'software', label: 'SOFT 软件', icon: <Package />, section: '运维', routes: [
    { path: '/software', element: s(SoftwarePage), label: "SOFT 软件" },
    { path: 'software/version', element: s(C_pages_software_Version), label: "版本查询" },
    { path: 'software/version/:id', element: s(C_pages_software_VersionDetail), label: "版本详情", hidden: true },
    { path: 'software/upgrade-plan', element: s(C_pages_software_UpgradePlan), label: "升级计划" },
    { path: 'software/upgrade-plan/:id', element: s(C_pages_software_UpgradeTaskDetail), label: "升级任务详情", hidden: true },
    { path: 'software/activation', element: s(C_pages_software_Activation), label: "激活计划" },
    { path: 'software/firmware', element: s(C_pages_software_Firmware), label: "固件库" },
    { path: 'software/rollback', element: s(C_pages_software_Rollback), label: "版本回退" },
  ] },
  { key: 'backup', label: 'BACK 备份', icon: <FileStack />, section: '运维', routes: [
    { path: '/backup', element: s(BackupPage), label: "BACK 备份" },
    { path: '/backup/tasks', element: s(C_pages_backup_Tasks), label: "备份任务" },
    { path: '/backup/tasks/:id', element: s(C_pages_backup_TaskDetail), label: "备份任务详情", hidden: true },
    { path: '/backup/schedule', element: s(C_pages_backup_Schedule), label: "定时备份" },
    { path: '/backup/ftp', element: s(C_pages_backup_Ftp), label: "FTP 通道" },
    { path: '/backup/restore', element: s(C_pages_backup_Restore), label: "数据恢复" },
    { path: '/backup/policy', element: s(C_pages_backup_Policy), label: "保留策略" },
    { path: '/backup/config-snapshots', element: s(C_pages_backup_ConfigSnapshots), label: "配置快照库" },
  ] },
  { key: 'ops', label: 'OPS 运维', icon: <Wrench />, section: '运维', routes: [
    { path: '/ops', element: s(OpsPage), label: "OPS 运维" },
    { path: 'ops/templates', element: s(C_pages_ops_Templates), label: "动作模板" },
    { path: 'ops/commands', element: s(C_pages_ops_CommandManagement), label: "指令流水" },
    { path: 'ops/tasks', element: s(C_pages_ops_TaskManagement), label: "编队任务" },
    { path: 'ops/network-diagnosis', element: s(C_pages_ops_NetworkDiagnosis), label: "网络诊断" },
    { path: 'ops/downloads', element: s(C_pages_ops_Downloads), label: "运维下载" },
    { path: 'ops/message-trace', element: s(C_pages_ops_MessageTrace), label: "报文跟踪" },
    { path: 'ops/message-trace/:taskId', element: s(C_pages_ops_MessageTraceDetail), label: "跟踪报文详情", hidden: true },
    { path: 'ops/aggregation-trigger', element: s(C_pages_ops_AggregationTrigger), label: "聚合触发" },
  ] },
  { key: 'transfer', label: 'XFER 传输', icon: <ArrowLeftRight />, section: '运维', routes: [
    { path: '/transfer/center', element: s(C_pages_transfer_Center), label: "文件传输中心" },
    { path: '/transfer/center/:taskId', element: s(C_pages_transfer_TaskDetail), label: "传输任务详情", hidden: true },
    { path: '/transfer/file-management', element: s(C_pages_transfer_FileManagement), label: "文件管理" },
    { path: '/transfer/template-management', element: s(C_pages_transfer_TemplateManagement), label: "模板配置" },
    { path: '/transfer/template-management/:typeCode', element: s(C_pages_transfer_TemplateDetail), label: "模板详情", hidden: true },
  ] },
  { key: 'performance', label: 'PERF 性能', icon: <LineChart />, section: '数据', routes: [
    { path: '/performance', element: s(PerformancePage), label: "PERF 性能" },
    { path: 'performance/kpi-standard', element: s(C_pages_performance_KpiStandard), label: "标准指标库" },
    { path: 'performance/kpi-standard/detail/:deviceType/:indicatorId', element: s(C_pages_performance_KpiStandardDetail), label: "指标详情", hidden: true },
    { path: 'performance/kpi-station', element: s(C_pages_performance_KpiStation), label: "基站测量" },
    { path: 'performance/query', element: s(C_pages_performance_PerformanceQuery), label: "实时取数" },
    { path: 'performance/charts', element: s(C_pages_performance_PerformanceCharts), label: "性能图表" },
    { path: 'performance/threshold', element: s(C_pages_performance_ThresholdConfig), label: "门限配置" },
    { path: 'performance/files', element: s(C_pages_performance_PerformanceFiles), label: "性能文件" },
    { path: 'performance/task-config', element: s(C_pages_performance_PerformanceTaskConfig), label: "采集任务" },
    { path: 'performance/device-view', element: s(C_pages_performance_DeviceView), label: "设备视图" },
    { path: 'performance/pm-adhoc', element: s(C_pages_performance_PmAdhoc), label: "自定义聚合" },
    { path: 'performance/pm-adhoc/new', element: s(C_pages_performance_PmAdhocWizard), label: "新建聚合任务", hidden: true },
    { path: 'performance/pm-adhoc/:id/edit', element: s(C_pages_performance_PmAdhocWizard), label: "编辑聚合任务", hidden: true },
    // 兼容重定向（对齐 v1 routes.tsx：旧 pm-dashboard 路径 → /performance）
    { path: 'performance/pm-dashboard', element: <Navigate to="/performance" replace />, label: "PM 看板（重定向）", hidden: true },
    { path: 'performance/pm-dashboard/:id', element: <Navigate to="/performance" replace />, label: "PM 看板（重定向）", hidden: true },
  ] },
  { key: 'mr', label: 'MR 测量', icon: <Radio />, section: '数据', routes: [
    { path: '/mr', element: s(MRPage), label: "MR 测量" },
    { path: 'mr/indicators', element: s(C_pages_mr_Indicators), label: "测量指标库" },
    { path: 'mr/indicators/:code', element: s(C_pages_mr_IndicatorDetail), label: "指标详情", hidden: true },
    { path: 'mr/device-mapping', element: s(C_pages_mr_DeviceMapping), label: "设备小区映射" },
    { path: 'mr/device-mapping/:id', element: s(C_pages_mr_MappingDetail), label: "映射详情", hidden: true },
    { path: 'mr/variables', element: s(C_pages_mr_Variables), label: "采集变量字典" },
    { path: 'mr/reports', element: s(C_pages_mr_Reports), label: "分析报表" },
    { path: 'mr/reports/:id', element: s(C_pages_mr_ReportDetail), label: "报表定义详情", hidden: true },
    { path: 'mr/files', element: s(C_pages_mr_Files), label: "测量文件" },
    { path: 'mr/files/:deviceSn', element: s(C_pages_mr_FilesDevice), label: "单设备文件", hidden: true },
  ] },
  { key: 'reports', label: 'RPT 报表', icon: <FileBarChart />, section: '数据', routes: [
    { path: '/reports', element: s(ReportsPage), label: "RPT 报表" },
    { path: '/reports/lte-standard', element: s(C_pages_reports_LteStandard), label: "LTE 标准报表" },
    { path: '/reports/station', element: s(C_pages_reports_Station), label: "基站报表" },
    { path: '/reports/station/:sn', element: s(C_pages_reports_StationDetail), label: "基站报表详情", hidden: true },
    { path: '/reports/historical-kpi', element: s(C_pages_reports_HistoricalKpi), label: "历史 KPI" },
    { path: '/reports/poll-stats', element: s(C_pages_reports_PollStatistics), label: "轮询统计" },
  ] },
  { key: 'files', label: 'FILE 文件', icon: <FolderOpen />, section: '数据', routes: [
    { path: '/files', element: s(FilesPage), label: "FILE 文件" },
    { path: 'file/config-retrieval', element: s(C_pages_files_ConfigRetrieval), label: "配置文件获取" },
    { path: 'file/config-retrieval/:taskId', element: s(C_pages_files_ConfigRetrievalDetail), label: "配置拉取任务详情", hidden: true },
    { path: 'file/config-distribution', element: s(C_pages_files_ConfigDistribution), label: "配置文件下发" },
    { path: 'file/config-distribution/:taskId', element: s(C_pages_files_ConfigDistributionDetail), label: "配置下发任务详情", hidden: true },
    { path: 'file/log-retrieval', element: s(C_pages_files_LogRetrieval), label: "日志获取" },
    { path: 'file/log-retrieval/:deviceSn', element: s(C_pages_files_LogRetrievalDetail), label: "设备日志详情", hidden: true },
    { path: 'file/perf-retrieval', element: s(C_pages_files_PerfRetrieval), label: "性能获取" },
    { path: 'file/perf-retrieval/:id', element: s(C_pages_files_PerfRetrievalDetail), label: "文件详情", hidden: true },
    { path: 'file/mr-retrieval', element: s(C_pages_files_MRRetrieval), label: "测量报告获取" },
    { path: 'file/mr-retrieval/:taskId', element: s(C_pages_files_MRRetrievalDetail), label: "MR 任务详情", hidden: true },
    { path: 'file/user-files', element: s(C_pages_files_UserFiles), label: "用户文件" },
    { path: 'file/device-files', element: s(C_pages_files_DeviceFiles), label: "网元文件" },
  ] },
  { key: 'product', label: 'PROD 产品', icon: <Boxes />, section: '数据', routes: [
    { path: 'product/products', element: s(C_pages_products_index), label: "产品装配件" },
    { path: 'product/products/:id', element: s(C_pages_products_detail), label: "装配件详情", hidden: true },
    { path: 'product/param-model', element: s(C_pages_param_model_index), label: "参数模型库" },
    { path: 'product/standard-params', element: s(C_pages_standard_params_index), label: "标准参数树" },
    { path: 'product/kpi-library', element: s(C_pages_kpi_library_index), label: "KPI 指标库" },
    { path: 'product/alarm-library', element: s(C_pages_alarm_library_index), label: "告警库" },
    { path: 'product/orphan-devices', element: s(C_pages_orphan_devices_index), label: "孤儿设备" },
  ] },
  { key: 'logs', label: 'LOG 日志', icon: <ScrollText />, section: '系统', routes: [
    { path: '/logs', element: s(LogsPage), label: "LOG 日志" },
    { path: '/logs/system', element: s(C_pages_logs_SystemLog), label: "系统日志" },
    { path: '/logs/operation', element: s(C_pages_logs_OperationLog), label: "操作审计" },
    { path: '/logs/device', element: s(C_pages_logs_DeviceLog), label: "基站日志" },
    { path: '/logs/device/:id', element: s(C_pages_logs_DeviceLogDetail), label: "基站日志详情", hidden: true },
    { path: '/logs/exception', element: s(C_pages_logs_ExceptionLog), label: "异常重启" },
    { path: '/logs/event', element: s(C_pages_logs_EventLog), label: "设备事件" },
    { path: '/logs/event/:id', element: s(C_pages_logs_EventLogDetail), label: "设备事件详情", hidden: true },
    { path: '/logs/config', element: s(C_pages_logs_LogConfig), label: "日志配置" },
  ] },
  { key: 'license', label: 'LIC 许可', icon: <KeySquare />, section: '系统', routes: [
    { path: '/license', element: s(LicensePage), label: "LIC 许可" },
    { path: '/license/history', element: s(C_pages_license_History), label: "授权历史", hidden: true },
  ] },
  { key: 'system', label: 'SYS 系统', icon: <Settings />, section: '系统', routes: [
    { path: '/system', element: s(SystemPage), label: "SYS 系统" },
    { path: 'system/device-class', element: s(C_pages_system_DeviceClassification), label: "设备分类" },
    { path: 'system/users', element: s(C_pages_system_UserManagement), label: "用户管理" },
    { path: 'system/users/:id', element: s(C_pages_system_UserDetail), label: "用户详情", hidden: true },
    { path: 'system/groups', element: s(C_pages_system_GroupManagement), label: "用户组" },
    { path: 'system/groups/:id', element: s(C_pages_system_GroupDetail), label: "用户组详情", hidden: true },
    { path: 'system/roles', element: s(C_pages_system_RolePermission), label: "角色权限" },
    { path: 'system/roles/:id', element: s(C_pages_system_RoleDetail), label: "角色详情", hidden: true },
    { path: 'system/operation-log', element: s(C_pages_system_OperationLog), label: "操作日志" },
    { path: 'system/config', element: s(C_pages_system_SystemConfig), label: "系统参数" },
    { path: 'system/ui-custom', element: s(C_pages_system_UICustomization), label: "界面定制" },
    { path: 'system/menus', element: s(C_pages_system_MenuManagement), label: "菜单管理" },
    { path: 'system/dashboard', element: s(C_pages_system_SystemDashboard), label: "系统态势" },
    { path: 'system/api-management', element: s(C_pages_system_ApiManagement), label: "接口管理" },
    { path: 'system/data-dictionary', element: s(C_pages_system_DataDictionary), label: "数据字典" },
    { path: 'system/data-dictionary/:type', element: s(C_pages_system_DataDictionaryDetail), label: "字典详情", hidden: true },
    { path: 'system/dict-loader', element: s(C_pages_system_DictLoader), label: "字典热重载" },
    { path: 'system/kpi-config', element: s(C_pages_system_KpiConfig), label: "KPI 指标配置" },
  ] },
]

export const SECTIONS = ['监控', '运维', '数据', '系统']
export const ALL_ROUTES: RouteDef[] = MODULES.flatMap((m) => m.routes)
