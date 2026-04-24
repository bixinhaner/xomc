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
  carrier: string;
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

interface BackendUpgradeTask {
  id: string;
  task_name: string;
  task_type: number;
  firmware_id?: string;
  file_name?: string;
  file_md5?: string;
  status: string;
  result?: string;
  operator_code: string;
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
}

interface BackendUpgradeSubTask {
  id: string;
  task_id: string;
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
  return {
    id: bt.id,
    taskName: bt.task_name,
    taskType: bt.task_type as TaskTypeValue,
    firmwareId: bt.firmware_id,
    fileName: bt.file_name,
    fileMd5: bt.file_md5,
    status: bt.status as TaskStatusType,
    result: bt.result as TaskResultType | undefined,
    operatorCode: bt.operator_code,
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
    formData.append('carrier', 'cmcc');
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
      carrier: string;
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
    formData.append('carrier', metadata.carrier);
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
  }): Promise<UpgradeTaskInfo> {
    const { data } = await http.post<BackendUpgradeTask>('/upgrade-tasks', {
      device_ids: req.deviceIds,
      firmware_id: req.firmwareId,
      task_name: req.taskName,
      task_type: req.taskType ?? 1,
      is_keep_config: req.isKeepConfig ?? true,
      concurrency: req.concurrency ?? 5,
    });
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

  async retryTask(id: string): Promise<void> {
    await http.post(`/upgrade-tasks/${id}/retry`);
  },

  // ---- Rollback (回退) ----

  async createRollback(req: {
    deviceIds: string[];
    taskName: string;
    operatorCode: string;
    createUser: string;
  }): Promise<UpgradeTaskInfo> {
    const { data } = await http.post<BackendUpgradeTask>('/upgrade-tasks/rollback', {
      device_ids: req.deviceIds,
      task_name: req.taskName,
      operator_code: req.operatorCode,
      create_user: req.createUser,
    });
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

  // ---- Legacy compatibility (delegated to mock) ----
  cancelUpgradePlan: softwareService.cancelUpgradePlan.bind(softwareService),
  precheck: softwareService.precheck.bind(softwareService),

  // ---- Legacy UpgradePlan (for mock compatibility) ----
  async getUpgradePlans(
    params: { status?: string; taskType?: number } & PageRequest
  ): Promise<PageResponse<UpgradeTaskInfo>> {
    return softwareApi.getUpgradeTasks({
      page: params.page,
      pageSize: params.pageSize,
      taskType: params.taskType,
      status: params.status,
    });
  },

  async getUpgradePlanById(id: string): Promise<UpgradeTaskInfo | null> {
    return softwareApi.getUpgradeTaskById(id);
  },
};
