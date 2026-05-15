import http from '../http';
import type {
  SoftwareVersion,
  UpgradePlan,
  UpgradeTaskInfo,
  UpgradeSubTaskInfo,
  VersionStatus,
  FileTypeValue,
  TaskTypeValue,
  TaskStatusType,
  TaskResultType,
  SubTaskStatusType,
} from '../../mock/data/software';
import type { PageRequest, PageResponse } from '../../types/pagination';
import { softwareService } from '../../mock/services/softwareService';

// ============================================================================
// Backend response types (snake_case — matches backend JSON exactly)
// ============================================================================

interface BackendFirmwareVersion {
  id: string;
  product_class: string;
  version: string;
  file_name: string;
  file_size: number;
  file_type: number;
  minio_path: string;
  compatible_oui: string[];
  md5_val: string;
  recommend: boolean;
  uploader: string;
  manufacturer: string;
  release_notes: string;
  description: string;
  status: string;
  created_at: string;
  updated_at: string;
}

interface BackendCanaryStage {
  percent: number;
  failure_threshold: number;
}

interface BackendStageHistoryEntry {
  stage: number;
  percent?: number;
  devices_in_stage?: number;
  success_count?: number;
  fail_count?: number;
  failure_rate?: number;
  action: string;
  at: string;
  reason?: string;
}

interface BackendUpgradeTask {
  id: string;
  task_name: string;
  task_type: number;
  firmware_id?: string;
  file_name?: string;
  file_md5?: string;
  status: string;
  result?: string;
  product_class: string;
  is_keep_config: boolean;
  create_status: string;
  create_user: string;
  total_count: number;
  success_count: number;
  fail_count: number;
  max_concurrent: number;
  started_at?: string;
  ended_at?: string;
  created_at: string;
  updated_at: string;

  // Canary fields (T-0019, mirrors backend T-0018 migration 000045)
  strategy?: string;
  canary_stages?: BackendCanaryStage[];
  current_stage?: number;
  stage_status?: string;
  stage_history?: BackendStageHistoryEntry[];
  auto_advance?: boolean;
  auto_advance_minutes?: number;
}

interface BackendUpgradeSubTask {
  id: string;
  task_id: string;
  task_name?: string;
  device_id: string;
  firmware_id?: string;
  status: string;
  error_message?: string;
  retry_count: number;
  max_retries: number;
  device_sn?: string;
  ori_version?: string;
  dest_version?: string;
  command_key?: string;
  failure_reason?: string;
  pre_suspend_status?: string;
  started_at?: string;
  completed_at?: string;
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

// ============================================================================
// Mapping functions: backend (snake_case) → frontend (camelCase)
// ============================================================================

function mapFirmwareStatus(status: string): VersionStatus {
  const map: Record<string, VersionStatus> = {
    current: 'current',
    deprecated: 'deprecated',
    beta: 'beta',
    archived: 'archived',
  };
  return map[status] || 'current';
}

function mapFirmware(bf: BackendFirmwareVersion): SoftwareVersion {
  return {
    id: bf.id,
    versionName: `${bf.product_class || ''} ${bf.version}`.trim(),
    versionCode: bf.version,
    fileName: bf.file_name,
    deviceType: bf.product_class || '',
    vendor: bf.compatible_oui?.[0] || '',
    releaseDate: bf.created_at,
    status: mapFirmwareStatus(bf.status),
    fileSize: bf.file_size,
    checksum: bf.md5_val || '',
    downloadUrl: bf.minio_path || '',
    releaseNotes: bf.release_notes || '',
    minHardwareVersion: '',
    features: [],
    bugFixes: [],
    fileType: bf.file_type as FileTypeValue,
    recommend: bf.recommend,
    uploader: bf.uploader,
    manufacturer: bf.manufacturer,
    description: bf.description,
  };
}

function mapFirmwareListResponse(
  resp: BackendListResponse<BackendFirmwareVersion>
): PageResponse<SoftwareVersion> {
  return {
    items: (resp.items || []).map(mapFirmware),
    total: resp.total,
    page: resp.page,
    pageSize: resp.page_size,
  };
}

function mapUpgradeTask(bt: BackendUpgradeTask): UpgradeTaskInfo {
  const result: UpgradeTaskInfo = {
    id: bt.id,
    taskName: bt.task_name,
    taskType: bt.task_type as TaskTypeValue,
    firmwareId: bt.firmware_id,
    fileName: bt.file_name,
    fileMd5: bt.file_md5,
    status: bt.status as TaskStatusType,
    result: bt.result as TaskResultType | undefined,
    productClass: bt.product_class,
    isKeepConfig: bt.is_keep_config,
    createStatus: bt.create_status,
    createUser: bt.create_user,
    totalCount: bt.total_count,
    successCount: bt.success_count,
    failCount: bt.fail_count,
    maxConcurrent: bt.max_concurrent,
    startedAt: bt.started_at,
    endedAt: bt.ended_at,
    createdAt: bt.created_at,
    updatedAt: bt.updated_at,
  };
  // Canary fields (T-0019, optional). Default strategy='full' when missing.
  if (bt.strategy) {
    result.strategy = bt.strategy as 'full' | 'canary';
  }
  if (bt.canary_stages) {
    result.canaryStages = bt.canary_stages.map((s) => ({
      percent: s.percent,
      failureThreshold: s.failure_threshold,
    }));
  }
  if (typeof bt.current_stage === 'number') {
    result.currentStage = bt.current_stage;
  }
  if (bt.stage_status) {
    result.stageStatus = bt.stage_status as UpgradeTaskInfo['stageStatus'];
  }
  if (bt.stage_history) {
    result.stageHistory = bt.stage_history.map((h) => ({
      stage: h.stage,
      percent: h.percent ?? 0,
      devicesInStage: h.devices_in_stage ?? 0,
      successCount: h.success_count ?? 0,
      failCount: h.fail_count ?? 0,
      failureRate: h.failure_rate ?? 0,
      action: h.action,
      at: h.at,
      reason: h.reason,
    }));
  }
  if (typeof bt.auto_advance === 'boolean') {
    result.autoAdvance = bt.auto_advance;
  }
  if (typeof bt.auto_advance_minutes === 'number') {
    result.autoAdvanceMinutes = bt.auto_advance_minutes;
  }
  return result;
}

function mapUpgradeTaskListResponse(
  resp: BackendListResponse<BackendUpgradeTask>
): PageResponse<UpgradeTaskInfo> {
  return {
    items: (resp.items || []).map(mapUpgradeTask),
    total: resp.total,
    page: resp.page,
    pageSize: resp.page_size,
  };
}

function mapUpgradeSubTask(bs: BackendUpgradeSubTask): UpgradeSubTaskInfo {
  return {
    id: bs.id,
    taskId: bs.task_id,
    taskName: bs.task_name,
    deviceId: bs.device_id,
    firmwareId: bs.firmware_id,
    status: bs.status as SubTaskStatusType,
    errorMessage: bs.error_message,
    retryCount: bs.retry_count,
    maxRetries: bs.max_retries,
    deviceSn: bs.device_sn,
    oriVersion: bs.ori_version,
    destVersion: bs.dest_version,
    commandKey: bs.command_key,
    failureReason: bs.failure_reason,
    preSuspendStatus: bs.pre_suspend_status,
    startedAt: bs.started_at,
    completedAt: bs.completed_at,
    createdAt: bs.created_at,
    updatedAt: bs.updated_at,
  };
}

function mapSubTaskListResponse(
  resp: BackendListResponse<BackendUpgradeSubTask>
): PageResponse<UpgradeSubTaskInfo> {
  return {
    items: (resp.items || []).map(mapUpgradeSubTask),
    total: resp.total,
    page: resp.page,
    pageSize: resp.page_size,
  };
}

// ============================================================================
// API functions
// ============================================================================

export const softwareApi = {
  // ---- Firmware (固件) ----

  async getVersions(
    params: { deviceType?: string; status?: string; vendor?: string; fileType?: number } & PageRequest
  ): Promise<PageResponse<SoftwareVersion>> {
    const query: Record<string, unknown> = {
      page: params.page,
      page_size: params.pageSize,
    };
    if (params.deviceType) query.product_class = params.deviceType;
    if (params.status) query.status = params.status;
    if (params.fileType !== undefined) query.file_type = params.fileType;

    const { data } = await http.get<BackendListResponse<BackendFirmwareVersion>>(
      '/firmware',
      { params: query }
    );
    return mapFirmwareListResponse(data);
  },

  async getVersionById(id: string): Promise<SoftwareVersion | null> {
    try {
      const { data } = await http.get<BackendFirmwareVersion>(`/firmware/${id}`);
      return mapFirmware(data);
    } catch {
      return null;
    }
  },

  async uploadVersion(
    data: Omit<SoftwareVersion, 'id' | 'releaseDate'>
  ): Promise<SoftwareVersion> {
    const formData = new FormData();
    formData.append('version', data.versionCode);
    if (data.deviceType) formData.append('product_class', data.deviceType);
    if (data.releaseNotes) formData.append('release_notes', data.releaseNotes);

    const { data: bf } = await http.post<BackendFirmwareVersion>(
      '/firmware',
      formData,
      {
        headers: { 'Content-Type': 'multipart/form-data' },
        timeout: 120000,
      }
    );
    return mapFirmware(bf);
  },

  async uploadFirmware(
    file: File,
    metadata: {
      version: string;
      productClass?: string;
      releaseNotes?: string;
      fileType?: number;
      recommend?: boolean;
      uploader?: string;
      manufacturer?: string;
      description?: string;
    }
  ): Promise<SoftwareVersion> {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('version', metadata.version);
    if (metadata.productClass)
      formData.append('product_class', metadata.productClass);
    if (metadata.releaseNotes)
      formData.append('release_notes', metadata.releaseNotes);
    if (metadata.fileType !== undefined)
      formData.append('file_type', String(metadata.fileType));
    if (metadata.recommend)
      formData.append('recommend', 'true');
    if (metadata.uploader)
      formData.append('uploader', metadata.uploader);
    if (metadata.manufacturer)
      formData.append('manufacturer', metadata.manufacturer);
    if (metadata.description)
      formData.append('description', metadata.description);

    const { data } = await http.post<BackendFirmwareVersion>(
      '/firmware',
      formData,
      {
        headers: { 'Content-Type': 'multipart/form-data' },
        timeout: 120000,
      }
    );
    return mapFirmware(data);
  },

  async deleteVersions(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.delete(`/firmware/${id}`);
    }
  },

  async toggleRecommend(id: string): Promise<SoftwareVersion> {
    const { data } = await http.put<BackendFirmwareVersion>(`/firmware/${id}/recommend`);
    return mapFirmware(data);
  },

  async downloadFirmware(id: string, fileName: string): Promise<void> {
    const response = await http.get(`/firmware/${id}/download`, {
      responseType: 'blob',
    });
    const url = window.URL.createObjectURL(new Blob([response.data as BlobPart]));
    const link = document.createElement('a');
    link.href = url;
    link.download = fileName || `firmware_${id}`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(url);
  },

  async updateFirmware(id: string, metadata: {
    productClass?: string;
    version?: string;
    recommend?: boolean;
    description?: string;
    releaseNotes?: string;
  }): Promise<SoftwareVersion> {
    const { data } = await http.put<BackendFirmwareVersion>(`/firmware/${id}`, {
      product_class: metadata.productClass,
      version: metadata.version,
      recommend: metadata.recommend,
      description: metadata.description,
      release_notes: metadata.releaseNotes,
    });
    return mapFirmware(data);
  },

  // ---- Upgrade Tasks (主任务) ----

  async getUpgradeTasks(
    params: { taskType?: number; status?: string; productClass?: string; createUser?: string } & PageRequest
  ): Promise<PageResponse<UpgradeTaskInfo>> {
    const query: Record<string, unknown> = {
      page: params.page,
      page_size: params.pageSize,
    };
    if (params.taskType !== undefined) query.task_type = params.taskType;
    if (params.status) query.status = params.status;
    if (params.productClass) query.product_class = params.productClass;
    if (params.createUser) query.create_user = params.createUser;

    const { data } = await http.get<BackendListResponse<BackendUpgradeTask>>(
      '/upgrade-tasks',
      { params: query }
    );
    return mapUpgradeTaskListResponse(data);
  },

  async getUpgradeTaskById(id: string): Promise<UpgradeTaskInfo | null> {
    try {
      const { data } = await http.get<BackendUpgradeTask>(`/upgrade-tasks/${id}`);
      return mapUpgradeTask(data);
    } catch {
      return null;
    }
  },

  async createUpgradeTask(req: {
    deviceIds: string[];
    firmwareId: string;
    taskName: string;
    taskType?: number;
    isKeepConfig?: boolean;
    concurrency?: number;
    createSuspended?: boolean;
  }): Promise<UpgradeTaskInfo> {
    const payload: Record<string, unknown> = {
      device_ids: req.deviceIds,
      firmware_id: req.firmwareId,
      task_name: req.taskName,
      task_type: req.taskType ?? 1,
      is_keep_config: req.isKeepConfig ?? true,
      concurrency: req.concurrency ?? 5,
    };
    if (req.createSuspended) {
      payload.create_suspended = true;
    }
    const { data } = await http.post<BackendUpgradeTask>('/upgrade-tasks', payload);
    return mapUpgradeTask(data);
  },

  async suspendTask(id: string): Promise<void> {
    await http.put(`/upgrade-tasks/${id}/suspend`);
  },

  async resumeTask(id: string): Promise<void> {
    await http.put(`/upgrade-tasks/${id}/resume`);
  },

  async terminateTask(id: string): Promise<void> {
    await http.put(`/upgrade-tasks/${id}/terminate`);
  },

  async deleteTask(id: string): Promise<void> {
    await http.delete(`/upgrade-tasks/${id}`);
  },

  async retryTask(id: string): Promise<void> {
    await http.post(`/upgrade-tasks/${id}/retry`);
  },

  // ---- Canary stage transitions (T-0019, mirrors backend T-0018) ----

  async advanceCanary(id: string): Promise<void> {
    await http.post(`/upgrade-tasks/${id}/advance`);
  },

  async pauseCanary(id: string): Promise<void> {
    await http.post(`/upgrade-tasks/${id}/pause-canary`);
  },

  async resumeCanary(id: string): Promise<void> {
    await http.post(`/upgrade-tasks/${id}/resume-canary`);
  },

  async abortCanary(id: string): Promise<void> {
    await http.post(`/upgrade-tasks/${id}/abort-canary`);
  },

  // ---- Rollback (回退) ----

  async createRollback(req: {
    deviceIds: string[];
    taskName: string;
    createUser: string;
    createSuspended?: boolean;
  }): Promise<UpgradeTaskInfo> {
    const body: Record<string, unknown> = {
      device_ids: req.deviceIds,
      task_name: req.taskName,
      create_user: req.createUser,
    };
    if (req.createSuspended) body.create_suspended = true;
    const { data } = await http.post<BackendUpgradeTask>('/upgrade-tasks/rollback', body);
    return mapUpgradeTask(data);
  },

  // ---- Sub Tasks (子任务) ----

  async getSubTasks(
    taskId: string,
    params: { status?: string } & PageRequest
  ): Promise<PageResponse<UpgradeSubTaskInfo>> {
    const query: Record<string, unknown> = {
      page: params.page,
      page_size: params.pageSize,
    };
    if (params.status) query.status = params.status;

    const { data } = await http.get<BackendListResponse<BackendUpgradeSubTask>>(
      `/upgrade-tasks/${taskId}/tasks`,
      { params: query }
    );
    return mapSubTaskListResponse(data);
  },

  async getSubTaskById(id: string): Promise<UpgradeSubTaskInfo | null> {
    try {
      const { data } = await http.get<BackendUpgradeSubTask>(`/upgrade-sub-tasks/${id}`);
      return mapUpgradeSubTask(data);
    } catch {
      return null;
    }
  },

  // ---- All Sub Tasks (跨任务设备列表) ----

  async getAllSubTasks(
    params: { taskName?: string; deviceSn?: string; status?: string; taskType?: number } & PageRequest
  ): Promise<PageResponse<UpgradeSubTaskInfo>> {
    const query: Record<string, unknown> = {
      page: params.page,
      page_size: params.pageSize,
    };
    if (params.taskName) query.task_name = params.taskName;
    if (params.deviceSn) query.device_sn = params.deviceSn;
    if (params.status) query.status = params.status;
    if (params.taskType !== undefined) query.task_type = params.taskType;

    const { data } = await http.get<BackendListResponse<BackendUpgradeSubTask>>(
      '/upgrade-sub-tasks',
      { params: query }
    );
    return mapSubTaskListResponse(data);
  },

  // ---- Legacy compatibility (delegated to mock) ----
  cancelUpgradePlan: softwareService.cancelUpgradePlan.bind(softwareService),
  precheck: softwareService.precheck.bind(softwareService),
  createUpgradePlan: softwareService.createUpgradePlan.bind(softwareService),

  // ---- Legacy UpgradePlan (delegated to mock for type consistency, T-0129 batch A-2) ----
  // 注：mock UpgradePlan vs real UpgradeTaskInfo 是不同 schema (planName/targetVersionId vs taskName/firmwareId 等)。
  // hook 端 useUpgradePlans 期望 UpgradePlan 类型，所以 real 端 delegate mock 保类型一致。
  // 若要真后端联调升级计划，需先做 schema 统一改造（est M 独立任务）。
  getUpgradePlans: softwareService.getUpgradePlans.bind(softwareService),
  getUpgradePlanById: softwareService.getUpgradePlanById.bind(softwareService),
};
