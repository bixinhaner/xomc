import http from '../http';
import type { PageRequest, PageResponse } from '../../types/pagination';

// --- Backend response types ---

interface BackendProvisioningTask {
  id: string;
  device_id: string;
  template_id: string | null;
  status: string;
  current_step: number;
  total_steps: number;
  error_message: string;
  retry_count: number;
  max_retries: number;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
}

// --- Frontend types ---

export interface ProvisioningTask {
  id: string;
  deviceId: string;
  templateId: string | null;
  status: string;
  currentStep: number;
  totalSteps: number;
  errorMessage: string;
  retryCount: number;
  maxRetries: number;
  startedAt: string | null;
  completedAt: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface CreateProvisioningTaskRequest {
  deviceId: string;
}

/**
 * UI 状态码（即插即用「执行状态」列表使用的简化状态机）：
 * 0-成功 1-失败 2-执行中 3-未执行 4-跳过。
 * 后端 ProvisioningState（discovered/identifying/matching/configuring/verifying/
 * discovering/syncing/completed/failed）映射到此简化态。
 */
export type ProvisioningTaskStatusCode = '0' | '1' | '2' | '3' | '4';

/**
 * 即插即用「执行状态」表的视图模型。后端 ProvisioningTask 字段较少
 * （无 productClass/policyName/executeType/licenseFile 等业务列），缺失字段在
 * mapTaskToExecuteView 中以空串占位，避免页面渲染崩溃。
 */
export interface ProvisioningExecuteView {
  taskId: string;
  deviceId: string;
  status: ProvisioningTaskStatusCode;
  startTime: string;
  endTime: string;
  /** 「Step N/M」形式的进度，由 currentStep/totalSteps 派生。 */
  executeProcedure: string;
  failureReason: string;
  retryCount: number;
  maxRetries: number;
}

/** 后端生命周期状态 → UI 简化状态码。 */
export function provisioningStatusCode(status: string): ProvisioningTaskStatusCode {
  switch (status) {
    case 'completed':
      return '0';
    case 'failed':
      return '1';
    default:
      // discovered / identifying / matching / configuring / verifying /
      // discovering / syncing —— 均视为「执行中」。
      return '2';
  }
}

/** 后端 ProvisioningTask → 即插即用执行状态视图模型。 */
export function mapTaskToExecuteView(t: ProvisioningTask): ProvisioningExecuteView {
  const procedure =
    t.totalSteps > 0 ? `${t.currentStep}/${t.totalSteps}` : '';
  return {
    taskId: t.id,
    deviceId: t.deviceId,
    status: provisioningStatusCode(t.status),
    startTime: t.startedAt ?? '',
    endTime: t.completedAt ?? '',
    executeProcedure: procedure,
    failureReason: t.errorMessage,
    retryCount: t.retryCount,
    maxRetries: t.maxRetries,
  };
}

// --- Mapping functions ---

function mapBackendTask(t: BackendProvisioningTask): ProvisioningTask {
  return {
    id: t.id,
    deviceId: t.device_id,
    templateId: t.template_id,
    status: t.status,
    currentStep: t.current_step,
    totalSteps: t.total_steps,
    errorMessage: t.error_message || '',
    retryCount: t.retry_count,
    maxRetries: t.max_retries,
    startedAt: t.started_at,
    completedAt: t.completed_at,
    createdAt: t.created_at,
    updatedAt: t.updated_at,
  };
}

// --- Exported service ---

export const provisionApi = {
  async getTasks(
    params: {
      status?: string;
      deviceId?: string;
    } & PageRequest
  ): Promise<PageResponse<ProvisioningTask>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.status) query.status = params.status;
    if (params.deviceId) query.device_id = params.deviceId;

    const { data } = await http.get<{ items: BackendProvisioningTask[]; total: number }>(
      '/provisioning/tasks',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendTask),
      total: data.total,
      page: params.page,
      pageSize: params.pageSize,
    };
  },

  async getTask(id: string): Promise<ProvisioningTask> {
    const { data } = await http.get<BackendProvisioningTask>(`/provisioning/tasks/${id}`);
    return mapBackendTask(data);
  },

  async createTask(req: CreateProvisioningTaskRequest): Promise<ProvisioningTask> {
    const body = { device_id: req.deviceId };
    const { data } = await http.post<BackendProvisioningTask>('/provisioning/tasks', body);
    return mapBackendTask(data);
  },

  async retryTask(id: string): Promise<ProvisioningTask> {
    const { data } = await http.post<BackendProvisioningTask>(`/provisioning/tasks/${id}/retry`);
    return mapBackendTask(data);
  },
};
