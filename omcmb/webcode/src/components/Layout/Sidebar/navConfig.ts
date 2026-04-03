export interface NavChild {
  key: string;
  label: string;
  path: string;
}

export interface NavGroup {
  key: string;
  label: string;
  iconName: string;
  children: NavChild[];
}

export type NavConfig = NavGroup[];

// Icon names are strings; actual icon components are resolved in NavMenu.tsx
// to avoid circular imports and keep this file pure data.
// label fields are i18n keys, resolved at render time via useT().

export const NAV_CONFIG: NavConfig = [
  {
    key: 'dashboard',
    label: 'nav.dashboard',
    iconName: 'DashboardOutlined',
    children: [
      { key: 'dashboard', label: 'nav.dashboard', path: '/dashboard' },
    ],
  },
  {
    key: 'device',
    label: 'nav.device',
    iconName: 'ClusterOutlined',
    children: [
      { key: 'device-list',      label: 'nav.device.list',       path: '/device/list' },
      // { key: 'device-register',  label: 'nav.device.register',   path: '/device/register' },  // 隐藏设备注册菜单
      { key: 'device-group',     label: 'nav.device.group',       path: '/device/group' },
      // { key: 'device-ne',        label: 'nav.device.ne',          path: '/device/ne' },           // 隐藏
      // { key: 'device-monitor',   label: 'nav.device.monitor',     path: '/device/monitor' },       // 隐藏
      { key: 'device-launch',    label: 'nav.device.commission',  path: '/device/commission' },
      // { key: 'device-transfer',  label: 'nav.device.handover',    path: '/device/handover' },      // 隐藏
      // { key: 'device-resource',  label: 'nav.device.stats',       path: '/device/stats' },         // 隐藏
      // { key: 'device-import',    label: 'nav.device.import',      path: '/device/import' },        // 隐藏
      { key: 'device-rule',      label: 'nav.device.rules',       path: '/device/rules' },
      { key: 'device-recycle',   label: 'nav.device.recycle',     path: '/device/recycle' },
    ],
  },
  {
    key: 'alarm',
    label: 'nav.alarm',
    iconName: 'AlertOutlined',
    children: [
      { key: 'alarm-current',      label: 'nav.alarm.current',        path: '/alarm/current' },
      { key: 'alarm-history',      label: 'nav.alarm.history',        path: '/alarm/history' },
      // { key: 'alarm-stats',        label: 'nav.alarm.statistics',     path: '/alarm/statistics' },  // 隐藏告警统计
      { key: 'alarm-rule',         label: 'nav.alarm.rules',          path: '/alarm/rules' },
      { key: 'alarm-knowledge',    label: 'nav.alarm.library',        path: '/alarm/library' },
      // { key: 'alarm-sync',         label: 'nav.alarm.sync',           path: '/alarm/sync' },           // 隐藏
      { key: 'alarm-notification', label: 'nav.alarm.notification',   path: '/alarm/notification' },
      // { key: 'alarm-interface',    label: 'nav.alarm.interfaceFault', path: '/alarm/interface-fault' }, // 隐藏
      { key: 'alarm-custom-stats', label: 'nav.alarm.customAlarm',    path: '/alarm/custom-stats' },
    ],
  },
  // {
  //   key: 'config',
  //   label: 'nav.config',
  //   iconName: 'SettingOutlined',
  //   children: [
  //     { key: 'config-sync',      label: 'nav.config.paramSync',       path: '/config/param-sync' },
  //     { key: 'config-realtime',  label: 'nav.config.liveParam',       path: '/config/live-param' },
  //     { key: 'config-batch-cat', label: 'nav.config.batchClass',      path: '/config/batch-class' },
  //     { key: 'config-batch-tpl', label: 'nav.config.batchTemplate',   path: '/config/batch-template' },
  //     { key: 'config-list',      label: 'nav.config.paramList',       path: '/config/param-list' },
  //     { key: 'config-cmd',       label: 'nav.config.commandMode',     path: '/config/command-mode' },
  //     { key: 'config-cell',      label: 'nav.config.cell',            path: '/config/cell' },
  //     { key: 'config-baseline',  label: 'nav.config.baseline',        path: '/config/baseline' },
  //     { key: 'config-general',    label: 'nav.config.common',          path: '/config/common' },
  //     { key: 'config-neighbor',   label: 'nav.config.neighbor',        path: '/config/neighbor' },
  //     { key: 'config-son',        label: 'nav.config.son',             path: '/config/son' },
  //     { key: 'config-epc',        label: 'nav.config.epc',             path: '/config/epc' },
  //     { key: 'config-reboot',     label: 'nav.config.reboot',          path: '/config/reboot' },
  //     { key: 'config-reset',      label: 'nav.config.factoryReset',    path: '/config/factory-reset' },
  //     { key: 'config-selfstart',  label: 'nav.config.selfStart',       path: '/config/self-start' },
  //     { key: 'config-cpe-diag',   label: 'nav.config.cpeDiagnostics',  path: '/config/cpe-diagnostics' },
  //     { key: 'config-cpe-freq',   label: 'nav.config.cpeFreqLock',     path: '/config/cpe-freq-lock' },
  //   ],
  // },  // 隐藏配置管理菜单
  {
    key: 'performance',
    label: 'nav.performance',
    iconName: 'LineChartOutlined',
    children: [
      { key: 'perf-query',      label: 'nav.performance.query',        path: '/performance/query' },
      { key: 'perf-chart',      label: 'nav.performance.charts',       path: '/performance/charts' },
      { key: 'perf-kpi-bs',     label: 'nav.performance.kpiStation',   path: '/performance/kpi-station' },
      { key: 'perf-kpi-std',    label: 'nav.performance.kpiStandard',  path: '/performance/kpi-standard' },
      // { key: 'perf-threshold',  label: 'nav.performance.threshold',    path: '/performance/threshold' },  // 隐藏
      // { key: 'perf-file',       label: 'nav.performance.files',        path: '/performance/files' },  // 隐藏
      // { key: 'perf-task',       label: 'nav.performance.taskConfig',   path: '/performance/task-config' },  // 隐藏
      // { key: 'perf-kpi-mgmt',   label: 'nav.performance.kpiMgmt',      path: '/performance/kpi-management' },  // 隐藏
      // { key: 'perf-query-tpl',  label: 'nav.performance.queryTemplates',path: '/performance/query-templates' },  // 隐藏
      // { key: 'perf-busy-hour',  label: 'nav.performance.busyHour',     path: '/performance/busy-hour' },  // 隐藏
    ],
  },
  {
    key: 'mml',
    label: 'nav.mml',
    iconName: 'CodeOutlined',
    children: [
      { key: 'mml-console', label: 'nav.mml.console',  path: '/mml/console' },
      { key: 'mml-script',  label: 'nav.mml.script',   path: '/mml/script' },
      // { key: 'mml-cmd',     label: 'nav.mml.commands',  path: '/mml/commands' },  // 隐藏
    ],
  },
  {
    key: 'topology',
    label: 'nav.topology',
    iconName: 'GlobalOutlined',
    children: [
      { key: 'topo-gis',      label: 'nav.topology.gisMap',    path: '/topology/gis-map' },
      { key: 'topo-canvas',   label: 'nav.topology.canvas',    path: '/topology/canvas' },
      { key: 'topo-domain',   label: 'nav.topology.domain',    path: '/topology/domain' },
      { key: 'topo-site',     label: 'nav.topology.site',      path: '/topology/site' },
      { key: 'topo-settings', label: 'nav.topology.settings',  path: '/topology/settings' },
      { key: 'topo-legend',   label: 'nav.topology.legend',    path: '/topology/legend' },
    ],
  },
  {
    key: 'backup',
    label: 'nav.backup',
    iconName: 'SaveOutlined',
    children: [
      { key: 'backup-task',     label: 'nav.backup.tasks',     path: '/backup/tasks' },
      { key: 'backup-plan',     label: 'nav.backup.schedule',  path: '/backup/schedule' },
      { key: 'backup-ftp',      label: 'nav.backup.ftp',       path: '/backup/ftp' },
      { key: 'backup-restore',  label: 'nav.backup.restore',   path: '/backup/restore' },
      { key: 'backup-strategy', label: 'nav.backup.policy',    path: '/backup/policy' },
    ],
  },
  {
    key: 'software',
    label: 'nav.software',
    iconName: 'CloudUploadOutlined',
    children: [
      // { key: 'sw-query',    label: 'nav.software.version',      path: '/software/version' },      // 隐藏版本查询
      { key: 'sw-upgrade',  label: 'nav.software.versionUpgrade',  path: '/software/upgrade-plan' },
      // { key: 'sw-activate', label: 'nav.software.activation',   path: '/software/activation' },    // 隐藏激活计划
      { key: 'sw-upload',   label: 'nav.software.upgradeFile',     path: '/software/firmware' },
      { key: 'sw-rollback', label: 'nav.software.versionRollback', path: '/software/rollback' },
    ],
  },
  // {
  //   key: 'file',
  //   label: 'nav.file',
  //   iconName: 'FolderOutlined',
  //   children: [
  //     { key: 'file-config-search',  label: 'nav.file.configRetrieval',    path: '/file/config-retrieval' },
  //     { key: 'file-config-dist',    label: 'nav.file.configDistribution', path: '/file/config-distribution' },
  //     { key: 'file-log-search',     label: 'nav.file.logRetrieval',       path: '/file/log-retrieval' },
  //     { key: 'file-perf-search',    label: 'nav.file.perfRetrieval',      path: '/file/perf-retrieval' },
  //     { key: 'file-mr-search',      label: 'nav.file.mrRetrieval',        path: '/file/mr-retrieval' },
  //     { key: 'file-user',           label: 'nav.file.userFiles',          path: '/file/user-files' },
  //     { key: 'file-device',         label: 'nav.file.deviceFiles',        path: '/file/device-files' },
  //   ],
  // },  // 隐藏文件管理菜单
  {
    key: 'log',
    label: 'nav.log',
    iconName: 'FileTextOutlined',
    children: [
      { key: 'log-device',    label: 'nav.log.device',     path: '/log/device' },
      { key: 'log-exception', label: 'nav.log.exception',  path: '/log/exception' },
      { key: 'log-event',     label: 'nav.log.event',      path: '/log/event' },
      // { key: 'log-operation',label: 'nav.log.operation',  path: '/log/operation' },  // 隐藏
      // { key: 'log-system',   label: 'nav.log.system',     path: '/log/system' },       // 隐藏
      // { key: 'log-config',   label: 'nav.log.config',     path: '/log/config' },       // 隐藏
    ],
  },
  {
    key: 'system',
    label: 'nav.system',
    iconName: 'ToolOutlined',
    children: [
      { key: 'sys-device-type', label: 'nav.system.deviceClass',    path: '/system/device-class' },
      { key: 'sys-user',        label: 'nav.system.users',          path: '/system/users' },
      // { key: 'sys-group',       label: 'nav.system.groups',         path: '/system/groups' },  // 隐藏用户组菜单
      { key: 'sys-role',        label: 'nav.system.roles',          path: '/system/roles' },
      { key: 'sys-menu',        label: 'nav.system.menus',          path: '/system/menus' },
      { key: 'sys-op-log',      label: 'nav.system.operationLog',   path: '/system/operation-log' },
      { key: 'sys-config',      label: 'nav.system.config',         path: '/system/config' },
      { key: 'sys-ui-custom',   label: 'nav.system.uiCustom',       path: '/system/ui-custom' },
      { key: 'sys-home',        label: 'nav.system.dashboard',      path: '/system/dashboard' },
      { key: 'sys-operator',    label: 'nav.system.operators',      path: '/system/operators' },
      { key: 'sys-cert',        label: 'nav.system.certificates',   path: '/system/certificates' },
      { key: 'sys-blacklist',   label: 'nav.system.blacklist',      path: '/system/blacklist' },
      { key: 'sys-migration',   label: 'nav.system.migration',      path: '/system/migration' },
      { key: 'sys-db-monitor',  label: 'nav.system.dbMonitor',      path: '/system/db-monitor' },
    ],
  },
  // {
  //   key: 'report',
  //   label: 'nav.report',
  //   iconName: 'BarChartOutlined',
  //   children: [
  //     { key: 'report-lte-std',  label: 'nav.report.lteStandard',   path: '/report/lte-standard' },
  //     { key: 'report-bs',       label: 'nav.report.station',       path: '/report/station' },
  //     { key: 'report-hist-kpi', label: 'nav.report.historicalKpi', path: '/report/historical-kpi' },
  //     { key: 'report-poll',     label: 'nav.report.pollStats',     path: '/report/poll-stats' },
  //   ],
  // },  // 隐藏报表管理菜单
  {
    key: 'mr',
    label: 'nav.mr',
    iconName: 'RadarChartOutlined',
    children: [
      { key: 'mr-index',  label: 'nav.mr.indicators',    path: '/mr/indicators' },
      { key: 'mr-device', label: 'nav.mr.deviceMapping',  path: '/mr/device-mapping' },
      { key: 'mr-var',    label: 'nav.mr.variables',      path: '/mr/variables' },
      { key: 'mr-report', label: 'nav.mr.reports',        path: '/mr/reports' },
      { key: 'mr-task',   label: 'nav.mr.tasks',          path: '/mr/tasks' },
      { key: 'mr-file',   label: 'nav.mr.files',          path: '/mr/files' },
    ],
  },
  {
    key: 'license',
    label: 'nav.license',
    iconName: 'SafetyOutlined',
    children: [
      { key: 'license-list',  label: 'nav.license.list',       path: '/license/list' },
      { key: 'license-op',    label: 'nav.license.operations',  path: '/license/operations' },
      { key: 'license-log',   label: 'nav.license.logs',        path: '/license/logs' },
    ],
  },
  // {
  //   key: 'ops',
  //   label: 'nav.ops',
  //   iconName: 'AppstoreOutlined',
  //   children: [
  //     { key: 'ops-template',   label: 'nav.ops.templates',        path: '/ops/templates' },
  //     { key: 'ops-cmd',        label: 'nav.ops.commands',         path: '/ops/commands' },
  //     { key: 'ops-task',       label: 'nav.ops.tasks',            path: '/ops/tasks' },
  //     { key: 'ops-diagnose',   label: 'nav.ops.networkDiagnosis', path: '/ops/network-diagnosis' },
  //     { key: 'ops-download',   label: 'nav.ops.downloads',        path: '/ops/downloads' },
  //   ],
  // },  // 隐藏运维管理菜单
  // {
  //   key: 'egw',
  //   label: 'nav.egw',
  //   iconName: 'GatewayOutlined',
  //   children: [
  //     { key: 'egw-monitor',       label: 'nav.egw.monitor',       path: '/egw/monitor' },
  //     { key: 'egw-maintenance',   label: 'nav.egw.maintenance',   path: '/egw/maintenance' },
  //     { key: 'egw-registration',  label: 'nav.egw.registration',  path: '/egw/registration' },
  //     { key: 'egw-upgrade',       label: 'nav.egw.upgrade',       path: '/egw/upgrade' },
  //   ],
  // },  // 隐藏网关管理菜单
  // {
  //   key: 'newegw',
  //   label: 'nav.newegw',
  //   iconName: 'DeploymentUnitOutlined',
  //   children: [
  //     { key: 'newegw-monitor',       label: 'nav.newegw.monitor',       path: '/newegw/monitor' },
  //     { key: 'newegw-topology',      label: 'nav.newegw.topology',      path: '/newegw/topology' },
  //     { key: 'newegw-maintenance',   label: 'nav.newegw.maintenance',   path: '/newegw/maintenance' },
  //     { key: 'newegw-registration',  label: 'nav.newegw.registration',  path: '/newegw/registration' },
  //     { key: 'newegw-upgrade',       label: 'nav.newegw.upgrade',       path: '/newegw/upgrade' },
  //     { key: 'newegw-access',        label: 'nav.newegw.accessControl', path: '/newegw/access-control' },
  //   ],
  // },  // 隐藏新型网关菜单
  // {
  //   key: 'sas',
  //   label: 'nav.sas',
  //   iconName: 'WifiOutlined',
  //   children: [
  //     { key: 'sas-cpi',        label: 'nav.sas.cpiConfig',   path: '/sas/cpi-config' },
  //     { key: 'sas-properties', label: 'nav.sas.properties',   path: '/sas/properties' },
  //     { key: 'sas-monitoring', label: 'nav.sas.monitoring',   path: '/sas/monitoring' },
  //   ],
  // },  // 隐藏频谱管理菜单
  // {
  //   key: 'ups',
  //   label: 'nav.ups',
  //   iconName: 'ThunderboltOutlined',
  //   children: [
  //     { key: 'ups-monitor',       label: 'nav.ups.monitor',       path: '/ups/monitor' },
  //     { key: 'ups-registration',  label: 'nav.ups.registration',  path: '/ups/registration' },
  //     { key: 'ups-upgrade',       label: 'nav.ups.upgrade',       path: '/ups/upgrade' },
  //   ],
  // },  // 隐藏电源管理菜单
  // {
  //   key: 'dhcp',
  //   label: 'nav.dhcp',
  //   iconName: 'ApartmentOutlined',
  //   children: [
  //     { key: 'dhcp-config',  label: 'nav.dhcp.serverConfig', path: '/dhcp/server-config' },
  //     { key: 'dhcp-clients', label: 'nav.dhcp.clientList',   path: '/dhcp/client-list' },
  //     { key: 'dhcp-service', label: 'nav.dhcp.service',      path: '/dhcp/service' },
  //   ],
  // },  // 隐藏DHCP管理菜单
  // {
  //   key: 'cau',
  //   label: 'nav.cau',
  //   iconName: 'CloudServerOutlined',
  //   children: [
  //     { key: 'cau-upgrade', label: 'nav.cau.upgrade', path: '/cau/upgrade' },
  //   ],
  // },  // 隐藏CAU管理菜单
  // {
  //   key: 'strategy',
  //   label: 'nav.strategy',
  //   iconName: 'AimOutlined',
  //   children: [
  //     { key: 'strategy-list',      label: 'nav.strategy.list',      path: '/strategy/list' },
  //     { key: 'strategy-execution', label: 'nav.strategy.execution', path: '/strategy/execution' },
  //     { key: 'strategy-import',    label: 'nav.strategy.import',    path: '/strategy/import' },
  //   ],
  // },  // 隐藏策略管理菜单
  // {
  //   key: 'advance',
  //   label: 'nav.advance',
  //   iconName: 'ExperimentOutlined',
  //   children: [
  //     { key: 'advance-anr',         label: 'nav.advance.anr',         path: '/advance/anr' },
  //     { key: 'advance-pci-conflict', label: 'nav.advance.pciConflict', path: '/advance/pci-conflict' },
  //   ],
  // },  // 隐藏高级功能菜单
];
