import http from '../http';
import type { BackupTask, BackupSchedule, FTPConfig, BackupPolicy } from '../../mock/data/backup';
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
      return data;
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
};
