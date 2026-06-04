import { mockDashboardSummary, mockDashboardChartData, mockDashboardWidgets } from '../data/dashboard';
import type { DashboardSummary, DashboardChartData, KPITimeSeriesPoint } from '../data/dashboard';
import { delay } from '../utils';

// KPI 配置（用于动态生成数据）
const KPI_CONFIG: Record<string, { base: number; variance: number }> = {
  // LTE Traffic
  LTE_PDCP_VOLUME_DL: { base: 450, variance: 80 },
  LTE_PDCP_VOLUME_UL: { base: 120, variance: 30 },
  LTE_PDCP_RATE_DL: { base: 76, variance: 15 },
  LTE_PDCP_RATE_UL: { base: 18, variance: 5 },
  // LTE Availability
  LTE_CELL_AVAILABLE: { base: 99.5, variance: 0.3 },
  // LTE Utilization
  LTE_PRB_UTIL_DL: { base: 65, variance: 15 },
  LTE_PRB_UTIL_UL: { base: 45, variance: 12 },
  // LTE Accessibility
  WIRELESS_SETUP_SR: { base: 98.5, variance: 1.5 },
  RRC_CONN_SETUP_SR: { base: 99.2, variance: 0.8 },
  ERAB_SETUP_SR: { base: 98.8, variance: 1.2 },
  CSFB_SR: { base: 97.5, variance: 2.5 },
  // LTE Retainability
  ERAB_DROP_RATE: { base: 0.05, variance: 0.02 },
  // LTE Mobility
  HO_INTRA_ENB_OUT_SR: { base: 98.2, variance: 1.8 },
  HO_INTRA_ENB_IN_SR: { base: 98.5, variance: 1.5 },
  HO_INTER_ENB_OUT_SR: { base: 97.8, variance: 2.2 },
  HO_INTER_ENB_IN_SR: { base: 98.0, variance: 2.0 },
  // NR Traffic
  NR_PDCP_VOLUME_DL: { base: 650, variance: 120 },
  NR_PDCP_VOLUME_UL: { base: 180, variance: 45 },
  NR_PDCP_RATE_DL: { base: 120, variance: 25 },
  NR_PDCP_RATE_UL: { base: 35, variance: 10 },
  // NR Utilization
  NR_PRB_UTIL_DL: { base: 55, variance: 18 },
  NR_PRB_UTIL_UL: { base: 38, variance: 15 },
  // GSM
  GSM_CALL_SETUP_SR: { base: 97, variance: 2.5 },
  GSM_CALL_DROP_RATE: { base: 0.08, variance: 0.03 },
  GSM_HO_SR: { base: 96.5, variance: 3.0 },
};

export const dashboardService = {
  async getDashboardData(): Promise<{
    summary: DashboardSummary;
    chartData: DashboardChartData;
    widgets: typeof mockDashboardWidgets;
  }> {
    await delay(200, 400);
    return {
      summary: { ...mockDashboardSummary },
      chartData: { ...mockDashboardChartData },
      widgets: { ...mockDashboardWidgets },
    };
  },

  async getSummary(): Promise<DashboardSummary> {
    await delay(100, 200);
    return { ...mockDashboardSummary };
  },

  async getChartData(): Promise<DashboardChartData> {
    await delay(150, 300);
    return { ...mockDashboardChartData };
  },

  async getAlarmTrend(days = 7): Promise<DashboardChartData['alarmTrend']> {
    await delay(100, 200);
    return mockDashboardChartData.alarmTrend.slice(-days);
  },

  async getDeviceStatusPie(): Promise<DashboardChartData['deviceStatusPie']> {
    await delay(80, 150);
    return mockDashboardChartData.deviceStatusPie;
  },

  async getTopAlarmDevices(): Promise<DashboardChartData['topAlarmDevices']> {
    await delay(80, 150);
    return mockDashboardChartData.topAlarmDevices;
  },

  async getKPITrend(kpiCode: string): Promise<Array<[string, number]>> {
    await delay(100, 200);
    return mockDashboardChartData.kpiTimeSeries[kpiCode] ?? [];
  },

  async getRegionStats(): Promise<DashboardChartData['deviceByRegion']> {
    await delay(80, 150);
    return mockDashboardChartData.deviceByRegion;
  },

  async getWidgets(): Promise<{ id: string; user_id: string; layout: unknown; created_at: string; updated_at: string }> {
    await delay(80, 150);
    return {
      id: '',
      user_id: '',
      layout: mockDashboardWidgets,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
  },

  async saveWidgets(layout: unknown): Promise<{ id: string; user_id: string; layout: unknown; created_at: string; updated_at: string }> {
    await delay(100, 200);
    return {
      id: '',
      user_id: '',
      layout,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
  },

  async getAlarmTypePie(): Promise<DashboardChartData['alarmTypePie']> {
    await delay(80, 150);
    return mockDashboardChartData.alarmTypePie;
  },

  async getKPITimeSeries(
    kpiNames?: string[],
    startTime?: string,
    endTime?: string
  ): Promise<DashboardChartData['kpiTimeSeries']> {
    await delay(100, 200);

    const result: DashboardChartData['kpiTimeSeries'] = {};

    // 解析时间范围
    const start = startTime ? new Date(startTime) : new Date(Date.now() - 24 * 60 * 60 * 1000);
    const end = endTime ? new Date(endTime) : new Date();

    // 计算时间范围和间隔（每小时一个点）
    const intervalMs = 60 * 60 * 1000; // 1小时
    const totalPoints = Math.ceil((end.getTime() - start.getTime()) / intervalMs) + 1;

    // 对每个请求的KPI动态生成数据
    if (kpiNames && kpiNames.length > 0) {
      for (const kpiName of kpiNames) {
        const kpiConfig = KPI_CONFIG[kpiName];
        const base = kpiConfig?.base ?? 100;
        const variance = kpiConfig?.variance ?? 20;

        // 生成覆盖完整时间范围的数据
        const timeSeries: KPITimeSeriesPoint[] = [];
        for (let i = 0; i < totalPoints; i++) {
          const pointTime = new Date(start.getTime() + i * intervalMs);
          // 确保不超过结束时间
          if (pointTime > end) break;

          const timeStr = pointTime.toISOString();
          const value = Math.max(0, base + (Math.random() - 0.5) * 2 * variance);
          timeSeries.push([timeStr, parseFloat(value.toFixed(2))]);
        }

        result[kpiName] = timeSeries;
      }
    }

    return result;
  },

  async getDeviceStatusByType(): Promise<Record<string, { online: number; offline: number; alarm: number }>> {
    await delay(80, 150);
    return {
      lte: { online: 120, offline: 15, alarm: 8 },
      nr: { online: 45, offline: 5, alarm: 3 },
      gsm: { online: 10, offline: 10, alarm: 0 },
    };
  },

  async getKPITrendComparison(kpiName: string, compareWith: 'yesterday' | 'last_week' = 'yesterday'): Promise<{
    current: Array<{ time: string; value: number }>;
    compare: Array<{ time: string; value: number }>;
    metadata: { kpi_name: string; compare_type: string; change_percent?: number };
  }> {
    await delay(100, 200);
    const baseData = mockDashboardChartData.kpiTimeSeries[kpiName] || [];
    const current = baseData.slice(-24).map(([time, value]) => ({ time, value }));
    const compare = baseData.slice(-48, -24).map(([time, value]) => ({ time, value }));
    return {
      current,
      compare,
      metadata: {
        kpi_name: kpiName,
        compare_type: compareWith,
        change_percent: Math.random() * 10 - 5,
      },
    };
  },
};
