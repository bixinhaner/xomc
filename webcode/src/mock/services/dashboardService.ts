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

  async getKPITimeSeries(): Promise<DashboardChartData['kpiTimeSeries']> {
    await delay(100, 200);
    return mockDashboardChartData.kpiTimeSeries;
  },
};
