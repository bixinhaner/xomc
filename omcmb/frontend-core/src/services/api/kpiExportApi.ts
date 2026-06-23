/**
 * KPI-EXPORT（KPI 数据导出）REST API 客户端 + Mock。
 *
 * 设计：~/Documents/notes/PM功能设计/kpi-export-design-20260604.md §6.1
 * 后端端点（挂在 /api/v1/pm 权限组下）：
 *   POST   /pm/exports            建导出任务
 *   GET    /pm/exports            列导出任务（任务管理 Tab）
 *   GET    /pm/exports/files      列已成功的导出文件（文件管理 Tab，强制 OnlyReady）
 *   GET    /pm/exports/:id/download  同源流式下载导出文件
 *   DELETE /pm/exports/:id        删除任务记录
 *
 * 说明：后端无独立"重试"端点；失败任务的"重试"由前端用原 sourceType + params 重新 create 实现。
 */

import http from '../http';
import type {
  KpiExportTask,
  KpiExportSource,
  KpiExportStatus,
  CreateKpiExportInput,
  BackendKpiExportTask,
} from '../../types/kpiExport';
import { mapBackendKpiExportTask } from '../../types/kpiExport';
import { filenameFromContentDisposition, saveBlob } from '../../utils/saveBlob';

interface ListResponse {
  items: BackendKpiExportTask[];
}

/** 列表过滤入参（任务管理 / 文件管理共用）。 */
export interface KpiExportListFilter {
  sourceType?: KpiExportSource;
  status?: KpiExportStatus;
  limit?: number;
  offset?: number;
}

function buildListParams(filter?: KpiExportListFilter): Record<string, unknown> {
  const params: Record<string, unknown> = {};
  if (filter?.sourceType) params.source_type = filter.sourceType;
  if (filter?.status) params.status = filter.status;
  if (filter?.limit !== undefined) params.limit = filter.limit;
  if (filter?.offset !== undefined) params.offset = filter.offset;
  return params;
}

export const kpiExportApi = {
  /** 建导出任务（落表 pending + 入队异步 job）。 */
  async create(input: CreateKpiExportInput): Promise<KpiExportTask> {
    const payload: Record<string, unknown> = {
      source_type: input.sourceType,
      params: input.params ?? {},
    };
    if (input.taskName) payload.task_name = input.taskName;
    const { data } = await http.post<BackendKpiExportTask>('/pm/exports', payload);
    return mapBackendKpiExportTask(data);
  },

  /** 列导出任务（任务管理 Tab，含各状态）。 */
  async listTasks(filter?: KpiExportListFilter): Promise<KpiExportTask[]> {
    const { data } = await http.get<ListResponse>('/pm/exports', {
      params: buildListParams(filter),
    });
    return (data.items ?? []).map(mapBackendKpiExportTask);
  },

  /** 列已成功且文件就绪的导出（文件管理 Tab）。 */
  async listFiles(filter?: KpiExportListFilter): Promise<KpiExportTask[]> {
    const { data } = await http.get<ListResponse>('/pm/exports/files', {
      params: buildListParams(filter),
    });
    return (data.items ?? []).map(mapBackendKpiExportTask);
  },

  async download(id: string, fileName?: string): Promise<void> {
    const response = await http.get(
      `/pm/exports/${encodeURIComponent(id)}/download`,
      { responseType: 'blob' },
    );
    const filename = filenameFromContentDisposition(
      response.headers['content-disposition'] as string | undefined,
      fileName || `kpi_export_${id}.csv`,
    );
    saveBlob(response.data as BlobPart, filename, 'text/csv;charset=utf-8');
  },

  /** 删除任务记录。 */
  async remove(id: string): Promise<void> {
    await http.delete(`/pm/exports/${encodeURIComponent(id)}`);
  },

  /** 失败重试：用原任务的来源 + 参数重新建一个导出任务（后端无独立重试端点）。 */
  async retry(task: KpiExportTask): Promise<KpiExportTask> {
    return this.create({
      sourceType: task.sourceType,
      params: task.params,
      taskName: task.taskName,
    });
  },
};

// ── Mock ────────────────────────────────────────────────────────────────────

const mockTasks: KpiExportTask[] = [
  {
    id: 'kpi-export-mock-1',
    taskName: 'KPI导出_仪表盘_20260604_101010',
    sourceType: 'dashboard',
    params: { granularity: 'hourly' },
    format: 'csv',
    status: 'succeeded',
    rowCount: 24,
    fileSize: 4687,
    createUser: 'mock-owner',
    createdAt: '2026-06-04T10:10:10Z',
    startedAt: '2026-06-04T10:10:12Z',
    finishedAt: '2026-06-04T10:10:18Z',
  },
];

export const kpiExportMock: typeof kpiExportApi = {
  async create(input) {
    const t: KpiExportTask = {
      id: `kpi-export-mock-${Math.random().toString(36).slice(2, 8)}`,
      taskName: input.taskName ?? 'KPI导出_仪表盘_mock',
      sourceType: input.sourceType,
      params: input.params ?? {},
      format: 'csv',
      status: 'pending',
      rowCount: 0,
      fileSize: 0,
      createUser: 'mock-owner',
      createdAt: new Date().toISOString(),
    };
    mockTasks.unshift(t);
    return t;
  },
  async listTasks() {
    return [...mockTasks];
  },
  async listFiles() {
    return mockTasks.filter((t) => t.status === 'succeeded');
  },
  async download() {
    /* no-op in mock */
  },
  async remove(id) {
    const i = mockTasks.findIndex((t) => t.id === id);
    if (i >= 0) mockTasks.splice(i, 1);
  },
  async retry(task) {
    return this.create({
      sourceType: task.sourceType,
      params: task.params,
      taskName: task.taskName,
    });
  },
};
