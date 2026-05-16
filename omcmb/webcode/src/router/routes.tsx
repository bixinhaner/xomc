import React, { Suspense } from 'react';
import { Navigate, type RouteObject } from 'react-router-dom';
import { Spin } from 'antd';
import AppShell from '@/components/Layout';
import PrivateRoute from './PrivateRoute';
import MenuBootstrap from '@/components/MenuBootstrap';
import ErrorBoundary from '@/components/common/ErrorBoundary';
import LoginPage from '@/pages/login';
import NotFound from '@/pages/error/NotFound';
import Forbidden from '@/pages/error/Forbidden';

// ---------------------------------------------------------------------------
// Real lazy-loaded page imports
// ---------------------------------------------------------------------------

// Dashboard
const Dashboard = React.lazy(() => import('@/pages/dashboard'));

// Device Management
const DeviceList         = React.lazy(() => import('@/pages/device/DeviceList'));
const DeviceRegister     = React.lazy(() => import('@/pages/device/DeviceRegistration'));
const DeviceGroup        = React.lazy(() => import('@/pages/device/DeviceGrouping'));
const DeviceDetail       = React.lazy(() => import('@/pages/device/DeviceDetail'));
const NEManagement       = React.lazy(() => import('@/pages/device/NEManagement'));
const OnlineMonitor      = React.lazy(() => import('@/pages/device/OnlineMonitoring'));
const Commissioning      = React.lazy(() => import('@/pages/device/Commissioning'));
const HandoverMgmt       = React.lazy(() => import('@/pages/device/HandoverManagement'));
const ResourceStats      = React.lazy(() => import('@/pages/device/ResourceStatistics'));
const ImportExport       = React.lazy(() => import('@/pages/device/ImportExport'));
const DeviceRules        = React.lazy(() => import('@/pages/device/DeviceRules'));
const RecycleBin         = React.lazy(() => import('@/pages/device/RecycleBin'));
const UeDetail           = React.lazy(() => import('@/pages/device/UeDetail'));

// Alarm Management
const CurrentAlarms      = React.lazy(() => import('@/pages/alarm/CurrentAlarms'));
const HistoricalAlarms   = React.lazy(() => import('@/pages/alarm/HistoricalAlarms'));
const AlarmStatistics    = React.lazy(() => import('@/pages/alarm/AlarmStatistics'));
const AlarmRules         = React.lazy(() => import('@/pages/alarm/AlarmRules'));
const AlarmSync          = React.lazy(() => import('@/pages/alarm/AlarmSync'));
const CustomAlarmStats   = React.lazy(() => import('@/pages/alarm/CustomAlarmStats'));

// Configuration Management
const ParamSync          = React.lazy(() => import('@/pages/config/ParamSync'));
const LiveParamConfig    = React.lazy(() => import('@/pages/config/LiveParamConfig'));
const BatchParamClass    = React.lazy(() => import('@/pages/config/BatchParamClass'));
const BatchParamTemplate = React.lazy(() => import('@/pages/config/BatchParamTemplate'));
const ParamList          = React.lazy(() => import('@/pages/config/ParamList'));
const CommandMode        = React.lazy(() => import('@/pages/config/CommandMode'));
const CellManagement     = React.lazy(() => import('@/pages/config/CellManagement'));
const BaselineMgmt       = React.lazy(() => import('@/pages/config/BaselineManagement'));
const CommonConfig       = React.lazy(() => import('@/pages/config/CommonConfig'));
const NeighborParams     = React.lazy(() => import('@/pages/config/NeighborParams'));
const NorthboundMgmt     = React.lazy(() => import('@/pages/config/NorthboundManagement'));
const AutoProvisioning   = React.lazy(() => import('@/pages/config/AutoProvisioning'));
const InteropTesting     = React.lazy(() => import('@/pages/config/InteropTesting'));

// Performance Management
const KPIStandard        = React.lazy(() => import('@/pages/performance/KPIStandardReport'));
const KPIIndicatorDetail = React.lazy(() => import('@/pages/performance/KPIStandardReport/IndicatorDetail'));
const KPIStation         = React.lazy(() => import('@/pages/performance/KPIStationReport'));
const KPIQuery            = React.lazy(() => import('@/pages/performance/KPIQuery'));
const PerformanceCharts  = React.lazy(() => import('@/pages/performance/PerformanceCharts'));
const ThresholdConfig    = React.lazy(() => import('@/pages/performance/ThresholdConfig'));
const PerformanceFiles   = React.lazy(() => import('@/pages/performance/PerformanceFiles'));
const PerfTaskConfig     = React.lazy(() => import('@/pages/performance/PerformanceTaskConfig'));

// MML Management
const MMLConsole         = React.lazy(() => import('@/pages/mml/Console'));
const MMLScript          = React.lazy(() => import('@/pages/mml/ScriptTask'));
const MMLTaskRecord      = React.lazy(() => import('@/pages/mml/TaskRecord'));
const MMLCommands        = React.lazy(() => import('@/pages/mml/CommandTree'));
const MMLPrivateCommand  = React.lazy(() => import('@/pages/mml/PrivateCommand'));
const MMLAdminCatalog    = React.lazy(() => import('@/pages/mml/admin/catalog'));

// Topology Management
const GISMapView         = React.lazy(() => import('@/pages/topology/GISMapView'));
const TopologyCanvasPage = React.lazy(() => import('@/pages/topology/TopologyCanvas'));
const DomainManagement   = React.lazy(() => import('@/pages/topology/DomainManagement'));
const SiteManagement     = React.lazy(() => import('@/pages/topology/SiteManagement'));
const TopologySettings   = React.lazy(() => import('@/pages/topology/TopologySettings'));
const LegendSystem       = React.lazy(() => import('@/pages/topology/LegendSystem'));

// Backup & Restore
const BackupTasks        = React.lazy(() => import('@/pages/backup/BackupTasks'));
const BackupSchedule     = React.lazy(() => import('@/pages/backup/BackupSchedule'));
const FTPConfig          = React.lazy(() => import('@/pages/backup/FTPConfig'));
const RestoreData        = React.lazy(() => import('@/pages/backup/RestoreData'));
const BackupPolicy       = React.lazy(() => import('@/pages/backup/BackupPolicy'));

// Software Management — 菜单已移除，保留 lazy import 以便路由可达
const VersionQuery       = React.lazy(() => import('@/pages/software/VersionQuery'));
const UpgradePlan        = React.lazy(() => import('@/pages/software/UpgradePlan'));
const ActivationPlan     = React.lazy(() => import('@/pages/software/ActivationPlan'));
const FirmwareUpload     = React.lazy(() => import('@/pages/software/FirmwareUpload'));
const VersionRollback    = React.lazy(() => import('@/pages/software/VersionRollback'));

// Unified File Transfer Preview
const FileTransferCenter = React.lazy(() => import('@/pages/transfer/FileTransferCenter'));
const TransferTemplateManagement = React.lazy(() => import('@/pages/transfer/TemplateDefinitionManagement'));

// File Management
const ConfigRetrieval    = React.lazy(() => import('@/pages/file/ConfigRetrieval'));
const ConfigDistribution = React.lazy(() => import('@/pages/file/ConfigDistribution'));
const LogRetrieval       = React.lazy(() => import('@/pages/file/LogRetrieval'));
const PerfRetrieval      = React.lazy(() => import('@/pages/file/PerfRetrieval'));
const MRRetrieval        = React.lazy(() => import('@/pages/file/MRRetrieval'));
const UserFiles          = React.lazy(() => import('@/pages/file/UserFiles'));
const DeviceFiles        = React.lazy(() => import('@/pages/file/DeviceFiles'));

// Log Management
const DeviceLog          = React.lazy(() => import('@/pages/log/DeviceLog'));
const ExceptionLog       = React.lazy(() => import('@/pages/log/ExceptionLog'));
const EventLog           = React.lazy(() => import('@/pages/log/EventLog'));
const OperationLog       = React.lazy(() => import('@/pages/log/OperationLog'));
const SystemLog          = React.lazy(() => import('@/pages/log/SystemLog'));
const LogConfig          = React.lazy(() => import('@/pages/log/LogConfig'));

// System Management
const DeviceClassification = React.lazy(() => import('@/pages/system/DeviceClassification'));
const UserManagement     = React.lazy(() => import('@/pages/system/UserManagement'));
const GroupManagement    = React.lazy(() => import('@/pages/system/GroupManagement'));
const RolePermission     = React.lazy(() => import('@/pages/system/RolePermission'));
const SysOperationLog    = React.lazy(() => import('@/pages/system/OperationLog'));
const SystemConfig       = React.lazy(() => import('@/pages/system/SystemConfig'));
const UICustomization    = React.lazy(() => import('@/pages/system/UICustomization'));
const MenuManagement     = React.lazy(() => import('@/pages/system/MenuManagement'));
const SystemDashboard    = React.lazy(() => import('@/pages/system/SystemDashboard'));
const ApiManagement      = React.lazy(() => import('@/pages/system/ApiManagement'));
const DataDictionary     = React.lazy(() => import('@/pages/system/DataDictionary'));

// Report Management
const LTEStandardReport  = React.lazy(() => import('@/pages/report/LTEStandardReport'));
const StationReport      = React.lazy(() => import('@/pages/report/StationReport'));
const HistoricalKPI      = React.lazy(() => import('@/pages/report/HistoricalKPI'));
const PollStatistics     = React.lazy(() => import('@/pages/report/PollStatistics'));

// MR Management
const MRIndicators       = React.lazy(() => import('@/pages/mr/Indicators'));
const MRDeviceMapping    = React.lazy(() => import('@/pages/mr/DeviceMapping'));
const MRVariables        = React.lazy(() => import('@/pages/mr/Variables'));
const MRReports          = React.lazy(() => import('@/pages/mr/Reports'));
const MRTasks            = React.lazy(() => import('@/pages/mr/Tasks'));
const MRFiles            = React.lazy(() => import('@/pages/mr/Files'));

// License Management
const LicenseList        = React.lazy(() => import('@/pages/license/LicenseList'));
const LicenseOperations  = React.lazy(() => import('@/pages/license/LicenseOperations'));
const LicenseLogs        = React.lazy(() => import('@/pages/license/LicenseLogs'));

// Device - Plug and Play
const PlugAndPlay        = React.lazy(() => import('@/pages/device/PlugAndPlay'));
const AddPolicyPage      = React.lazy(() => import('@/pages/device/PlugAndPlay/AddPolicyPage'));

// Notifications
const NotificationsPage  = React.lazy(() => import('@/pages/notifications'));

// Ops Management
const OpsTemplates       = React.lazy(() => import('@/pages/ops/Templates'));
const OpsCommands        = React.lazy(() => import('@/pages/ops/CommandManagement'));
const OpsTasks           = React.lazy(() => import('@/pages/ops/TaskManagement'));
const NetworkDiagnosis   = React.lazy(() => import('@/pages/ops/NetworkDiagnosis'));
const OpsDownloads       = React.lazy(() => import('@/pages/ops/Downloads'));
const OpsMessageTrace    = React.lazy(() => import('@/pages/ops/MessageTrace'));

// T-0098-P4 Product Center (super_admin only)
const ProductsPage       = React.lazy(() => import('@/pages/product/products'));
const ParamModelPage     = React.lazy(() => import('@/pages/product/param-model'));
const KpiLibraryPage     = React.lazy(() => import('@/pages/product/kpi-library'));
const AlarmLibraryPage   = React.lazy(() => import('@/pages/product/alarm-library'));
const OrphanDevicesPage  = React.lazy(() => import('@/pages/product/orphan-devices'));

// ---------------------------------------------------------------------------
// Loading fallback
// ---------------------------------------------------------------------------
const PageLoader = () => (
  <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%', minHeight: 300 }}>
    <Spin size="large" />
  </div>
);

function withSuspense(Component: React.ComponentType) {
  return (
    <ErrorBoundary>
      <Suspense fallback={<PageLoader />}>
        <Component />
      </Suspense>
    </ErrorBoundary>
  );
}

// T-0098-P4-02：super_admin 治理路由组守卫包装
function withSuperAdmin(Component: React.ComponentType) {
  return (
    <PrivateRoute requireSuperAdmin>
      <ErrorBoundary>
        <Suspense fallback={<PageLoader />}>
          <Component />
        </Suspense>
      </ErrorBoundary>
    </PrivateRoute>
  );
}

function withAdminRole(Component: React.ComponentType) {
  return (
    <PrivateRoute requireAdmin>
      <ErrorBoundary>
        <Suspense fallback={<PageLoader />}>
          <Component />
        </Suspense>
      </ErrorBoundary>
    </PrivateRoute>
  );
}

// ---------------------------------------------------------------------------
// Route definitions
// ---------------------------------------------------------------------------
export const routes: RouteObject[] = [
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    path: '/',
    element: (
      <PrivateRoute>
        <MenuBootstrap>
          <AppShell />
        </MenuBootstrap>
      </PrivateRoute>
    ),
    children: [
      { index: true, element: <Navigate to="/dashboard" replace /> },
      { path: 'dashboard', element: withSuspense(Dashboard) },
      { path: '403', element: <Forbidden /> },

      // Device Management
      { path: 'device/list',       element: withSuspense(DeviceList) },
      { path: 'device/register',   element: withSuspense(DeviceRegister) },
      { path: 'device/group',      element: withSuspense(DeviceGroup) },
      { path: 'device/detail/:sn', element: withSuspense(DeviceDetail) },
      { path: 'device/ne',         element: withSuspense(NEManagement) },
      { path: 'device/monitor',    element: withSuspense(OnlineMonitor) },
      { path: 'device/commission', element: withSuspense(Commissioning) },
      { path: 'device/handover',   element: withSuspense(HandoverMgmt) },
      { path: 'device/stats',      element: withSuspense(ResourceStats) },
      { path: 'device/import',     element: withSuspense(ImportExport) },
      { path: 'device/rules',      element: withSuspense(DeviceRules) },
      { path: 'device/recycle',    element: withSuspense(RecycleBin) },
      { path: 'device/ue-detail/:sn', element: withSuspense(UeDetail) },

      // Alarm Management
      { path: 'alarm/current',    element: withSuspense(CurrentAlarms) },
      { path: 'alarm/history',    element: withSuspense(HistoricalAlarms) },
      { path: 'alarm/statistics', element: withSuspense(AlarmStatistics) },
      { path: 'alarm/rules',      element: withSuspense(AlarmRules) },
      { path: 'alarm/sync',       element: withSuspense(AlarmSync) },
      { path: 'alarm/custom-stats', element: withSuspense(CustomAlarmStats) },

      // Configuration Management
      { path: 'config/param-sync',       element: withSuspense(ParamSync) },
      { path: 'config/live-param',       element: withSuspense(LiveParamConfig) },
      { path: 'config/batch-class',      element: withSuspense(BatchParamClass) },
      { path: 'config/batch-template',   element: withSuspense(BatchParamTemplate) },
      { path: 'config/param-list',       element: withSuspense(ParamList) },
      { path: 'config/command-mode',     element: withSuspense(CommandMode) },
      { path: 'config/cell',             element: withSuspense(CellManagement) },
      { path: 'config/baseline',         element: withSuspense(BaselineMgmt) },
      { path: 'config/common',           element: withSuspense(CommonConfig) },
      { path: 'config/neighbor',         element: withSuspense(NeighborParams) },
      { path: 'config/northbound',       element: withSuspense(NorthboundMgmt) },
      { path: 'config/auto-provision',   element: withSuspense(AutoProvisioning) },
      { path: 'config/interop-test',     element: withSuspense(InteropTesting) },

      // Performance Management
      { path: 'performance/kpi-standard',  element: withSuspense(KPIStandard) },
      { path: 'performance/kpi-standard/detail/:deviceType/:indicatorId', element: withSuspense(KPIIndicatorDetail) },
      { path: 'performance/kpi-station',   element: withSuspense(KPIStation) },
      { path: 'performance/query',         element: withSuspense(KPIQuery) },
      { path: 'performance/charts',        element: withSuspense(PerformanceCharts) },
      { path: 'performance/threshold',     element: withSuspense(ThresholdConfig) },
      { path: 'performance/files',         element: withSuspense(PerformanceFiles) },
      { path: 'performance/task-config',   element: withSuspense(PerfTaskConfig) },

      // MML Management
      { path: 'mml/console',         element: withSuspense(MMLConsole) },
      { path: 'mml/script',          element: withSuspense(MMLScript) },
      { path: 'mml/task-records',    element: withSuspense(MMLTaskRecord) },
      { path: 'mml/commands',        element: withSuspense(MMLCommands) },
      { path: 'mml/private-command', element: withSuspense(MMLPrivateCommand) },
      { path: 'mml/admin/catalog',   element: withAdminRole(MMLAdminCatalog) },

      // Topology Management
      { path: 'topology/gis-map',  element: withSuspense(GISMapView) },
      { path: 'topology/canvas',   element: withSuspense(TopologyCanvasPage) },
      { path: 'topology/domain',   element: withSuspense(DomainManagement) },
      { path: 'topology/site',     element: withSuspense(SiteManagement) },
      { path: 'topology/settings', element: withSuspense(TopologySettings) },
      { path: 'topology/legend',   element: withSuspense(LegendSystem) },

      // Backup & Restore
      { path: 'backup/tasks',    element: withSuspense(BackupTasks) },
      { path: 'backup/schedule', element: withSuspense(BackupSchedule) },
      { path: 'backup/ftp',      element: withSuspense(FTPConfig) },
      { path: 'backup/restore',  element: withSuspense(RestoreData) },
      { path: 'backup/policy',   element: withSuspense(BackupPolicy) },

      // Software Management — 菜单已移除，路由保留以便直接 URL 访问
      { path: 'software/version',       element: withSuspense(VersionQuery) },
      { path: 'software/upgrade-plan',  element: withSuspense(UpgradePlan) },
      { path: 'software/activation',    element: withSuspense(ActivationPlan) },
      { path: 'software/firmware',      element: withSuspense(FirmwareUpload) },
      { path: 'software/rollback',      element: withSuspense(VersionRollback) },

      // File Management
      { path: 'file/config-retrieval',    element: withSuspense(ConfigRetrieval) },
      { path: 'file/config-distribution', element: withSuspense(ConfigDistribution) },
      { path: 'file/log-retrieval',       element: withSuspense(LogRetrieval) },
      { path: 'file/perf-retrieval',      element: withSuspense(PerfRetrieval) },
      { path: 'file/mr-retrieval',        element: withSuspense(MRRetrieval) },
      { path: 'file/user-files',          element: withSuspense(UserFiles) },
      { path: 'file/device-files',        element: withSuspense(DeviceFiles) },

      // Log Management
      { path: 'log/device',      element: withSuspense(DeviceLog) },
      { path: 'log/exception',   element: withSuspense(ExceptionLog) },
      { path: 'log/event',       element: withSuspense(EventLog) },
      { path: 'log/operation',   element: withSuspense(OperationLog) },
      { path: 'log/system',      element: withSuspense(SystemLog) },
      { path: 'log/config',      element: withSuspense(LogConfig) },

      // System Management
      { path: 'system/device-class',   element: withSuspense(DeviceClassification) },
      { path: 'system/users',          element: withSuspense(UserManagement) },
      { path: 'system/groups',         element: withSuspense(GroupManagement) },
      { path: 'system/roles',          element: withSuspense(RolePermission) },
      { path: 'system/operation-log',  element: withSuspense(SysOperationLog) },
      { path: 'system/config',         element: withSuspense(SystemConfig) },
      { path: 'system/ui-custom',      element: withSuspense(UICustomization) },
      { path: 'system/menus',          element: withSuspense(MenuManagement) },
      { path: 'system/dashboard',      element: withSuspense(SystemDashboard) },
      { path: 'system/api-management', element: withSuspense(ApiManagement) },
      { path: 'system/data-dictionary', element: withSuspense(DataDictionary) },

      // Report Management
      { path: 'report/lte-standard',   element: withSuspense(LTEStandardReport) },
      { path: 'report/station',         element: withSuspense(StationReport) },
      { path: 'report/historical-kpi',  element: withSuspense(HistoricalKPI) },
      { path: 'report/poll-stats',      element: withSuspense(PollStatistics) },

      // MR Management
      { path: 'mr/indicators',     element: withSuspense(MRIndicators) },
      { path: 'mr/device-mapping', element: withSuspense(MRDeviceMapping) },
      { path: 'mr/variables',      element: withSuspense(MRVariables) },
      { path: 'mr/reports',        element: withSuspense(MRReports) },
      { path: 'mr/tasks',          element: withSuspense(MRTasks) },
      { path: 'mr/files',          element: withSuspense(MRFiles) },

      // License Management
      { path: 'license/list',       element: withSuspense(LicenseList) },
      { path: 'license/operations', element: withSuspense(LicenseOperations) },
      { path: 'license/logs',       element: withSuspense(LicenseLogs) },

      // Device - Plug and Play
      { path: 'device/plug-and-play', element: withSuspense(PlugAndPlay) },
      { path: 'device/plug-and-play/add', element: withSuspense(AddPolicyPage) },
      { path: 'device/plug-and-play/edit/:id', element: withSuspense(AddPolicyPage) },
      { path: 'device/plug-and-play/view/:id', element: withSuspense(AddPolicyPage) },

      // Notifications
      { path: 'notifications',         element: withSuspense(NotificationsPage) },

      // Transfer Management
      { path: 'transfer/center',             element: withSuspense(FileTransferCenter) },
      { path: 'transfer/template-management', element: withAdminRole(TransferTemplateManagement) },

      // Ops Management
      { path: 'ops/templates',         element: withSuspense(OpsTemplates) },
      { path: 'ops/commands',          element: withSuspense(OpsCommands) },
      { path: 'ops/tasks',             element: withSuspense(OpsTasks) },
      { path: 'ops/network-diagnosis', element: withSuspense(NetworkDiagnosis) },
      { path: 'ops/downloads',         element: withSuspense(OpsDownloads) },
      { path: 'ops/message-trace',     element: withSuspense(OpsMessageTrace) },

      // T-0098-P4 Product Center (super_admin only)
      { path: 'product/products',       element: withSuperAdmin(ProductsPage) },
      { path: 'product/param-model',    element: withSuperAdmin(ParamModelPage) },
      { path: 'product/kpi-library',    element: withSuperAdmin(KpiLibraryPage) },
      { path: 'product/alarm-library',  element: withSuperAdmin(AlarmLibraryPage) },
      { path: 'product/orphan-devices', element: withSuperAdmin(OrphanDevicesPage) },

      // 403 / 404
      { path: '403', element: <Forbidden /> },
      { path: '*', element: <NotFound /> },
    ],
  },
];
