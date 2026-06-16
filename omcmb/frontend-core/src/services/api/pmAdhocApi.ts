/**
 * T-0164-P7 / G7 adhoc 任务 REST API 客户端 + Mock。
 */

import http from '../http';
import type {
  AdhocTask,
  AdhocResultRow,
  AdhocTaskRun,
  AdhocStatus,
  AdhocDimension,
  AdhocFilterOptions,
  CreateAdhocTaskInput,
  UpdateAdhocTaskInput,
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

/**
 * 把 params 序列化成 query string，数组 → 重复键（k=a&k=b，无方括号），标量原样。
 * 用于 results 端点的 object_ldns 多值（值合法含逗号，不能 CSV-join，见 issue #401）。
 * 每个键/值都 encodeURIComponent，避免 UUID/逗号/等号被破坏。
 */
function serializeRepeatedParams(params: Record<string, unknown>): string {
  const parts: string[] = [];
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null) continue;
    const ek = encodeURIComponent(key);
    if (Array.isArray(value)) {
      for (const v of value) {
        if (v === undefined || v === null) continue;
        parts.push(`${ek}=${encodeURIComponent(String(v))}`);
      }
    } else {
      parts.push(`${ek}=${encodeURIComponent(String(value))}`);
    }
  }
  return parts.join('&');
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
  async update(id: string, input: UpdateAdhocTaskInput): Promise<{ id: string }> {
    // T-0194：编辑任务定义。metric_paths 必带；其余字段仅自建任务有意义（内置后端忽略）。
    const payload: Record<string, unknown> = {
      metric_paths: input.metricPaths,
    };
    if (input.name !== undefined) payload.name = input.name;
    if (input.deviceSns !== undefined) payload.device_sns = input.deviceSns;
    if (input.granularities !== undefined) payload.granularities = input.granularities;
    if (input.windowStart) payload.window_start = input.windowStart;
    if (input.windowEnd) payload.window_end = input.windowEnd;
    if (input.objectLdns && input.objectLdns.length > 0) {
      payload.object_ldns = input.objectLdns;
    }
    const { data } = await http.patch<{ id: string }>(`/pm/adhoc/tasks/${id}`, payload);
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
    productIds?: string[],
    objectLdns?: string[],
  ): Promise<{ rows: AdhocResultRow[]; total: number }> {
    // 大时间段（页签1 仪表盘）：startTime/endTime 为 RFC3339，透传为 start_time/end_time query 参数。
    // PM-DASH-DIMFILTER：维度子集过滤——本端点 query 是手动 snake_case 构造（不靠 Axios 自动转换），故新参数手写 snake_case。
    //
    // 两个维度参数传法不同（issue #401 修复后定型）：
    //   - product_ids：纯 UUID，值内永不含逗号 → 发 CSV（join(',')），后端 parseCSVQuery 逗号切分还原多值。
    //   - object_ldns：设备组维度值本身合法含一个逗号（'DeviceGroup=<uuid>,Tech=<tech>'），不能再用 CSV——
    //     否则单个值被逗号切成两段、后端 object_ldn = ANY(...) 匹配不上完整存储值 → 0 行（issue #401 根因）。
    //     故 object_ldns 发「重复参数」形态 ?object_ldns=a&object_ldns=b，整值保留不拆，后端 parseRepeatedQuery
    //     用 gin QueryArray 逐值取回。下方 paramsSerializer 把数组序列化成无方括号的重复键（Axios 默认会发
    //     object_ldns[]=a&object_ldns[]=b，方括号键被 QueryArray("object_ldns") 收不到 → 过滤静默失效）。
    const params: Record<string, unknown> = { limit, offset };
    if (startTime) params.start_time = startTime;
    if (endTime) params.end_time = endTime;
    if (productIds?.length) params.product_ids = productIds.join(',');
    if (objectLdns?.length) params.object_ldns = objectLdns;
    const { data } = await http.get<ResultsResponse>(`/pm/adhoc/tasks/${id}/results`, {
      params,
      // 数组按重复键序列化（object_ldns=a&object_ldns=b），标量原样拼接——不引第三方 qs 依赖。
      paramsSerializer: (p: Record<string, unknown>) => serializeRepeatedParams(p),
    });
    const rows = (data.items ?? []).map(mapBackendAdhocResult);
    // T-0194：total 是后端真实 COUNT(*)，rows.length<total 即被 limit 截断（前端据此提示）。
    return { rows, total: data.total ?? rows.length };
  },
  async filterOptions(id: string): Promise<AdhocFilterOptions> {
    // PM-DASH-DIMFILTER：列出本任务实际聚合到的子集选项（后端 SELECT DISTINCT，不被结果上限截断）。
    // 响应字段 dimension/options/value/label 均单词，camelCase 转换不影响，直接用。
    const { data } = await http.get<AdhocFilterOptions>(
      `/pm/adhoc/tasks/${id}/filter-options`,
    );
    return { dimension: data.dimension, options: data.options ?? [] };
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
  async update(id, input) {
    const t = mockTasks.find((x) => x.id === id);
    if (!t) throw new Error('not found');
    t.metricPaths = input.metricPaths;
    if (!t.isBuiltin) {
      if (input.name !== undefined) t.name = input.name;
      if (input.deviceSns !== undefined) t.deviceSns = input.deviceSns;
      if (input.granularities !== undefined) t.granularities = input.granularities;
      if (input.windowStart !== undefined) t.windowStart = input.windowStart;
      if (input.windowEnd !== undefined) t.windowEnd = input.windowEnd;
      t.objectLdns = input.objectLdns && input.objectLdns.length > 0 ? input.objectLdns : undefined;
    }
    t.updatedAt = new Date().toISOString();
    return { id };
  },
  async cancel(id) {
    const t = mockTasks.find((x) => x.id === id);
    if (t) t.status = 'canceled';
  },
  async results() {
    return { rows: [], total: 0 };
  },
  async filterOptions() {
    return { dimension: 'device' as AdhocDimension, options: [] };
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
