import http from '../http';

// ============================================================================
// Backend response types (snake_case — matches backend JSON exactly)
// ============================================================================

interface BackendLogFile {
  id: string;
  device_id: string | null;
  device_sn: string;
  log_type: 'running' | 'fault';
  file_name: string;
  object_path: string;
  bucket: string;
  file_size: number;
  fault_reason: string;
  fault_detail: string;
  task_id: string | null;
  is_deleted: boolean;
  collected_at: string;
  created_at: string;
  updated_at: string;
}

interface BackendListResponse {
  items: BackendLogFile[] | null;
  total: number;
  page: number;
  size: number;
}

// ============================================================================
// Frontend types (camelCase)
// ============================================================================

export interface StationLogFile {
  id: string;
  deviceId: string | null;
  deviceSn: string;
  logType: 'running' | 'fault';
  fileName: string;
  objectPath: string;
  bucket: string;
  fileSize: number;
  faultReason: string;
  faultDetail: string;
  taskId: string | null;
  isDeleted: boolean;
  collectedAt: string;
  createdAt: string;
  updatedAt: string;
}

export interface StationLogListParams {
  deviceId?: string;
  logType?: 'running' | 'fault';
  page?: number;
  pageSize?: number;
}

export interface StationLogListResponse {
  items: StationLogFile[];
  total: number;
  page: number;
  size: number;
}

// ============================================================================
// Mappers
// ============================================================================

function mapBackendLogFile(b: BackendLogFile): StationLogFile {
  return {
    id: b.id,
    deviceId: b.device_id,
    deviceSn: b.device_sn,
    logType: b.log_type,
    fileName: b.file_name,
    objectPath: b.object_path,
    bucket: b.bucket,
    fileSize: b.file_size,
    faultReason: b.fault_reason,
    faultDetail: b.fault_detail,
    taskId: b.task_id,
    isDeleted: b.is_deleted,
    collectedAt: b.collected_at,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

// ============================================================================
// API service
// ============================================================================

const BASE = '/station-logs';

export const stationLogApi = {
  /**
   * 查询日志文件列表
   */
  list(params: StationLogListParams): Promise<StationLogListResponse> {
    const query: Record<string, string | number> = {};
    if (params.deviceId) query.device_id = params.deviceId;
    if (params.logType) query.log_type = params.logType;
    if (params.page) query.page = params.page;
    if (params.pageSize) query.page_size = params.pageSize;

    return http.get<BackendListResponse>(BASE, { params: query }).then((res) => ({
      items: (res.data.items ?? []).map(mapBackendLogFile),
      total: res.data.total,
      page: res.data.page,
      size: res.data.size,
    }));
  },

  /**
   * 获取预签名下载 URL（有效期 1 小时）
   */
  getDownloadUrl(id: string): Promise<string> {
    return http
      .get<{ url: string }>(`${BASE}/${id}/download`)
      .then((res) => res.data.url);
  },

  /**
   * 删除日志文件
   */
  delete(id: string): Promise<void> {
    return http.delete(`${BASE}/${id}`).then(() => undefined);
  },
};
