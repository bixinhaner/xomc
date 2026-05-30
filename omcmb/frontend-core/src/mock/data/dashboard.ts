import { generateTimeSeries } from '../utils';

export interface DashboardSummary {
  deviceCounts: {
    total: number;
    online: number;
    offline: number;
    alarm: number;
  };
  alarmCounts: {
    critical: number;
    major: number;
    minor: number;
    warning: number;
    total: number;
  };
  kpiSummary: {
    rrcSuccRate: number;
    erabSuccRate: number;
    hoSuccRate: number;
    dlThroughput: number;
    ulThroughput: number;
    radioDrop: number;
    prbUtil: number;
    voLteSuccRate: number;
  };
  taskSummary: {
    running: number;
    pending: number;
    success: number;
    failed: number;
  };
}

export interface DashboardChartData {
  alarmTrend: Array<{ date: string; critical: number; major: number; minor: number; warning: number }>;
  deviceStatusPie: Array<{ name: string; value: number }>;
  kpiTimeSeries: Record<string, Array<[string, number]>>;
  alarmTypePie: Array<{ name: string; value: number }>;
  deviceByRegion: Array<{ region: string; total: number; online: number; offline: number }>;
  topAlarmDevices: Array<{ deviceSN: string; technology: string; deviceName: string; alarmCount: number; severity: string }>;
}

export const mockDashboardSummary: DashboardSummary = {
  deviceCounts: {
    total: 200,
    online: 170,
    offline: 30,
    alarm: 45,
  },
  alarmCounts: {
    critical: 12,
    major: 30,
    minor: 45,
    warning: 63,
    total: 150,
  },
  kpiSummary: {
    rrcSuccRate: 97.8,
    erabSuccRate: 96.5,
    hoSuccRate: 95.1,
    dlThroughput: 89.3,
    ulThroughput: 35.7,
    radioDrop: 0.8,
    prbUtil: 62.4,
    voLteSuccRate: 98.2,
  },
  taskSummary: {
    running: 3,
    pending: 2,
    success: 45,
    failed: 5,
  },
};

function generateAlarmTrend(): DashboardChartData['alarmTrend'] {
  const trend = [];
  for (let i = 6; i >= 0; i--) {
    const d = new Date(Date.now() - i * 86400000);
    const dateStr = d.toISOString().slice(0, 10);
    trend.push({
      date: dateStr,
      critical: Math.floor(Math.random() * 5 + 8),
      major: Math.floor(Math.random() * 15 + 20),
      minor: Math.floor(Math.random() * 20 + 30),
      warning: Math.floor(Math.random() * 25 + 40),
    });
  }
  return trend;
}

export const mockDashboardChartData: DashboardChartData = {
  alarmTrend: generateAlarmTrend(),
  deviceStatusPie: [
    { name: '在线正常', value: 125 },
    { name: '在线告警', value: 45 },
    { name: '离线', value: 30 },
  ],
  kpiTimeSeries: {
    rrcSuccRate: generateTimeSeries(7, 60, 97.5, 2.5),
    erabSuccRate: generateTimeSeries(7, 60, 96.8, 3.0),
    hoSuccRate: generateTimeSeries(7, 60, 95.2, 4.0),
    dlThroughput: generateTimeSeries(7, 60, 85.0, 20.0),
    ulThroughput: generateTimeSeries(7, 60, 35.0, 10.0),
    prbUtil: generateTimeSeries(7, 60, 62.0, 15.0),
  },
  alarmTypePie: [
    { name: '无线告警', value: 42 },
    { name: '传输告警', value: 35 },
    { name: '硬件告警', value: 28 },
    { name: '系统告警', value: 20 },
    { name: '射频告警', value: 15 },
    { name: '其他告警', value: 10 },
  ],
  deviceByRegion: [
    { region: '华北', total: 45, online: 40, offline: 5 },
    { region: '华东', total: 60, online: 52, offline: 8 },
    { region: '华南', total: 50, online: 43, offline: 7 },
    { region: '西南', total: 25, online: 22, offline: 3 },
    { region: '西北', total: 20, online: 13, offline: 7 },
  ],
  topAlarmDevices: [
    { deviceSN: '120288069823C4B0020', technology: 'lte', deviceName: '120288069823C4B0020', alarmCount: 8, severity: 'critical' },
    { deviceSN: '1202000534228GNB0010', technology: 'nr', deviceName: '1202000534228GNB0010', alarmCount: 6, severity: 'major' },
    { deviceSN: '120288069823C4B0040', technology: 'lte', deviceName: '120288069823C4B0040', alarmCount: 5, severity: 'major' },
    { deviceSN: '1202000534228E0020', technology: 'lte', deviceName: '1202000534228E0020', alarmCount: 4, severity: 'minor' },
    { deviceSN: '120288069823C4B0080', technology: 'lte', deviceName: '120288069823C4B0080', alarmCount: 3, severity: 'minor' },
  ],
};

export const mockDashboardWidgets = {
  onlineRate: {
    value: 85.0,
    trend: 1.2,
    unit: '%',
    label: '设备在线率',
  },
  alarmHandleRate: {
    value: 78.3,
    trend: 3.5,
    unit: '%',
    label: '告警处置率',
  },
  networkAvailability: {
    value: 99.95,
    trend: 0.01,
    unit: '%',
    label: '网络可用性',
  },
  avgAlarmClearTime: {
    value: 42,
    trend: -5,
    unit: '分钟',
    label: '平均告警清除时间',
  },
};
