import { lazy, Suspense, type ComponentType, type ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import {
  AlertTriangle, ArrowLeftRight, Bell, Boxes, Cpu, FileBarChart, FileStack, FolderOpen, Globe, KeySquare, LayoutDashboard, LineChart, Package, Radio, ScrollText, Settings, Sliders, Terminal, Wrench,
} from 'lucide-react'

const DashboardPage = lazy(() => import('@/pages/dashboard').then((m) => ({ default: m.DashboardPage })))
const DevicesPage = lazy(() => import('@/pages/devices').then((m) => ({ default: m.DevicesPage })))
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

const C_pages_alarms_AlarmRuleDetail = lazy(() => import('@/pages/alarms/AlarmRuleDetail'))
const C_pages_alarms_AlarmRules = lazy(() => import('@/pages/alarms/AlarmRules'))
const C_pages_alarms_AlarmStatistics = lazy(() => import('@/pages/alarms/AlarmStatistics'))
const C_pages_alarms_AlarmSync = lazy(() => import('@/pages/alarms/AlarmSync'))
const C_pages_alarms_HistoricalAlarms = lazy(() => import('@/pages/alarms/HistoricalAlarms'))
const C_pages_alarms_CustomAlarmStats = lazy(() => import('@/pages/alarms/CustomAlarmStats'))
const C_pages_backup_BackupPolicy = lazy(() => import('@/pages/backup/BackupPolicy'))
const C_pages_backup_BackupSchedule = lazy(() => import('@/pages/backup/BackupSchedule'))
const C_pages_backup_BackupTaskDetail = lazy(() => import('@/pages/backup/BackupTaskDetail'))
const C_pages_backup_BackupTasks = lazy(() => import('@/pages/backup/BackupTasks'))
const C_pages_backup_ConfigSnapshotLibrary = lazy(() => import('@/pages/backup/ConfigSnapshotLibrary'))
const C_pages_backup_FTPConfig = lazy(() => import('@/pages/backup/FTPConfig'))
const C_pages_backup_RestoreData = lazy(() => import('@/pages/backup/RestoreData'))
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
const C_pages_config_ProvisioningTaskDetail = lazy(() => import('@/pages/config/ProvisioningTaskDetail'))
const C_pages_config_TemplateDetail = lazy(() => import('@/pages/config/TemplateDetail'))
const C_pages_device_AbnormalReboot = lazy(() => import('@/pages/device/AbnormalReboot'))
const C_pages_device_AddPolicyPage = lazy(() => import('@/pages/device/AddPolicyPage'))
const C_pages_device_Commissioning = lazy(() => import('@/pages/device/Commissioning'))
const C_pages_device_DeviceDetail = lazy(() => import('@/pages/device/DeviceDetail'))
const C_pages_device_DeviceGrouping = lazy(() => import('@/pages/device/DeviceGrouping'))
const C_pages_device_DeviceRegistration = lazy(() => import('@/pages/device/DeviceRegistration'))
const C_pages_device_HandoverManagement = lazy(() => import('@/pages/device/HandoverManagement'))
const C_pages_device_ImportExport = lazy(() => import('@/pages/device/ImportExport'))
const C_pages_device_NEManagement = lazy(() => import('@/pages/device/NEManagement'))
const C_pages_device_OnlineMonitoring = lazy(() => import('@/pages/device/OnlineMonitoring'))
const C_pages_device_PlugAndPlay = lazy(() => import('@/pages/device/PlugAndPlay'))
const C_pages_device_RecycleBin = lazy(() => import('@/pages/device/RecycleBin'))
const C_pages_device_ResourceStatistics = lazy(() => import('@/pages/device/ResourceStatistics'))
const C_pages_device_UeDetail = lazy(() => import('@/pages/device/UeDetail'))
const C_pages_files_ConfigDistribution = lazy(() => import('@/pages/files/ConfigDistribution'))
const C_pages_files_ConfigRetrieval = lazy(() => import('@/pages/files/ConfigRetrieval'))
const C_pages_files_DeviceFiles = lazy(() => import('@/pages/files/DeviceFiles'))
const C_pages_files_FileDetail = lazy(() => import('@/pages/files/FileDetail'))
const C_pages_files_LogRetrieval = lazy(() => import('@/pages/files/LogRetrieval'))
const C_pages_files_MRDetail = lazy(() => import('@/pages/files/MRDetail'))
const C_pages_files_MRRetrieval = lazy(() => import('@/pages/files/MRRetrieval'))
const C_pages_files_PerfDetail = lazy(() => import('@/pages/files/PerfDetail'))
const C_pages_files_PerfRetrieval = lazy(() => import('@/pages/files/PerfRetrieval'))
const C_pages_files_UserFiles = lazy(() => import('@/pages/files/UserFiles'))
const C_pages_license_History = lazy(() => import('@/pages/license/History'))
const C_pages_logs_DeviceLog = lazy(() => import('@/pages/logs/DeviceLog'))
const C_pages_logs_DeviceLogDetail = lazy(() => import('@/pages/logs/DeviceLogDetail'))
const C_pages_logs_EventLog = lazy(() => import('@/pages/logs/EventLog'))
const C_pages_logs_EventLogDetail = lazy(() => import('@/pages/logs/EventLogDetail'))
const C_pages_logs_ExceptionLog = lazy(() => import('@/pages/logs/ExceptionLog'))
const C_pages_logs_ExceptionLogDetail = lazy(() => import('@/pages/logs/ExceptionLogDetail'))
const C_pages_logs_LogConfig = lazy(() => import('@/pages/logs/LogConfig'))
const C_pages_logs_OperationLog = lazy(() => import('@/pages/logs/OperationLog'))
const C_pages_logs_SystemLog = lazy(() => import('@/pages/logs/SystemLog'))
const C_pages_mml_CommandDetail = lazy(() => import('@/pages/mml/CommandDetail'))
const C_pages_mml_CommandTree = lazy(() => import('@/pages/mml/CommandTree'))
const C_pages_mml_Console = lazy(() => import('@/pages/mml/Console'))
const C_pages_mml_PrivateCommand = lazy(() => import('@/pages/mml/PrivateCommand'))
const C_pages_mml_PrivateCommandDetail = lazy(() => import('@/pages/mml/PrivateCommandDetail'))
const C_pages_mml_ScriptTask = lazy(() => import('@/pages/mml/ScriptTask'))
const C_pages_mml_TaskRecord = lazy(() => import('@/pages/mml/TaskRecord'))
const C_pages_mml_admin_Catalog = lazy(() => import('@/pages/mml/admin/Catalog'))
const C_pages_mml_admin_CatalogCommandDetail = lazy(() => import('@/pages/mml/admin/CatalogCommandDetail'))
const C_pages_mr_DeviceFiles = lazy(() => import('@/pages/mr/DeviceFiles'))
const C_pages_mr_DeviceMapping = lazy(() => import('@/pages/mr/DeviceMapping'))
const C_pages_mr_DeviceMappingDetail = lazy(() => import('@/pages/mr/DeviceMappingDetail'))
const C_pages_mr_Files = lazy(() => import('@/pages/mr/Files'))
const C_pages_mr_Indicators = lazy(() => import('@/pages/mr/Indicators'))
const C_pages_mr_ReportDetail = lazy(() => import('@/pages/mr/ReportDetail'))
const C_pages_mr_Reports = lazy(() => import('@/pages/mr/Reports'))
const C_pages_mr_Variables = lazy(() => import('@/pages/mr/Variables'))
const C_pages_notifications_History = lazy(() => import('@/pages/notifications/History'))
const C_pages_notifications_HistoryDetail = lazy(() => import('@/pages/notifications/HistoryDetail'))
const C_pages_notifications_TemplateCreate = lazy(() => import('@/pages/notifications/TemplateCreate'))
const C_pages_notifications_TemplateEdit = lazy(() => import('@/pages/notifications/TemplateEdit'))
const C_pages_notifications_Templates = lazy(() => import('@/pages/notifications/Templates'))
const C_pages_notifications_index = lazy(() => import('@/pages/notifications/index'))
const C_pages_ops_AggregationTrigger = lazy(() => import('@/pages/ops/AggregationTrigger'))
const C_pages_ops_CommandDetail = lazy(() => import('@/pages/ops/CommandDetail'))
const C_pages_ops_CommandManagement = lazy(() => import('@/pages/ops/CommandManagement'))
const C_pages_ops_Downloads = lazy(() => import('@/pages/ops/Downloads'))
const C_pages_ops_MessageTrace = lazy(() => import('@/pages/ops/MessageTrace'))
const C_pages_ops_MessageTraceDetail = lazy(() => import('@/pages/ops/MessageTraceDetail'))
const C_pages_ops_NetworkDiagnosis = lazy(() => import('@/pages/ops/NetworkDiagnosis'))
const C_pages_ops_TaskDetail = lazy(() => import('@/pages/ops/TaskDetail'))
const C_pages_ops_TaskManagement = lazy(() => import('@/pages/ops/TaskManagement'))
const C_pages_ops_TemplateDetail = lazy(() => import('@/pages/ops/TemplateDetail'))
const C_pages_ops_Templates = lazy(() => import('@/pages/ops/Templates'))
const C_pages_performance_DeviceView = lazy(() => import('@/pages/performance/DeviceView'))
const C_pages_performance_KPIIndicatorDetail = lazy(() => import('@/pages/performance/KPIIndicatorDetail'))
const C_pages_performance_KPIQuery = lazy(() => import('@/pages/performance/KPIQuery'))
const C_pages_performance_KPIStandardReport = lazy(() => import('@/pages/performance/KPIStandardReport'))
const C_pages_performance_KPIStationReport = lazy(() => import('@/pages/performance/KPIStationReport'))
const C_pages_performance_PerformanceCharts = lazy(() => import('@/pages/performance/PerformanceCharts'))
const C_pages_performance_PerformanceFiles = lazy(() => import('@/pages/performance/PerformanceFiles'))
const C_pages_performance_PerformanceTaskConfig = lazy(() => import('@/pages/performance/PerformanceTaskConfig'))
const C_pages_performance_PmAdhoc = lazy(() => import('@/pages/performance/PmAdhoc'))
const C_pages_performance_PmAdhocWizard = lazy(() => import('@/pages/performance/PmAdhocWizard'))
const C_pages_performance_ThresholdConfig = lazy(() => import('@/pages/performance/ThresholdConfig'))
const C_pages_product_AlarmDefinitions = lazy(() => import('@/pages/product/AlarmDefinitions'))
const C_pages_product_AlarmLibrary = lazy(() => import('@/pages/product/AlarmLibrary'))
const C_pages_product_KpiIndicators = lazy(() => import('@/pages/product/KpiIndicators'))
const C_pages_product_KpiLibrary = lazy(() => import('@/pages/product/KpiLibrary'))
const C_pages_product_OrphanDevices = lazy(() => import('@/pages/product/OrphanDevices'))
const C_pages_product_ParamModel = lazy(() => import('@/pages/product/ParamModel'))
const C_pages_product_ParamModelMappings = lazy(() => import('@/pages/product/ParamModelMappings'))
const C_pages_product_ProductDetail = lazy(() => import('@/pages/product/ProductDetail'))
const C_pages_product_Products = lazy(() => import('@/pages/product/Products'))
const C_pages_product_StandardParams = lazy(() => import('@/pages/product/StandardParams'))
const C_pages_reports_HistoricalKpi = lazy(() => import('@/pages/reports/HistoricalKpi'))
const C_pages_reports_LteStandardReport = lazy(() => import('@/pages/reports/LteStandardReport'))
const C_pages_reports_PollStatDetail = lazy(() => import('@/pages/reports/PollStatDetail'))
const C_pages_reports_PollStatistics = lazy(() => import('@/pages/reports/PollStatistics'))
const C_pages_reports_ReportRecordDetail = lazy(() => import('@/pages/reports/ReportRecordDetail'))
const C_pages_reports_StationDetail = lazy(() => import('@/pages/reports/StationDetail'))
const C_pages_reports_StationReport = lazy(() => import('@/pages/reports/StationReport'))
const C_pages_software_ActivationPlan = lazy(() => import('@/pages/software/ActivationPlan'))
const C_pages_software_FirmwareUpload = lazy(() => import('@/pages/software/FirmwareUpload'))
const C_pages_software_UpgradePlan = lazy(() => import('@/pages/software/UpgradePlan'))
const C_pages_software_UpgradeTaskDetail = lazy(() => import('@/pages/software/UpgradeTaskDetail'))
const C_pages_software_VersionDetail = lazy(() => import('@/pages/software/VersionDetail'))
const C_pages_software_VersionQuery = lazy(() => import('@/pages/software/VersionQuery'))
const C_pages_software_VersionRollback = lazy(() => import('@/pages/software/VersionRollback'))
const C_pages_system_ApiManagement = lazy(() => import('@/pages/system/ApiManagement'))
const C_pages_system_DataDictionary = lazy(() => import('@/pages/system/DataDictionary'))
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
const C_pages_topology_GisMap = lazy(() => import('@/pages/topology/GisMap'))
const C_pages_topology_LegendSystem = lazy(() => import('@/pages/topology/LegendSystem'))
const C_pages_topology_SiteDetail = lazy(() => import('@/pages/topology/SiteDetail'))
const C_pages_topology_SiteManagement = lazy(() => import('@/pages/topology/SiteManagement'))
const C_pages_topology_TopologyCanvas = lazy(() => import('@/pages/topology/TopologyCanvas'))
const C_pages_topology_TopologySettings = lazy(() => import('@/pages/topology/TopologySettings'))
const C_pages_transfer_FileManagement = lazy(() => import('@/pages/transfer/FileManagement'))
const C_pages_transfer_FileTransferCenter = lazy(() => import('@/pages/transfer/FileTransferCenter'))
const C_pages_transfer_TaskDetail = lazy(() => import('@/pages/transfer/TaskDetail'))
const C_pages_transfer_TemplateDetail = lazy(() => import('@/pages/transfer/TemplateDetail'))
const C_pages_transfer_TemplateManagement = lazy(() => import('@/pages/transfer/TemplateManagement'))

const s = (C: ComponentType): ReactNode => (
  <Suspense fallback={<div className="p-8 text-sm text-muted-foreground">加载中…</div>}>
    <C />
  </Suspense>
)

export type RouteDef = { path: string; element: ReactNode; label: string; hidden?: boolean }
export type ModuleDef = { key: string; label: string; icon: ReactNode; section: string; routes: RouteDef[] }

// 数据驱动路由+导航唯一事实源（由 metadata 生成；对齐 v1 177 路由）。
export const MODULES: ModuleDef[] = [
  { key: 'dashboard', label: '控制台', icon: <LayoutDashboard />, section: '监控', routes: [
    { path: '/dashboard', element: s(DashboardPage), label: "控制台" },
  ] },
  { key: 'device', label: '设备管理', icon: <Cpu />, section: '监控', routes: [
    { path: '/devices', element: s(DevicesPage), label: "设备管理" },
    { path: '/devices/detail/:sn', element: s(C_pages_device_DeviceDetail), label: "设备详情", hidden: true },
    { path: '/devices/group', element: s(C_pages_device_DeviceGrouping), label: "设备分组" },
    { path: '/devices/register', element: s(C_pages_device_DeviceRegistration), label: "设备注册" },
    { path: '/devices/ne', element: s(C_pages_device_NEManagement), label: "网元管理" },
    { path: '/devices/monitor', element: s(C_pages_device_OnlineMonitoring), label: "在线监控" },
    { path: '/devices/stats', element: s(C_pages_device_ResourceStatistics), label: "资源统计" },
    { path: '/devices/commission', element: s(C_pages_device_Commissioning), label: "开站调测" },
    { path: '/devices/handover', element: s(C_pages_device_HandoverManagement), label: "交维管理" },
    { path: '/devices/import', element: s(C_pages_device_ImportExport), label: "导入导出" },
    { path: '/devices/recycle', element: s(C_pages_device_RecycleBin), label: "回收站" },
    { path: '/devices/abnormal-reboot', element: s(C_pages_device_AbnormalReboot), label: "启动记录" },
    { path: '/devices/ue-detail/:sn', element: s(C_pages_device_UeDetail), label: "UE 详情", hidden: true },
    { path: '/devices/plug-and-play', element: s(C_pages_device_PlugAndPlay), label: "即插即用" },
    { path: '/devices/plug-and-play/add', element: s(C_pages_device_AddPolicyPage), label: "新增开站策略", hidden: true },
    { path: '/devices/plug-and-play/edit/:id', element: s(C_pages_device_AddPolicyPage), label: "编辑开站任务", hidden: true },
    { path: '/devices/plug-and-play/view/:id', element: s(C_pages_device_AddPolicyPage), label: "开站任务详情", hidden: true },
  ] },
  { key: 'alarms', label: '告警中心', icon: <AlertTriangle />, section: '监控', routes: [
    { path: '/alarms', element: s(AlarmsPage), label: "告警中心" },
    { path: '/alarms/history', element: s(C_pages_alarms_HistoricalAlarms), label: "历史告警" },
    { path: '/alarms/statistics', element: s(C_pages_alarms_AlarmStatistics), label: "告警统计" },
    { path: '/alarms/rules', element: s(C_pages_alarms_AlarmRules), label: "告警规则" },
    { path: '/alarms/rules/:id', element: s(C_pages_alarms_AlarmRuleDetail), label: "规则详情", hidden: true },
    { path: '/alarms/sync', element: s(C_pages_alarms_AlarmSync), label: "告警同步" },
    { path: '/alarms/custom-stats', element: s(C_pages_alarms_CustomAlarmStats), label: "自定义统计" },
  ] },
  { key: 'notifications', label: '通知中心', icon: <Bell />, section: '监控', routes: [
    { path: '/notifications', element: s(C_pages_notifications_index), label: "通知中心" },
    { path: '/notifications/templates', element: s(C_pages_notifications_Templates), label: "通知模板" },
    { path: '/notifications/templates/new', element: s(C_pages_notifications_TemplateCreate), label: "新建模板", hidden: true },
    { path: '/notifications/templates/:id', element: s(C_pages_notifications_TemplateEdit), label: "编辑模板", hidden: true },
    { path: '/notifications/history', element: s(C_pages_notifications_History), label: "发送历史" },
    { path: '/notifications/history/:id', element: s(C_pages_notifications_HistoryDetail), label: "发送记录详情", hidden: true },
  ] },
  { key: 'topology', label: '拓扑视图', icon: <Globe />, section: '监控', routes: [
    { path: '/topology', element: s(TopologyPage), label: "拓扑视图" },
    { path: '/topology/gis-map', element: s(C_pages_topology_GisMap), label: "GIS 地图" },
    { path: '/topology/canvas', element: s(C_pages_topology_TopologyCanvas), label: "拓扑画布" },
    { path: '/topology/domain', element: s(C_pages_topology_DomainManagement), label: "域管理" },
    { path: '/topology/site', element: s(C_pages_topology_SiteManagement), label: "站点管理" },
    { path: '/topology/site/:id', element: s(C_pages_topology_SiteDetail), label: "站点详情", hidden: true },
    { path: '/topology/settings', element: s(C_pages_topology_TopologySettings), label: "拓扑设置" },
    { path: '/topology/legend', element: s(C_pages_topology_LegendSystem), label: "图例系统" },
  ] },
  { key: 'config', label: '配置管理', icon: <Sliders />, section: '运维', routes: [
    { path: '/config', element: s(ConfigPage), label: "配置管理" },
    { path: '/config/param-sync', element: s(C_pages_config_ParamSync), label: "参数同步" },
    { path: '/config/live-param', element: s(C_pages_config_LiveParamConfig), label: "在线参数配置" },
    { path: '/config/batch-class', element: s(C_pages_config_BatchParamClass), label: "按参数类批量配置" },
    { path: '/config/batch-template', element: s(C_pages_config_BatchParamTemplate), label: "模板批量配置" },
    { path: '/config/batch-template/:id', element: s(C_pages_config_TemplateDetail), label: "模板详情", hidden: true },
    { path: '/config/param-list', element: s(C_pages_config_ParamList), label: "参数列表" },
    { path: '/config/command-mode', element: s(C_pages_config_CommandMode), label: "命令行模式" },
    { path: '/config/cell', element: s(C_pages_config_CellManagement), label: "小区管理" },
    { path: '/config/baseline', element: s(C_pages_config_BaselineManagement), label: "配置基线管理" },
    { path: '/config/baseline/:id', element: s(C_pages_config_BaselineDetail), label: "基线详情", hidden: true },
    { path: '/config/common', element: s(C_pages_config_CommonConfig), label: "通用配置" },
    { path: '/config/neighbor', element: s(C_pages_config_NeighborParams), label: "邻区参数" },
    { path: '/config/northbound', element: s(C_pages_config_NorthboundManagement), label: "北向/OSS 管理" },
    { path: '/config/auto-provision', element: s(C_pages_config_AutoProvisioning), label: "自动开站" },
    { path: '/config/auto-provision/:id', element: s(C_pages_config_ProvisioningTaskDetail), label: "开站任务详情", hidden: true },
    { path: '/config/interop-test', element: s(C_pages_config_InteropTesting), label: "互操作测试" },
  ] },
  { key: 'mml', label: 'MML 脚本', icon: <Terminal />, section: '运维', routes: [
    { path: '/mml', element: s(MMLPage), label: "MML 脚本" },
    { path: '/mml/script', element: s(C_pages_mml_ScriptTask), label: "脚本库" },
    { path: '/mml/task-records', element: s(C_pages_mml_TaskRecord), label: "任务记录" },
    { path: '/mml/console', element: s(C_pages_mml_Console), label: "MML控制台" },
    { path: '/mml/commands-tree', element: s(C_pages_mml_CommandTree), label: "命令树" },
    { path: '/mml/command-detail/:id', element: s(C_pages_mml_CommandDetail), label: "命令详情", hidden: true },
    { path: '/mml/private-command', element: s(C_pages_mml_PrivateCommand), label: "私有命令" },
    { path: '/mml/private-command/:id', element: s(C_pages_mml_PrivateCommandDetail), label: "私有命令详情", hidden: true },
    { path: '/mml/admin/catalog', element: s(C_pages_mml_admin_Catalog), label: "命令字典管理" },
    { path: '/mml/admin/command/:id', element: s(C_pages_mml_admin_CatalogCommandDetail), label: "字典命令详情", hidden: true },
  ] },
  { key: 'software', label: '软件版本', icon: <Package />, section: '运维', routes: [
    { path: '/software', element: s(SoftwarePage), label: "软件版本" },
    { path: '/software/version', element: s(C_pages_software_VersionQuery), label: "版本查询" },
    { path: '/software/version/:id', element: s(C_pages_software_VersionDetail), label: "版本详情", hidden: true },
    { path: '/software/upgrade-plan', element: s(C_pages_software_UpgradePlan), label: "升级计划" },
    { path: '/software/upgrade-plan/:id', element: s(C_pages_software_UpgradeTaskDetail), label: "任务详情", hidden: true },
    { path: '/software/activation', element: s(C_pages_software_ActivationPlan), label: "激活计划" },
    { path: '/software/firmware', element: s(C_pages_software_FirmwareUpload), label: "固件管理" },
    { path: '/software/rollback', element: s(C_pages_software_VersionRollback), label: "版本回退" },
  ] },
  { key: 'backup', label: '备份管理', icon: <FileStack />, section: '运维', routes: [
    { path: '/backup', element: s(BackupPage), label: "备份管理" },
    { path: '/backup/tasks', element: s(C_pages_backup_BackupTasks), label: "备份任务" },
    { path: '/backup/tasks/:id', element: s(C_pages_backup_BackupTaskDetail), label: "备份任务详情", hidden: true },
    { path: '/backup/schedule', element: s(C_pages_backup_BackupSchedule), label: "备份调度" },
    { path: '/backup/ftp', element: s(C_pages_backup_FTPConfig), label: "FTP 配置" },
    { path: '/backup/restore', element: s(C_pages_backup_RestoreData), label: "配置还原" },
    { path: '/backup/policy', element: s(C_pages_backup_BackupPolicy), label: "备份策略" },
    { path: '/backup/config-snapshots', element: s(C_pages_backup_ConfigSnapshotLibrary), label: "配置快照库" },
  ] },
  { key: 'ops', label: '运维工具箱', icon: <Wrench />, section: '运维', routes: [
    { path: '/ops', element: s(OpsPage), label: "运维工具箱" },
    { path: '/ops/templates', element: s(C_pages_ops_Templates), label: "运维模板" },
    { path: '/ops/templates/:id', element: s(C_pages_ops_TemplateDetail), label: "模板详情", hidden: true },
    { path: '/ops/tasks', element: s(C_pages_ops_TaskManagement), label: "自动化任务" },
    { path: '/ops/tasks/:id', element: s(C_pages_ops_TaskDetail), label: "任务详情", hidden: true },
    { path: '/ops/commands', element: s(C_pages_ops_CommandManagement), label: "命令记录" },
    { path: '/ops/commands/:id', element: s(C_pages_ops_CommandDetail), label: "命令详情", hidden: true },
    { path: '/ops/network-diagnosis', element: s(C_pages_ops_NetworkDiagnosis), label: "网络诊断" },
    { path: '/ops/downloads', element: s(C_pages_ops_Downloads), label: "运维下载" },
    { path: '/ops/message-trace', element: s(C_pages_ops_MessageTrace), label: "报文跟踪" },
    { path: '/ops/message-trace/:id', element: s(C_pages_ops_MessageTraceDetail), label: "报文跟踪详情", hidden: true },
    { path: '/ops/aggregation-trigger', element: s(C_pages_ops_AggregationTrigger), label: "PM 聚合触发" },
  ] },
  { key: 'transfer', label: '文件传输', icon: <ArrowLeftRight />, section: '运维', routes: [
    { path: '/transfer/center', element: s(C_pages_transfer_FileTransferCenter), label: "文件传输中心" },
    { path: '/transfer/center/task/:id', element: s(C_pages_transfer_TaskDetail), label: "任务详情", hidden: true },
    { path: '/transfer/file-management', element: s(C_pages_transfer_FileManagement), label: "文件管理" },
    { path: '/transfer/template-management', element: s(C_pages_transfer_TemplateManagement), label: "模板定义管理" },
    { path: '/transfer/template-management/:typeCode', element: s(C_pages_transfer_TemplateDetail), label: "模板详情", hidden: true },
  ] },
  { key: 'performance', label: '性能管理', icon: <LineChart />, section: '数据', routes: [
    { path: '/performance', element: s(PerformancePage), label: "性能管理" },
    { path: '/performance/kpi-standard', element: s(C_pages_performance_KPIStandardReport), label: "KPI 标准库" },
    { path: '/performance/kpi-standard/detail/:deviceType/:indicatorId', element: s(C_pages_performance_KPIIndicatorDetail), label: "指标详情", hidden: true },
    { path: '/performance/kpi-station', element: s(C_pages_performance_KPIStationReport), label: "基站性能上报" },
    { path: '/performance/query', element: s(C_pages_performance_KPIQuery), label: "指标查询" },
    { path: '/performance/charts', element: s(C_pages_performance_PerformanceCharts), label: "性能趋势图" },
    { path: '/performance/threshold', element: s(C_pages_performance_ThresholdConfig), label: "阈值配置" },
    { path: '/performance/files', element: s(C_pages_performance_PerformanceFiles), label: "PM 文件" },
    { path: '/performance/task-config', element: s(C_pages_performance_PerformanceTaskConfig), label: "采集任务配置" },
    { path: '/performance/device-view', element: s(C_pages_performance_DeviceView), label: "设备性能查看" },
    { path: '/performance/pm-adhoc', element: s(C_pages_performance_PmAdhoc), label: "自定义聚合任务" },
    { path: '/performance/pm-adhoc/new', element: s(C_pages_performance_PmAdhocWizard), label: "新建聚合任务", hidden: true },
    { path: '/performance/pm-adhoc/:id/edit', element: s(C_pages_performance_PmAdhocWizard), label: "编辑聚合任务", hidden: true },
    // v1 兼容重定向：旧 pm-dashboard URL → 性能管理（非真页面）
    { path: '/performance/pm-dashboard', element: <Navigate to="/performance" replace />, label: "PM 仪表盘（重定向）", hidden: true },
    { path: '/performance/pm-dashboard/:id', element: <Navigate to="/performance" replace />, label: "PM 仪表盘（重定向）", hidden: true },
  ] },
  { key: 'mr', label: '测量报告', icon: <Radio />, section: '数据', routes: [
    { path: '/mr', element: s(MRPage), label: "测量报告" },
    { path: '/mr/indicators', element: s(C_pages_mr_Indicators), label: "指标库" },
    { path: '/mr/device-mapping', element: s(C_pages_mr_DeviceMapping), label: "设备小区映射" },
    { path: '/mr/device-mapping/:id', element: s(C_pages_mr_DeviceMappingDetail), label: "映射详情", hidden: true },
    { path: '/mr/variables', element: s(C_pages_mr_Variables), label: "测量变量" },
    { path: '/mr/reports', element: s(C_pages_mr_Reports), label: "分析报告" },
    { path: '/mr/reports/:id', element: s(C_pages_mr_ReportDetail), label: "报告详情", hidden: true },
    { path: '/mr/files', element: s(C_pages_mr_Files), label: "采集文件" },
    { path: '/mr/files/:sn', element: s(C_pages_mr_DeviceFiles), label: "设备文件明细", hidden: true },
  ] },
  { key: 'reports', label: '报表中心', icon: <FileBarChart />, section: '数据', routes: [
    { path: '/reports', element: s(ReportsPage), label: "报表中心" },
    { path: '/reports/lte-standard', element: s(C_pages_reports_LteStandardReport), label: "LTE 标准报表" },
    { path: '/reports/station', element: s(C_pages_reports_StationReport), label: "单站报表" },
    { path: '/reports/historical-kpi', element: s(C_pages_reports_HistoricalKpi), label: "历史 KPI" },
    { path: '/reports/poll-stats', element: s(C_pages_reports_PollStatistics), label: "轮询统计" },
    { path: '/reports/record/:id', element: s(C_pages_reports_ReportRecordDetail), label: "报表记录详情", hidden: true },
    { path: '/reports/station/:region', element: s(C_pages_reports_StationDetail), label: "区域明细", hidden: true },
    { path: '/reports/poll-stats/:id', element: s(C_pages_reports_PollStatDetail), label: "采集任务详情", hidden: true },
  ] },
  { key: 'files', label: '文件管理', icon: <FolderOpen />, section: '数据', routes: [
    { path: '/files', element: s(FilesPage), label: "文件管理" },
    { path: '/files/config-retrieval', element: s(C_pages_files_ConfigRetrieval), label: "配置文件获取" },
    { path: '/files/config-distribution', element: s(C_pages_files_ConfigDistribution), label: "配置文件下发" },
    { path: '/files/mr-retrieval', element: s(C_pages_files_MRRetrieval), label: "MR 文件获取" },
    { path: '/files/user-files', element: s(C_pages_files_UserFiles), label: "用户文件" },
    { path: '/files/device-files', element: s(C_pages_files_DeviceFiles), label: "设备文件" },
    { path: '/files/log-retrieval', element: s(C_pages_files_LogRetrieval), label: "日志获取" },
    { path: '/files/perf-retrieval', element: s(C_pages_files_PerfRetrieval), label: "性能文件获取" },
    { path: '/files/detail/:id', element: s(C_pages_files_FileDetail), label: "文件详情", hidden: true },
    { path: '/files/mr-detail/:sn', element: s(C_pages_files_MRDetail), label: "MR 文件详情", hidden: true },
    { path: '/files/perf-detail/:sn', element: s(C_pages_files_PerfDetail), label: "性能文件详情", hidden: true },
  ] },
  { key: 'product', label: '产品中心', icon: <Boxes />, section: '数据', routes: [
    { path: '/product/products', element: s(C_pages_product_Products), label: "产品装配件" },
    { path: '/product/products/:id', element: s(C_pages_product_ProductDetail), label: "产品详情", hidden: true },
    { path: '/product/param-model', element: s(C_pages_product_ParamModel), label: "参数模型库" },
    { path: '/product/param-model/:name', element: s(C_pages_product_ParamModelMappings), label: "参数映射", hidden: true },
    { path: '/product/standard-params', element: s(C_pages_product_StandardParams), label: "标准参数树" },
    { path: '/product/kpi-library', element: s(C_pages_product_KpiLibrary), label: "KPI 指标库" },
    { path: '/product/kpi-library/:tech/:platform', element: s(C_pages_product_KpiIndicators), label: "KPI 指标详情", hidden: true },
    { path: '/product/alarm-library', element: s(C_pages_product_AlarmLibrary), label: "告警库" },
    { path: '/product/alarm-library/:neType', element: s(C_pages_product_AlarmDefinitions), label: "告警定义", hidden: true },
    { path: '/product/orphan-devices', element: s(C_pages_product_OrphanDevices), label: "孤儿设备" },
  ] },
  { key: 'logs', label: '系统日志', icon: <ScrollText />, section: '系统', routes: [
    { path: '/logs', element: s(LogsPage), label: "系统日志" },
    { path: '/logs/system', element: s(C_pages_logs_SystemLog), label: "系统日志" },
    { path: '/logs/operation', element: s(C_pages_logs_OperationLog), label: "操作审计" },
    { path: '/logs/device', element: s(C_pages_logs_DeviceLog), label: "基站日志" },
    { path: '/logs/device/:id', element: s(C_pages_logs_DeviceLogDetail), label: "基站日志详情", hidden: true },
    { path: '/logs/exception', element: s(C_pages_logs_ExceptionLog), label: "异常重启" },
    { path: '/logs/exception/:id', element: s(C_pages_logs_ExceptionLogDetail), label: "异常重启详情", hidden: true },
    { path: '/logs/event', element: s(C_pages_logs_EventLog), label: "设备事件" },
    { path: '/logs/event/:id', element: s(C_pages_logs_EventLogDetail), label: "设备事件详情", hidden: true },
    { path: '/logs/config', element: s(C_pages_logs_LogConfig), label: "保留配置" },
  ] },
  { key: 'license', label: '许可证', icon: <KeySquare />, section: '系统', routes: [
    { path: '/license', element: s(LicensePage), label: "许可证" },
    { path: '/license/history', element: s(C_pages_license_History), label: "许可证历史", hidden: true },
  ] },
  { key: 'system', label: '系统管理', icon: <Settings />, section: '系统', routes: [
    { path: '/system', element: s(SystemPage), label: "系统管理" },
    { path: '/system/device-class', element: s(C_pages_system_DeviceClassification), label: "设备分类" },
    { path: '/system/users', element: s(C_pages_system_UserManagement), label: "用户管理" },
    { path: '/system/users/:id', element: s(C_pages_system_UserDetail), label: "用户详情", hidden: true },
    { path: '/system/groups', element: s(C_pages_system_GroupManagement), label: "用户组" },
    { path: '/system/groups/:id', element: s(C_pages_system_GroupDetail), label: "用户组详情", hidden: true },
    { path: '/system/roles', element: s(C_pages_system_RolePermission), label: "角色与权限" },
    { path: '/system/roles/:id', element: s(C_pages_system_RoleDetail), label: "角色详情", hidden: true },
    { path: '/system/operation-log', element: s(C_pages_system_OperationLog), label: "操作日志" },
    { path: '/system/config', element: s(C_pages_system_SystemConfig), label: "系统配置" },
    { path: '/system/ui-custom', element: s(C_pages_system_UICustomization), label: "界面定制" },
    { path: '/system/menus', element: s(C_pages_system_MenuManagement), label: "菜单管理" },
    { path: '/system/dashboard', element: s(C_pages_system_SystemDashboard), label: "系统概览" },
    { path: '/system/api-management', element: s(C_pages_system_ApiManagement), label: "API 管理" },
    { path: '/system/data-dictionary', element: s(C_pages_system_DataDictionary), label: "数据字典" },
    { path: '/system/dict-loader', element: s(C_pages_system_DictLoader), label: "字典 Loader 重载" },
    { path: '/system/kpi-config', element: s(C_pages_system_KpiConfig), label: "首页 KPI 配置" },
  ] },
]

export const SECTIONS = ['监控', '运维', '数据', '系统']
export const ALL_ROUTES: RouteDef[] = MODULES.flatMap((m) => m.routes)
