import http from '../http';

// ============================================================================
// T-0158: 设备异常重启记录（station_fault_logs）API
//
// 与 stationLogApi 故意分开：异常重启记录有独立的过滤维度（record_status /
// device_type / 时间区间）和独立的菜单/权限语义；混用会让两边互相牵制。
// ============================================================================

export type RecordStatus = 'detected' | 'file_received' | 'collection_failed';

// ManualCollectionStatus 与后端的 station_fault_logs.manual_collection_status
// 列对齐：'0' 未收集 / 已完成，'1' 收集中，'2' 失败。本期 MVP 只展示，不支持触发。
export type ManualCollectionStatus = '0' | '1' | '2';

// ----------------------------------------------------------------------------
// Backend response types (snake_case)
// ----------------------------------------------------------------------------

interface BackendAbnormalReboot {
  id: string;
  device_id: string | null;
  device_sn: string;
  device_name?: string;
  device_type?: string; // eNB / gNB
  is_gnb: boolean;
  operate_ip?: string;
  software_version?: string;
  file_name?: string;
  object_path?: string;
  bucket?: string;
  file_size: number;
  fault_reason?: string; // HaltReason.MainReason
  fault_detail?: string; // HaltReason.DetailReason
  runtime_before_reboot?: number; // 秒
  record_status: RecordStatus;
  collection_fail_reason?: string;
  manual_collection_status: ManualCollectionStatus;
  is_deleted: boolean;
  collected_at: string;
  created_at: string;
  updated_at: string;
}

interface BackendListResponse {
  items: BackendAbnormalReboot[] | null;
  total: number;
  page: number;
  size: number;
}

// ----------------------------------------------------------------------------
// Frontend types (camelCase)
// ----------------------------------------------------------------------------

export interface AbnormalReboot {
  id: string;
  deviceId: string | null;
  deviceSn: string;
  deviceName: string;
  deviceType: string;
  isGnb: boolean;
  operateIp: string;
  softwareVersion: string;
  fileName: string;
  objectPath: string;
  bucket: string;
  fileSize: number;
  haltMainReason: string;
  haltDetailReason: string;
  runtimeBeforeReboot: number;
  recordStatus: RecordStatus;
  collectionFailReason: string;
  manualCollectionStatus: ManualCollectionStatus;
  isDeleted: boolean;
  collectedAt: string;
  createdAt: string;
  updatedAt: string;
}

export interface AbnormalRebootListParams {
  deviceId?: string;
  deviceSn?: string;
  recordStatus?: RecordStatus;
  deviceType?: string; // eNB / gNB
  startTime?: string; // RFC3339
  endTime?: string;
  page?: number;
  pageSize?: number;
}

export interface AbnormalRebootListResponse {
  items: AbnormalReboot[];
  total: number;
  page: number;
  size: number;
}

// ----------------------------------------------------------------------------
// Mappers
// ----------------------------------------------------------------------------

export function mapBackendAbnormalReboot(b: BackendAbnormalReboot): AbnormalReboot {
  return {
    id: b.id,
    deviceId: b.device_id,
    deviceSn: b.device_sn,
    deviceName: b.device_name ?? '',
    deviceType: b.device_type ?? '',
    isGnb: b.is_gnb,
    operateIp: b.operate_ip ?? '',
    softwareVersion: b.software_version ?? '',
    fileName: b.file_name ?? '',
    objectPath: b.object_path ?? '',
    bucket: b.bucket ?? '',
    fileSize: b.file_size,
    haltMainReason: b.fault_reason ?? '',
    haltDetailReason: b.fault_detail ?? '',
    runtimeBeforeReboot: b.runtime_before_reboot ?? 0,
    recordStatus: b.record_status,
    collectionFailReason: b.collection_fail_reason ?? '',
    manualCollectionStatus: b.manual_collection_status,
    isDeleted: b.is_deleted,
    collectedAt: b.collected_at,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

// ----------------------------------------------------------------------------
// API service
// ----------------------------------------------------------------------------

const BASE = '/device-abnormal-reboots';

export const deviceAbnormalRebootApi = {
  list(params: AbnormalRebootListParams): Promise<AbnormalRebootListResponse> {
    const query: Record<string, string | number> = {};
    if (params.deviceId) query.device_id = params.deviceId;
    if (params.deviceSn) query.device_sn = params.deviceSn;
    if (params.recordStatus) query.record_status = params.recordStatus;
    if (params.deviceType) query.device_type = params.deviceType;
    if (params.startTime) query.start_time = params.startTime;
    if (params.endTime) query.end_time = params.endTime;
    if (params.page) query.page = params.page;
    if (params.pageSize) query.page_size = params.pageSize;

    return http.get<BackendListResponse>(BASE, { params: query }).then((res) => ({
      items: (res.data.items ?? []).map(mapBackendAbnormalReboot),
      total: res.data.total,
      page: res.data.page,
      size: res.data.size,
    }));
  },

  getById(id: string): Promise<AbnormalReboot> {
    return http
      .get<BackendAbnormalReboot>(`${BASE}/${id}`)
      .then((res) => mapBackendAbnormalReboot(res.data));
  },

  getDownloadUrl(id: string): Promise<string> {
    return http.get<{ url: string }>(`${BASE}/${id}/download`).then((res) => res.data.url);
  },

  delete(id: string): Promise<void> {
    return http.delete(`${BASE}/${id}`).then(() => undefined);
  },
};
