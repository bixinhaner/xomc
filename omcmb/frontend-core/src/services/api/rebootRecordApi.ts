import http from '../http';

// ============================================================================
// 统一重启记录 API（event_logs ∪ station_fault_logs 的只读合并视图）
//
// 后端 /reboot-records 把"普通重启"(event_logs) 与"异常重启"(station_fault_logs)
// 两张互斥表 UNION ALL 合成一份重启记录，按 rebootType 过滤；不写入。
//   - 普通重启：event_logs，reason = event_reason，无 HaltReason / 运行时长
//   - 异常重启：station_fault_logs，reason = HaltMainReason，detailReason = HaltDetailReason
// ============================================================================

// 重启类型：normal 仅正常 / abnormal 仅异常 / all（或不传）全部
export type RebootType = 'all' | 'normal' | 'abnormal';

interface BackendRebootRecord {
  id: string;
  source: 'event' | 'fault';
  is_abnormal: boolean;
  device_sn: string;
  device_name?: string;
  device_type?: string;
  operate_ip?: string;
  software_version?: string;
  reason?: string;
  detail_reason?: string;
  runtime_before_reboot: number;
  reboot_time: string;
}

interface BackendListResponse {
  items: BackendRebootRecord[] | null;
  total: number;
  page: number;
  size: number;
}

export interface RebootRecord {
  id: string;
  source: 'event' | 'fault';
  isAbnormal: boolean;
  deviceSn: string;
  deviceName: string;
  deviceType: string;
  operateIp: string;
  softwareVersion: string;
  reason: string;
  detailReason: string;
  runtimeBeforeReboot: number;
  rebootTime: string;
}

export interface RebootRecordListParams {
  deviceSn?: string;
  deviceType?: string;
  rebootType?: RebootType;
  startTime?: string; // RFC3339
  endTime?: string;
  page?: number;
  pageSize?: number;
}

export interface RebootRecordListResponse {
  items: RebootRecord[];
  total: number;
  page: number;
  size: number;
}

function mapBackendRebootRecord(b: BackendRebootRecord): RebootRecord {
  return {
    id: b.id,
    source: b.source,
    isAbnormal: b.is_abnormal,
    deviceSn: b.device_sn,
    deviceName: b.device_name ?? '',
    deviceType: b.device_type ?? '',
    operateIp: b.operate_ip ?? '',
    softwareVersion: b.software_version ?? '',
    reason: b.reason ?? '',
    detailReason: b.detail_reason ?? '',
    runtimeBeforeReboot: b.runtime_before_reboot ?? 0,
    rebootTime: b.reboot_time,
  };
}

// ---- 按设备聚合统计：总次数 + 异常次数 ----

interface BackendDeviceRebootStat {
  device_sn: string;
  device_name?: string;
  device_type?: string;
  total_count: number;
  abnormal_count: number;
  latest_at?: string | null;
}

export interface DeviceRebootStat {
  deviceSn: string;
  deviceName: string;
  deviceType: string;
  totalCount: number;
  abnormalCount: number;
  latestAt: string;
}

export interface RebootRecordStatParams {
  deviceSn?: string;
  deviceType?: string;
  rebootType?: RebootType;
  startTime?: string;
  endTime?: string;
}

export interface RebootRecordStatResponse {
  items: DeviceRebootStat[];
  total: number;
}

function mapBackendDeviceRebootStat(b: BackendDeviceRebootStat): DeviceRebootStat {
  return {
    deviceSn: b.device_sn,
    deviceName: b.device_name ?? '',
    deviceType: b.device_type ?? '',
    totalCount: b.total_count,
    abnormalCount: b.abnormal_count,
    latestAt: b.latest_at ?? '',
  };
}

const BASE = '/reboot-records';

// 只有 normal/abnormal 才下发 reboot_type；all 省略（后端默认全部）
function applyCommonQuery(
  query: Record<string, string | number>,
  params: { deviceSn?: string; deviceType?: string; rebootType?: RebootType; startTime?: string; endTime?: string },
): void {
  if (params.deviceSn) query.device_sn = params.deviceSn;
  if (params.deviceType) query.device_type = params.deviceType;
  if (params.rebootType && params.rebootType !== 'all') query.reboot_type = params.rebootType;
  if (params.startTime) query.start_time = params.startTime;
  if (params.endTime) query.end_time = params.endTime;
}

export const rebootRecordApi = {
  list(params: RebootRecordListParams): Promise<RebootRecordListResponse> {
    const query: Record<string, string | number> = {};
    applyCommonQuery(query, params);
    if (params.page) query.page = params.page;
    if (params.pageSize) query.page_size = params.pageSize;

    return http.get<BackendListResponse>(BASE, { params: query }).then((res) => ({
      items: (res.data.items ?? []).map(mapBackendRebootRecord),
      total: res.data.total,
      page: res.data.page,
      size: res.data.size,
    }));
  },

  statByDevice(params: RebootRecordStatParams): Promise<RebootRecordStatResponse> {
    const query: Record<string, string | number> = {};
    applyCommonQuery(query, params);

    return http
      .get<{ items: BackendDeviceRebootStat[] | null; total: number }>(`${BASE}/statistics`, {
        params: query,
      })
      .then((res) => ({
        items: (res.data.items ?? []).map(mapBackendDeviceRebootStat),
        total: res.data.total,
      }));
  },
};
