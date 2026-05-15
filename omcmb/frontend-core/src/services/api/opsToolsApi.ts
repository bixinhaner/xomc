import http from '../http';
import type { OpsTemplate, OpsCommandRecord, OpsTask, OpsStep } from '../../mock/data/opsTools';
import type { PageRequest, PageResponse } from '../../types/pagination';

// ---------------------------------------------------------------------------
// Backend response types (snake_case)
// ---------------------------------------------------------------------------

interface BackendOpsTemplate {
  id: string;
  template_name: string;
  description: string;
  category: string;
  target_device_types: string[];
  steps: Array<Record<string, unknown>>;
  estimated_duration: number;
  creator: string;
  use_count: number;
  tags: string[];
  created_at: string;
  updated_at: string;
}

interface BackendOpsTask {
  id: string;
  task_name: string;
  template_id?: string;
  device_sns: string[];
  status: string;
  current_step: number;
  total_steps: number;
  progress: number;
  success_count: number;
  fail_count: number;
  total_count: number;
  creator: string;
  message: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

interface BackendOpsCommandRecord {
  id: string;
  command_text: string;
  device_sn: string;
  device_name: string;
  operator: string;
  execute_time: string;
  duration: number;
  success: boolean;
  output: string;
  error_message: string;
  created_at: string;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// ---------------------------------------------------------------------------
// Mapping functions: backend -> frontend
// ---------------------------------------------------------------------------

function mapBackendTemplate(bt: BackendOpsTemplate): OpsTemplate {
  return {
    id: bt.id,
    templateName: bt.template_name,
    description: bt.description || '',
    category: bt.category || '',
    targetDeviceTypes: bt.target_device_types || [],
    steps: (bt.steps || []).map((s: Record<string, unknown>, idx: number) => ({
      stepNo: (s.step_no as number) ?? (s.stepNo as number) ?? idx + 1,
      stepName: (s.step_name as string) ?? (s.stepName as string) ?? '',
      stepType: ((s.step_type as string) ?? (s.stepType as string) ?? 'mml') as OpsStep['stepType'],
      command: (s.command as string) ?? undefined,
      condition: (s.condition as string) ?? undefined,
      waitSeconds: (s.wait_seconds as number) ?? (s.waitSeconds as number) ?? undefined,
      notifyTarget: (s.notify_target as string) ?? (s.notifyTarget as string) ?? undefined,
      description: (s.description as string) ?? '',
      rollbackCommand: (s.rollback_command as string) ?? (s.rollbackCommand as string) ?? undefined,
    })),
    estimatedDuration: bt.estimated_duration || 0,
    creator: bt.creator || '',
    createTime: bt.created_at,
    updateTime: bt.updated_at,
    useCount: bt.use_count || 0,
    tags: bt.tags || [],
  };
}

function mapBackendTask(bt: BackendOpsTask): OpsTask {
  return {
    id: bt.id,
    taskName: bt.task_name,
    templateId: bt.template_id || undefined,
    deviceSns: bt.device_sns || [],
    status: bt.status as OpsTask['status'],
    currentStep: bt.current_step,
    totalSteps: bt.total_steps,
    progress: bt.progress,
    successCount: bt.success_count,
    failCount: bt.fail_count,
    totalCount: bt.total_count,
    createdAt: bt.created_at,
    startedAt: bt.started_at || undefined,
    completedAt: bt.completed_at || undefined,
    creator: bt.creator || '',
    message: bt.message || undefined,
  };
}

function mapBackendCommandRecord(br: BackendOpsCommandRecord): OpsCommandRecord {
  return {
    id: br.id,
    commandText: br.command_text,
    deviceSn: br.device_sn,
    deviceName: br.device_name || '',
    operator: br.operator || '',
    executeTime: br.execute_time,
    duration: br.duration,
    success: br.success,
    output: br.output || '',
    errorMessage: br.error_message || undefined,
  };
}

// ---------------------------------------------------------------------------
// Mapping functions: frontend -> backend payloads
// ---------------------------------------------------------------------------

function mapTemplateToBackend(
  data: Partial<OpsTemplate>
): Record<string, unknown> {
  const payload: Record<string, unknown> = {};
  if (data.templateName !== undefined) payload.template_name = data.templateName;
  if (data.description !== undefined) payload.description = data.description;
  if (data.category !== undefined) payload.category = data.category;
  if (data.targetDeviceTypes !== undefined) payload.target_device_types = data.targetDeviceTypes;
  if (data.steps !== undefined) payload.steps = data.steps;
  if (data.estimatedDuration !== undefined) payload.estimated_duration = data.estimatedDuration;
  if (data.creator !== undefined) payload.creator = data.creator;
  if (data.tags !== undefined) payload.tags = data.tags;
  return payload;
}

// ---------------------------------------------------------------------------
// Public API (13 methods matching 13 endpoints)
// ---------------------------------------------------------------------------

export const opsToolsApi = {
  // --- Templates (5 endpoints) ---

  async getTemplates(
    params: { category?: string; keyword?: string; targetDeviceType?: string } & PageRequest
  ): Promise<PageResponse<OpsTemplate>> {
    const query: Record<string, unknown> = {
      page: params.page,
      page_size: params.pageSize,
    };
    if (params.category) query.category = params.category;
    if (params.keyword) query.keyword = params.keyword;

    const { data } = await http.get<BackendListResponse<BackendOpsTemplate>>(
      '/ops/templates',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendTemplate),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getTemplateById(id: string): Promise<OpsTemplate | null> {
    try {
      const { data } = await http.get<BackendOpsTemplate>(
        `/ops/templates/${id}`
      );
      return mapBackendTemplate(data);
    } catch {
      return null;
    }
  },

  async createTemplate(
    data: Omit<OpsTemplate, 'id' | 'createTime' | 'updateTime' | 'useCount'>
  ): Promise<OpsTemplate> {
    const payload = mapTemplateToBackend(data);
    const { data: bt } = await http.post<BackendOpsTemplate>(
      '/ops/templates',
      payload
    );
    return mapBackendTemplate(bt);
  },

  async updateTemplate(id: string, data: Partial<OpsTemplate>): Promise<OpsTemplate> {
    const payload = mapTemplateToBackend(data);
    const { data: bt } = await http.put<BackendOpsTemplate>(
      `/ops/templates/${id}`,
      payload
    );
    return mapBackendTemplate(bt);
  },

  async deleteTemplates(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.delete(`/ops/templates/${id}`);
    }
  },

  // --- Command Records (2 endpoints) ---

  async getCommandRecords(
    params: { deviceSn?: string; operator?: string; success?: boolean } & PageRequest
  ): Promise<PageResponse<OpsCommandRecord>> {
    const query: Record<string, unknown> = {
      page: params.page,
      page_size: params.pageSize,
    };
    if (params.deviceSn) query.deviceSn = params.deviceSn;
    if (params.operator) query.operator = params.operator;
    if (params.success !== undefined) query.success = params.success;

    const { data } = await http.get<BackendListResponse<BackendOpsCommandRecord>>(
      '/ops/command-records',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendCommandRecord),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async addCommandRecord(
    data: Omit<OpsCommandRecord, 'id'>
  ): Promise<OpsCommandRecord> {
    const payload = {
      command_text: data.commandText,
      device_sn: data.deviceSn,
      device_name: data.deviceName,
      operator: data.operator,
      execute_time: data.executeTime,
      duration: data.duration,
      success: data.success,
      output: data.output,
      error_message: data.errorMessage,
    };
    const { data: br } = await http.post<BackendOpsCommandRecord>(
      '/ops/command-records',
      payload
    );
    return mapBackendCommandRecord(br);
  },

  // --- Tasks (6 endpoints) ---

  async getTasks(
    params: { status?: string; templateId?: string; keyword?: string; creator?: string } & PageRequest
  ): Promise<PageResponse<OpsTask>> {
    const query: Record<string, unknown> = {
      page: params.page,
      page_size: params.pageSize,
    };
    if (params.status) query.status = params.status;
    if (params.templateId) query.templateId = params.templateId;
    if (params.keyword) query.keyword = params.keyword;
    if (params.creator) query.creator = params.creator;

    const { data } = await http.get<BackendListResponse<BackendOpsTask>>(
      '/ops/tasks',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendTask),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getTaskById(id: string): Promise<OpsTask | null> {
    try {
      const { data } = await http.get<BackendOpsTask>(
        `/ops/tasks/${id}`
      );
      return mapBackendTask(data);
    } catch {
      return null;
    }
  },

  async createTask(
    data: Omit<OpsTask, 'id' | 'status' | 'currentStep' | 'progress' | 'successCount' | 'failCount' | 'createdAt'>
  ): Promise<OpsTask> {
    const payload = {
      task_name: data.taskName,
      template_id: data.templateId || undefined,
      device_sns: data.deviceSns,
      total_steps: data.totalSteps,
      total_count: data.totalCount,
      creator: data.creator,
      message: data.message,
    };
    const { data: bt } = await http.post<BackendOpsTask>(
      '/ops/tasks',
      payload
    );
    return mapBackendTask(bt);
  },

  async cancelTask(id: string): Promise<void> {
    await http.post(`/ops/tasks/${id}/cancel`);
  },

  async pauseTask(id: string): Promise<void> {
    await http.post(`/ops/tasks/${id}/pause`);
  },

  async resumeTask(id: string): Promise<void> {
    await http.post(`/ops/tasks/${id}/resume`);
  },
};
