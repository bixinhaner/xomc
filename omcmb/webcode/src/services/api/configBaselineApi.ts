import http from '../http';
import type { BaselineConfig, ConfigTask, NeighborParam, ConfigParam } from '@/types/config';
import type { PageRequest, PageResponse } from '@/types/pagination';

// ---------------------------------------------------------------------------
// Backend response types (snake_case)
// ---------------------------------------------------------------------------

interface BackendBaseline {
  id: string;
  baseline_name: string;
  description?: string;
  device_type?: string;
  version?: string;
  params: ConfigParam[];
  creator?: string;
  status: 'draft' | 'active' | 'deprecated';
  created_at: string;
  updated_at: string;
}

interface BackendConfigTask {
  id: string;
  task_name: string;
  task_type: 'param-sync' | 'batch-config' | 'baseline-apply' | 'neighbor-update';
  device_sns: string[];
  template_id?: string;
  baseline_id?: string;
  params?: ConfigParam[];
  status: string;
  progress: number;
  success_count: number;
  fail_count: number;
  total_count: number;
  creator?: string;
  message?: string;
  created_at: string;
  updated_at: string;
}

interface BackendNeighborParam {
  id: string;
  source_cell_id: string;
  source_cell_name?: string;
  target_cell_id: string;
  target_cell_name?: string;
  neighbor_type: 'intra-freq' | 'inter-freq' | 'inter-rat';
  params: Record<string, string | number | boolean>;
  created_at: string;
  updated_at: string;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// ---------------------------------------------------------------------------
// Mapping functions: backend (snake_case) -> frontend (camelCase)
// ---------------------------------------------------------------------------

function mapBackendBaseline(b: BackendBaseline): BaselineConfig {
  return {
    id: b.id,
    baselineName: b.baseline_name,
    description: b.description ?? '',
    deviceType: b.device_type ?? '',
    version: b.version ?? '',
    params: b.params ?? [],
    creator: b.creator ?? '',
    status: b.status,
    createTime: b.created_at,
    updateTime: b.updated_at,
  };
}

function mapBackendConfigTask(t: BackendConfigTask): ConfigTask {
  return {
    id: t.id,
    taskName: t.task_name,
    taskType: t.task_type,
    deviceSns: t.device_sns ?? [],
    templateId: t.template_id,
    baselineId: t.baseline_id,
    params: t.params,
    status: t.status === 'completed' ? 'success' : (t.status as ConfigTask['status']),
    progress: t.progress,
    successCount: t.success_count,
    failCount: t.fail_count,
    totalCount: t.total_count,
    createdAt: t.created_at,
    updatedAt: t.updated_at,
    creator: t.creator ?? '',
    message: t.message,
  };
}

function mapBackendNeighborParam(n: BackendNeighborParam): NeighborParam {
  return {
    id: n.id,
    sourceCellId: n.source_cell_id,
    sourceCellName: n.source_cell_name ?? '',
    targetCellId: n.target_cell_id,
    targetCellName: n.target_cell_name ?? '',
    neighborType: n.neighbor_type,
    params: n.params ?? {},
    createTime: n.created_at,
    updateTime: n.updated_at,
  };
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

export const configBaselineApi = {
  // --- Baselines ---

  async getBaselines(
    params: { deviceType?: string; status?: string } & PageRequest
  ): Promise<PageResponse<BaselineConfig>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.deviceType) query.device_type = params.deviceType;
    if (params.status) query.status = params.status;

    const { data } = await http.get<BackendListResponse<BackendBaseline>>(
      '/config/baselines',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendBaseline),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getBaselineById(id: string): Promise<BaselineConfig | null> {
    try {
      const { data } = await http.get<BackendBaseline>(
        `/config/baselines/${id}`
      );
      return mapBackendBaseline(data);
    } catch {
      return null;
    }
  },

  async createBaseline(
    data: Omit<BaselineConfig, 'id' | 'createTime' | 'updateTime'>
  ): Promise<BaselineConfig> {
    const payload = {
      baseline_name: data.baselineName,
      description: data.description || undefined,
      device_type: data.deviceType || undefined,
      version: data.version || undefined,
      params: data.params,
      creator: data.creator || undefined,
      status: data.status || 'draft',
    };
    const { data: b } = await http.post<BackendBaseline>(
      '/config/baselines',
      payload
    );
    return mapBackendBaseline(b);
  },

  async updateBaseline(
    id: string,
    data: Partial<BaselineConfig>
  ): Promise<BaselineConfig> {
    const payload: Record<string, unknown> = {};
    if (data.baselineName !== undefined) payload.baseline_name = data.baselineName;
    if (data.description !== undefined) payload.description = data.description;
    if (data.deviceType !== undefined) payload.device_type = data.deviceType;
    if (data.version !== undefined) payload.version = data.version;
    if (data.params !== undefined) payload.params = data.params;
    if (data.creator !== undefined) payload.creator = data.creator;
    if (data.status !== undefined) payload.status = data.status;

    const { data: b } = await http.put<BackendBaseline>(
      `/config/baselines/${id}`,
      payload
    );
    return mapBackendBaseline(b);
  },

  async deleteBaselines(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.delete(`/config/baselines/${id}`);
    }
  },

  // --- Config Tasks ---

  async getTasks(
    params: PageRequest
  ): Promise<PageResponse<ConfigTask>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };

    const { data } = await http.get<BackendListResponse<BackendConfigTask>>(
      '/config/tasks',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendConfigTask),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async createTask(
    data: Omit<ConfigTask, 'id' | 'createdAt' | 'updatedAt' | 'status' | 'progress' | 'successCount' | 'failCount'>
  ): Promise<ConfigTask> {
    const payload = {
      task_name: data.taskName,
      task_type: data.taskType,
      device_sns: data.deviceSns,
      template_id: data.templateId || undefined,
      baseline_id: data.baselineId || undefined,
      params: data.params,
      creator: data.creator || undefined,
      total_count: data.totalCount,
    };
    const { data: t } = await http.post<BackendConfigTask>(
      '/config/tasks',
      payload
    );
    return mapBackendConfigTask(t);
  },

  // --- Neighbors ---

  async getNeighbors(
    params: { sourceCellId?: string } & PageRequest
  ): Promise<PageResponse<NeighborParam>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.sourceCellId) query.source_cell_id = params.sourceCellId;

    const { data } = await http.get<BackendListResponse<BackendNeighborParam>>(
      '/config/neighbors',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendNeighborParam),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },
};
