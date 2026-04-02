import React, { Suspense } from 'react';
import { Navigate, type RouteObject } from 'react-router-dom';
import { Spin } from 'antd';
import AppShell from '@/components/Layout';
import PrivateRoute from './PrivateRoute';
import ErrorBoundary from '@/components/common/ErrorBoundary';
import LoginPage from '@/pages/login';
import NotFound from '@/pages/error/NotFound';

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
const AlarmLibrary       = React.lazy(() => import('@/pages/alarm/AlarmSupportLibrary'));
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
const DataModelMgmt      = React.lazy(() => import('@/pages/config/DataModelManagement'));
const NorthboundMgmt     = React.lazy(() => import('@/pages/config/NorthboundManagement'));
const AutoProvisioning   = React.lazy(() => import('@/pages/config/AutoProvisioning'));
const InteropTesting     = React.lazy(() => import('@/pages/config/InteropTesting'));

// Performance Management
const KPIStandard        = React.lazy(() => import('@/pages/performance/KPIStandardReport'));
const KPIStation         = React.lazy(() => import('@/pages/performance/KPIStationReport'));
const KPIQuery            = React.lazy(() => import('@/pages/performance/KPIQuery'));
const PerformanceCharts  = React.lazy(() => import('@/pages/performance/PerformanceCharts'));
const ThresholdConfig    = React.lazy(() => import('@/pages/performance/ThresholdConfig'));
const PerformanceFiles   = React.lazy(() => import('@/pages/performance/PerformanceFiles'));
const PerfTaskConfig     = React.lazy(() => import('@/pages/performance/PerformanceTaskConfig'));

// MML Management
const MMLConsole         = React.lazy(() => import('@/pages/mml/Console'));
const MMLScript          = React.lazy(() => import('@/pages/mml/ScriptTask'));
const MMLCommands        = React.lazy(() => import('@/pages/mml/CommandTree'));

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

// Software Management
const VersionQuery       = React.lazy(() => import('@/pages/software/VersionQuery'));
const UpgradePlan        = React.lazy(() => import('@/pages/software/UpgradePlan'));
const ActivationPlan     = React.lazy(() => import('@/pages/software/ActivationPlan'));
const FirmwareUpload     = React.lazy(() => import('@/pages/software/FirmwareUpload'));
const VersionRollback    = React.lazy(() => import('@/pages/software/VersionRollback'));

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

// Ops Management
const OpsTemplates       = React.lazy(() => import('@/pages/ops/Templates'));
const OpsCommands        = React.lazy(() => import('@/pages/ops/CommandManagement'));
const OpsTasks           = React.lazy(() => import('@/pages/ops/TaskManagement'));
const NetworkDiagnosis   = React.lazy(() => import('@/pages/ops/NetworkDiagnosis'));
const OpsDownloads       = React.lazy(() => import('@/pages/ops/Downloads'));

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
        <AppShell />
      </PrivateRoute>
    ),
    children: [
      { index: true, element: <Navigate to="/dashboard" replace /> },
      { path: 'dashboard', element: withSuspense(Dashboard) },

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
      { path: 'alarm/library',    element: withSuspense(AlarmLibrary) },
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
      { path: 'config/data-model',       element: withSuspense(DataModelMgmt) },
      { path: 'config/northbound',       element: withSuspense(NorthboundMgmt) },
      { path: 'config/auto-provision',   element: withSuspense(AutoProvisioning) },
      { path: 'config/interop-test',     element: withSuspense(InteropTesting) },

      // Performance Management
      { path: 'performance/kpi-standard',  element: withSuspense(KPIStandard) },
      { path: 'performance/kpi-station',   element: withSuspense(KPIStation) },
      { path: 'performance/query',         element: withSuspense(KPIQuery) },
      { path: 'performance/charts',        element: withSuspense(PerformanceCharts) },
      { path: 'performance/threshold',     element: withSuspense(ThresholdConfig) },
      { path: 'performance/files',         element: withSuspense(PerformanceFiles) },
      { path: 'performance/task-config',   element: withSuspense(PerfTaskConfig) },

      // MML Management
      { path: 'mml/console',   element: withSuspense(MMLConsole) },
      { path: 'mml/script',    element: withSuspense(MMLScript) },
      { path: 'mml/commands',  element: withSuspense(MMLCommands) },

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

      // Software Management
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

      // Ops Management
      { path: 'ops/templates',         element: withSuspense(OpsTemplates) },
      { path: 'ops/commands',          element: withSuspense(OpsCommands) },
      { path: 'ops/tasks',             element: withSuspense(OpsTasks) },
      { path: 'ops/network-diagnosis', element: withSuspense(NetworkDiagnosis) },
      { path: 'ops/downloads',         element: withSuspense(OpsDownloads) },

      // 404
      { path: '*', element: <NotFound /> },
    ],
  },
];
