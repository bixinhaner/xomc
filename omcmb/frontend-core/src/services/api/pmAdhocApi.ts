/**
 * T-0164-P7 / G7 adhoc 任务 REST API 客户端 + Mock。
 */

import http from '../http';
import type {
  AdhocTask,
  AdhocResultRow,
  CreateAdhocTaskInput,
  BackendAdhocTask,
  BackendAdhocResultRow,
} from '../../types/pmAdhoc';
import { mapBackendAdhocTask, mapBackendAdhocResult } from '../../types/pmAdhoc';

interface ListResponse {
  items: BackendAdhocTask[];
  total: number;
}

interface ResultsResponse {
  items: BackendAdhocResultRow[];
  total: number;
}

export const pmAdhocApi = {
  async list(): Promise<AdhocTask[]> {
    const { data } = await http.get<ListResponse>('/pm/adhoc/tasks');
    return (data.items ?? []).map(mapBackendAdhocTask);
  },
  async get(id: string): Promise<AdhocTask> {
    const { data } = await http.get<BackendAdhocTask>(`/pm/adhoc/tasks/${id}`);
    return mapBackendAdhocTask(data);
  },
  async create(input: CreateAdhocTaskInput): Promise<{ id: string }> {
    const { data } = await http.post<{ id: string }>('/pm/adhoc/tasks', {
      name: input.name,
      mode: input.mode,
      cron_expr: input.cronExpr,
      device_sns: input.deviceSns,
      metric_paths: input.metricPaths,
      granularities: input.granularities,
      window_start: input.windowStart,
      window_end: input.windowEnd,
      dimension: input.dimension,
    });
    return data;
  },
  async cancel(id: string): Promise<void> {
    await http.delete(`/pm/adhoc/tasks/${id}`);
  },
  async results(id: string, limit = 100, offset = 0): Promise<AdhocResultRow[]> {
    const { data } = await http.get<ResultsResponse>(`/pm/adhoc/tasks/${id}/results`, {
      params: { limit, offset },
    });
    return (data.items ?? []).map(mapBackendAdhocResult);
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
    status: 'succeeded',
    progress: 100,
    creator: 'mock-owner',
    createdAt: '2026-05-23T01:00:00Z',
    updatedAt: '2026-05-23T01:01:30Z',
  },
];

export const pmAdhocMock: typeof pmAdhocApi = {
  async list() {
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
      windowStart: input.windowStart,
      windowEnd: input.windowEnd,
      dimension: input.dimension ?? 'device',
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
};
