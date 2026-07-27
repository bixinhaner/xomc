/**
 * Dashboard Mock 数据
 *
 * 设计原则：
 * 1. 数据结构与后端 API 对齐（device_stats, alarm_stats, kpi_overview）
 * 2. 使用生成器函数产生有变化的数据，便于测试不同场景
 * 3. 使用英文键名，支持 i18n 国际化
 * 4. KPI 名称与后端指标库保持一致（RRC_CONN_SETUP_SR, NR_PDCP_RATE_DL 等）
 * 5. 支持动态 KPI 字段扩展
 */

import { generateTimeSeries } from '../utils';
import type { KPIDelta } from '../../types/dashboard';

// ============================================================================
// 类型定义
// ============================================================================

/**
 * 设备状态统计
 */
export interface DeviceStats {
  total: number;
  online: number;
  offline: number;
  alarm: number;
}

/**
 * 告警状态统计
 */
export interface AlarmStats {
  critical: number;
  major: number;
  minor: number;
  warning: number;
  total: number;
}

/**
 * KPI 概览（支持动态字段）
 * 后端返回的 kpi_overview 是 map[string]float64，支持自定义 KPI
 */
export interface KPIOverview {
  [key: string]: number | undefined;
}

/**
 * 任务状态统计
 */
export interface TaskStats {
  running: number;
  pending: number;
  success: number;
  failed: number;
}

/**
 * Dashboard 汇总数据
 */
export interface DashboardSummary {
  deviceCounts: DeviceStats;
  alarmCounts: AlarmStats;
  kpiSummary: KPIOverview;
  kpiDeltas: Record<string, KPIDelta>;  // KPI趋势数据（新增）
  taskSummary: TaskStats;
}

/**
 * 告警趋势数据点
 */
export interface AlarmTrendItem {
  date: string;
  critical: number;
  major: number;
  minor: number;
  warning: number;
}

/**
 * 设备状态分布（用于饼图）
 */
export interface DeviceStatusItem {
  name: string;
  value: number;
}

/**
 * KPI 时序数据点
 */
export type KPITimeSeriesPoint = [string, number];

/**
 * 告警类型分布
 */
export interface AlarmTypeItem {
  name: string;
  value: number;
}

/**
 * 区域设备统计
 */
export interface RegionDeviceStats {
  region: string;
  total: number;
  online: number;
  offline: number;
}

/**
 * Top 告警设备
 */
export interface TopAlarmDevice {
  deviceSN: string;
  technology: string;
  deviceName: string;
  alarmCount: number;
  severity: string;
}

/**
 * Dashboard 图表数据
 */
export interface DashboardChartData {
  alarmTrend: AlarmTrendItem[];
  deviceStatusPie: DeviceStatusItem[];
  kpiTimeSeries: Record<string, KPITimeSeriesPoint[]>;
  alarmTypePie: AlarmTypeItem[];
  deviceByRegion: RegionDeviceStats[];
  topAlarmDevices: TopAlarmDevice[];
}

// ============================================================================
// 配置常量
// ============================================================================

/**
 * KPI 指标配置
 * 与后端指标库保持一致
 * 支持 LTE (eNB)、NR (gNB)、GSM 三种制式的完整Panel指标
 */
const KPI_CONFIG = {
  // ============================================================================
  // LTE (eNB) 指标 - 6个Panel共17个指标
  // ============================================================================

  // Traffic Panel (4个) - 业务量
  K900010015: { base: 450, variance: 80 },    // LTE PDCP 下行流量 (GB)
  K900010016: { base: 85, variance: 20 },     // LTE PDCP 上行流量 (GB)
  K900010040: { base: 125, variance: 25 },    // LTE PDCP 下行速率 (Mbps)
  K900010041: { base: 35, variance: 10 },     // LTE PDCP 上行速率 (Mbps)

  // Availability Panel (1个) - 可用性
  K900010076: { base: 99.5, variance: 0.3 },  // LTE 小区可用率 (%)

  // Utilization Panel (2个) - 利用率
  K900010014: { base: 55, variance: 15 },     // LTE 下行PRB利用率 (%)
  K900010013: { base: 35, variance: 12 },     // LTE 上行PRB利用率 (%)

  // Accessibility Panel (4个) - 接入性
  K900010006: { base: 98.5, variance: 1.5 },  // 无线建立成功率 (%)
  K900010002: { base: 98, variance: 2 },      // RRC 建立成功率 (%)
  K900010005: { base: 97, variance: 3 },      // E-RAB 建立成功率 (%)
  K900010029: { base: 96, variance: 4 },      // CSFB 成功率 (%)

  // Retainability Panel (1个) - 保持性
  K900010027: { base: 0.15, variance: 0.2 },  // E-RAB 掉线率 (%)

  // Mobility Panel (4个) - 移动性
  K900010017: { base: 98, variance: 2 },      // 同基站切换-切出成功率 (%)
  K900010022: { base: 98, variance: 2 },      // 同基站切换-切入成功率 (%)
  K900010021: { base: 97, variance: 3 },      // 异基站切换-切出成功率 (%)
  K900010026: { base: 97, variance: 3 },      // 异基站切换-切入成功率 (%)

  // ============================================================================
  // NR (gNB) 指标 - 2个Panel共6个指标
  // ============================================================================

  // Traffic Panel (4个)
  KGNB0511: { base: 650, variance: 120 },     // NR PDCP 下行流量 (GB)
  KGNB0510: { base: 120, variance: 30 },      // NR PDCP 上行流量 (GB)
  KGNB0517: { base: 85, variance: 20 },       // NR PDCP 下行速率 (Mbps)
  KGNB0516: { base: 35, variance: 10 },       // NR PDCP 上行速率 (Mbps)

  // Utilization Panel (2个)
  KGNB0506: { base: 60, variance: 15 },       // NR PRB 下行利用率 (%)
  KGNB0505: { base: 40, variance: 12 },       // NR PRB 上行利用率 (%)

  // ============================================================================
  // GSM (2G) 指标 - 3个Panel共3个指标
  // ============================================================================

  // Accessibility Panel (1个)
  KGSM0102: { base: 97, variance: 2.5 },      // GSM 呼叫建立成功率 (%)

  // Retainability Panel (1个)
  KGSM0103: { base: 0.8, variance: 0.4 },     // GSM 呼叫掉线率 (%)

  // Mobility Panel (1个)
  KGSM0101: { base: 96, variance: 3 },        // GSM 切换成功率 (%)
} as const;

/**
 * 设备状态分布配置
 */
const DEVICE_STATUS_CONFIG = [
  { name: 'online_normal', baseValue: 125, variance: 20 },   // 在线正常
  { name: 'online_alarm', baseValue: 45, variance: 15 },      // 在线告警
  { name: 'offline', baseValue: 30, variance: 10 },           // 离线
] as const;

/**
 * 告警类型分布配置
 */
const ALARM_TYPE_CONFIG = [
  { name: 'wireless', baseValue: 42, variance: 12 },      // 无线告警
  { name: 'transmission', baseValue: 35, variance: 10 },  // 传输告警
  { name: 'hardware', baseValue: 28, variance: 8 },        // 硬件告警
  { name: 'system', baseValue: 20, variance: 6 },          // 系统告警
  { name: 'rf', baseValue: 15, variance: 5 },              // 射频告警
  { name: 'other', baseValue: 10, variance: 3 },           // 其他告警
] as const;

/**
 * 区域设备配置
 */
const REGION_CONFIG = [
  { region: 'North', baseTotal: 45, onlineRate: 0.89 },    // 华北
  { region: 'East', baseTotal: 60, onlineRate: 0.87 },      // 华东
  { region: 'South', baseTotal: 50, onlineRate: 0.86 },     // 华南
  { region: 'Southwest', baseTotal: 25, onlineRate: 0.88 }, // 西南
  { region: 'Northwest', baseTotal: 20, onlineRate: 0.65 }, // 西北
] as const;

/**
 * Top 告警设备种子数据
 */
const TOP_ALARM_DEVICE_SEEDS = [
  { deviceSN: '120288069823C4B0020', technology: 'lte', severity: 'critical' },
  { deviceSN: '1202000534228GNB0010', technology: 'nr', severity: 'major' },
  { deviceSN: '120288069823C4B0040', technology: 'lte', severity: 'major' },
  { deviceSN: '1202000534228E0020', technology: 'lte', severity: 'minor' },
  { deviceSN: '120288069823C4B0080', technology: 'lte', severity: 'minor' },
] as const;

// ============================================================================
// 工具函数
// ============================================================================

/**
 * 生成带变化的随机值
 */
function randomVariation(base: number, variance: number): number {
  const value = base + (Math.random() - 0.5) * 2 * variance;
  return Math.max(0, parseFloat(value.toFixed(2)));
}

/**
 * 生成随机整数
 */
function randomInt(min: number, max: number): number {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

/**
 * 生成告警趋势数据
 */
function generateAlarmTrend(days: number = 7): AlarmTrendItem[] {
  const trend: AlarmTrendItem[] = [];
  for (let i = days - 1; i >= 0; i--) {
    const d = new Date(Date.now() - i * 86400000);
    const dateStr = d.toISOString().slice(0, 10);
    trend.push({
      date: dateStr,
      critical: randomInt(8, 15),
      major: randomInt(25, 40),
      minor: randomInt(35, 55),
      warning: randomInt(50, 75),
    });
  }
  return trend;
}

/**
 * 生成 KPI 概览数据
 */
function generateKPIOverview(): KPIOverview {
  const overview: KPIOverview = {};
  for (const [kpiName, config] of Object.entries(KPI_CONFIG)) {
    overview[kpiName] = randomVariation(config.base, config.variance);
  }
  return overview;
}

/**
 * 生成设备状态分布
 */
function generateDeviceStatusPie(): DeviceStatusItem[] {
  return DEVICE_STATUS_CONFIG.map((config) => ({
    name: config.name,
    value: randomInt(
      Math.max(0, config.baseValue - config.variance),
      config.baseValue + config.variance
    ),
  }));
}

/**
 * 生成告警类型分布
 */
function generateAlarmTypePie(): AlarmTypeItem[] {
  return ALARM_TYPE_CONFIG.map((config) => ({
    name: config.name,
    value: randomInt(
      Math.max(0, config.baseValue - config.variance),
      config.baseValue + config.variance
    ),
  }));
}

/**
 * 生成区域设备统计
 */
function generateRegionStats(): RegionDeviceStats[] {
  return REGION_CONFIG.map((config) => {
    const total = randomInt(
      Math.max(0, config.baseTotal - 5),
      config.baseTotal + 5
    );
    const onlineRate = randomVariation(config.onlineRate, 0.05);
    const online = Math.round(total * Math.max(0, Math.min(1, onlineRate)));
    return {
      region: config.region,
      total,
      online,
      offline: total - online,
    };
  });
}

/**
 * 生成 Top 告警设备
 */
function generateTopAlarmDevices(): TopAlarmDevice[] {
  return TOP_ALARM_DEVICE_SEEDS.map((seed) => ({
    deviceSN: seed.deviceSN,
    technology: seed.technology,
    deviceName: seed.deviceSN,
    alarmCount: randomInt(3, 10),
    severity: seed.severity,
  }));
}

/**
 * 生成 KPI 时序数据
 */
function generateKPITimeSeries(): Record<string, KPITimeSeriesPoint[]> {
  const series: Record<string, KPITimeSeriesPoint[]> = {};
  for (const [kpiName, config] of Object.entries(KPI_CONFIG)) {
    // 根据指标类型选择不同的时间粒度
    const intervalMinutes = kpiName.includes('RATE') || kpiName.includes('UTIL') ? 60 : 60;
    // 生成14天数据以确保包含完整的Today和Yesterday数据
    series[kpiName] = generateTimeSeries(14, intervalMinutes, config.base, config.variance);
  }
  return series;
}

// ============================================================================
// 导出的 Mock 数据
// ============================================================================

/**
 * 生成 Dashboard 汇总数据
 */
export function generateDashboardSummary(): DashboardSummary {
  const total = randomInt(190, 210);
  const online = randomInt(160, 180);
  const offline = total - online;
  const alarm = randomInt(40, 50);

  return {
    deviceCounts: {
      total,
      online,
      offline,
      alarm,
    },
    alarmCounts: {
      critical: randomInt(10, 15),
      major: randomInt(28, 35),
      minor: randomInt(40, 50),
      warning: randomInt(58, 68),
      total: randomInt(145, 165),
    },
    kpiSummary: generateKPIOverview(),
    kpiDeltas: generateKPIDeltas(total, randomInt(145, 165)),  // 添加 KPI 趋势数据
    taskSummary: {
      running: randomInt(2, 4),
      pending: randomInt(1, 3),
      success: randomInt(42, 48),
      failed: randomInt(4, 6),
    },
  };
}

/**
 * 生成 KPI 趋势增量数据（Mock）
 */
function generateKPIDeltas(currentTotalDevices: number, currentTotalAlarms: number): Record<string, KPIDelta> {
  // 生成一些变化趋势的假数据
  const prevTotalDevices = Math.round(currentTotalDevices * (1 + (Math.random() - 0.5) * 0.1));
  const prevTotalAlarms = Math.round(currentTotalAlarms * (1 + (Math.random() - 0.5) * 0.15));

  const deviceDelta: KPIDelta = {
    currentValue: currentTotalDevices,
    previousValue: prevTotalDevices,
    changePercent: ((currentTotalDevices - prevTotalDevices) / prevTotalDevices) * 100,
    trend: currentTotalDevices > prevTotalDevices ? 'up' : currentTotalDevices < prevTotalDevices ? 'down' : 'stable',
    compareType: 'last_week',
    hasComparison: prevTotalDevices !== 0,
  };

  const alarmDelta: KPIDelta = {
    currentValue: currentTotalAlarms,
    previousValue: prevTotalAlarms,
    changePercent: ((currentTotalAlarms - prevTotalAlarms) / prevTotalAlarms) * 100,
    trend: currentTotalAlarms > prevTotalAlarms ? 'up' : currentTotalAlarms < prevTotalAlarms ? 'down' : 'stable',
    compareType: 'yesterday',
    hasComparison: prevTotalAlarms !== 0,
  };

  return {
    total_devices: deviceDelta,
    active_alarms: alarmDelta,
  };
}

/**
 * 生成 Dashboard 图表数据
 */
export function generateDashboardChartData(): DashboardChartData {
  return {
    alarmTrend: generateAlarmTrend(),
    deviceStatusPie: generateDeviceStatusPie(),
    kpiTimeSeries: generateKPITimeSeries(),
    alarmTypePie: generateAlarmTypePie(),
    deviceByRegion: generateRegionStats(),
    topAlarmDevices: generateTopAlarmDevices(),
  };
}

// ============================================================================
// 向后兼容的静态导出（保持原有导出方式）
// ============================================================================

/**
 * 静态 Mock 数据（每次调用返回相同的生成结果）
 * 使用生成函数确保数据一致性
 */
let _cachedSummary: DashboardSummary | null = null;
let _cachedChartData: DashboardChartData | null = null;

/**
 * 重置缓存，强制重新生成数据
 * 用于测试不同场景或KPI_CONFIG更新后
 */
export function resetDashboardCache(): void {
  _cachedSummary = null;
  _cachedChartData = null;
}

// 立即执行一次缓存重置，确保KPI_CONFIG更新后生成新数据
resetDashboardCache();

export const mockDashboardSummary: DashboardSummary = (() => {
  if (!_cachedSummary) {
    _cachedSummary = generateDashboardSummary();
  }
  return { ..._cachedSummary };
})();

export const mockDashboardChartData: DashboardChartData = (() => {
  if (!_cachedChartData) {
    _cachedChartData = generateDashboardChartData();
  }
  return {
    alarmTrend: [..._cachedChartData.alarmTrend],
    deviceStatusPie: [..._cachedChartData.deviceStatusPie],
    kpiTimeSeries: { ..._cachedChartData.kpiTimeSeries },
    alarmTypePie: [..._cachedChartData.alarmTypePie],
    deviceByRegion: [..._cachedChartData.deviceByRegion],
    topAlarmDevices: [..._cachedChartData.topAlarmDevices],
  };
})();

/**
 * Dashboard Widget 配置
 */
export const mockDashboardWidgets = {
  onlineRate: {
    value: 85.0,
    trend: 1.2,
    unit: '%',
    label: 'onlineRate',
  },
  alarmHandleRate: {
    value: 78.3,
    trend: 3.5,
    unit: '%',
    label: 'alarmHandleRate',
  },
  networkAvailability: {
    value: 99.95,
    trend: 0.01,
    unit: '%',
    label: 'networkAvailability',
  },
  avgAlarmClearTime: {
    value: 42,
    trend: -5,
    unit: 'minutes',
    label: 'avgAlarmClearTime',
  },
};

// ============================================================================
// 测试辅助函数
// ============================================================================

/**
 * 生成特定场景的测试数据
 */
export function generateScenarioData(scenario: 'healthy' | 'warning' | 'critical'): DashboardSummary {
  const base = generateDashboardSummary();

  switch (scenario) {
    case 'healthy':
      // 健康场景：高在线率、低告警、高成功率
      return {
        ...base,
        deviceCounts: {
          total: 200,
          online: 185,
          offline: 15,
          alarm: 10,
        },
        alarmCounts: {
          critical: 2,
          major: 5,
          minor: 3,
          warning: 5,
          total: 15,
        },
        kpiSummary: {
          RRC_CONN_SETUP_SR: 99.5,
          ERAB_SETUP_SR: 99.2,
          NR_SA_HO_SR: 98.8,
          NR_PDCP_RATE_DL: 95.0,
          NR_PDCP_RATE_UL: 45.0,
          NR_PRB_UTIL_DL: 55.0,
          CALL_DROP_RATE: 0.05,
        },
      };

    case 'warning':
      // 关注场景：中等告警、部分指标下降
      return {
        ...base,
        alarmCounts: {
          critical: 15,
          major: 35,
          minor: 50,
          warning: 70,
          total: 170,
        },
        kpiSummary: {
          RRC_CONN_SETUP_SR: 97.0,
          ERAB_SETUP_SR: 95.5,
          NR_SA_HO_SR: 94.0,
          NR_PDCP_RATE_DL: 70.0,
          NR_PDCP_RATE_UL: 30.0,
          NR_PRB_UTIL_DL: 75.0,
          CALL_DROP_RATE: 0.5,
        },
      };

    case 'critical':
      // 严重场景：大量告警、指标严重下降
      return {
        ...base,
        deviceCounts: {
          total: 200,
          online: 140,
          offline: 60,
          alarm: 80,
        },
        alarmCounts: {
          critical: 45,
          major: 60,
          minor: 45,
          warning: 30,
          total: 180,
        },
        kpiSummary: {
          RRC_CONN_SETUP_SR: 92.0,
          ERAB_SETUP_SR: 90.5,
          NR_SA_HO_SR: 88.0,
          NR_PDCP_RATE_DL: 45.0,
          NR_PDCP_RATE_UL: 15.0,
          NR_PRB_UTIL_DL: 90.0,
          CALL_DROP_RATE: 2.5,
        },
      };

    default:
      return base;
  }
}
