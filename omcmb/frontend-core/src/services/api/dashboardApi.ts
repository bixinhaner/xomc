import http from '../http';
import type {
  DashboardSummary,
  KPIDelta,
  EfficiencyMetrics,
  HeatmapData,
  AlarmHeatmapBySeverity,
  KPIDefinitionsResponse,
  BackendKPILayout,
  KPILayout,
  KPILayoutPanel,
  DashboardKPIGranularity,
  DashboardKPITimeSeriesSnapshot,
  DashboardPeriodProgress,
} from '../../types/dashboard';
import type { DashboardChartData } from '../../mock/data/dashboard';

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

// Backend returns kpi_overview as map[string]float64 with dynamic keys
// like "NR_PDCP_RATE_DL", "RRC_CONN_SETUP_SR", etc.
interface BackendKPIOverview {
  [key: string]: number | undefined;
}

interface BackendRecentAlarm {
  device_sn: string;
  technology: string;
  device_name: string;
  alarm_count: number;
  severity: string;
}

interface BackendTopAlarmDevice {
  device_sn: string;
  technology: string;
  alarm_count: number;
  critical: number;
  major: number;
  minor: number;
  warning: number;
}

interface BackendDashboardSummary {
  device_stats: BackendDeviceStats;
  alarm_stats: BackendAlarmStats;
  kpi_overview: BackendKPIOverview;
  kpi_deltas: Record<string, BackendKPIDelta>;
  recent_alarms: BackendRecentAlarm[];
  pm_slot_health: BackendPMSlotHealth[];
  timestamp: string;
}

interface BackendPMSlotHealth {
  slot_end: string;
  technology: string;
  carrier: string;
  expected_devices: number;
  received_devices: number;
  coverage_ratio: number;
  status: 'complete' | 'partial' | 'missing' | 'bootstrap_ignored';
  evaluated_at: string;
}

/** 后端 KPI 趋势增量数据 */
interface BackendKPIDelta {
  current_value: number;
  previous_value: number;
  change_percent: number;
  trend: string;  // "up" | "down" | "stable"
  compare_type: string;  // "yesterday" | "last_week"
  has_comparison: boolean;
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

/** Device status counts for a single technology */
interface BackendDeviceStatusCounts {
  online: number;
  offline: number;
  alarm: number;
}

/** GET /dashboard/device-status-by-type returns { [technology]: { online, offline, alarm } } */
type BackendDeviceStatusByType = Record<string, BackendDeviceStatusCounts>;

interface BackendKPITrendItem {
  time: string;
  value: number;
}

/** KPI trend comparison response - 今日vs昨日对比 */
interface BackendKPITrendComparison {
  current: BackendKPITrendItem[];
  compare: BackendKPITrendItem[];
  metadata: {
    kpi_name: string;
    compare_type: string; // 后端返回普通字符串，需要断言为字面量类型
    change_percent?: number;
  };
}

/** 前端使用的 KPI 趋势对比类型（正确的字面量类型） */
interface KPITrendComparison {
  current: Array<{ time: string; value: number }>;
  compare: Array<{ time: string; value: number }>;
  metadata: {
    kpi_name: string;
    compare_type: 'yesterday' | 'last_week';
    change_percent?: number;
  };
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
  partial?: boolean;
}

/** GET /dashboard/kpi-time-series returns { kpiName: [{ time, value }, ...], ... } */
type BackendKPITimeSeriesResponse = Record<string, BackendKPITimeSeriesEntry[]>;

interface BackendDashboardPeriodProgress {
  task_id: string;
  task_version_id: string;
  granularity: DashboardKPIGranularity;
  window_start: string;
  window_end: string;
  entity_key: string;
  revision: number;
  version_effective_from: string;
  version_effective_to: string | null;
  received_slots: number;
  expected_slots: number;
  version_expected_slots: number;
  coverage_ratio: number;
  version_slice_complete: boolean;
  period_complete: boolean;
  state: 'partial';
}

interface BackendKPITimeSeriesSnapshot {
  series: BackendKPITimeSeriesResponse;
  period_progress: BackendDashboardPeriodProgress[];
  progress_state: DashboardKPITimeSeriesSnapshot['progressState'];
}

type ApiEnvelope<T> = {
  ret?: number;
  msg?: string;
  data?: T;
};

// --- Mapping functions ---

function mapBackendSummary(b: BackendDashboardSummary): DashboardSummary {
  // 透传 kpi_overview 全部动态键（如 UE_ACTIVE），再叠加命名快捷字段保持老调用点兼容。
  // 注意：'in' 探测保留 number 0 与 undefined 的区别（前者是真值，后者表示该 KPI 无数据）。
  const overview = b.kpi_overview ?? {};
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
      ...overview,
      rrcSuccRate: overview.RRC_CONN_SETUP_SR ?? 0,
      erabSuccRate: overview.ERAB_SETUP_SR ?? 0,
      hoSuccRate: overview.NR_SA_HO_SR ?? 0,
      dlThroughput: overview.NR_PDCP_RATE_DL ?? 0,
      ulThroughput: 0, // 数据库中暂无上行速率KPI
      radioDrop: overview.CALL_DROP_RATE ?? 0,
      prbUtil: overview.NR_PRB_UTIL_DL ?? 0,
      voLteSuccRate: 0, // 数据库中暂无VoLTE KPI
    },
    kpiDeltas: Object.entries(b.kpi_deltas || {}).reduce((acc, [kpiName, delta]) => {
      acc[kpiName] = {
        currentValue: delta.current_value,
        previousValue: delta.previous_value,
        changePercent: delta.change_percent,
        trend: delta.trend as 'up' | 'down' | 'stable',
        compareType: delta.compare_type as 'yesterday' | 'last_week',
        hasComparison: delta.has_comparison,
      };
      return acc;
    }, {} as Record<string, KPIDelta>),
    taskSummary: {
      running: 0,
      pending: 0,
      success: 0,
      failed: 0,
    },
    pmSlotHealth: (b.pm_slot_health ?? []).map((slot) => ({
      slotEnd: slot.slot_end,
      technology: slot.technology,
      carrier: slot.carrier,
      expectedDevices: slot.expected_devices,
      receivedDevices: slot.received_devices,
      coverageRatio: slot.coverage_ratio,
      status: slot.status,
      evaluatedAt: slot.evaluated_at,
    })),
  };
}

function mapTopAlarmDevices(
  devices: BackendTopAlarmDevice[]
): DashboardChartData['topAlarmDevices'] {
  return devices.map((device) => ({
    deviceSN: device.device_sn,
    technology: device.technology,
    alarmCount: device.alarm_count,
    critical: device.critical,
    major: device.major,
    minor: device.minor,
    warning: device.warning,
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

/**
 * 映射 KPI 趋势对比数据
 * 将后端的普通字符串 compare_type 转换为字面量类型
 */
function mapKPITrendComparison(
  data: BackendKPITrendComparison
): KPITrendComparison {
  return {
    current: data.current,
    compare: data.compare,
    metadata: {
      kpi_name: data.metadata.kpi_name,
      compare_type: data.metadata.compare_type as 'yesterday' | 'last_week',
      change_percent: data.metadata.change_percent,
    },
  };
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

function mapDashboardPeriodProgress(
  item: BackendDashboardPeriodProgress,
): DashboardPeriodProgress {
  return {
    taskId: item.task_id,
    taskVersionId: item.task_version_id,
    granularity: item.granularity,
    windowStart: item.window_start,
    windowEnd: item.window_end,
    entityKey: item.entity_key,
    revision: item.revision,
    versionEffectiveFrom: item.version_effective_from,
    versionEffectiveTo: item.version_effective_to,
    receivedSlots: item.received_slots,
    expectedSlots: item.expected_slots,
    versionExpectedSlots: item.version_expected_slots,
    coverageRatio: item.coverage_ratio,
    versionSliceComplete: item.version_slice_complete,
    periodComplete: item.period_complete,
    state: item.state,
  };
}

function mapAlarmTypePie(
  items: BackendAlarmTypePieItem[]
): DashboardChartData['alarmTypePie'] {
  return items.map((item) => ({ name: item.name, value: item.value }));
}

function unwrapDashboardEnvelope<T>(payload: T | ApiEnvelope<T>): T {
  if (
    payload &&
    typeof payload === 'object' &&
    'ret' in payload &&
    'data' in payload &&
    (payload as ApiEnvelope<T>).ret === 1
  ) {
    return (payload as ApiEnvelope<T>).data as T;
  }
  return payload as T;
}

/**
 * 映射全局 KPI 布局（issue #213 S2）。
 * 把后端 snake_case 信封映射为前端 camelCase 形状；panels 原样透传（已是 camelCase）。
 * 后端可能返回 layout 为 null / 缺 panels，统一兜底成空数组（首页回退由调用方处理）。
 */
function mapBackendKPILayout(b: BackendKPILayout): KPILayout {
  return {
    tech: b.tech,
    panels: b.layout?.panels ?? [],
    updatedAt: b.updated_at,
  };
}

/**
 * 把前端布局形状映射回后端存盘信封（issue #213 S3）。
 * panels 字段与后端一致（title/metrics/x,y/w,h/chartType），原样透传；
 * 仅包上 layout.panels 信封（updated_at/updated_by 由后端按当前管理员写入，前端不传）。
 */
function mapKPILayoutToBackend(panels: KPILayoutPanel[]): { panels: KPILayoutPanel[] } {
  return {
    panels: panels.map((p) => ({
      title: p.title,
      metrics: [...p.metrics],
      x: p.x,
      y: p.y,
      w: p.w,
      h: p.h,
      chartType: p.chartType,
    })),
  };
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
        // 暂时注释：PRB利用率、无线质量指标图表已隐藏，不需要获取全部KPI时序数据
        // 如需恢复显示，取消下面注释即可
        // dashboardApi.getKPITimeSeries(),
        {} as DashboardChartData['kpiTimeSeries'],
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
  async getAlarmTrend(
    days: number = 7,
    metric: 'raised' | 'active' = 'raised',
  ): Promise<DashboardChartData['alarmTrend']> {
    const { data } = await http.get<BackendAlarmTrendItem[]>(
      '/dashboard/alarm-trend',
      { params: { days, metric } }
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

  /** Device status by technology type from GET /dashboard/device-status-by-type */
  async getDeviceStatusByType(): Promise<BackendDeviceStatusByType> {
    const { data } = await http.get<BackendDeviceStatusByType>(
      '/dashboard/device-status-by-type'
    );
    return data;
  },

  /** Top alarm devices from current uncleared alarms */
  async getTopAlarmDevices(): Promise<DashboardChartData['topAlarmDevices']> {
    const { data } = await http.get<BackendTopAlarmDevice[]>(
      '/dashboard/top-alarm-devices'
    );
    return mapTopAlarmDevices(data || []);
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
    endTime?: string,
    granularity?: DashboardKPIGranularity,
    technology?: string,
  ): Promise<DashboardChartData['kpiTimeSeries']> {
    const defaultNames = [
      'RRC_CONN_SETUP_SR',
      'ERAB_SETUP_SR',
      'NR_SA_HO_SR',
      'NR_PDCP_RATE_DL',
      'NR_PRB_UTIL_DL',
    ];
    const names = kpiNames ?? defaultNames;
    const { data } = await http.get<BackendKPITimeSeriesResponse>(
      '/dashboard/kpi-time-series',
      {
        params: {
          kpi_names: names.join(','),
          ...(startTime ? { start_time: startTime } : {}),
          ...(endTime ? { end_time: endTime } : {}),
          ...(granularity ? { granularity } : {}),
          ...(technology ? { technology } : {}),
        },
      }
    );
    return mapKPITimeSeries(data);
  },

  async getKPITimeSeriesWithProgress(
    kpiNames: string[],
    startTime: string,
    endTime: string,
    granularity: DashboardKPIGranularity,
    technology?: string,
  ): Promise<DashboardKPITimeSeriesSnapshot> {
    const { data } = await http.get<BackendKPITimeSeriesSnapshot>(
      '/dashboard/kpi-time-series',
      {
        params: {
          kpi_names: kpiNames.join(','),
          start_time: startTime,
          end_time: endTime,
          granularity,
          ...(technology ? { technology } : {}),
          include_partial: true,
        },
      },
    );
    const payload = unwrapDashboardEnvelope(data);
    return {
      series: mapKPITimeSeries(payload.series),
      periodProgress: (payload.period_progress ?? []).map(mapDashboardPeriodProgress),
      progressState: payload.progress_state,
    };
  },

  /**
   * 获取 Dashboard KPI 动态定义（issue #213 Phase1）
   * GET /dashboard/kpi/definitions
   *
   * 返回首页全部 KPI 的定义（symbolic key + K 编号 + 中文名 + 单位，按制式/Panel 分组），
   * 供前端动态加载指标列表。响应为 snake_case（拦截器不 camelize），与后端 JSON 一致。
   */
  async getKPIDefinitions(): Promise<KPIDefinitionsResponse> {
    const { data } = await http.get<KPIDefinitionsResponse>(
      '/dashboard/kpi/definitions'
    );
    return data;
  },

  /**
   * 获取首页 KPI 折线图区的全局布局（issue #213 S2）
   * GET /dashboard/kpi-layout?tech=lte|nr|gsm
   *
   * 全局单套、按制式各一行，所有登录用户可读。响应为 snake_case 信封，
   * 经 mapBackendKPILayout 映射为前端 camelCase 形状（panels 原样透传）。
   */
  async getKPILayout(tech: string): Promise<KPILayout> {
    const { data } = await http.get<BackendKPILayout>('/dashboard/kpi-layout', {
      params: { tech },
    });
    return mapBackendKPILayout(data);
  },

  /**
   * 保存首页 KPI 折线图区的全局布局（issue #213 S3，仅管理员）
   * PUT /dashboard/kpi-layout
   *
   * 后端从 JSON body 读 tech（制式）与 layout（panels 信封），最后写入生效（不做版本锁）。
   * handler 再校验一次管理员身份；非管理员被拒（403）。存完对所有用户生效。
   */
  async saveKPILayout(tech: string, panels: KPILayoutPanel[]): Promise<KPILayout> {
    const { data } = await http.put<BackendKPILayout>('/dashboard/kpi-layout', {
      tech,
      layout: mapKPILayoutToBackend(panels),
    });
    return mapBackendKPILayout(data);
  },

  /**
   * 获取 KPI 趋势对比数据（今日vs昨日）
   * 用于 KPITrendChart 组件
   */
  async getKPITrendComparison(
    kpiName: string,
    compareWith: 'yesterday' | 'last_week' = 'yesterday'
  ): Promise<KPITrendComparison> {
    const { data } = await http.get<BackendKPITrendComparison>(
      '/dashboard/kpi-trend',
      { params: { kpi_name: kpiName, compare_with: compareWith } }
    );
    return mapKPITrendComparison(data);
  },

  // ==================== Phase 2 新增 API ====================

  /**
   * 获取告警处理效率指标（MTTA、MTTR、确认率、清除率）
   * GET /dashboard/alarm-efficiency
   */
  async getAlarmEfficiency(): Promise<EfficiencyMetrics> {
    const { data } = await http.get<EfficiencyMetrics>('/dashboard/alarm-efficiency');
    return data;
  },

  /**
   * 获取告警热度图数据（按星期几和小时统计）
   * GET /dashboard/alarm-heatmap?days=30
   */
  async getAlarmHeatmap(params: { days: number }): Promise<HeatmapData> {
    const { data } = await http.get<HeatmapData | ApiEnvelope<HeatmapData>>('/dashboard/alarm-heatmap', { params });
    return unwrapDashboardEnvelope(data);
  },

  /**
   * 获取按严重程度分组的告警热度图数据
   * GET /dashboard/alarm-heatmap-by-severity?days=30&severity=critical
   */
  async getAlarmHeatmapBySeverity(params: {
    days: number;
    severity?: string;
  }): Promise<AlarmHeatmapBySeverity> {
    const { data } = await http.get<AlarmHeatmapBySeverity | ApiEnvelope<AlarmHeatmapBySeverity>>(
      '/dashboard/alarm-heatmap-by-severity',
      { params }
    );
    return unwrapDashboardEnvelope(data);
  },
};
