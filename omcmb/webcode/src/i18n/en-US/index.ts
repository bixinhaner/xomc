const messages: Record<string, string> = {
  // -------------------------------------------------------------------------
  // Common actions
  // -------------------------------------------------------------------------
  'common.search':         'Search',
  'common.reset':          'Reset',
  'common.add':            'Add',
  'common.edit':           'Edit',
  'common.delete':         'Delete',
  'common.export':         'Export',
  'common.import':         'Import',
  'common.confirm':        'Confirm',
  'common.cancel':         'Cancel',
  'common.save':           'Save',
  'common.submit':         'Submit',
  'common.refresh':        'Refresh',
  'common.close':          'Close',
  'common.back':           'Back',
  'common.next':           'Next',
  'common.prev':           'Previous',
  'common.finish':         'Finish',
  'common.view':           'View',
  'common.detail':         'Detail',
  'common.copy':           'Copy',
  'common.download':       'Download',
  'common.upload':         'Upload',
  'common.execute':        'Execute',
  'common.deploy':         'Deploy',
  'common.approve':        'Approve',
  'common.reject':         'Reject',
  'common.enable':         'Enable',
  'common.disable':        'Disable',
  'common.batchDelete':    'Batch Delete',
  'common.batchExport':    'Batch Export',
  'common.more':           'More',
  'common.loading':        'Loading...',
  'common.noData':         'No Data',
  'common.placeholder':    'Please enter',
  'common.pleaseSelect':   'Please select',
  'common.all':            'All',
  'common.yes':            'Yes',
  'common.no':             'No',
  'common.unknown':        'Unknown',

  // -------------------------------------------------------------------------
  // Table headers
  // -------------------------------------------------------------------------
  'table.sn':              'Serial Number',
  'table.name':            'Name',
  'table.status':          'Status',
  'table.createTime':      'Created At',
  'table.updateTime':      'Updated At',
  'table.operation':       'Actions',
  'table.index':           'No.',
  'table.description':     'Description',
  'table.vendor':          'Vendor',
  'table.type':            'Type',
  'table.region':          'Region',
  'table.site':            'Site',
  'table.ip':              'IP Address',
  'table.version':         'Version',
  'table.operator':        'Operator',
  'table.result':          'Result',
  'table.time':            'Time',
  'table.total':           'Total',
  'table.success':         'Success',
  'table.failed':          'Failed',
  'table.pending':         'Pending',

  // -------------------------------------------------------------------------
  // Status labels
  // -------------------------------------------------------------------------
  'status.online':         'Online',
  'status.offline':        'Offline',
  'status.active':         'Active',
  'status.inactive':       'Inactive',
  'status.locked':         'Locked',
  'status.enabled':        'Enabled',
  'status.disabled':       'Disabled',
  'status.managed':        'Managed',
  'status.unmanaged':      'Unmanaged',
  'status.preManaged':     'Pre-managed',
  'status.commissioned':   'Commissioned',
  'status.uncommissioned': 'Uncommissioned',
  'status.decommissioned': 'Decommissioned',
  'status.pending':        'Pending',
  'status.running':        'Running',
  'status.success':        'Success',
  'status.failed':         'Failed',
  'status.cancelled':      'Cancelled',

  // -------------------------------------------------------------------------
  // Alarm severity
  // -------------------------------------------------------------------------
  'alarm.severity.critical': 'Critical',
  'alarm.severity.major':    'Major',
  'alarm.severity.minor':    'Minor',
  'alarm.severity.warning':  'Warning',
  'alarm.ackStatus.acknowledged':   'Acknowledged',
  'alarm.ackStatus.unacknowledged': 'Unacknowledged',

  // -------------------------------------------------------------------------
  // Navigation — module names
  // -------------------------------------------------------------------------
  'nav.dashboard':     'Dashboard',
  'nav.device':        'Device Management',
  'nav.alarm':         'Alarm Management',
  'nav.config':        'Config Management',
  'nav.performance':   'Performance Management',
  'nav.mml':           'MML Management',
  'nav.topology':      'Topology Management',
  'nav.backup':        'Backup & Restore',
  'nav.software':      'Software Management',
  'nav.file':          'File Management',
  'nav.log':           'Log Management',
  'nav.system':        'System Management',
  'nav.report':        'Report Management',
  'nav.mr':            'MR Management',
  'nav.license':       'License Management',
  'nav.ops':           'Ops Management',

  // -------------------------------------------------------------------------
  // Navigation — sub-pages: Device
  // -------------------------------------------------------------------------
  'nav.device.list':       'Device List',
  'nav.device.register':   'Device Registration',
  'nav.device.group':      'Device Groups',
  'nav.device.detail':     'Device Detail',
  'nav.device.ne':         'NE Management',
  'nav.device.monitor':    'Online Monitor',
  'nav.device.commission': 'Commissioning',
  'nav.device.handover':   'Handover Management',
  'nav.device.stats':      'Resource Statistics',
  'nav.device.import':     'Import & Export',
  'nav.device.rules':      'Device Rules',
  'nav.device.recycle':    'Recycle Bin',

  // -------------------------------------------------------------------------
  // Navigation — sub-pages: Alarm
  // -------------------------------------------------------------------------
  'nav.alarm.current':    'Current Alarms',
  'nav.alarm.history':    'Historical Alarms',
  'nav.alarm.statistics': 'Alarm Statistics',
  'nav.alarm.rules':      'Alarm Rules',
  'nav.alarm.library':    'Alarm Library',
  'nav.alarm.sync':       'Alarm Sync',

  // -------------------------------------------------------------------------
  // Navigation — sub-pages: Config
  // -------------------------------------------------------------------------
  'nav.config.paramSync':       'Param Sync',
  'nav.config.liveParam':       'Live Param Config',
  'nav.config.batchClass':      'Batch Param Class',
  'nav.config.batchTemplate':   'Batch Param Template',
  'nav.config.paramList':       'Param List',
  'nav.config.commandMode':     'Command Mode',
  'nav.config.cell':            'Cell Management',
  'nav.config.baseline':        'Baseline Management',
  'nav.config.common':          'Common Config',
  'nav.config.neighbor':        'Neighbor Params',

  // -------------------------------------------------------------------------
  // Navigation — sub-pages: Performance
  // -------------------------------------------------------------------------
  'nav.performance.kpiStandard':  'KPI Standard Report',
  'nav.performance.kpiStation':   'KPI Station Report',
  'nav.performance.extraction':   'Extraction Wizard',
  'nav.performance.charts':       'Performance Charts',
  'nav.performance.threshold':    'Threshold Config',
  'nav.performance.files':        'Performance Files',
  'nav.performance.taskConfig':   'Task Config',

  // -------------------------------------------------------------------------
  // Navigation — sub-pages: MML
  // -------------------------------------------------------------------------
  'nav.mml.console':   'MML Console',
  'nav.mml.script':    'Script Task',
  'nav.mml.commands':  'Command Tree',

  // -------------------------------------------------------------------------
  // Navigation — sub-pages: Topology
  // -------------------------------------------------------------------------
  'nav.topology.gisMap':   'GIS Map',
  'nav.topology.canvas':   'Topology Canvas',
  'nav.topology.domain':   'Domain Management',
  'nav.topology.site':     'Site Management',
  'nav.topology.settings': 'Topology Settings',
  'nav.topology.legend':   'Legend System',

  // -------------------------------------------------------------------------
  // Navigation — sub-pages: Backup
  // -------------------------------------------------------------------------
  'nav.backup.tasks':    'Backup Tasks',
  'nav.backup.schedule': 'Backup Schedule',
  'nav.backup.ftp':      'FTP Config',
  'nav.backup.restore':  'Restore Data',
  'nav.backup.policy':   'Backup Policy',

  // -------------------------------------------------------------------------
  // Navigation — sub-pages: Software
  // -------------------------------------------------------------------------
  'nav.software.version':      'Version Query',
  'nav.software.upgradePlan':  'Upgrade Plan',
  'nav.software.activation':   'Activation Plan',
  'nav.software.firmware':     'Firmware Upload',

  // -------------------------------------------------------------------------
  // Navigation — sub-pages: File
  // -------------------------------------------------------------------------
  'nav.file.configRetrieval':    'Config Retrieval',
  'nav.file.configDistribution': 'Config Distribution',
  'nav.file.logRetrieval':       'Log Retrieval',
  'nav.file.perfRetrieval':      'Perf Retrieval',
  'nav.file.mrRetrieval':        'MR Retrieval',
  'nav.file.userFiles':          'User Files',
  'nav.file.deviceFiles':        'Device Files',

  // -------------------------------------------------------------------------
  // Navigation — sub-pages: Log
  // -------------------------------------------------------------------------
  'nav.log.neMessage':  'NE Message Log',
  'nav.log.heartbeat':  'Heartbeat Log',
  'nav.log.alarm':      'Alarm Log',
  'nav.log.operation':  'Operation Log',
  'nav.log.system':     'System Log',
  'nav.log.config':     'Log Config',

  // -------------------------------------------------------------------------
  // Navigation — sub-pages: System
  // -------------------------------------------------------------------------
  'nav.system.deviceClass':   'Device Classification',
  'nav.system.users':         'User Management',
  'nav.system.roles':         'Role Permission',
  'nav.system.operationLog':  'Operation Log',
  'nav.system.config':        'System Config',
  'nav.system.dataDict':      'Data Dictionary',
  'nav.system.notifications': 'Notification Settings',
  'nav.system.dashboard':     'System Dashboard',

  // -------------------------------------------------------------------------
  // Navigation — sub-pages: Report
  // -------------------------------------------------------------------------
  'nav.report.lteStandard':   'LTE Standard Report',
  'nav.report.station':       'Station Report',
  'nav.report.historicalKpi': 'Historical KPI',
  'nav.report.pollStats':     'Poll Statistics',

  // -------------------------------------------------------------------------
  // Navigation — sub-pages: MR
  // -------------------------------------------------------------------------
  'nav.mr.indicators':    'MR Indicators',
  'nav.mr.deviceMapping': 'Device Mapping',
  'nav.mr.variables':     'Variable Management',
  'nav.mr.reports':       'MR Reports',
  'nav.mr.tasks':         'MR Tasks',
  'nav.mr.files':         'MR Files',

  // -------------------------------------------------------------------------
  // Navigation — sub-pages: License
  // -------------------------------------------------------------------------
  'nav.license.list':       'License List',
  'nav.license.operations': 'License Operations',
  'nav.license.logs':       'License Logs',

  // -------------------------------------------------------------------------
  // Navigation — sub-pages: Ops
  // -------------------------------------------------------------------------
  'nav.ops.templates':         'Ops Templates',
  'nav.ops.commands':          'Ops Commands',
  'nav.ops.tasks':             'Ops Tasks',
  'nav.ops.networkDiagnosis':  'Network Diagnosis',
  'nav.ops.downloads':         'Ops Downloads',

  // -------------------------------------------------------------------------
  // Device labels
  // -------------------------------------------------------------------------
  'device.type.eNB':        'eNB Base Station',
  'device.type.gNB':        'gNB Base Station',
  'device.type.CPE':        'CPE Terminal',
  'device.type.eGW':        'Enterprise Gateway',
  'device.type.all':        'All Types',
  'device.connStatus':      'Connection Status',
  'device.engStatus':       'Engineering Status',
  'device.mgmtStatus':      'Management Status',
  'device.vendor':          'Vendor',
  'device.productType':     'Product Type',
  'device.networkType':     'Network Type',
  'device.model':           'Device Model',
  'device.softwareVersion': 'Software Version',
  'device.ipAddress':       'IP Address',
  'device.subnet':          'Subnet',
  'device.lastOnlineTime':  'Last Online Time',
  'device.longitude':       'Longitude',
  'device.latitude':        'Latitude',

  // -------------------------------------------------------------------------
  // Alarm page labels
  // -------------------------------------------------------------------------
  'alarm.id':           'Alarm ID',
  'alarm.code':         'Alarm Code',
  'alarm.name':         'Alarm Name',
  'alarm.severity':     'Severity',
  'alarm.deviceSn':     'Device SN',
  'alarm.deviceName':   'Device Name',
  'alarm.neType':       'NE Type',
  'alarm.content':      'Alarm Content',
  'alarm.time':         'Alarm Time',
  'alarm.clearTime':    'Clear Time',
  'alarm.duration':     'Duration',
  'alarm.ackStatus':    'Ack Status',
  'alarm.ackUser':      'Ack User',
  'alarm.ackTime':      'Ack Time',
  'alarm.ackNote':      'Ack Note',
  'alarm.source':       'Alarm Source',
  'alarm.location':     'Alarm Location',
  'alarm.type':         'Alarm Type',
  'alarm.acknowledge':  'Acknowledge',
  'alarm.clear':        'Clear Alarm',
  'alarm.filter':       'Alarm Filter',
  'alarm.total':        'Total Alarms',
  'alarm.active':       'Active Alarms',

  // -------------------------------------------------------------------------
  // User / System labels
  // -------------------------------------------------------------------------
  'user.username':      'Username',
  'user.displayName':   'Display Name',
  'user.email':         'Email',
  'user.phone':         'Phone',
  'user.role':          'Role',
  'user.status':        'Status',
  'user.lastLogin':     'Last Login',
  'user.createTime':    'Created At',
  'user.password':      'Password',
  'user.newPassword':   'New Password',
  'user.confirmPwd':    'Confirm Password',
  'user.role.admin':    'Administrator',
  'user.role.operator': 'Operator',
  'user.role.viewer':   'Viewer',
  'user.role.auditor':  'Auditor',

  // -------------------------------------------------------------------------
  // Login page
  // -------------------------------------------------------------------------
  'login.title':         'OMC Network Management System',
  'login.username':      'Username',
  'login.password':      'Password',
  'login.submit':        'Sign In',
  'login.rememberMe':    'Remember Me',
  'login.forgotPwd':     'Forgot Password',
  'login.usernameTip':   'Please enter username',
  'login.passwordTip':   'Please enter password',
  'login.success':       'Login successful',
  'login.failed':        'Invalid username or password',

  // -------------------------------------------------------------------------
  // Task panel
  // -------------------------------------------------------------------------
  'task.panel.title':     'Task Panel',
  'task.single':          'Single Tasks',
  'task.batch':           'Batch Tasks',
  'task.export':          'Export Tasks',
  'task.progress':        'Progress',
  'task.status':          'Status',
  'task.startTime':       'Start Time',
  'task.endTime':         'End Time',
  'task.message':         'Message',
  'task.clear':           'Clear',
  'task.clearCompleted':  'Clear Completed',

  // -------------------------------------------------------------------------
  // Theme / locale / timezone
  // -------------------------------------------------------------------------
  'settings.theme.light':   'Light Theme',
  'settings.theme.dark':    'Dark Theme',
  'settings.locale.zhCN':   '中文',
  'settings.locale.enUS':   'English',
  'settings.timezone.utc':  'UTC Time',
  'settings.timezone.local':'Local Time',

  // -------------------------------------------------------------------------
  // Error pages
  // -------------------------------------------------------------------------
  'error.404.title':   'Page Not Found',
  'error.404.message': 'Sorry, the page you are looking for does not exist.',
  'error.403.title':   'Access Denied',
  'error.403.message': 'Sorry, you do not have permission to access this page.',
  'error.500.title':   'Server Error',
  'error.500.message': 'A server error occurred. Please try again later.',
  'error.backHome':    'Back to Home',

  // -------------------------------------------------------------------------
  // KPI / Performance labels
  // -------------------------------------------------------------------------
  'perf.kpiName':     'KPI Name',
  'perf.kpiCode':     'KPI Code',
  'perf.unit':        'Unit',
  'perf.category':    'Category',
  'perf.threshold':   'Threshold',
  'perf.granularity': 'Granularity',
  'perf.timeRange':   'Time Range',
  'perf.value':       'Value',
  'perf.timestamp':   'Timestamp',

  // -------------------------------------------------------------------------
  // Config labels
  // -------------------------------------------------------------------------
  'config.paramName':   'Param Name',
  'config.paramCode':   'Param Code',
  'config.paramValue':  'Param Value',
  'config.defaultValue':'Default Value',
  'config.paramType':   'Param Type',
  'config.readonly':    'Read-only',
  'config.template':    'Template',
  'config.baseline':    'Baseline',
  'config.apply':       'Apply',
  'config.sync':        'Sync',
  'config.compare':     'Compare',

  // -------------------------------------------------------------------------
  // Tab context menu
  // -------------------------------------------------------------------------
  'tab.closeOthers': 'Close Others',
  'tab.closeAll':    'Close All',
  'tab.closeRight':  'Close to Right',

  // -------------------------------------------------------------------------
  // App / Header
  // -------------------------------------------------------------------------
  'app.title':            'OMC Network Management',
  'header.currentView':   'Current View:',
  'header.notification':  'Notifications',
  'header.switchToDark':  'Switch to Dark Theme',
  'header.switchToDim':   'Switch to Dim Theme',
  'header.switchToLight': 'Switch to Light Theme',
  'header.timezone':      'TZ:',
  'header.localTimezone': 'Local',
  'header.timezoneTitle': 'Current timezone: {tz}, click to switch',
  'header.alarmTitle':    '{label} alarms: {count}',

  // -------------------------------------------------------------------------
  // User dropdown
  // -------------------------------------------------------------------------
  'user.changePassword':  'Change Password',
  'user.logout':          'Logout',
  'user.notLoggedIn':     'Not Logged In',
  'user.switchToEn':      'Switch to English',
  'user.switchToZh':      'Switch to Chinese',

  // -------------------------------------------------------------------------
  // Dashboard
  // -------------------------------------------------------------------------
  'dashboard.totalDevices':      'Total Devices',
  'dashboard.onlineDevices':     'Online Devices',
  'dashboard.activeAlarms':      'Active Alarms',
  'dashboard.runningTasks':      'Running Tasks',
  'dashboard.vsLastWeek':        'vs last week',
  'dashboard.onlineRate':        'Online Rate',
  'dashboard.vsYesterday':       'vs yesterday',
  'dashboard.alarmSummary':      'Alarm Summary',
  'dashboard.viewAll':           'View All',
  'dashboard.alarmDistribution': 'Alarm Distribution',
  'dashboard.deviceStatusByType':'Device Status by Type',
  'dashboard.alarmTrend7d':      '7-Day Alarm Trend',
  'dashboard.top10AlarmDevices': 'TOP 10 Alarm Devices',
  'dashboard.deviceMap':         'Device Map',
  'dashboard.quickAccess':       'Quick Access',
  'dashboard.sysAdmin':          'System Admin',
  'dashboard.todayOps':          "Today's Ops",
  'dashboard.processedAlarms':   'Processed Alarms',
  'dashboard.lastLogin':         'Last login:',
  'dashboard.alarmCount':        'Alarm Count',

  // Dashboard chart labels
  'dashboard.chart.online':  'Online',
  'dashboard.chart.offline': 'Offline',
  'dashboard.chart.alarm':   'Alarm',

  // -------------------------------------------------------------------------
  // Common page labels
  // -------------------------------------------------------------------------
  'common.batchConfig':     'Batch Config',
  'common.batchAck':        'Batch Acknowledge',
  'common.batchClear':      'Batch Clear',
  'common.addDevice':       'Add Device',
  'common.confirmDelete':   'Confirm Delete',
  'common.deleteConfirmMsg':'Are you sure you want to delete the selected {count} records? This cannot be undone.',
  'common.deleteSuccess':   'Deleted successfully',
  'common.exportInProgress':'Exporting...',
  'common.featureInDev':    'Feature in development',
  'common.realTimeConn':    'Real-time Connected',
  'common.unacked':         '{count} unacknowledged',
  'common.ackConfirmMsg':   'Are you sure you want to acknowledge the selected {count} alarms?',
  'common.ackSuccess':      'Acknowledged {count} alarms',
  'common.clearConfirmMsg': 'Are you sure you want to clear the selected {count} alarms?',
  'common.clearSuccess':    'Cleared {count} alarms',
  'common.hasAlarm':        'Has Alarm',
  'common.noAlarm':         'No Alarm',

  // -------------------------------------------------------------------------
  // FilterBar
  // -------------------------------------------------------------------------
  'filter.expand':      'Expand',
  'filter.collapse':    'Collapse',
  'filter.query':       'Query',
  'filter.enterField':  'Enter {label}',
  'filter.selectField': 'Select {label}',
  'dateRange.start':    'Start',
  'dateRange.end':      'End',

  // -------------------------------------------------------------------------
  // DataTable extras
  // -------------------------------------------------------------------------
  'table.density':           'Density',
  'table.density.compact':   'Compact',
  'table.density.default':   'Default',
  'table.density.comfortable':'Comfortable',
  'table.columnSettings':    'Column Settings',
  'table.columnDisplay':     'Column Display',
  'table.showAll':           'Show All',
  'table.selected':          '{count} selected',
  'table.totalItems':        '{total} total',
  'table.copied':            'Copied',
  'table.searchPlaceholder': 'Search...',

  // -------------------------------------------------------------------------
  // EmptyState
  // -------------------------------------------------------------------------
  'empty.noData':          'No Data',
  'empty.noDataDesc':      'The list is empty, please try again later',
  'empty.noResult':        'No Results Found',
  'empty.noResultDesc':    'Try adjusting your search criteria',
  'empty.loadFailed':      'Load Failed',
  'empty.loadFailedDesc':  'An error occurred while loading data, please retry',
  'empty.noPermission':    'No Permission',
  'empty.noPermissionDesc':'You do not have permission to access this content',

  // -------------------------------------------------------------------------
  // ErrorBoundary
  // -------------------------------------------------------------------------
  'error.pageError':        'Page Error',
  'error.tryRefresh':       'Try refreshing the page or contact system admin',
  'error.retry':            'Retry',
  'error.networkError':     'Network Error',
  'error.networkErrorDesc': 'Unable to connect to the server. Please check your network and try again.',

  // -------------------------------------------------------------------------
  // Sidebar
  // -------------------------------------------------------------------------
  'sidebar.expand':    'Expand Sidebar',
  'sidebar.collapse':  'Collapse Sidebar',

  // -------------------------------------------------------------------------
  // Layout customization
  // -------------------------------------------------------------------------
  'layout.title':             'Layout Settings',
  'layout.subtitle':          'Customize sidebar and tab bar positions',
  'layout.sidebarPosition':   'Sidebar Position',
  'layout.sidebarLeft':       'Left',
  'layout.sidebarRight':      'Right',
  'layout.sidebarTop':        'Top',
  'layout.tabBarPosition':    'Tab Bar Position',
  'layout.tabBarTop':         'Top',
  'layout.tabBarBottom':      'Bottom',
  'layout.tabBarLeft':        'Left',
  'layout.preview':           'Layout Preview',
  'layout.resetDefault':      'Reset to Default',

  // -------------------------------------------------------------------------
  // Visual styles
  // -------------------------------------------------------------------------
  'style.title':              'Visual Style',
  'style.subtitle':           'Choose a complete visual style',
  'style.classic':            'Classic Business',
  'style.classicDesc':        'Dark navy navigation, professional and reliable',
  'style.tech':               'Tech Future',
  'style.techDesc':           'Full dark cyberpunk, neon cyan accent',
  'style.fresh':              'Modern Fresh',
  'style.freshDesc':          'All-white navigation, indigo accent, SaaS style',
  'style.cyberpunk':          'Cyberpunk',
  'style.cyberpunkDesc':      'Ultimate cool Cyberpunk / Matrix style',
  'style.minions':            'Minions',
  'style.minionsDesc':        'Cute and lively Minions theme',
  'style.tiffany':            'Tiffany',
  'style.tiffanyDesc':        'Elegant and luxurious Tiffany style',
  'style.rmb':                'RMB',
  'style.rmbDesc':            'Dignified Chinese red and gold, Guochao style',
  'effects3d.toggle':         '3D Effects',
  'effects3d.description':    'Enable card tilt, particle background, floating animations and other 3D visual effects',
  'header.switchToTech':      'Switch to Tech Future style',
  'header.switchToFresh':     'Switch to Modern Fresh style',
  'header.switchToCyberpunk': 'Switch to Cyberpunk style',
  'header.switchToMinions':   'Switch to Minions style',
  'header.switchToTiffany':   'Switch to Tiffany style',
  'header.switchToRmb':       'Switch to RMB style',
  'header.switchToClassic':   'Switch to Classic Business style',

  // -------------------------------------------------------------------------
  // SplitPanel
  // -------------------------------------------------------------------------
  'panel.expandTop':    'Expand Top Panel',
  'panel.collapseTop':  'Collapse Top Panel',
  'panel.expandBottom': 'Expand Bottom Panel',
  'panel.collapseBottom':'Collapse Bottom Panel',

  // -------------------------------------------------------------------------
  // Topology
  // -------------------------------------------------------------------------
  'topology.zoomIn':    'Zoom In',
  'topology.zoomOut':   'Zoom Out',
  'topology.resetView': 'Reset View',
  'topology.noData':    'No Topology Data',
  'topology.status.online':  'Online',
  'topology.status.offline': 'Offline',
  'topology.status.alarm':   'Alarm',
  'topology.status.maintenance': 'Maintenance',

  // -------------------------------------------------------------------------
  // AlarmFilter
  // -------------------------------------------------------------------------
  'alarm.filter.title':    'Alarm Filter',
  'alarm.filter.apply':    'Apply Filter',
  'alarm.filter.resetAll': 'Reset All',
  'alarm.filter.basic':    'Basic',
  'alarm.filter.device':   'Device',
  'alarm.filter.advanced': 'Advanced',
  'alarm.filter.alarmType':'Alarm Type',
  'alarm.filter.alarmNameCode': 'Alarm name/code/content',
  'alarm.filter.deviceSn': 'Enter or paste device SN, press Enter to confirm',
  'alarm.filter.deviceModel': 'Enter device model',
  'alarm.filter.severity':     'Severity',
  'alarm.filter.keyword':      'Alarm Keyword',
  'alarm.filter.deviceSnPlaceholder': 'Enter or paste device SN, press Enter',
  'alarm.filter.location':     'Location',
  'alarm.filter.locationId':   'Alarm Location',
  'alarm.filter.locationDetail':'Location Detail',
  'alarm.filter.region':       'Region',
  'alarm.filter.timeRange':    'Time Range',
  'alarm.filter.alarmTime':    'Alarm Time',
  'alarm.filter.clearTime':    'Clear Time',
  'alarm.filter.sortBy':       'Sort By',
  'alarm.filter.maxCount':     'Max Count',
  'alarm.filter.includeCleared':      'Include Cleared',
  'alarm.filter.includeAcknowledged': 'Include Acknowledged',
  'alarm.filter.allTypes':     'All Types',
  'alarm.filter.type.device':  'Device Alarm',
  'alarm.filter.type.link':    'Link Alarm',
  'alarm.filter.type.performance': 'Performance Alarm',
  'alarm.filter.type.security':'Security Alarm',
  'alarm.filter.type.environment': 'Environment Alarm',
  'alarm.filter.alarmCode':    'Alarm Code',
  'alarm.filter.countItems':   '{count} items',
  'alarm.filter.noLimit':      'No Limit',

  // Topology node types
  'topology.nodeType.eNB':    'Base Station',
  'topology.nodeType.domain': 'Domain',
  'topology.nodeType.site':   'Site',

  // -------------------------------------------------------------------------
  // Device specific
  // -------------------------------------------------------------------------
  'device.name':             'Device Name',
  'device.sn':               'Device SN',
  'device.region':           'Region',
  'device.alarmLevel':       'Alarm Level',
  'device.lastOnline':       'Last Online',
  'device.engStatus.commissioned':   'Commissioned',
  'device.engStatus.uncommissioned': 'Uncommissioned',
  'device.engStatus.decommissioned': 'Decommissioned',
  'device.networkMode':      'Network Mode',
  'device.count.total':      'Total',

  // -------------------------------------------------------------------------
  // Alarm specific
  // -------------------------------------------------------------------------
  'alarm.occurTime':    'Occur Time',
  'alarm.severity.none':'None',

  // -------------------------------------------------------------------------
  // Time duration
  // -------------------------------------------------------------------------
  'time.seconds': '{n}s',
  'time.minutes': '{n}min',
  'time.hours':   '{n}h',
  'time.days':    '{n}d',

  // -------------------------------------------------------------------------
  // License module
  // -------------------------------------------------------------------------
  'license.idOrName':               'License ID/Name',
  'license.idOrNamePlaceholder':    'Enter license ID or name',
  'license.perpetual':              'Perpetual',
  'license.subscription':           'Subscription',
  'license.trialType':              'Trial',
  'license.evaluation':             'Evaluation',
  'license.expired':                'Expired',
  'license.trial':                  'Trial',
  'license.revoked':                'Revoked',
  'license.capacityUsage':          'Capacity Usage',
  'license.validity':               'Validity',
  'license.permanent':              'Permanent',
  'license.permanentValid':         'Permanent',
  'license.detail':                 'License Detail',
  'license.licenseName':            'License Name',
  'license.productName':            'Product Name',
  'license.licenseType':            'License Type',
  'license.capacity':               'Capacity',
  'license.used':                   'Used',
  'license.licensor':               'Licensor',
  'license.issueDate':              'Issue Date',
  'license.expiryDate':             'Expiry Date',
  'license.features':               'Features',
  'license.notes':                  'Notes',
  'license.selectToViewDetail':     'Click "Detail" button in the table above to view license details',
  'license.revoke':                 'Revoke',
  'license.revokeSuccess':          'License revoked',
  'license.activate':               'Activate',
  'license.query':                  'Query',
  'license.renew':                  'Renew',
  'license.operationType':          'Operation Type',
  'license.remark':                 'Remark',
  'license.operationsSubtitle':     'Import, activate and revoke licenses',
  'license.importLicense':          'Import License',
  'license.activateLicense':        'Activate License',
  'license.revokeLicense':          'Revoke License',
  'license.operationHistory':       'Operation History',
  'license.selectFileFirst':        'Please select a license file first',
  'license.importFile':             'Import file',
  'license.importSuccess':          'License file imported successfully',
  'license.enterActivationCode':    'Please enter license activation code',
  'license.activateSuccess':        'License activated successfully',
  'license.enterRevokeId':          'Please enter the License ID to revoke',
  'license.confirmRevoke':          'Confirm Revocation',
  'license.confirmRevokeMsg':       'Confirm revoking License "{id}"? The license will become invalid immediately.',
  'license.dragFileHere':           'Click or drag license file to this area',
  'license.supportedFormats':       'Supported formats: .lic / .dat / .xml / .key',
  'license.activateDescription':    'Enter the license activation code. The code is usually provided by the vendor in format OMC-XXXX-XXXX-XXXX.',
  'license.pasteActivationCode':    'Paste the complete license activation code',
  'license.revokeWarning':          'Warning: Revocation is irreversible! The license will become invalid immediately and protected features will be unavailable.',
  'license.enterRevokeIdPlaceholder': 'Enter the License ID to revoke (e.g. OMC-BASIC-HB-2024-001)',
  'license.keyword':                'Keyword',
  'license.keywordPlaceholder':     'Enter License ID / Operator',
  'license.timeRange':              'Time Range',
  'license.logsSubtitle':           'View license operation history',
  'license.clientIp':               'Client IP',

  // -------------------------------------------------------------------------
  // MR module
  // -------------------------------------------------------------------------
  'mr.indicatorNameCode':           'Indicator Name/Code',
  'mr.indicatorNameCodePlaceholder': 'Enter indicator name or code',
  'mr.category':                    'Category',
  'mr.categoryPlaceholder':         'Enter category',
  'mr.categoryExample':             'e.g. Collection Config, Filter Config',
  'mr.radioAccess':                 'Radio Access',
  'mr.mobility':                    'Mobility',
  'mr.coverageQuality':             'Coverage Quality',
  'mr.systemLoad':                  'System Load',
  'mr.interference':                'Interference',
  'mr.indicatorName':               'Indicator Name',
  'mr.indicatorCode':               'Indicator Code',
  'mr.unit':                        'Unit',
  'mr.unitPlaceholder':             'e.g. seconds, dBm (optional)',
  'mr.valueRange':                  'Value Range',
  'mr.valueRangePlaceholder':       'e.g. 1-1000 or A/B/C',
  'mr.mrType':                      'MR Type',
  'mr.indicatorsSubtitle':          'View and manage MR measurement indicator definitions',
  'mr.reportName':                  'Report Name',
  'mr.reportNamePlaceholder':       'Enter report name',
  'mr.analysisType':                'Analysis Type',
  'mr.coverageAnalysis':            'Coverage Analysis',
  'mr.interferenceAnalysis':        'Interference Analysis',
  'mr.mobilityAnalysis':            'Mobility Analysis',
  'mr.loadAnalysis':                'Load Analysis',
  'mr.qualityAnalysis':             'Quality Analysis',
  'mr.generated':                   'Generated',
  'mr.generating':                  'Generating',
  'mr.generateFailed':              'Generation Failed',
  'mr.generateTime':                'Generated At',
  'mr.timeRange':                   'Time Range',
  'mr.deviceScope':                 'Device Scope',
  'mr.fileSize':                    'File Size',
  'mr.downloadTaskCreated':         'Download task created',
  'mr.newReport':                   'New Report',
  'mr.newReportFeature':            'New report feature',
  'mr.reportsSubtitle':             'View MR data analysis reports',
  'mr.taskName':                    'Task Name',
  'mr.taskNamePlaceholder':         'Enter task name',
  'mr.collectType':                 'Collection Type',
  'mr.paused':                      'Paused',
  'mr.completed':                   'Completed',
  'mr.schedule':                    'Schedule',
  'mr.progress':                    'Progress',
  'mr.lastRun':                     'Last Run',
  'mr.creator':                     'Creator',
  'mr.pause':                       'Pause',
  'mr.taskCreateSuccess':           'MR collection task created successfully',
  'mr.newTask':                     'New Task',
  'mr.tasksSubtitle':               'Manage MR data collection tasks',
  'mr.newCollectTask':              'New MR Collection Task',
  'mr.deviceCount':                 'Device Count',
  'mr.deviceCountPlaceholder':      'Expected device count',
  'mr.collectSchedule':             'Collection Schedule',
  'mr.every15min':                  'Every 15 minutes',
  'mr.everyHour':                   'Every hour',
  'mr.dailyAt01':                   'Daily at 01:00',
  'mr.oneTime':                     'One-time execution',
  'mr.varName':                     'Variable Name',
  'mr.varNamePlaceholder':          'Enter variable name or description',
  'mr.varNameRequired':             'Please enter variable name',
  'mr.varNamePattern':              'Variable name can only contain uppercase letters, digits and underscores, starting with a letter or underscore',
  'mr.varNameExample':              'e.g. MR_PERIOD',
  'mr.varType':                     'Variable Type',
  'mr.varTypeInteger':              'Integer',
  'mr.varTypeFloat':                'Float',
  'mr.varTypeEnum':                 'Enum',
  'mr.varTypeBoolean':              'Boolean',
  'mr.varTypeString':               'String',
  'mr.defaultValue':                'Default Value',
  'mr.defaultValuePlaceholder':     'Enter default value',
  'mr.descriptionPlaceholder':      'Enter variable description',
  'mr.varUpdateSuccess':            'Variable configuration updated',
  'mr.varCreateSuccess':            'Variable created',
  'mr.addVariable':                 'Add Variable',
  'mr.editVariable':                'Edit Variable',
  'mr.variablesSubtitle':           'Manage MR measurement report collection variables',
  'mr.fileName':                    'File Name',
  'mr.fileNamePlaceholder':         'Enter file name',
  'mr.deviceSnPlaceholder':         'Enter device SN',
  'mr.collectTime':                 'Collection Time',
  'mr.uploadTime':                  'Upload Time',
  'mr.recordCount':                 'Record Count',
  'mr.filesSubtitle':               'View MR data files reported by devices',
  'mr.batchDownload':               'Batch download {count} files',

  // -------------------------------------------------------------------------
  // Ops module
  // -------------------------------------------------------------------------
  'ops.templateName':               'Template Name',
  'ops.templateNamePlaceholder':    'Enter template name',
  'ops.targetDevice':               'Target Device',
  'ops.applicableDevices':          'Applicable Devices',
  'ops.applicableDeviceTypes':      'Applicable Device Types',
  'ops.stepCount':                  'Steps',
  'ops.estimatedDuration':          'Est. Duration',
  'ops.estimatedDurationSeconds':   'Est. Duration (seconds)',
  'ops.durationExample':            'e.g. 300',
  'ops.useCount':                   'Usage Count',
  'ops.downloadCount':              'Downloads',
  'ops.templateCreateSuccess':      'Template created successfully',
  'ops.deleteConfirmContent':       'This action cannot be undone. Are you sure?',
  'ops.newTemplate':                'New Template',
  'ops.newOpsTemplate':             'New Ops Template',
  'ops.templatesSubtitle':          'Manage commissioning and generic operation templates',
  'ops.commissionTemplates':        'Commissioning Templates',
  'ops.genericTemplates':           'Generic Templates',
  'ops.templateDetail':             'Template Detail',
  'ops.tags':                       'Tags',
  'ops.executionSteps':             'Execution Steps ({count} total)',
  'ops.stepTypeMml':                'MML Command',
  'ops.stepTypeCheck':              'Condition Check',
  'ops.stepTypeWait':               'Wait',
  'ops.stepTypeNotify':             'Notification',
  'ops.stepTypeScript':             'Script',
  'ops.condition':                  'Condition',
  'ops.waitSeconds':                'Wait {seconds} seconds',
  'ops.rollback':                   'Rollback',
  'ops.minutes':                    'min',
  'ops.seconds':                    'sec',
  'ops.catInspection':              'Inspection',
  'ops.catFault':                   'Fault Handling',
  'ops.catPerformance':             'Performance Optimization',
  'ops.catSoftware':                'Software Management',
  'ops.catNetwork':                 'Network Configuration',
  'ops.catMaintenance':             'Maintenance',
  'ops.descriptionPlaceholder':     'Enter template description',
  'ops.commandOrDevice':            'Command/Device',
  'ops.commandOrDevicePlaceholder': 'Enter command text or device SN',
  'ops.deviceSnPlaceholder':        'Enter device SN',
  'ops.operatorPlaceholder':        'Enter operator',
  'ops.executeResult':              'Execution Result',
  'ops.executeTime':                'Execution Time',
  'ops.command':                    'Command',
  'ops.duration':                   'Duration',
  'ops.outputSummary':              'Output Summary',
  'ops.commandDetail':              'Command Execution Detail',
  'ops.executeSuccess':             'Execution Succeeded',
  'ops.executeFailed':              'Execution Failed',
  'ops.executeInfo':                'Execution Info',
  'ops.outputResult':               'Output Result',
  'ops.noOutput':                   '(No output)',
  'ops.executeFailedNoOutput':      'Command execution failed, no valid output.',
  'ops.commandsSubtitle':           'View MML command execution records',
  'ops.downloadReady':              'Ready',
  'ops.downloadExpired':            'Expired',
  'ops.fileType':                   'File Type',
  'ops.fileTypeConfig':             'Config File',
  'ops.fileTypeLog':                'Log File',
  'ops.fileTypeFirmware':           'Firmware',
  'ops.fileTypeMR':                 'MR Data',
  'ops.fileTypePerf':               'Performance Data',
  'ops.reportType':                 'Report Type',
  'ops.reportTypeLTE':              'LTE Standard Report',
  'ops.reportTypeStation':          'Station Report',
  'ops.reportTypeKPI':              'KPI History',
  'ops.reportTypeMR':               'MR Analysis Report',
  'ops.reportTypePoll':             'Poll Statistics',
  'ops.fileNameOrSn':               'File Name/Device SN',
  'ops.fileNameOrSnPlaceholder':    'Enter file name or device SN',
  'ops.reportName':                 'Report Name',
  'ops.reportNamePlaceholder':      'Enter report name',
  'ops.sourceDevice':               'Source Device',
  'ops.expireTime':                 'Expiry Time',
  'ops.generateTime':               'Generated At',
  'ops.statsPeriod':                'Statistics Period',
  'ops.startDownload':              'Downloading',
  'ops.retry':                      'Retry',
  'ops.fileRegenerated':            'File regenerated',
  'ops.reportRegenerated':          'Report regenerated',
  'ops.resourceFileDownload':       'Resource File Download',
  'ops.reportDownload':             'Report Download',
  'ops.downloadsSubtitle':          'Resource files and report download management',
  'ops.networkDiagnosisSubtitle':   'Ping / Traceroute network connectivity diagnosis tools',
  'ops.pingParams':                 'Ping Parameters',
  'ops.tracerouteParams':           'Traceroute Parameters',
  'ops.selectSourceDevice':         'Select source device (default: OMC)',
  'ops.targetAddress':              'Target Address',
  'ops.enterTargetAddress':         'Enter target IP or domain',
  'ops.ipOrDomain':                 'IP address or domain',
  'ops.count':                      'Count',
  'ops.intervalSeconds':            'Interval(s)',
  'ops.timeoutSeconds':             'Timeout(s)',
  'ops.maxHops':                    'Max Hops',
  'ops.startPing':                  'Start Ping',
  'ops.startTraceroute':            'Start Traceroute',
  'ops.stop':                       'Stop',
  'ops.pingStats':                  'Ping Statistics',
  'ops.transmitted':                'Transmitted',
  'ops.received':                   'Received',
  'ops.packetLoss':                 'Packet Loss',
  'ops.minRtt':                     'Min RTT',
  'ops.avgRtt':                     'Avg RTT',
  'ops.maxRtt':                     'Max RTT',
  'ops.tracerouteComplete':         'Traceroute complete. Route trace results are displayed in the terminal above.',
  'ops.allOmcServer':               'All (OMC Server)',
};

export default messages;
