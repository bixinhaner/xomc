import { mockDashboardSummary, mockDashboardChartData, mockDashboardWidgets } from '../data/dashboard';
import type { DashboardSummary, DashboardChartData } from '../data/dashboard';
import { delay } from '../utils';

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

    // 如果指定了 kpiNames，只返回指定的 KPI 数据
    if (kpiNames && kpiNames.length > 0) {
      const result: DashboardChartData['kpiTimeSeries'] = {};
      for (const name of kpiNames) {
        result[name] = mockDashboardChartData.kpiTimeSeries[name] || [];
      }
      return result;
    }

    return mockDashboardChartData.kpiTimeSeries;
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
    console.log('getKPITrendComparison called with:', kpiName);
    console.log('Available keys:', Object.keys(mockDashboardChartData.kpiTimeSeries));
    const baseData = mockDashboardChartData.kpiTimeSeries[kpiName] || [];
    console.log('baseData length:', baseData.length);
    const current = baseData.slice(-24).map(([time, value]) => ({ time, value }));
    const compare = baseData.slice(-48, -24).map(([time, value]) => ({ time, value }));
    console.log('Returning data:', { current: current.length, compare: compare.length });
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
