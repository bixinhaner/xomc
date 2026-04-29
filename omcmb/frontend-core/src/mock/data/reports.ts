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

// =============================================================================
// Station-level KPI mock (T-0022)
//
// Used by webcode StationReport / HistoricalKPI pages while the backend
// /pm/kpi/by-station endpoint is being finalised. When the real endpoint lands
// the pages should switch to a hook over reportsApi or pmApi without changing
// shape.
// =============================================================================

export interface StationKPIRecord {
  id: string;
  stationName: string;
  stationSn: string;
  region: string;
  date: string;
  accessRate: number;
  hoSuccessRate: number;
  prbUtilization: number;
  dropRate: number;
  volteMos: number;
  onlineUsers: number;
  dataVolume: number;
}

export const mockStationKPIs: StationKPIRecord[] = [
  { id: 'sk-001', stationName: '北京-eNB-0001', stationSn: 'ENB00001', region: '北京', date: '2024-06-01', accessRate: 99.85, hoSuccessRate: 99.92, prbUtilization: 45.2, dropRate: 0.02, volteMos: 4.21, onlineUsers: 234, dataVolume: 1024 * 150 },
  { id: 'sk-002', stationName: '北京-eNB-0002', stationSn: 'ENB00002', region: '北京', date: '2024-06-01', accessRate: 99.21, hoSuccessRate: 98.88, prbUtilization: 67.8, dropRate: 0.08, volteMos: 4.05, onlineUsers: 312, dataVolume: 1024 * 220 },
  { id: 'sk-003', stationName: '上海-eNB-0001', stationSn: 'ENB00010', region: '上海', date: '2024-06-01', accessRate: 99.95, hoSuccessRate: 99.98, prbUtilization: 38.5, dropRate: 0.01, volteMos: 4.35, onlineUsers: 189, dataVolume: 1024 * 120 },
  { id: 'sk-004', stationName: '广州-eNB-0001', stationSn: 'ENB00020', region: '广州', date: '2024-06-01', accessRate: 98.76, hoSuccessRate: 98.45, prbUtilization: 82.3, dropRate: 0.15, volteMos: 3.92, onlineUsers: 425, dataVolume: 1024 * 380 },
  { id: 'sk-005', stationName: '北京-gNB-0001', stationSn: 'GNB00001', region: '北京', date: '2024-06-01', accessRate: 99.98, hoSuccessRate: 99.99, prbUtilization: 28.9, dropRate: 0.005, volteMos: 4.48, onlineUsers: 98, dataVolume: 1024 * 580 },
  { id: 'sk-006', stationName: '深圳-eNB-0001', stationSn: 'ENB00030', region: '深圳', date: '2024-06-01', accessRate: 99.34, hoSuccessRate: 99.12, prbUtilization: 71.5, dropRate: 0.06, volteMos: 4.12, onlineUsers: 356, dataVolume: 1024 * 290 },
];

// =============================================================================
// Historical KPI time-series mock (T-0022)
// =============================================================================

export const HISTORICAL_KPI_OPTIONS = [
  { labelKey: 'kpi.accessRate', value: 'accessRate' },
  { labelKey: 'kpi.handoverSuccessRate', value: 'hoSuccessRate' },
  { labelKey: 'kpi.prbUtilization', value: 'prbUtil' },
  { labelKey: 'kpi.dropRate', value: 'dropRate' },
  { labelKey: 'kpi.onlineUsers', value: 'onlineUsers' },
  { labelKey: 'kpi.dlThroughput', value: 'dlThroughput' },
  { labelKey: 'kpi.ulThroughput', value: 'ulThroughput' },
] as const;

interface KPIRangeConfig {
  base: number;
  range: number;
  min: number;
  max: number;
}

const HISTORICAL_KPI_CONFIGS: Record<string, KPIRangeConfig> = {
  accessRate: { base: 99.5, range: 1, min: 97, max: 100 },
  hoSuccessRate: { base: 99, range: 1.5, min: 97, max: 100 },
  prbUtil: { base: 60, range: 30, min: 20, max: 95 },
  dropRate: { base: 0.05, range: 0.1, min: 0, max: 0.3 },
  onlineUsers: { base: 250, range: 100, min: 100, max: 500 },
  dlThroughput: { base: 80, range: 40, min: 20, max: 200 },
  ulThroughput: { base: 25, range: 15, min: 5, max: 80 },
};

export function generateHistoricalKPITimePoints(count: number, granularity: '15min' | '1h' | '1d'): string[] {
  const points: string[] = [];
  const now = new Date('2024-06-01T00:00:00.000Z');
  const intervalMs =
    granularity === '15min' ? 15 * 60 * 1000 : granularity === '1h' ? 60 * 60 * 1000 : 24 * 60 * 60 * 1000;
  for (let i = 0; i < count; i++) {
    const d = new Date(now.getTime() + i * intervalMs);
    if (granularity === '1d') {
      points.push(d.toISOString().slice(0, 10));
    } else {
      points.push(d.toISOString().slice(0, 16).replace('T', ' '));
    }
  }
  return points;
}

export function generateHistoricalKPIData(kpi: string, count: number): number[] {
  const cfg = HISTORICAL_KPI_CONFIGS[kpi] ?? HISTORICAL_KPI_CONFIGS.accessRate;
  const out: number[] = [];
  for (let i = 0; i < count; i++) {
    const noise = (Math.sin(i * 0.5) + Math.sin(i * 0.13)) / 2;
    let v = cfg.base + noise * cfg.range;
    if (v < cfg.min) v = cfg.min;
    if (v > cfg.max) v = cfg.max;
    out.push(Number(v.toFixed(3)));
  }
  return out;
}

// =============================================================================
// LTE Standard Report records (T-0022)
//
// Display-shape used by webcode LTEStandardReport. The shape is intentionally
// distinct from `ReportRecord` because the page renders category-bucketed
// listings rather than the raw generated-record stream returned by
// `/reports/records`. Once the backend exposes a categorised endpoint the
// shape can be unified.
// =============================================================================

export type LTEReportStatus = 'generated' | 'generating' | 'failed' | 'scheduled';

export interface LTEReportRecord {
  id: string;
  reportName: string;
  reportType: string;
  period: string;
  generatedTime: string;
  status: LTEReportStatus;
  fileSize?: number;
  category: string;
}

export const mockLTEReportRecords: LTEReportRecord[] = [
  { id: 'rr-001', reportName: '华北区域eNB日KPI报表_20240601', reportType: '日报', period: '2024-06-01', generatedTime: '2024-06-02T01:00:00.000Z', status: 'generated', fileSize: 1024 * 512, category: 'kpi-daily' },
  { id: 'rr-002', reportName: '华北区域eNB日KPI报表_20240602', reportType: '日报', period: '2024-06-02', generatedTime: '2024-06-03T01:00:00.000Z', status: 'generated', fileSize: 1024 * 480, category: 'kpi-daily' },
  { id: 'rr-003', reportName: '全网周KPI报表_W22_2024', reportType: '周报', period: '2024-W22', generatedTime: '2024-06-03T06:00:00.000Z', status: 'generated', fileSize: 1024 * 1024 * 2, category: 'kpi-weekly' },
  { id: 'rr-004', reportName: '基站可用性日报_20240601', reportType: '日报', period: '2024-06-01', generatedTime: '2024-06-02T02:00:00.000Z', status: 'generated', fileSize: 1024 * 256, category: 'avail-station' },
  { id: 'rr-005', reportName: '全网月KPI报表_2024-05', reportType: '月报', period: '2024-05', generatedTime: '2024-06-01T08:00:00.000Z', status: 'generated', fileSize: 1024 * 1024 * 8, category: 'kpi-monthly' },
  { id: 'rr-006', reportName: 'PRB利用率周报_W23_2024', reportType: '周报', period: '2024-W23', generatedTime: '', status: 'generating', category: 'cap-prb' },
  { id: 'rr-007', reportName: 'VoLTE质量日报_20240603', reportType: '日报', period: '2024-06-03', generatedTime: '', status: 'failed', category: 'qual-voice' },
];
