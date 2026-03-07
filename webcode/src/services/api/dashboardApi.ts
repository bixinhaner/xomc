import http from '../http';
import type { DashboardSummary, DashboardChartData } from '@/mock/data/dashboard';

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

// Backend response types for Sprint 7 chart endpoints

interface BackendAlarmTrendItem {
  date: string;
  critical: number;
  major: number;
  minor: number;
  warning: number;
}

/** GET /dashboard/device-status returns a map of status label → count */
type BackendDeviceStatusMap = Record<string, number>;

interface BackendKPITrendItem {
  time: string;
  value: number;
}

interface BackendRegionStatsItem {
  region: string;
  device_count: number;
  online_count: number;
  alarm_count: number;
}

// Backend response types for Phase A2 endpoints

interface BackendWidgetLayout {
  id: string;
  user_id: string;
  layout: unknown;
  created_at: string;
  updated_at: string;
}

interface BackendAlarmTypePieItem {
  name: string;
  value: number;
}

interface BackendKPITimeSeriesEntry {
  time: string;
  value: number;
}

/** GET /dashboard/kpi-time-series returns { kpiName: [{ time, value }, ...], ... } */
type BackendKPITimeSeriesResponse = Record<string, BackendKPITimeSeriesEntry[]>;

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

function mapDeviceStatusMap(
  statusMap: BackendDeviceStatusMap
): DashboardChartData['deviceStatusPie'] {
  return Object.entries(statusMap).map(([name, value]) => ({ name, value }));
}

function mapKPITrend(
  items: BackendKPITrendItem[]
): Array<[string, number]> {
  return items.map((item) => [item.time, item.value]);
}

function mapRegionStats(
  items: BackendRegionStatsItem[]
): DashboardChartData['deviceByRegion'] {
  return items.map((item) => ({
    region: item.region,
    total: item.device_count,
    online: item.online_count,
    offline: item.device_count - item.online_count,
  }));
}

function mapKPITimeSeries(
  data: BackendKPITimeSeriesResponse
): DashboardChartData['kpiTimeSeries'] {
  const result: Record<string, Array<[string, number]>> = {};
  for (const [kpiName, entries] of Object.entries(data)) {
    result[kpiName] = entries.map((e) => [e.time, e.value]);
  }
  return result;
}

function mapAlarmTypePie(
  items: BackendAlarmTypePieItem[]
): DashboardChartData['alarmTypePie'] {
  return items.map((item) => ({ name: item.name, value: item.value }));
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

  /** Fetch full dashboard data — summary + charts + widgets all from backend */
  async getDashboardData(): Promise<{
    summary: DashboardSummary;
    chartData: DashboardChartData;
    widgets: BackendWidgetLayout;
  }> {
    const [summary, alarmTrend, deviceStatusPie, topAlarmDevices, deviceByRegion, alarmTypePie, kpiTimeSeries, widgets] =
      await Promise.all([
        dashboardApi.getSummary(),
        dashboardApi.getAlarmTrend(),
        dashboardApi.getDeviceStatusPie(),
        dashboardApi.getTopAlarmDevices(),
        dashboardApi.getRegionStats(),
        dashboardApi.getAlarmTypePie(),
        dashboardApi.getKPITimeSeries(),
        dashboardApi.getWidgets(),
      ]);
    return {
      summary,
      chartData: {
        alarmTrend,
        deviceStatusPie,
        kpiTimeSeries,
        alarmTypePie,
        deviceByRegion,
        topAlarmDevices,
      },
      widgets,
    };
  },

  /** Chart data — all from real backend endpoints */
  async getChartData(): Promise<DashboardChartData> {
    const [alarmTrend, deviceStatusPie, topAlarmDevices, deviceByRegion, alarmTypePie, kpiTimeSeries] =
      await Promise.all([
        dashboardApi.getAlarmTrend(),
        dashboardApi.getDeviceStatusPie(),
        dashboardApi.getTopAlarmDevices(),
        dashboardApi.getRegionStats(),
        dashboardApi.getAlarmTypePie(),
        dashboardApi.getKPITimeSeries(),
      ]);
    return {
      alarmTrend,
      deviceStatusPie,
      kpiTimeSeries,
      alarmTypePie,
      deviceByRegion,
      topAlarmDevices,
    };
  },

  /** Alarm trend from GET /dashboard/alarm-trend */
  async getAlarmTrend(days: number = 7): Promise<DashboardChartData['alarmTrend']> {
    const { data } = await http.get<BackendAlarmTrendItem[]>(
      '/dashboard/alarm-trend',
      { params: { days } }
    );
    return data;
  },

  /** Device status pie from GET /dashboard/device-status */
  async getDeviceStatusPie(): Promise<DashboardChartData['deviceStatusPie']> {
    const { data } = await http.get<BackendDeviceStatusMap>(
      '/dashboard/device-status'
    );
    return mapDeviceStatusMap(data);
  },

  /** Top alarm devices — extracted from /dashboard/summary recent_alarms */
  async getTopAlarmDevices(): Promise<DashboardChartData['topAlarmDevices']> {
    const { data } = await http.get<BackendDashboardSummary>(
      '/dashboard/summary'
    );
    return mapRecentAlarmsToTopDevices(data.recent_alarms || []);
  },

  /** KPI trend from GET /dashboard/kpi-trend */
  async getKPITrend(kpiCode: string, days: number = 7): Promise<Array<[string, number]>> {
    const { data } = await http.get<BackendKPITrendItem[]>(
      '/dashboard/kpi-trend',
      { params: { kpi_name: kpiCode, days } }
    );
    return mapKPITrend(data);
  },

  /** Region stats from GET /dashboard/region-stats */
  async getRegionStats(): Promise<DashboardChartData['deviceByRegion']> {
    const { data } = await http.get<BackendRegionStatsItem[]>(
      '/dashboard/region-stats'
    );
    return mapRegionStats(data);
  },

  /** Widget layout from GET /dashboard/widgets */
  async getWidgets(): Promise<BackendWidgetLayout> {
    const { data } = await http.get<BackendWidgetLayout>('/dashboard/widgets');
    return data;
  },

  /** Save widget layout via PUT /dashboard/widgets */
  async saveWidgets(layout: unknown): Promise<BackendWidgetLayout> {
    const { data } = await http.put<BackendWidgetLayout>('/dashboard/widgets', {
      layout,
    });
    return data;
  },

  /** Alarm type pie from GET /dashboard/alarm-type-pie */
  async getAlarmTypePie(): Promise<DashboardChartData['alarmTypePie']> {
    const { data } = await http.get<BackendAlarmTypePieItem[]>(
      '/dashboard/alarm-type-pie'
    );
    return mapAlarmTypePie(data);
  },

  /** KPI time series from GET /dashboard/kpi-time-series */
  async getKPITimeSeries(
    kpiNames?: string[],
    startTime?: string,
    endTime?: string
  ): Promise<DashboardChartData['kpiTimeSeries']> {
    const defaultNames = [
      'rrcSuccRate',
      'erabSuccRate',
      'hoSuccRate',
      'dlThroughput',
      'ulThroughput',
      'prbUtil',
    ];
    const names = kpiNames ?? defaultNames;
    const { data } = await http.get<BackendKPITimeSeriesResponse>(
      '/dashboard/kpi-time-series',
      {
        params: {
          kpi_names: names.join(','),
          ...(startTime ? { start_time: startTime } : {}),
          ...(endTime ? { end_time: endTime } : {}),
        },
      }
    );
    return mapKPITimeSeries(data);
  },
};
