import { useQuery, useQueries, useQueryClient, useMutation } from '@tanstack/react-query';
import { dashboardService } from '../../mock/services/dashboardService';
import { dashboardApi } from '../../services/api/dashboardApi';
import { createApiSwitchWithMock } from '../../services/apiSwitch';
import type { DashboardSummary, DashboardChartData } from '../../mock/data/dashboard';
import type {
  TrendComparisonData,
  MultiTrendComparisonData,
  UseKPITrendComparisonV2Result,
  UseMultiKPITrendComparisonResult,
  KPILayoutPanel,
  DashboardKPIGranularity,
} from '../../types/dashboard';
import { useMemo, useEffect } from 'react';
import { useSystemTimezone, useSystemTimezoneValue } from './useSystemTimezone';
import { fetchDeviceList } from './useDevices';
import { nowInSystemTimezone } from '../../utils/systemTime';
import dayjs from 'dayjs';

// ============================================================================
// 类型别名
// ============================================================================

type KPITimeSeriesData = Record<string, unknown[]>;
type KPITimeSeriesParams = {
  kpi_names: string[];
  start_time: string;
  end_time: string;
  granularity?: DashboardKPIGranularity;
  technology?: string;
};

const api = createApiSwitchWithMock(dashboardService, dashboardApi);

export const DASHBOARD_REFRESH_INTERVAL_MS = 5 * 60 * 1000;
export const DASHBOARD_FRESH_TIME_MS = 4 * 60 * 1000 + 30 * 1000;

const dashboardPollingOptions = {
  retry: false,
  staleTime: DASHBOARD_FRESH_TIME_MS,
  refetchInterval: DASHBOARD_REFRESH_INTERVAL_MS,
  refetchIntervalInBackground: false,
  refetchOnWindowFocus: false,
  refetchOnReconnect: false,
} as const;

export function buildDashboardKPIQueryKey(params?: Partial<KPITimeSeriesParams>) {
  return ['dashboard', 'kpi-time-series', params] as const;
}

interface DashboardKPITimeSeriesClient {
  getKPITimeSeries(
    kpiNames?: string[],
    startTime?: string,
    endTime?: string,
    granularity?: DashboardKPIGranularity,
    technology?: string,
  ): Promise<DashboardChartData['kpiTimeSeries']>;
}

export function buildDashboardKPIQueryOptions(
  params?: Partial<KPITimeSeriesParams>,
  enabled = true,
  client: DashboardKPITimeSeriesClient = api,
) {
  return {
    queryKey: buildDashboardKPIQueryKey(params),
    queryFn: () => client.getKPITimeSeries(
      params?.kpi_names,
      params?.start_time,
      params?.end_time,
      params?.granularity,
      params?.technology,
    ),
    enabled: enabled && Boolean(params?.kpi_names?.length),
    ...dashboardPollingOptions,
  };
}

function dashboardNow(now: Date, systemTimezone?: string) {
  const value = dayjs(now);
  if (systemTimezone) {
    const zoned = value.tz(systemTimezone);
    if (zoned.isValid()) return zoned;
  }
  return value.utc();
}

export function buildDashboardDayRanges(now: Date, systemTimezone?: string) {
  const currentEnd = dashboardNow(now, systemTimezone);
  const currentStart = currentEnd.startOf('day');
  const compareStart = currentStart.subtract(1, 'day');
  return {
    current: { start_time: currentStart.format(), end_time: currentEnd.format() },
    compare: { start_time: compareStart.format(), end_time: currentStart.format() },
  };
}

export function buildDashboardWeekRange(now: Date, systemTimezone?: string) {
  const end = dashboardNow(now, systemTimezone).startOf('day');
  const start = end.subtract(7, 'day');
  return {
    start_time: start.format(),
    end_time: end.format(),
    // 查询只取七个已完成自然日；横轴额外保留今天，让用户看清数据滚动到何处。
    // 今天尚无 daily 桶时，图表会映射为 null，不补零也不画假点。
    dateKeys: Array.from({ length: 8 }, (_, index) => start.add(index, 'day').format('YYYY-MM-DD')),
  };
}

export interface DashboardKPIWindow {
  start_time: string;
  end_time: string;
  bucketKeys: string[];
}

function dashboardBucketKey(
  value: ReturnType<typeof dashboardNow>,
  granularity: DashboardKPIGranularity,
) {
  return granularity === 'hourly'
    ? value.format('YYYY-MM-DDTHH')
    : value.format('YYYY-MM-DD');
}

export function buildDashboardKPIWindow(
  now: Date,
  granularity: DashboardKPIGranularity,
  systemTimezone?: string,
): DashboardKPIWindow {
  const current = dashboardNow(now, systemTimezone);
  let end;
  let bucketEnd;
  let count;
  let unit: 'hour' | 'day' | 'week';

  if (granularity === 'hourly') {
    end = current.startOf('hour');
    bucketEnd = end;
    count = 24;
    unit = 'hour';
  } else if (granularity === 'daily') {
    bucketEnd = current.startOf('day');
    end = current;
    count = 30;
    unit = 'day';
  } else {
    const weekday = current.day() || 7;
    bucketEnd = current.subtract(weekday - 1, 'day').startOf('day');
    end = current;
    count = 12;
    unit = 'week';
  }
  const start = granularity === 'hourly'
    ? bucketEnd.subtract(count, unit)
    : bucketEnd.subtract(count - 1, unit);

  return {
    start_time: start.format(),
    end_time: end.format(),
    bucketKeys: Array.from(
      { length: count },
      (_, index) => dashboardBucketKey(start.add(index, unit), granularity),
    ),
  };
}

export function isDashboardBusinessTimezoneReady(systemTimezone?: string): boolean {
  return Boolean(systemTimezone?.trim());
}

export interface DashboardDataResponse {
  summary: DashboardSummary;
  chartData: DashboardChartData;
  widgets: Record<string, unknown>;
}

export function useDashboardData() {
  return useQuery<DashboardDataResponse>({
    queryKey: ['dashboard', 'all'],
    queryFn: () => api.getDashboardData() as unknown as Promise<DashboardDataResponse>,
    ...dashboardPollingOptions,
  });
}

export function useDashboardSummary(apiScope: string, userScope: string | undefined) {
  return useQuery({
    queryKey: ['dashboard', 'summary', apiScope, userScope],
    queryFn: () => api.getSummary(),
    enabled: Boolean(userScope),
    ...dashboardPollingOptions,
  });
}

/**
 * Dashboard 设备卡片复用设备列表的全量 stats，避免被重型 Summary 的告警、
 * KPI 和历史趋势查询阻塞。无筛选参数时与设备列表首页统计口径及数据权限一致。
 */
export function useDashboardDeviceStats(apiScope: string, userScope: string | undefined) {
  return useQuery({
    queryKey: ['dashboard', 'device-stats', apiScope, userScope],
    queryFn: async () => {
      const { stats } = await fetchDeviceList({ page: 1, pageSize: 1 });
      if (stats.online_count + stats.offline_count !== stats.total) {
        throw new Error('device list returned incomplete page-level stats');
      }
      if (!Number.isFinite(stats.current_ue_count)) {
        throw new Error('device list returned no current UE stats');
      }
      return stats;
    },
    enabled: Boolean(userScope),
    ...dashboardPollingOptions,
  });
}

export function useDashboardChartData() {
  return useQuery({
    queryKey: ['dashboard', 'charts'],
    queryFn: () => api.getChartData(),
  });
}

export function useAlarmTrend(
  days = 7,
  metric: 'raised' | 'active' = 'raised',
  enabled = true,
) {
  return useQuery({
    queryKey: ['dashboard', 'alarm-trend', days, metric],
    queryFn: () => api.getAlarmTrend(days, metric),
    enabled,
  });
}

export function useDeviceStatusPie() {
  return useQuery({
    queryKey: ['dashboard', 'device-status'],
    queryFn: () => api.getDeviceStatusPie(),
  });
}

export function useDeviceStatusByType() {
  return useQuery({
    queryKey: ['dashboard', 'device-status-by-type'],
    queryFn: () => api.getDeviceStatusByType(),
    ...dashboardPollingOptions,
  });
}

export function useTopAlarmDevices(enabled = true) {
  return useQuery({
    queryKey: ['dashboard', 'top-alarm-devices'],
    queryFn: () => api.getTopAlarmDevices(),
    enabled,
  });
}

export function useKPITrend(kpiCode: string) {
  return useQuery({
    queryKey: ['dashboard', 'kpi-trend', kpiCode],
    queryFn: () => api.getKPITrend(kpiCode),
    enabled: Boolean(kpiCode),
  });
}

export function useRegionStats() {
  return useQuery({
    queryKey: ['dashboard', 'region-stats'],
    queryFn: () => api.getRegionStats(),
  });
}

/**
 * 获取 Dashboard KPI 动态定义（issue #213 Phase1）
 *
 * 返回首页全部 KPI 的定义（symbolic key + K 编号 + 中文名 + 单位，按制式/Panel 分组），
 * 供前端动态加载指标列表。定义相对静态，缓存较久、不轮询。
 */
export function useKPIDefinitions(enabled = true) {
  return useQuery({
    queryKey: ['dashboard', 'kpi-definitions'],
    queryFn: () => api.getKPIDefinitions(),
    enabled,
    staleTime: 5 * 60 * 1000, // 定义不常变，缓存 5 分钟
  });
}

/**
 * 获取首页 KPI 折线图区的全局布局（issue #213 S2）
 *
 * 按当前制式读全局布局（管理员配一次、所有人看同一份）。布局相对静态，
 * 缓存较久、不轮询。读不到/为空/出错时由首页回退内置默认（保证永不空白）。
 *
 * @param tech 制式：lte / nr / gsm
 * @param enabled 是否启用查询
 */
export function useKPILayout(tech: string, enabled = true) {
  return useQuery({
    queryKey: ['dashboard', 'kpi-layout', tech],
    queryFn: () => api.getKPILayout(tech),
    enabled: enabled && Boolean(tech),
    staleTime: 5 * 60 * 1000, // 布局不常变，缓存 5 分钟
  });
}

/**
 * 保存首页 KPI 折线图区的全局布局（issue #213 S3，仅管理员）
 *
 * 把当前制式整套布局写回后端（最后写入生效），成功后失效该制式的读布局缓存，
 * 使首页下次开页读到新布局（存完对所有用户生效）。后端再校验管理员身份，
 * 非管理员被拒（403）由调用方 toast 提示。
 */
export function useSaveKPILayout() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ tech, panels }: { tech: string; panels: KPILayoutPanel[] }) =>
      api.saveKPILayout(tech, panels),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({
        queryKey: ['dashboard', 'kpi-layout', variables.tech],
      });
    },
  });
}

/**
 * 获取 KPI 时序数据
 * @param params 包含 kpi_names, start_time, end_time
 * @param enabled 是否启用查询
 */
export function useKPITimeSeries(
  params?: {
    kpi_names?: string[];
    start_time?: string;
    end_time?: string;
    granularity?: DashboardKPIGranularity;
    technology?: string;
  },
  enabled = true
) {
  return useQuery(buildDashboardKPIQueryOptions(params, enabled));
}

export function useDashboardKPIWindowSeries(
  kpiNames: string[],
  granularity: DashboardKPIGranularity,
  enabled = true,
  technology?: string,
) {
  const { systemTimezone } = useSystemTimezone();
  const query = useQuery({
    queryKey: ['dashboard', 'kpi-window-series', kpiNames, granularity, technology, systemTimezone],
    queryFn: async () => {
      const window = buildDashboardKPIWindow(new Date(), granularity, systemTimezone);
      if (granularity === 'hourly') {
        const raw = await api.getKPITimeSeries(
          kpiNames,
          window.start_time,
          window.end_time,
          granularity,
          technology,
        );
        return {
          raw, window, periodProgress: [],
          progressState: 'not_applicable' as const,
        };
      }
      const snapshot = await api.getKPITimeSeriesWithProgress(
        kpiNames,
        window.start_time,
        window.end_time,
        granularity,
        technology,
      );
      return {
        raw: snapshot.series,
        window,
        periodProgress: snapshot.periodProgress,
        progressState: snapshot.progressState,
      };
    },
    enabled: enabled
      && kpiNames.length > 0
      && isDashboardBusinessTimezoneReady(systemTimezone),
    ...dashboardPollingOptions,
  });

  const data = useMemo(() => {
    if (!query.data?.raw) return undefined;
    return kpiNames.reduce<MultiTrendComparisonData>((acc, name) => {
      acc[name] = calculateTrendComparison(
        query.data.raw,
        {},
        name,
        'last_week',
      );
      return acc;
    }, {});
  }, [kpiNames, query.data]);

  return {
    data,
    window: query.data?.window,
    periodProgress: query.data?.periodProgress,
    progressState: query.data?.progressState,
    isLoading: query.isLoading,
    error: query.error,
  };
}

/**
 * 获取 KPI 趋势对比数据（今日vs昨日）
 * 用于 KPITrendChart 组件
 * 注意：此接口依赖于后端修复粒度问题，可能无法返回数据
 * 推荐使用 useKPITrendComparisonV2 替代
 */
export function useKPITrendComparison(
  kpiName: string,
  compareWith: 'yesterday' | 'last_week' = 'yesterday',
  enabled = true
) {
  return useQuery({
    queryKey: ['dashboard', 'kpi-trend-comparison', kpiName, compareWith],
    queryFn: () => api.getKPITrendComparison(kpiName, compareWith),
    enabled: enabled && Boolean(kpiName),
    staleTime: 30000,
  });
}

// ============================================================================
// 工具函数（提取自 useKPITrendComparisonV2，便于测试和复用）
// ============================================================================

/**
 * 计算 KPI 趋势对比的时间范围参数
 *
 * 注意：此函数为内部工具函数，仅供 useKPITrendComparisonV2 和 useKPIGroupTrend 使用。
 * 外部请使用相应的 Hook 而非直接调用此函数。
 *
 * @param kpiName - KPI 指标名称
 * @param compareWith - 对比类型：yesterday（昨日对比）或 last_week（上周对比）
 * @returns 当前时段和对比时段的查询参数
 *
 * @internal
 */
function calculateTrendTimeRanges(
  kpiName: string,
  compareWith: 'yesterday' | 'last_week'
): { currentParams: KPITimeSeriesParams; compareParams: KPITimeSeriesParams } {
  const now = new Date();
  let currentStart: Date;
  let compareStart: Date;
  let compareEnd: Date;

  if (compareWith === 'yesterday') {
    // 当前：今天 00:00 到现在
    currentStart = new Date(now);
    currentStart.setHours(0, 0, 0, 0);
    compareStart = new Date(currentStart);
    compareStart.setDate(compareStart.getDate() - 1); // 昨天 00:00
    compareEnd = new Date(compareStart);
    compareEnd.setHours(23, 59, 59, 999); // 昨天 23:59:59
  } else {
    // last_week: 本周一到现在 vs 上周一到上周日
    const weekday = now.getDay() || 7; // 周日 = 7
    const currentMonday = new Date(now);
    currentMonday.setDate(now.getDate() - weekday + 1);
    currentMonday.setHours(0, 0, 0, 0);
    currentStart = currentMonday;

    compareStart = new Date(currentMonday);
    compareStart.setDate(compareStart.getDate() - 7);
    compareEnd = new Date(compareStart);
    compareEnd.setDate(compareEnd.getDate() + 6);
    compareEnd.setHours(23, 59, 59, 999);
  }

  return {
    currentParams: {
      kpi_names: [kpiName],
      start_time: currentStart.toISOString(),
      end_time: now.toISOString(),
    },
    compareParams: {
      kpi_names: [kpiName],
      start_time: compareStart.toISOString(),
      end_time: compareEnd.toISOString(),
    },
  };
}

/**
 * 数据归一化：兼容 Mock（元组）和真实 API（对象）两种格式
 *
 * @param point - KPI 数据点
 * @returns 标准化的数据点 { time, value }
 */
function normalizeKPIDataPoint(point: unknown): { time: string; value: number } {
  if (Array.isArray(point)) {
    return { time: point[0] as string, value: point[1] as number };
  }
  return point as { time: string; value: number };
}

/**
 * 计算 KPI 趋势对比数据
 *
 * @param currentData - 当前时段的时序数据
 * @param compareData - 对比时段的时序数据
 * @param kpiName - KPI 指标名称
 * @param compareWith - 对比类型
 * @returns 趋势对比结果
 */
function calculateTrendComparison(
  currentData: KPITimeSeriesData,
  compareData: KPITimeSeriesData,
  kpiName: string,
  compareWith: 'yesterday' | 'last_week'
): TrendComparisonData {
  const current = currentData[kpiName] || [];
  const compare = compareData[kpiName] || [];

  // 计算变化百分比
  let changePercent: number | undefined;
  if (current.length > 0 && compare.length > 0) {
    const currentAvg = current.reduce(
      (sum: number, point: unknown) => sum + normalizeKPIDataPoint(point).value, 0
    ) / current.length;
    const compareAvg = compare.reduce(
      (sum: number, point: unknown) => sum + normalizeKPIDataPoint(point).value, 0
    ) / compare.length;

    if (compareAvg !== 0) {
      changePercent = ((currentAvg - compareAvg) / compareAvg) * 100;
    }
  }

  return {
    current: current.map(normalizeKPIDataPoint),
    compare: compare.map(normalizeKPIDataPoint),
    metadata: {
      kpi_name: kpiName,
      compare_type: compareWith,
      change_percent: changePercent,
    },
  };
}

// ============================================================================
// KPI 趋势对比 Hooks（v3.5 优化版本）
// ============================================================================

/**
 * 获取 KPI 趋势对比数据（今日vs昨日）V2版本
 *
 * 复用 getKPITimeSeries 接口，避免使用有粒度问题的 kpi-trend 接口
 *
 * @param kpiName - KPI 指标名称
 * @param compareWith - 对比类型：yesterday（昨日对比）或 last_week（上周对比）
 * @param enabled - 是否启用查询
 * @returns 包含 data, isLoading, errors 的结果对象
 *
 * @example
 * ```ts
 * const { data, isLoading, errors } = useKPITrendComparisonV2('dl_throughput', 'yesterday');
 * if (data) {
 *   console.log('当前值:', data.current);
 *   console.log('对比值:', data.compare);
 *   console.log('变化百分比:', data.metadata.change_percent);
 * }
 * ```
 *
 * @version 3.5.0 - 使用 useQueries 统一管理并行查询，提升错误处理能力
 */
export function useKPITrendComparisonV2(
  kpiName: string,
  compareWith: 'yesterday' | 'last_week' = 'yesterday',
  enabled = true
): UseKPITrendComparisonV2Result {
  // 获取 queryClient 用于预取
  const queryClient = useQueryClient();

  // 1. 计算时间范围参数（使用 useMemo 避免重复计算）
  const { currentParams, compareParams } = useMemo(
    () => calculateTrendTimeRanges(kpiName, compareWith),
    [kpiName, compareWith]
  );

  // 2. 使用 useQueries 统一管理并行查询
  const queries = useQueries({
    queries: [
      {
        queryKey: ['dashboard', 'kpi-time-series', currentParams],
        queryFn: () =>
          api.getKPITimeSeries(
            currentParams.kpi_names,
            currentParams.start_time,
            currentParams.end_time
          ),
        enabled,
        staleTime: 30000,
      },
      {
        queryKey: ['dashboard', 'kpi-time-series', compareParams],
        queryFn: () =>
          api.getKPITimeSeries(
            compareParams.kpi_names,
            compareParams.start_time,
            compareParams.end_time
          ),
        enabled,
        staleTime: 30000,
      },
    ],
  });

  // 3. 统一加载状态和错误处理
  const isLoading = queries.some((q) => q.isLoading);
  const errors = queries.filter((q) => q.error).map((q) => q.error);

  // 4. 计算对比数据
  const comparison = useMemo(() => {
    if (!queries[0].data || !queries[1].data) {
      return undefined;
    }
    return calculateTrendComparison(
      queries[0].data,
      queries[1].data,
      kpiName,
      compareWith
    );
  }, [queries[0].data, queries[1].data, kpiName, compareWith]);

  // 5. 智能预取策略（方案三）
  // 当用户查看昨日对比时，预取上周对比数据，提升切换体验
  useEffect(() => {
    if (compareWith === 'yesterday' && comparison && enabled) {
      // 计算上周对比的参数
      const prefetchParams = calculateTrendTimeRanges(kpiName, 'last_week');

      // 预取上周当前时段数据
      queryClient.prefetchQuery({
        queryKey: ['dashboard', 'kpi-time-series', prefetchParams.currentParams],
        queryFn: () =>
          api.getKPITimeSeries(
            prefetchParams.currentParams.kpi_names,
            prefetchParams.currentParams.start_time,
            prefetchParams.currentParams.end_time
          ),
        staleTime: 30000,
      });

      // 预取上周对比时段数据
      queryClient.prefetchQuery({
        queryKey: ['dashboard', 'kpi-time-series', prefetchParams.compareParams],
        queryFn: () =>
          api.getKPITimeSeries(
            prefetchParams.compareParams.kpi_names,
            prefetchParams.compareParams.start_time,
            prefetchParams.compareParams.end_time
          ),
        staleTime: 30000,
      });
    }
  }, [comparison, compareWith, kpiName, enabled, queryClient]);

  return {
    data: comparison,
    isLoading,
    errors,
  };
}

/**
 * 获取多个 KPI 的趋势对比数据
 *
 * 适用于需要同时展示多个指标的场景（如上下行速率双线对比）。
 * 相比多次调用 useKPITrendComparisonV2，此 Hook 只需 2 次网络请求（而非 2N 次）。
 *
 * @param kpiNames - KPI 指标名称数组
 * @param compareWith - 对比类型：yesterday（昨日对比）或 last_week（上周对比）
 * @param enabled - 是否启用查询
 * @returns 包含 data（字典结构）, isLoading, errors 的结果对象
 *
 * @example
 * ```ts
 * const { data, isLoading, errors } = useMultiKPITrendComparison(
 *   ['dl_throughput', 'ul_throughput'],
 *   'yesterday'
 * );
 * if (data) {
 *   console.log('下行速率趋势:', data.dl_throughput);
 *   console.log('上行速率趋势:', data.ul_throughput);
 * }
 * ```
 *
 * @version 3.5.0 - 新增
 */
export function useMultiKPITrendComparison(
  kpiNames: string[],
  compareWith: 'yesterday' | 'last_week' = 'yesterday',
  enabled = true,
  windowDateKey?: string,
  technology?: string,
): UseMultiKPITrendComparisonResult {
  const systemTimezone = useSystemTimezoneValue();
  // 1. 计算时间范围参数（支持多个 KPI）
  const { currentParams, compareParams } = useMemo(() => {
    if (compareWith === 'yesterday') {
      const ranges = buildDashboardDayRanges(new Date(), systemTimezone);
      return {
        currentParams: { kpi_names: kpiNames, technology, ...ranges.current },
        compareParams: { kpi_names: kpiNames, technology, ...ranges.compare },
      };
    }
    const now = nowInSystemTimezone(systemTimezone);
    const weekday = now.day() || 7;
    const currentMonday = now.subtract(weekday - 1, 'day').startOf('day');
    const compareStart = currentMonday.subtract(7, 'day');

    return {
      currentParams: {
        kpi_names: kpiNames,
        start_time: currentMonday.format(),
        end_time: now.format(),
        technology,
      },
      compareParams: {
        kpi_names: kpiNames,
        start_time: compareStart.format(),
        end_time: currentMonday.format(),
        technology,
      },
    };
  }, [kpiNames, compareWith, systemTimezone, windowDateKey, technology]);

  // 2. 两次请求获取所有 KPI 的时序数据
  const currentTimeSeries = useKPITimeSeries(currentParams, enabled);
  const compareTimeSeries = useKPITimeSeries(compareParams, enabled);

  // 3. 计算每个 KPI 的对比数据
  const comparisons = useMemo(() => {
    if (!currentTimeSeries.data || !compareTimeSeries.data) {
      return undefined;
    }

    return kpiNames.reduce<MultiTrendComparisonData>((acc, name) => {
      acc[name] = calculateTrendComparison(
        currentTimeSeries.data,
        compareTimeSeries.data,
        name,
        compareWith
      );
      return acc;
    }, {});
  }, [currentTimeSeries.data, compareTimeSeries.data, kpiNames, compareWith]);

  return {
    data: comparisons,
    isLoading: currentTimeSeries.isLoading || compareTimeSeries.isLoading,
    errors: [
      ...(currentTimeSeries.error ? [currentTimeSeries.error] : []),
      ...(compareTimeSeries.error ? [compareTimeSeries.error] : []),
    ],
  };
}

export function useMultiKPIWeekSeries(
  kpiNames: string[],
  enabled = true,
  windowDateKey?: string,
  technology?: string,
) {
  // 周窗口依赖业务自然日，必须直接订阅已鉴权的时区查询结果，不能只读可能尚未
  // 初始化的 store 缓存，否则首次请求会回落 UTC，daily 点整体左移一天。
  const { systemTimezone } = useSystemTimezone();
  const window = useMemo(() => {
    const range = buildDashboardWeekRange(new Date(), systemTimezone);
    return { startTime: range.start_time, endTime: range.end_time, dateKeys: range.dateKeys };
  }, [systemTimezone, windowDateKey]);

  const query = useKPITimeSeries({
    kpi_names: kpiNames,
    start_time: window.startTime,
    end_time: window.endTime,
    granularity: 'daily',
    technology,
  }, enabled && isDashboardBusinessTimezoneReady(systemTimezone));

  const data = useMemo(() => {
    if (!query.data) return undefined;
    return kpiNames.reduce<MultiTrendComparisonData>((acc, name) => {
      acc[name] = calculateTrendComparison(query.data, {}, name, 'last_week');
      return acc;
    }, {});
  }, [kpiNames, query.data]);

  return { data, isLoading: query.isLoading, error: query.error, dateKeys: window.dateKeys };
}

// ============================================================================
// 仪表板 KPI 概览（优化版本：一次获取所有KPI趋势数据）
// ============================================================================

/**
 * 仪表板 KPI 概览配置
 */
interface DashboardKPIOverviewConfig {
  /** 速率图表的时间对比模式 */
  throughputCompare?: 'yesterday' | 'last_week';
  /** 质量指标图表的时间对比模式 */
  qualityCompare?: 'yesterday' | 'last_week';
  /** PRB 利用率图表的时间对比模式 */
  prbCompare?: 'yesterday' | 'last_week';
  /** 是否启用查询 */
  enabled?: boolean;
}

/**
 * 仪表板 KPI 概览 - 支持多个图表独立时间对比模式
 *
 * 此 Hook 专为仪表板页面优化，支持三个图表各自独立的时间对比模式。
 *
 * @param config - 配置对象，包含各图表的时间对比模式和启用状态
 * @returns 包含所有 6 个 KPI 趋势数据的结果对象
 *
 * @example
 * ```ts
 * const {
 *   throughputKPIs,    // { dlThroughput, ulThroughput }
 *   qualityKPIs,       // { rrcSuccRate, erabSuccRate, hoSuccRate }
 *   prbKPIs,           // { prbUtil }
 *   isLoading
 * } = useDashboardKPIOverview({
 *   throughputCompare: 'yesterday',
 *   qualityCompare: 'last_week',
 *   prbCompare: 'yesterday',
 * });
 * ```
 *
 * @version 3.6.1 - 支持多图表独立时间对比，API 请求数 = 时间对比模式数 × 2
 */
export function useDashboardKPIOverview(config: DashboardKPIOverviewConfig = {}) {
  const {
    throughputCompare = 'yesterday',
    qualityCompare = 'yesterday',
    prbCompare = 'yesterday',
    enabled = true,
  } = config;

  // 按图表分组 KPI 名称
  const THROUGHPUT_KPIS = ['NR_PDCP_RATE_DL', 'NR_PDCP_RATE_UL'] as const;
  const QUALITY_KPIS = ['RRC_CONN_SETUP_SR', 'ERAB_SETUP_SR', 'NR_SA_HO_SR'] as const;
  const PRB_KPIS = ['NR_PRB_UTIL_DL'] as const;

  // 收集所有唯一的时间对比模式（避免重复请求）
  const uniqueCompares = useMemo(() => {
    return Array.from(new Set([throughputCompare, qualityCompare, prbCompare]));
  }, [throughputCompare, qualityCompare, prbCompare]);

  // 为每个唯一的时间对比模式创建查询参数
  const compareParamsMap = useMemo(() => {
    const now = new Date();
    const map = new Map<'yesterday' | 'last_week', { currentParams: KPITimeSeriesParams; compareParams: KPITimeSeriesParams }>();

    uniqueCompares.forEach((compareWith) => {
      let currentStart: Date;
      let compareStart: Date;
      let compareEnd: Date;

      if (compareWith === 'yesterday') {
        currentStart = new Date(now);
        currentStart.setHours(0, 0, 0, 0);
        compareStart = new Date(currentStart);
        compareStart.setDate(compareStart.getDate() - 1);
        compareEnd = new Date(compareStart);
        compareEnd.setHours(23, 59, 59, 999);
      } else {
        const weekday = now.getDay() || 7;
        const currentMonday = new Date(now);
        currentMonday.setDate(now.getDate() - weekday + 1);
        currentMonday.setHours(0, 0, 0, 0);
        currentStart = currentMonday;

        compareStart = new Date(currentMonday);
        compareStart.setDate(compareStart.getDate() - 7);
        compareEnd = new Date(compareStart);
        compareEnd.setDate(compareEnd.getDate() + 6);
        compareEnd.setHours(23, 59, 59, 999);
      }

      map.set(compareWith, {
        currentParams: {
          kpi_names: [...THROUGHPUT_KPIS, ...QUALITY_KPIS, ...PRB_KPIS],
          start_time: currentStart.toISOString(),
          end_time: now.toISOString(),
        },
        compareParams: {
          kpi_names: [...THROUGHPUT_KPIS, ...QUALITY_KPIS, ...PRB_KPIS],
          start_time: compareStart.toISOString(),
          end_time: compareEnd.toISOString(),
        },
      });
    });

    return map;
  }, [uniqueCompares]);

  // 为每个唯一的时间对比模式创建查询（每个模式2个请求）
  const queries = useQueries({
    queries: Array.from(compareParamsMap.entries()).flatMap(([compareWith, params]) => [
      {
        queryKey: ['dashboard', 'kpi-time-series', 'current', compareWith],
        queryFn: () =>
          api.getKPITimeSeries(
            params.currentParams.kpi_names,
            params.currentParams.start_time,
            params.currentParams.end_time
          ),
        enabled,
        staleTime: 30000,
      },
      {
        queryKey: ['dashboard', 'kpi-time-series', 'compare', compareWith],
        queryFn: () =>
          api.getKPITimeSeries(
            params.compareParams.kpi_names,
            params.compareParams.start_time,
            params.compareParams.end_time
          ),
        enabled,
        staleTime: 30000,
      },
    ]),
  });

  // 辅助函数：根据时间对比模式获取对应的数据
  const getKPIsByCompare = (
    compareWith: 'yesterday' | 'last_week',
    kpiNames: readonly string[]
  ): Record<string, TrendComparisonData> | undefined => {
    const idx = uniqueCompares.indexOf(compareWith);
    if (idx === -1) return undefined;
    const currentData = queries[idx * 2]?.data;
    const compareData = queries[idx * 2 + 1]?.data;

    if (!currentData || !compareData) return undefined;

    const result: Record<string, TrendComparisonData> = {};
    kpiNames.forEach((kpiName) => {
      result[kpiName] = calculateTrendComparison(currentData, compareData, kpiName, compareWith);
    });
    return result;
  };

  // 为每个图表分组计算对比数据
  const throughputKPIs = useMemo(
    () => getKPIsByCompare(throughputCompare, THROUGHPUT_KPIS),
    [queries, throughputCompare, uniqueCompares]
  );

  const qualityKPIs = useMemo(
    () => getKPIsByCompare(qualityCompare, QUALITY_KPIS),
    [queries, qualityCompare, uniqueCompares]
  );

  const prbKPIs = useMemo(
    () => getKPIsByCompare(prbCompare, PRB_KPIS),
    [queries, prbCompare, uniqueCompares]
  );

  // 统一加载状态
  const isLoading = queries.some((q) => q.isLoading);
  const errors = queries.filter((q) => q.error).map((q) => q.error);

  return {
    throughputKPIs,
    qualityKPIs,
    prbKPIs,
    isLoading,
    errors,
  };
}

// ============================================================================
// 简化版：单个图表组 KPI 趋势（每个图表独立调用）
// ============================================================================

/**
 * KPI 分组趋势数据 - 简单的按名称索引的记录类型
 */
type KPITrendDataMap = Record<string, TrendComparisonData | undefined>;

/**
 * 获取一组 KPI 的趋势对比数据
 *
 * 简单设计：每个图表独立调用，互不影响。
 *
 * @param kpiNames - KPI 名称数组
 * @param compareWith - 对比类型
 * @param enabled - 是否启用
 * @returns [数据映射表, 加载状态]
 *
 * @example
 * ```ts
 * // 速率图表独立使用
 * const [throughputData, isLoading] = useKPIGroupTrend(
 *   ['NR_PDCP_RATE_DL', 'NR_PDCP_RATE_UL'],
 *   throughputTimeRange
 * );
 * ```
 */
export function useKPIGroupTrend(
  kpiNames: string[],
  compareWith: 'yesterday' | 'last_week' = 'yesterday',
  enabled = true
): [KPITrendDataMap, boolean] {
  // 计算时间范围
  const { currentParams, compareParams } = useMemo(
    () => calculateTrendTimeRanges(kpiNames[0] || '', compareWith),
    [kpiNames.join(','), compareWith]
  );

  // 并行请求当前和对比时段数据
  const queries = useQueries({
    queries: [
      {
        queryKey: ['dashboard', 'kpi-time-series', 'current', compareWith, kpiNames],
        queryFn: () =>
          api.getKPITimeSeries(
            kpiNames,
            currentParams.start_time,
            currentParams.end_time
          ),
        enabled,
        staleTime: 30000,
      },
      {
        queryKey: ['dashboard', 'kpi-time-series', 'compare', compareWith, kpiNames],
        queryFn: () =>
          api.getKPITimeSeries(
            kpiNames,
            compareParams.start_time,
            compareParams.end_time
          ),
        enabled,
        staleTime: 30000,
      },
    ],
  });

  // 计算每个 KPI 的对比数据
  const result = useMemo(() => {
    const currentData = queries[0].data;
    const compareData = queries[1].data;

    // 当查询被禁用或数据未就绪时返回空对象
    if (!currentData || !compareData) {
      return {};
    }

    const map: KPITrendDataMap = {};
    kpiNames.forEach((kpiName) => {
      map[kpiName] = calculateTrendComparison(
        currentData,
        compareData,
        kpiName,
        compareWith
      );
    });
    return map;
  }, [queries, kpiNames, compareWith]);

  const isLoading = queries.some(q => q.isLoading);

  return [result, isLoading];
}
