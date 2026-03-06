import http from '../http';
import type { PageRequest, PageResponse } from '@/types/pagination';

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
