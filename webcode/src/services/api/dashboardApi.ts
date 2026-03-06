import http from '../http';
import type { DashboardSummary, DashboardChartData } from '@/mock/data/dashboard';
import { dashboardService } from '@/mock/services/dashboardService';

// --- Backend response types ---

interface BackendDeviceStats {
  total: number;
  online: number;
  offline: number;
  alarm: number;
}

interface BackendAlarmStats {
  critical: number;
  major: number;
  minor: number;
  warning: number;
  total: number;
}

interface BackendKPIOverview {
  rrc_succ_rate?: number;
  erab_succ_rate?: number;
  ho_succ_rate?: number;
  dl_throughput?: number;
  ul_throughput?: number;
  radio_drop?: number;
  prb_util?: number;
  volte_succ_rate?: number;
}

interface BackendRecentAlarm {
  device_name: string;
  alarm_count: number;
  severity: string;
}

interface BackendDashboardSummary {
  device_stats: BackendDeviceStats;
  alarm_stats: BackendAlarmStats;
  kpi_overview: BackendKPIOverview;
  recent_alarms: BackendRecentAlarm[];
  timestamp: string;
}

// --- Mapping functions ---

function mapBackendSummary(b: BackendDashboardSummary): DashboardSummary {
  return {
    deviceCounts: {
      total: b.device_stats.total,
      online: b.device_stats.online,
      offline: b.device_stats.offline,
      alarm: b.device_stats.alarm,
    },
    alarmCounts: {
      critical: b.alarm_stats.critical,
      major: b.alarm_stats.major,
      minor: b.alarm_stats.minor,
      warning: b.alarm_stats.warning,
      total: b.alarm_stats.total,
    },
    kpiSummary: {
      rrcSuccRate: b.kpi_overview.rrc_succ_rate ?? 0,
      erabSuccRate: b.kpi_overview.erab_succ_rate ?? 0,
      hoSuccRate: b.kpi_overview.ho_succ_rate ?? 0,
      dlThroughput: b.kpi_overview.dl_throughput ?? 0,
      ulThroughput: b.kpi_overview.ul_throughput ?? 0,
      radioDrop: b.kpi_overview.radio_drop ?? 0,
      prbUtil: b.kpi_overview.prb_util ?? 0,
      voLteSuccRate: b.kpi_overview.volte_succ_rate ?? 0,
    },
    taskSummary: {
      running: 0,
      pending: 0,
      success: 0,
      failed: 0,
    },
  };
}

function mapRecentAlarmsToTopDevices(
  recentAlarms: BackendRecentAlarm[]
): DashboardChartData['topAlarmDevices'] {
  return recentAlarms.map((a) => ({
    deviceName: a.device_name,
    alarmCount: a.alarm_count,
    severity: a.severity,
  }));
}

// --- Exported service ---

export const dashboardApi = {
  /** Fetch dashboard summary from backend and map to DashboardSummary */
  async getSummary(): Promise<DashboardSummary> {
    const { data } = await http.get<BackendDashboardSummary>(
      '/dashboard/summary'
    );
    return mapBackendSummary(data);
  },

  /** Fetch full dashboard data — summary comes from backend, charts delegate to mock */
  async getDashboardData(): Promise<{
    summary: DashboardSummary;
    chartData: DashboardChartData;
    widgets: Awaited<ReturnType<typeof dashboardService.getDashboardData>>['widgets'];
  }> {
    const [summary, mockData] = await Promise.all([
      dashboardApi.getSummary(),
      dashboardService.getDashboardData(),
    ]);
    return {
      summary,
      chartData: mockData.chartData,
      widgets: mockData.widgets,
    };
  },

  /** Chart data — no dedicated backend endpoint, delegate to mock */
  getChartData: dashboardService.getChartData.bind(dashboardService),

  /** Alarm trend — no dedicated backend endpoint, delegate to mock */
  getAlarmTrend: dashboardService.getAlarmTrend.bind(dashboardService),

  /** Device status pie — no dedicated backend endpoint, delegate to mock */
  getDeviceStatusPie: dashboardService.getDeviceStatusPie.bind(dashboardService),

  /** Top alarm devices — extracted from /dashboard/summary recent_alarms */
  async getTopAlarmDevices(): Promise<DashboardChartData['topAlarmDevices']> {
    const { data } = await http.get<BackendDashboardSummary>(
      '/dashboard/summary'
    );
    return mapRecentAlarmsToTopDevices(data.recent_alarms || []);
  },

  /** KPI trend — no dedicated backend endpoint, delegate to mock */
  getKPITrend: dashboardService.getKPITrend.bind(dashboardService),

  /** Region stats — no dedicated backend endpoint, delegate to mock */
  getRegionStats: dashboardService.getRegionStats.bind(dashboardService),
};
