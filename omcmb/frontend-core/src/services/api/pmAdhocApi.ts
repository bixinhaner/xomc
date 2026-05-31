/**
 * T-0164-P7 / G7 adhoc 任务 REST API 客户端 + Mock。
 */

import http from '../http';
import type {
  AdhocTask,
  AdhocResultRow,
  AdhocTaskRun,
  AdhocStatus,
  CreateAdhocTaskInput,
  BackendAdhocTask,
  BackendAdhocResultRow,
  BackendAdhocTaskRun,
} from '../../types/pmAdhoc';
import {
  mapBackendAdhocTask,
  mapBackendAdhocResult,
  mapBackendAdhocTaskRun,
} from '../../types/pmAdhoc';

interface ListResponse {
  items: BackendAdhocTask[];
  total: number;
}

interface ResultsResponse {
  items: BackendAdhocResultRow[];
  total: number;
}

interface RunsResponse {
  items: BackendAdhocTaskRun[];
  total: number;
}

/** list 过滤入参（T-0186：分内置区/自建区）。 */
export interface AdhocListFilter {
  isBuiltin?: boolean;
}

export const pmAdhocApi = {
  async list(filter?: AdhocListFilter): Promise<AdhocTask[]> {
    const params: Record<string, unknown> = {};
    if (filter?.isBuiltin !== undefined) params.is_builtin = filter.isBuiltin;
    const { data } = await http.get<ListResponse>('/pm/adhoc/tasks', { params });
    return (data.items ?? []).map(mapBackendAdhocTask);
  },
  async get(id: string): Promise<AdhocTask> {
    const { data } = await http.get<BackendAdhocTask>(`/pm/adhoc/tasks/${id}`);
    return mapBackendAdhocTask(data);
  },
  async create(input: CreateAdhocTaskInput): Promise<{ id: string }> {
    // T-0185：window 仅在有值时发（oneshot）；continuous 不带 → 后端开窗滚动聚合。
    const payload: Record<string, unknown> = {
      name: input.name,
      mode: input.mode,
      cron_expr: input.cronExpr,
      device_sns: input.deviceSns,
      metric_paths: input.metricPaths,
      granularities: input.granularities,
      dimension: input.dimension,
      technology: input.technology,
      is_builtin: input.isBuiltin,
      expire_days: input.expireDays,
    };
    if (input.windowStart) payload.window_start = input.windowStart;
    if (input.windowEnd) payload.window_end = input.windowEnd;
    // T-0193：小区/PLMN 白名单——非空才透传（空=不过滤，保持现状语义）。
    if (input.objectLdns && input.objectLdns.length > 0) {
      payload.object_ldns = input.objectLdns;
    }
    const { data } = await http.post<{ id: string }>('/pm/adhoc/tasks', payload);
    return data;
  },
  async cancel(id: string): Promise<void> {
    await http.delete(`/pm/adhoc/tasks/${id}`);
  },
  async results(
    id: string,
    limit = 100,
    offset = 0,
    startTime?: string,
    endTime?: string,
  ): Promise<AdhocResultRow[]> {
    // 大时间段（页签1 仪表盘）：startTime/endTime 为 RFC3339，透传为 start_time/end_time query 参数。
    const params: Record<string, unknown> = { limit, offset };
    if (startTime) params.start_time = startTime;
    if (endTime) params.end_time = endTime;
    const { data } = await http.get<ResultsResponse>(`/pm/adhoc/tasks/${id}/results`, {
      params,
    });
    return (data.items ?? []).map(mapBackendAdhocResult);
  },
  async runs(id: string, limit = 50, offset = 0): Promise<AdhocTaskRun[]> {
    const { data } = await http.get<RunsResponse>(`/pm/adhoc/tasks/${id}/runs`, {
      params: { limit, offset },
    });
    return (data.items ?? []).map(mapBackendAdhocTaskRun);
  },
};

// Mock — 简化版（不实现完整生命周期，仅保证 UI 可调）
const mockTasks: AdhocTask[] = [
  {
    id: 'adhoc-mock-1',
    name: '小时级 RRC 成功率（模拟）',
    mode: 'oneshot',
    deviceSns: ['BLQ-001', 'BLQ-002'],
    metricPaths: ['L.RRC.SuccRate'],
    granularities: ['hourly'],
    windowStart: '2026-05-22T00:00:00Z',
    windowEnd: '2026-05-23T00:00:00Z',
    dimension: 'device',
    technology: 'lte',
    isBuiltin: false,
    expireDays: 60,
    status: 'succeeded',
    progress: 100,
    creator: 'mock-owner',
    createdAt: '2026-05-23T01:00:00Z',
    updatedAt: '2026-05-23T01:01:30Z',
  },
];

export const pmAdhocMock: typeof pmAdhocApi = {
  async list(filter?: AdhocListFilter) {
    if (filter?.isBuiltin !== undefined) {
      return mockTasks.filter((t) => t.isBuiltin === filter.isBuiltin);
    }
    return [...mockTasks];
  },
  async get(id) {
    const t = mockTasks.find((x) => x.id === id);
    if (!t) throw new Error('not found');
    return t;
  },
  async create(input) {
    const id = `adhoc-mock-${Math.random().toString(36).slice(2, 8)}`;
    mockTasks.push({
      id,
      name: input.name,
      mode: input.mode,
      cronExpr: input.cronExpr,
      deviceSns: input.deviceSns,
      metricPaths: input.metricPaths,
      granularities: input.granularities,
      windowStart: input.windowStart ?? '',
      windowEnd: input.windowEnd ?? '',
      dimension: input.dimension ?? 'device',
      technology: input.technology,
      isBuiltin: input.isBuiltin ?? false,
      expireDays: input.expireDays ?? 60,
      objectLdns: input.objectLdns && input.objectLdns.length > 0 ? input.objectLdns : undefined,
      status: 'pending',
      progress: 0,
      creator: 'mock-owner',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    });
    return { id };
  },
  async cancel(id) {
    const t = mockTasks.find((x) => x.id === id);
    if (t) t.status = 'canceled';
  },
  async results() {
    return [];
  },
  async runs(id: string) {
    // 简化 mock：返回两条运行记录（一成一败）便于 UI 调试
    const now = Date.now();
    return [
      {
        id: `${id}-run-2`,
        taskId: id,
        runSeq: 2,
        granularity: 'hourly',
        dimension: 'device',
        windowStart: new Date(now - 3600_000).toISOString(),
        windowEnd: new Date(now).toISOString(),
        status: 'succeeded' as AdhocStatus,
        queuedAt: new Date(now - 120_000).toISOString(),
        startedAt: new Date(now - 110_000).toISOString(),
        finishedAt: new Date(now - 100_000).toISOString(),
        rowsTotal: 24,
      },
      {
        id: `${id}-run-1`,
        taskId: id,
        runSeq: 1,
        granularity: 'hourly',
        dimension: 'device',
        status: 'failed' as AdhocStatus,
        queuedAt: new Date(now - 7200_000).toISOString(),
        startedAt: new Date(now - 7190_000).toISOString(),
        finishedAt: new Date(now - 7180_000).toISOString(),
        error: '聚合源数据缺失（模拟失败）',
        rowsTotal: 0,
      },
    ];
  },
};
