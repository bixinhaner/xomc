import http from '../http';
import type { BackupTask, BackupSchedule, FTPConfig, BackupPolicy, RestoreTask, RestoreStatus } from '../../mock/data/backup';
import { DEFAULT_BACKUP_POLICY } from '../../mock/data/backup';
import type { PageRequest, PageResponse } from '../../types/pagination';

// ---------------------------------------------------------------------------
// Backend response types
// ---------------------------------------------------------------------------

interface BackendBackupTask {
  id: string;
  task_type: string;      // full, incremental, config_only
  target_type: string;    // device, group
  target_ids: string[];
  status: string;         // pending, running, completed, failed, cancelled
  progress: number;
  file_path: string;
  error_message: string;
  started_at: string;
  completed_at: string;
  created_at: string;
  updated_at: string;
}

interface BackendBackupSchedule {
  id: string;
  name: string;
  cron_expr: string;
  enabled: boolean;
  task_type: string;      // full, incremental, config_only
  target_type: string;    // device, group
  target_ids: string[];
  created_at: string;
  updated_at: string;
}

interface BackendFTPConfig {
  id: string;
  config_name: string;
  host: string;
  port: number;
  username: string;
  protocol: string;
  remote_path: string;
  passive: boolean;
  enabled: boolean;
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

// T-0072 / T-0078: backend restore_tasks row shape (snake_case before axios
// interceptor camelCase conversion). Mapping into the frontend RestoreTask is
// explicit so we control nullability normalization (error_message → undefined).
interface BackendRestoreTask {
  id: string;
  source_bucket: string;
  source_object_path: string;
  target_device_sns: string[];
  status: string;
  progress: number;
  error_message?: string | null;
  started_at?: string | null;
  completed_at?: string | null;
  created_at: string;
  updated_at: string;
  created_by?: string | null;
}

// ---------------------------------------------------------------------------
// Mapping helpers: backend task_type <-> frontend backupType
// ---------------------------------------------------------------------------

/** Backend task_type values use underscores; frontend uses hyphens. */
function mapBackendTaskType(
  backendType: string
): BackupTask['backupType'] {
  if (backendType === 'config_only') return 'config-only';
  if (backendType === 'incremental') return 'incremental';
  return 'full';
}

function mapFrontendBackupType(
  frontendType: BackupTask['backupType']
): string {
  if (frontendType === 'config-only') return 'config_only';
  return frontendType; // 'full' | 'incremental' pass through
}

// ---------------------------------------------------------------------------
// Mapping helpers: backend status <-> frontend status
// ---------------------------------------------------------------------------

function mapBackendTaskStatus(
  status: string
): BackupTask['status'] {
  if (status === 'completed') return 'success';
  return status as BackupTask['status'];
}

function mapFrontendTaskStatus(status: string): string {
  if (status === 'success') return 'completed';
  return status;
}

// ---------------------------------------------------------------------------
// Mapping functions: backend -> frontend
// ---------------------------------------------------------------------------

function mapBackendTask(bt: BackendBackupTask): BackupTask {
  const targetCount = bt.target_ids?.length ?? 0;
  const isCompleted = bt.status === 'completed';
  const isFailed = bt.status === 'failed';

  return {
    id: bt.id,
    taskName: '',                                          // backend has no task name
    taskType: 'manual',                                    // default; backend has no direct equivalent
    deviceSns: bt.target_ids || [],
    status: mapBackendTaskStatus(bt.status),
    progress: bt.progress ?? 0,
    successCount: isCompleted ? targetCount : 0,
    failCount: isFailed ? targetCount : 0,
    totalCount: targetCount,
    backupType: mapBackendTaskType(bt.task_type),
    storageLocation: bt.file_path || '',
    createdAt: bt.created_at,
    updatedAt: bt.updated_at || bt.created_at,
    creator: '',                                           // backend has no creator field
    message: bt.error_message || undefined,
  };
}

function mapBackendSchedule(bs: BackendBackupSchedule): BackupSchedule {
  return {
    id: bs.id,
    scheduleName: bs.name,
    deviceGroups: bs.target_ids || [],
    backupType: mapBackendTaskType(bs.task_type),
    cronExpression: bs.cron_expr,
    cronDescription: '',                                   // backend has no cron description
    enabled: bs.enabled,
    retentionDays: 0,                                      // backend has no retention field
    storageLocation: '',                                   // backend has no storage location
    nextRunTime: '',                                       // backend has no next run time
    createTime: bs.created_at,
    creator: '',                                           // backend has no creator
  };
}

function mapBackendFTPConfig(bf: BackendFTPConfig): FTPConfig {
  return {
    id: bf.id,
    configName: bf.config_name,
    host: bf.host,
    port: bf.port,
    username: bf.username,
    protocol: bf.protocol as FTPConfig['protocol'],
    remotePath: bf.remote_path,
    passive: bf.passive,
    enabled: bf.enabled,
    createTime: bf.created_at,
  };
}

function mapToBackendFTPConfig(f: Partial<FTPConfig>): Record<string, unknown> {
  const payload: Record<string, unknown> = {};
  if (f.configName !== undefined) payload.config_name = f.configName;
  if (f.host !== undefined) payload.host = f.host;
  if (f.port !== undefined) payload.port = f.port;
  if (f.username !== undefined) payload.username = f.username;
  if (f.protocol !== undefined) payload.protocol = f.protocol;
  if (f.remotePath !== undefined) payload.remote_path = f.remotePath;
  if (f.passive !== undefined) payload.passive = f.passive;
  if (f.enabled !== undefined) payload.enabled = f.enabled;
  return payload;
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

export const backupApi = {
  // --- Tasks ---

  async getTasks(
    params: { status?: string; taskType?: string } & PageRequest
  ): Promise<PageResponse<BackupTask>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.status) query.status = mapFrontendTaskStatus(params.status);
    if (params.taskType) query.task_type = params.taskType;

    const { data } = await http.get<BackendListResponse<BackendBackupTask>>(
      '/backup/tasks',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendTask),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getTaskById(id: string): Promise<BackupTask | null> {
    try {
      const { data } = await http.get<BackendBackupTask>(
        `/backup/tasks/${id}`
      );
      return mapBackendTask(data);
    } catch {
      return null;
    }
  },

  async createTask(
    data: Omit<
      BackupTask,
      'id' | 'createdAt' | 'updatedAt' | 'status' | 'progress' | 'successCount' | 'failCount'
    >
  ): Promise<BackupTask> {
    const payload = {
      task_type: mapFrontendBackupType(data.backupType),
      target_type: 'device' as const,
      target_ids: data.deviceSns,
    };
    const { data: bt } = await http.post<BackendBackupTask>(
      '/backup/tasks',
      payload
    );
    return mapBackendTask(bt);
  },

  async cancelTask(id: string): Promise<void> {
    await http.post(`/backup/tasks/${id}/cancel`);
  },

  async deleteTasks(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.delete(`/backup/tasks/${id}`);
    }
  },

  // --- Schedules ---

  async getSchedules(
    params: { enabled?: boolean } & PageRequest
  ): Promise<PageResponse<BackupSchedule>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.enabled !== undefined) query.enabled = params.enabled;

    const { data } = await http.get<BackendListResponse<BackendBackupSchedule>>(
      '/backup/schedules',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendSchedule),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async createSchedule(
    data: Omit<BackupSchedule, 'id' | 'createTime'>
  ): Promise<BackupSchedule> {
    const payload = {
      name: data.scheduleName,
      cron_expr: data.cronExpression,
      enabled: data.enabled,
      task_type: mapFrontendBackupType(data.backupType),
      target_type: 'group' as const,
      target_ids: data.deviceGroups,
    };
    const { data: bs } = await http.post<BackendBackupSchedule>(
      '/backup/schedules',
      payload
    );
    return mapBackendSchedule(bs);
  },

  async updateSchedule(
    id: string,
    data: Partial<BackupSchedule>
  ): Promise<BackupSchedule> {
    const payload: Record<string, unknown> = {};
    if (data.scheduleName !== undefined) payload.name = data.scheduleName;
    if (data.cronExpression !== undefined) payload.cron_expr = data.cronExpression;
    if (data.enabled !== undefined) payload.enabled = data.enabled;
    if (data.backupType !== undefined)
      payload.task_type = mapFrontendBackupType(data.backupType);
    if (data.deviceGroups !== undefined) {
      payload.target_type = 'group';
      payload.target_ids = data.deviceGroups;
    }

    const { data: bs } = await http.put<BackendBackupSchedule>(
      `/backup/schedules/${id}`,
      payload
    );
    return mapBackendSchedule(bs);
  },

  async deleteSchedules(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.delete(`/backup/schedules/${id}`);
    }
  },

  // --- FTP Configs ---

  async getFTPConfigs(
    params: PageRequest
  ): Promise<PageResponse<FTPConfig>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };

    const { data } = await http.get<BackendListResponse<BackendFTPConfig>>(
      '/backup/ftp-configs',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendFTPConfig),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async createFTPConfig(
    data: Omit<FTPConfig, 'id' | 'createTime'>
  ): Promise<FTPConfig> {
    const payload = mapToBackendFTPConfig(data);
    const { data: bf } = await http.post<BackendFTPConfig>(
      '/backup/ftp-configs',
      payload
    );
    return mapBackendFTPConfig(bf);
  },

  async updateFTPConfig(
    id: string,
    data: Partial<FTPConfig>
  ): Promise<FTPConfig> {
    const payload = mapToBackendFTPConfig(data);
    const { data: bf } = await http.put<BackendFTPConfig>(
      `/backup/ftp-configs/${id}`,
      payload
    );
    return mapBackendFTPConfig(bf);
  },

  async deleteFTPConfigs(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.delete(`/backup/ftp-configs/${id}`);
    }
  },

  async testFTPConnection(
    id: string
  ): Promise<{ success: boolean; message: string }> {
    const { data } = await http.post<{ success: boolean; message: string }>(
      `/backup/ftp-configs/${id}/test`
    );
    return data;
  },

  // --- Policy (T-0071, singleton) ---

  /**
   * Fetches the singleton backup policy. When the backend returns a 404
   * (table empty) we surface canonical defaults so first-render UI matches
   * what a fresh PUT would create. Other errors propagate.
   */
  async getPolicy(): Promise<BackupPolicy> {
    try {
      const { data } = await http.get<BackupPolicy>('/backup/policy');
      // Axios interceptor already converts snake_case → camelCase, so the
      // response shape is the frontend BackupPolicy interface. ftp_config_id
      // null → ftpConfigId null, preserved.
      // T-0088 fallback: pre-T-0084 backend may not return alertSeverity;
      // default to 'major' so the form Select renders deterministically.
      return { ...data, alertSeverity: data.alertSeverity ?? 'major' };
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status;
      if (status === 404) return { ...DEFAULT_BACKUP_POLICY };
      throw err;
    }
  },

  async updatePolicy(payload: BackupPolicy): Promise<BackupPolicy> {
    const { data } = await http.put<BackupPolicy>('/backup/policy', payload);
    return data;
  },

  // --- Restore (T-0072 / T-0078) ---

  /**
   * Creates a restore task. Backend validates path traversal + restricts the
   * source bucket to "config_backup"; FE form should mirror these rules so a
   * 400 is rare.
   */
  async createRestore(req: {
    bucket: string;
    objectPath: string;
    targetDeviceSns: string[];
  }): Promise<RestoreTask> {
    const { data } = await http.post<BackendRestoreTask>('/backup/restore', {
      bucket: req.bucket,
      object_path: req.objectPath,
      target_device_sns: req.targetDeviceSns,
    });
    return mapBackendRestoreTask(data);
  },

  async createRestoreByTaskID(req: {
    backupTaskId: string;
    targetDeviceSns: string[];
  }): Promise<RestoreTask> {
    const { data } = await http.post<BackendRestoreTask>('/backup/restore/by-task-id', {
      backup_task_id: req.backupTaskId,
      target_device_sns: req.targetDeviceSns,
    });
    return mapBackendRestoreTask(data);
  },

  /**
   * T-0164 B5: 按设备快照恢复。每台设备从 config_snapshots 取自己最新一份。
   * 缺失快照的设备会触发后端整批拒绝（HTTP 4xx + missing 列表），
   * 调用方应捕获 axios 错误并读取 response.data.missing。
   */
  async createRestoreBySnapshot(req: {
    targetDeviceSns: string[];
  }): Promise<{ task: RestoreTask | null; missing: string[] }> {
    const { data } = await http.post<{
      task?: BackendRestoreTask;
      missing?: string[];
    }>('/backup/restore/by-snapshot', {
      target_device_sns: req.targetDeviceSns,
    });
    return {
      task: data.task ? mapBackendRestoreTask(data.task) : null,
      missing: data.missing ?? [],
    };
  },

  async listRestoreTasks(
    params: PageRequest & { status?: RestoreStatus }
  ): Promise<PageResponse<RestoreTask>> {
    const { data } = await http.get<BackendListResponse<BackendRestoreTask>>(
      '/backup/restore-tasks',
      { params }
    );
    return {
      items: (data.items || []).map(mapBackendRestoreTask),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getRestoreTask(id: string): Promise<RestoreTask> {
    const { data } = await http.get<BackendRestoreTask>(
      `/backup/restore-tasks/${id}`
    );
    return mapBackendRestoreTask(data);
  },
};

// ---------------------------------------------------------------------------
// Mapping helpers (T-0078): restore_tasks
// ---------------------------------------------------------------------------

// Whitelist of valid restore status values mirroring the DB CHECK constraint.
// Any unknown value from the backend (e.g. a future status the FE doesn't
// recognize yet) collapses to 'pending' so the UI avoids `undefined`
// indexing into STATUS_TAG and producing a runtime crash on Tag color (review M1 fix).
const KNOWN_RESTORE_STATUSES: ReadonlySet<RestoreStatus> = new Set<RestoreStatus>([
  'pending',
  'running',
  'completed',
  'failed',
  'cancelled',
]);

function mapBackendRestoreTask(b: BackendRestoreTask): RestoreTask {
  const rawStatus = b.status as RestoreStatus;
  const status: RestoreStatus = KNOWN_RESTORE_STATUSES.has(rawStatus)
    ? rawStatus
    : 'pending';
  return {
    id: b.id,
    sourceBucket: b.source_bucket,
    sourceObjectPath: b.source_object_path,
    targetDeviceSns: b.target_device_sns ?? [],
    status,
    progress: b.progress ?? 0,
    errorMessage: b.error_message ?? undefined,
    startedAt: b.started_at ?? undefined,
    completedAt: b.completed_at ?? undefined,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
    createdBy: b.created_by ?? undefined,
  };
}
