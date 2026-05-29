/**
 * G6-Gap-7 / G6 Phase 4：Panel 数据加载 hook（真 API）。
 *
 * 调用 /pm/metrics/aggregated（后端走 G5 aggregator → 按粒度路由聚合表，
 * 15min 退回 pm_metrics 原表；hourly+ 直查物化表）。
 *
 * compareMode 处理（v2 待补，G6 Phase 4 简化先只渲染 primary series）：
 *   - 'previous_window' / 'same_window_other_devices' 在 v2 触发第二次 API 调用，
 *     生成 'compare' kind series。当前版本透传 panel.compareMode 但不渲染对比线。
 *
 * 限制（v1）：
 *   - 后端 handler 只支持单 OUI+SN / 单 device_group_id；panel.deviceSns 取第 0 项。
 *   - 多设备并列对比走 dimension='device_group' + deviceGroupIds。
 */

import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { createApiSwitch } from '../../services/apiSwitch';
import { pmDashboardApi, pmDashboardMock } from '../../services/api/pmDashboardApi';
import type {
  AggregatedQueryParams,
  AggregatedRow,
  CompareMode,
  Granularity,
  Panel,
  PanelTimeRange,
} from '../../types/pmDashboard';

const api = createApiSwitch(pmDashboardMock, pmDashboardApi);

export interface PanelSeriesPoint {
  // X 轴标签（时间桶或设备名）。
  label: string;
  // Y 轴值；null 表示缺采，前端按"缺采"渲染（不画 0）。
  value: number | null;
  // 入库延迟秒数；非空且超阈值时 PanelRenderer 画角标（G6-Gap-9）。
  ingestLagSeconds?: number;
}

export interface PanelSeries {
  name: string;
  // 对比模式下 kind=primary 是主 series；compare 是对比 series。
  kind: 'primary' | 'compare';
  points: PanelSeriesPoint[];
}

export interface PanelSeriesData {
  // 已展开为多 series（单 series + 对比 series；或多设备对比时多个 compare）。
  series: PanelSeries[];
  // 原始聚合行（long 格式）；导出 Excel 时按指标查询页透视格式输出。
  rows: AggregatedRow[];
  // metricPath（KPI=K 编号）→ 友好显示名（PLMN 级带标记）。渲染器据此把标题/系列名映射回友好名。
  displayNameByPath: Record<string, string>;
  // 当前 panel 是否处于对比模式（非空 = 渲染时多 series 处理）。
  compareMode: CompareMode | undefined;
  isLoading: boolean;
  isError: boolean;
  error?: unknown;
}

// 把 panel.timeRange + 兜底窗口换算成 RFC3339 startTime/endTime。
function resolveTimeWindow(
  timeRange: PanelTimeRange | undefined,
  granularity: Granularity,
): { startTime?: string; endTime?: string } {
  const now = Date.now();
  if (timeRange && 'absolute_start' in timeRange && 'absolute_end' in timeRange) {
    return { startTime: timeRange.absolute_start, endTime: timeRange.absolute_end };
  }
  let startOffsetMs = defaultWindowMs(granularity);
  let endOffsetMs = 0;
  if (timeRange && 'start_offset' in timeRange) {
    const so = parseOffset(timeRange.start_offset);
    if (so > 0) startOffsetMs = so;
    if (timeRange.end_offset) {
      const eo = parseOffset(timeRange.end_offset);
      if (eo >= 0) endOffsetMs = eo;
    }
  }
  return {
    startTime: new Date(now - startOffsetMs).toISOString(),
    endTime: new Date(now - endOffsetMs).toISOString(),
  };
}

// '-1h' / '-7d' / '-2w' / '-3M' → ms（正数）。无法解析返 0。
function parseOffset(s: string): number {
  const m = s.match(/^-?(\d+)([hdwM])$/);
  if (!m) return 0;
  const n = parseInt(m[1], 10);
  switch (m[2]) {
    case 'h':
      return n * 3_600_000;
    case 'd':
      return n * 86_400_000;
    case 'w':
      return n * 7 * 86_400_000;
    case 'M':
      return n * 30 * 86_400_000;
  }
  return 0;
}

function defaultWindowMs(g: Granularity): number {
  switch (g) {
    case '15min':
      return 2 * 3_600_000;
    case 'hourly':
      return 24 * 3_600_000;
    case 'daily':
      return 30 * 86_400_000;
    case 'weekly':
      return 12 * 7 * 86_400_000;
    case 'monthly':
      return 12 * 30 * 86_400_000;
  }
}

function formatBucketLabel(iso: string, g: Granularity): string {
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, '0');
  const MM = pad(d.getMonth() + 1);
  const DD = pad(d.getDate());
  const HH = pad(d.getHours());
  const mm = pad(d.getMinutes());
  switch (g) {
    case '15min':
      return `${HH}:${mm}`;
    case 'hourly':
      return `${MM}-${DD} ${HH}:00`;
    case 'daily':
    case 'weekly':
      return `${MM}-${DD}`;
    case 'monthly':
      return `${d.getFullYear()}-${MM}`;
  }
}

function buildQueryParams(panel: Panel, granularity: Granularity): AggregatedQueryParams | null {
  if (!panel.metricPaths || panel.metricPaths.length === 0) return null;
  const { startTime, endTime } = resolveTimeWindow(panel.timeRange, granularity);
  const params: AggregatedQueryParams = {
    granularity,
    dimension: panel.dimension,
    metricPaths: panel.metricPaths,
    startTime,
    endTime,
    limit: 5000,
  };
  if (panel.dimension === 'device_group' && panel.deviceGroupIds && panel.deviceGroupIds.length > 0) {
    params.deviceGroupId = panel.deviceGroupIds[0];
  } else if (panel.deviceSns && panel.deviceSns.length > 0) {
    params.deviceSn = panel.deviceSns[0];
  }
  return params;
}

// rows 里收集 metricPath → displayName（KPI 友好名 / PLMN 标记）；counter 行 displayName = metricPath。
function buildDisplayNameMap(rows: AggregatedRow[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const r of rows) {
    if (r.displayName && !out[r.metricPath]) out[r.metricPath] = r.displayName;
  }
  return out;
}

// AggregatedRow[] → PanelSeries[]：按 metricPath 分组，按 time 升序，去重同桶取最后一条。
// series.name 用友好名（displayName），不露 K 编号。
function rowsToSeries(
  rows: AggregatedRow[],
  panel: Panel,
  granularity: Granularity,
  nameByPath: Record<string, string>,
): PanelSeries[] {
  const byMetric = new Map<string, AggregatedRow[]>();
  for (const r of rows) {
    if (!byMetric.has(r.metricPath)) byMetric.set(r.metricPath, []);
    byMetric.get(r.metricPath)!.push(r);
  }
  return panel.metricPaths.map((metric) => {
    const list = (byMetric.get(metric) ?? [])
      .slice()
      .sort((a, b) => new Date(a.time).getTime() - new Date(b.time).getTime());
    const points: PanelSeriesPoint[] = list.map((r) => {
      const lagSec = Math.max(0, Math.floor((new Date(r.ingestTime).getTime() - new Date(r.time).getTime()) / 1000));
      return {
        label: formatBucketLabel(r.time, granularity),
        value: r.metricValue,
        ingestLagSeconds: lagSec > 0 ? lagSec : undefined,
      };
    });
    return { name: nameByPath[metric] ?? metric, kind: 'primary', points };
  });
}

export function usePmPanelData(panel: Panel, activeGranularity: Granularity): PanelSeriesData {
  const params = useMemo(() => buildQueryParams(panel, activeGranularity), [panel, activeGranularity]);

  const query = useQuery({
    queryKey: [
      'pm',
      'panel-aggregated',
      panel.id,
      activeGranularity,
      params?.dimension,
      params?.deviceSn,
      params?.deviceGroupId,
      params?.metricPaths?.join(','),
      params?.startTime,
      params?.endTime,
    ],
    queryFn: () => api.queryAggregated(params!),
    enabled: params != null,
    staleTime: 30_000,
  });

  return useMemo(() => {
    const rows = query.data ?? [];
    const displayNameByPath = buildDisplayNameMap(rows);
    const series = rowsToSeries(rows, panel, activeGranularity, displayNameByPath);
    return {
      series,
      rows,
      displayNameByPath,
      compareMode: panel.compareMode,
      isLoading: query.isLoading,
      isError: query.isError,
      error: query.error,
    };
  }, [query.data, query.isLoading, query.isError, query.error, panel, activeGranularity]);
}
