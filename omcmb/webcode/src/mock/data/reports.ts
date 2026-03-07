export type ReportType = 'performance' | 'alarm' | 'device' | 'capacity' | 'security';
export type ReportStatus = 'draft' | 'published' | 'archived';
export type ReportFormat = 'pdf' | 'excel' | 'html' | 'csv';
export type ReportPeriod = 'daily' | 'weekly' | 'monthly' | 'quarterly' | 'custom';

export interface ReportDefinition {
  id: string;
  reportName: string;
  reportType: ReportType;
  description: string;
  format: ReportFormat[];
  period: ReportPeriod;
  kpiCodes?: string[];
  deviceGroups?: string[];
  autoGenerate: boolean;
  cronExpression?: string;
  status: ReportStatus;
  createTime: string;
  creator: string;
  lastGenTime?: string;
}

export interface ReportRecord {
  id: string;
  reportDefinitionId: string;
  reportName: string;
  period: string;
  generateTime: string;
  fileSize: number;
  downloadUrl: string;
  format: ReportFormat;
  status: 'generating' | 'ready' | 'failed';
}

export const mockReportDefinitions: ReportDefinition[] = [
  {
    id: 'rdef-001',
    reportName: '全网性能日报',
    reportType: 'performance',
    description: '全网所有设备KPI性能日度分析报告',
    format: ['pdf', 'excel'],
    period: 'daily',
    kpiCodes: ['RRC_SUCC_RATE', 'ERAB_SUCC_RATE', 'HO_SUCC_RATE', 'DL_THROUGHPUT', 'UL_THROUGHPUT'],
    autoGenerate: true,
    cronExpression: '0 6 * * *',
    status: 'published',
    createTime: '2024-01-01T00:00:00.000Z',
    creator: 'admin',
    lastGenTime: '2024-06-14T06:00:00.000Z',
  },
  {
    id: 'rdef-002',
    reportName: '告警周报',
    reportType: 'alarm',
    description: '每周告警统计分析，包含告警趋势和TOP10告警',
    format: ['pdf', 'excel'],
    period: 'weekly',
    autoGenerate: true,
    cronExpression: '0 8 * * 1',
    status: 'published',
    createTime: '2024-01-01T00:00:00.000Z',
    creator: 'admin',
    lastGenTime: '2024-06-10T08:00:00.000Z',
  },
  {
    id: 'rdef-003',
    reportName: '设备资产月报',
    reportType: 'device',
    description: '全网设备资产清单和状态月度统计',
    format: ['excel', 'csv'],
    period: 'monthly',
    autoGenerate: true,
    cronExpression: '0 7 1 * *',
    status: 'published',
    createTime: '2024-01-01T00:00:00.000Z',
    creator: 'admin',
    lastGenTime: '2024-06-01T07:00:00.000Z',
  },
  {
    id: 'rdef-004',
    reportName: '容量规划季报',
    reportType: 'capacity',
    description: '网络容量分析和扩容建议季度报告',
    format: ['pdf'],
    period: 'quarterly',
    kpiCodes: ['PRB_UTIL', 'MAX_UE_COUNT', 'DL_THROUGHPUT', 'UL_THROUGHPUT'],
    autoGenerate: false,
    status: 'published',
    createTime: '2024-01-01T00:00:00.000Z',
    creator: 'operator01',
    lastGenTime: '2024-04-01T00:00:00.000Z',
  },
  {
    id: 'rdef-005',
    reportName: '安全审计报告',
    reportType: 'security',
    description: '操作安全审计日志月度分析',
    format: ['pdf', 'html'],
    period: 'monthly',
    autoGenerate: true,
    cronExpression: '0 9 1 * *',
    status: 'published',
    createTime: '2024-02-01T00:00:00.000Z',
    creator: 'admin',
    lastGenTime: '2024-06-01T09:00:00.000Z',
  },
  {
    id: 'rdef-006',
    reportName: '华北区性能周报',
    reportType: 'performance',
    description: '华北大区设备性能周度分析',
    format: ['pdf', 'excel'],
    period: 'weekly',
    kpiCodes: ['RRC_SUCC_RATE', 'ERAB_SUCC_RATE', 'HO_SUCC_RATE'],
    deviceGroups: ['grp-north'],
    autoGenerate: true,
    cronExpression: '0 8 * * 1',
    status: 'published',
    createTime: '2024-03-01T00:00:00.000Z',
    creator: 'operator01',
    lastGenTime: '2024-06-10T08:00:00.000Z',
  },
  {
    id: 'rdef-007',
    reportName: '5G gNB专项报告',
    reportType: 'performance',
    description: '5G gNB性能专项分析月报',
    format: ['pdf', 'excel'],
    period: 'monthly',
    kpiCodes: ['RRC_SUCC_RATE', 'DL_THROUGHPUT', 'UL_THROUGHPUT', 'AVG_SINR', 'PRB_UTIL'],
    deviceGroups: ['grp-5g'],
    autoGenerate: false,
    status: 'draft',
    createTime: '2024-05-01T00:00:00.000Z',
    creator: 'operator02',
  },
  {
    id: 'rdef-008',
    reportName: '告警处置效率报告',
    reportType: 'alarm',
    description: '告警响应时间和处置效率分析',
    format: ['pdf'],
    period: 'monthly',
    autoGenerate: false,
    status: 'published',
    createTime: '2024-04-01T00:00:00.000Z',
    creator: 'admin',
    lastGenTime: '2024-06-01T00:00:00.000Z',
  },
  {
    id: 'rdef-009',
    reportName: '自定义性能分析',
    reportType: 'performance',
    description: '可自定义时间范围和KPI的性能分析报告',
    format: ['excel', 'csv', 'html'],
    period: 'custom',
    autoGenerate: false,
    status: 'draft',
    createTime: '2024-06-01T00:00:00.000Z',
    creator: 'operator01',
  },
  {
    id: 'rdef-010',
    reportName: '归档-2023年度报告',
    reportType: 'performance',
    description: '2023年全年网络运行综合年报（已归档）',
    format: ['pdf'],
    period: 'custom',
    autoGenerate: false,
    status: 'archived',
    createTime: '2024-01-15T00:00:00.000Z',
    creator: 'admin',
    lastGenTime: '2024-01-20T10:00:00.000Z',
  },
];

export const mockReportRecords: ReportRecord[] = [
  { id: 'rrec-001', reportDefinitionId: 'rdef-001', reportName: '全网性能日报-2024-06-14', period: '2024-06-14', generateTime: '2024-06-14T06:05:00.000Z', fileSize: 1024 * 1024 * 3, downloadUrl: '/reports/performance_daily_20240614.pdf', format: 'pdf', status: 'ready' },
  { id: 'rrec-002', reportDefinitionId: 'rdef-001', reportName: '全网性能日报-2024-06-13', period: '2024-06-13', generateTime: '2024-06-13T06:03:00.000Z', fileSize: 1024 * 1024 * 2, downloadUrl: '/reports/performance_daily_20240613.pdf', format: 'pdf', status: 'ready' },
  { id: 'rrec-003', reportDefinitionId: 'rdef-002', reportName: '告警周报-2024W24', period: '2024-W24', generateTime: '2024-06-10T08:10:00.000Z', fileSize: 1024 * 1024 * 5, downloadUrl: '/reports/alarm_weekly_2024W24.pdf', format: 'pdf', status: 'ready' },
  { id: 'rrec-004', reportDefinitionId: 'rdef-003', reportName: '设备资产月报-2024-06', period: '2024-06', generateTime: '2024-06-01T07:15:00.000Z', fileSize: 1024 * 512, downloadUrl: '/reports/device_monthly_202406.xlsx', format: 'excel', status: 'ready' },
  { id: 'rrec-005', reportDefinitionId: 'rdef-005', reportName: '安全审计报告-2024-06', period: '2024-06', generateTime: '2024-06-01T09:20:00.000Z', fileSize: 1024 * 1024 * 1, downloadUrl: '/reports/security_audit_202406.pdf', format: 'pdf', status: 'ready' },
  { id: 'rrec-006', reportDefinitionId: 'rdef-004', reportName: '容量规划季报-2024Q1', period: '2024-Q1', generateTime: '2024-04-05T10:00:00.000Z', fileSize: 1024 * 1024 * 8, downloadUrl: '/reports/capacity_2024Q1.pdf', format: 'pdf', status: 'ready' },
  { id: 'rrec-007', reportDefinitionId: 'rdef-001', reportName: '全网性能日报-2024-06-12', period: '2024-06-12', generateTime: '2024-06-12T06:08:00.000Z', fileSize: 1024 * 1024 * 3, downloadUrl: '/reports/performance_daily_20240612.pdf', format: 'pdf', status: 'ready' },
  { id: 'rrec-008', reportDefinitionId: 'rdef-006', reportName: '华北区性能周报-2024W24', period: '2024-W24', generateTime: '2024-06-10T08:15:00.000Z', fileSize: 1024 * 1024 * 2, downloadUrl: '/reports/north_weekly_2024W24.pdf', format: 'pdf', status: 'ready' },
  { id: 'rrec-009', reportDefinitionId: 'rdef-001', reportName: '全网性能日报-2024-06-15（生成中）', period: '2024-06-15', generateTime: new Date().toISOString(), fileSize: 0, downloadUrl: '', format: 'pdf', status: 'generating' },
  { id: 'rrec-010', reportDefinitionId: 'rdef-008', reportName: '告警处置效率报告-2024-06', period: '2024-06', generateTime: '2024-06-01T12:00:00.000Z', fileSize: 1024 * 1024 * 2, downloadUrl: '/reports/alarm_efficiency_202406.pdf', format: 'pdf', status: 'ready' },
];

export const mockReportSampleData = {
  kpiSummary: {
    rrcSuccRate: { value: 97.8, trend: 0.3, status: 'good' },
    erabSuccRate: { value: 96.5, trend: -0.2, status: 'good' },
    hoSuccRate: { value: 95.1, trend: 0.5, status: 'good' },
    dlThroughput: { value: 89.3, unit: 'Mbps', trend: 2.1, status: 'good' },
    ulThroughput: { value: 35.7, unit: 'Mbps', trend: 0.8, status: 'good' },
    radioDrop: { value: 0.8, trend: -0.1, status: 'good' },
  },
  alarmSummary: {
    totalAlarms: 650,
    byDay: [45, 52, 38, 61, 43, 29, 55],
    bySeverity: { critical: 12, major: 89, minor: 210, warning: 339 },
    top5Types: [
      { name: '小区不可用', count: 45 },
      { name: 'S1链路中断', count: 38 },
      { name: '射频单元故障', count: 29 },
      { name: 'GPS失锁', count: 25 },
      { name: '温度过高', count: 22 },
    ],
    avgClearTime: 42,
  },
  deviceSummary: {
    totalDevices: 200,
    online: 170,
    offline: 30,
    byType: { eNB: 80, gNB: 60, CPE: 40, eGW: 20 },
    byRegion: { 华北: 45, 华东: 60, 华南: 50, 西南: 25, 西北: 20 },
  },
};
