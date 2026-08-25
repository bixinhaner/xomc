/**
 * PM 聚合指标查询 REST API 客户端。
 *
 * 旧拖拽仪表盘（dashboard/panel CRUD + 用户偏好）已随后端模块下线（ISSUE-403），
 * 本文件仅保留指标查询页命脉 queryAggregated。
 *
 * 后端路由（cmd/app/provider 注册到 permGroup("pm")）：
 *   GET /api/v1/pm/metrics/aggregated   按粒度路由聚合查询（被 usePmQuery 消费）
 */

import http from '../http';
import type {
  Granularity,
  AggregatedQueryParams,
  AggregatedQueryResult,
  AggregatedRow,
  BackendAggregatedResponse,
} from '../../types/pmDashboard';
import { mapBackendAggregatedMeta, mapBackendAggregatedRow } from '../../types/pmDashboard';
import { serializeRepeatedParams } from '../../utils/queryParams';

const aggregatedQueryTimeoutMs = 60_000;

// ── 真实 API 服务对象 ────────────────────────────────────────────────

export const pmDashboardApi = {
  // G6 Phase 4: 走 G5 aggregator → 按粒度路由聚合表（hourly+ 直查物化表，15min 退回 pm_metrics 原表）。
  async queryAggregated(
    params: AggregatedQueryParams,
  ): Promise<AggregatedQueryResult> {
    // metricPaths 后端期望 comma-separated；其它 snake_case 参数手工拼，避免被 http 拦截器误转。
    const qp: Record<string, unknown> = {
      granularity: params.granularity,
      dimension: params.dimension,
      device_oui: params.deviceOui,
      device_sn: params.deviceSn,
      device_sns: params.deviceSns && params.deviceSns.length > 0
        ? params.deviceSns.join(',')
        : undefined,
      device_group_id: params.deviceGroupId,
      metric_paths: params.metricPaths && params.metricPaths.length > 0
        ? params.metricPaths.join(',')
        : undefined,
      metric_type: params.metricType,
      technologies: params.technologies && params.technologies.length > 0
        ? params.technologies.join(',')
        : params.technology,
      start_time: params.startTime,
      end_time: params.endTime,
      limit: params.limit,
      offset: params.offset,
      page_by: params.pageBy,
      count_mode: params.countMode,
      fill_empty: params.fillEmpty ? 'true' : undefined,
      // #599：星期/小时段后端过滤（全选/空不传 = 不过滤，向后兼容）。
      weekdays: params.weekdays?.length && params.weekdays.length < 7
        ? params.weekdays.join(',')
        : undefined,
      hours: params.hours?.length && params.hours.length < 24
        ? params.hours.join(',')
        : undefined,
      // #619：测量对象后端过滤（空不传 = 不过滤，向后兼容）。
      // LDN 值自身合法含逗号（如 Cellid=x,PLMN=y），不能 CSV-join——
      // 走「重复键」形态 ?object_ldns=a&object_ldns=b（值整体 encode），后端 QueryArray 取回（与 #401 修复同模式）。
      object_ldns: params.objectLdns?.length ? params.objectLdns : undefined,
    };
    const { data } = await http.get<BackendAggregatedResponse>(
      '/pm/metrics/aggregated',
      {
        params: qp,
        timeout: aggregatedQueryTimeoutMs,
        // 数组按重复键序列化（object_ldns=a&object_ldns=b），标量原样——保留 LDN 值内逗号。
        paramsSerializer: (p: Record<string, unknown>) => serializeRepeatedParams(p),
      },
    );
    const rows = (data.items ?? []).map(mapBackendAggregatedRow);
    return {
      rows,
      total: data.total ?? rows.length,
      truncated: data.truncated ?? false,
      meta: mapBackendAggregatedMeta(data),
    };
  },
};

// ── Mock 服务（VITE_USE_MOCK=true 启用）────────────────────────────────

export const pmDashboardMock: typeof pmDashboardApi = {
  // Mock：每个 metric 拉出 deterministic 序列。粒度按入参 8 桶。
  async queryAggregated(params): Promise<AggregatedQueryResult> {
    const buckets = mockBuckets(params.granularity);
    const paths = params.metricPaths && params.metricPaths.length > 0
      ? params.metricPaths
      : ['MOCK.Counter'];
    const out: AggregatedRow[] = [];
    paths.forEach((path) => {
      buckets.forEach((iso, i) => {
        out.push({
          deviceOui: params.deviceOui ?? 'MOCK',
          deviceSn: params.deviceSn ?? 'MOCK-0001',
          deviceGroupId: undefined,
          metricPath: path,
          metricType: (params.metricType ?? 'counter') as 'counter' | 'kpi',
          metricValue: Math.round((90 + ((i * 3) % 11) + Math.sin(i) * 2) * 100) / 100,
          statisType: 'sum',
          granularity: params.granularity,
          time: iso,
          startTime: iso,
          endTime: iso,
          ingestTime: iso,
          objectLdn: null,
          extra: {},
        });
      });
    });
    return {
      rows: out,
      total: out.length,
      truncated: false,
      meta: {
        requestedStartTime: params.startTime,
        requestedEndTime: params.endTime,
        actualStartTime: out[0]?.startTime ?? null,
        actualEndTime: out[out.length - 1]?.endTime ?? null,
        granularity: params.granularity,
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      },
    };
  },
};

// ── Helpers ────────────────────────────────────────────────────────────

function mockBuckets(g: Granularity): string[] {
  const now = Date.now();
  const stepMs = g === '15min'
    ? 15 * 60_000
    : g === 'hourly'
    ? 3_600_000
    : g === 'daily'
    ? 86_400_000
    : g === 'weekly'
    ? 7 * 86_400_000
    : 30 * 86_400_000; // monthly approx
  return Array.from({ length: 8 }, (_, i) => new Date(now - (7 - i) * stepMs).toISOString());
}
