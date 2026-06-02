import http from '../http';

// ============================================================================
// 事件日志 API（设备活动审计流水 event_logs 表）
//
// 与 deviceAbnormalRebootApi 分开：
//   - deviceAbnormalRebootApi：异常重启 / station_fault_logs / 带文件管理
//   - eventLogApi：通用事件审计 / event_logs / 本期只接 1 BOOT 普通重启
// ============================================================================

export type EventType = 'boot' | string; // 留扩展余地
export type EventLevel = 'info' | 'warning' | 'error' | 'success';

interface BackendEventLog {
  id: string;
  device_id: string | null;
  device_sn: string;
  device_name?: string;
  device_type?: string;
  is_gnb: boolean;
  operate_ip?: string;
  software_version?: string;
  event_type: EventType;
  event_reason?: string;
  event_level: EventLevel;
  event_data?: unknown;
  occurred_at: string;
  created_at: string;
}

interface BackendListResponse {
  items: BackendEventLog[] | null;
  total: number;
  page: number;
  size: number;
}

export interface EventLog {
  id: string;
  deviceId: string | null;
  deviceSn: string;
  deviceName: string;
  deviceType: string;
  isGnb: boolean;
  operateIp: string;
  softwareVersion: string;
  eventType: EventType;
  eventReason: string;
  eventLevel: EventLevel;
  eventData?: unknown;
  occurredAt: string;
  createdAt: string;
}

export interface EventLogListParams {
  deviceSn?: string;
  eventType?: EventType;
  startTime?: string; // RFC3339
  endTime?: string;
  page?: number;
  pageSize?: number;
}

export interface EventLogListResponse {
  items: EventLog[];
  total: number;
  page: number;
  size: number;
}

// ---- 按设备聚合的重启次数统计 ----

interface BackendDeviceRebootStat {
  device_sn: string;
  device_name?: string;
  reboot_count: number;
  latest_at?: string | null;
}

export interface DeviceRebootStat {
  deviceSn: string;
  deviceName: string;
  rebootCount: number;
  latestAt: string;
}

// 统计跟随当前筛选条件，但不分页（后端一次返回全部设备聚合行）
export interface EventLogStatParams {
  deviceSn?: string;
  eventType?: EventType;
  startTime?: string; // RFC3339
  endTime?: string;
}

export interface EventLogStatResponse {
  items: DeviceRebootStat[];
  total: number;
}

export function mapBackendDeviceRebootStat(b: BackendDeviceRebootStat): DeviceRebootStat {
  return {
    deviceSn: b.device_sn,
    deviceName: b.device_name ?? '',
    rebootCount: b.reboot_count,
    latestAt: b.latest_at ?? '',
  };
}

export function mapBackendEventLog(b: BackendEventLog): EventLog {
  return {
    id: b.id,
    deviceId: b.device_id,
    deviceSn: b.device_sn,
    deviceName: b.device_name ?? '',
    deviceType: b.device_type ?? '',
    isGnb: b.is_gnb,
    operateIp: b.operate_ip ?? '',
    softwareVersion: b.software_version ?? '',
    eventType: b.event_type,
    eventReason: b.event_reason ?? '',
    eventLevel: b.event_level,
    eventData: b.event_data,
    occurredAt: b.occurred_at,
    createdAt: b.created_at,
  };
}

const BASE = '/event-logs';

export const eventLogApi = {
  list(params: EventLogListParams): Promise<EventLogListResponse> {
    const query: Record<string, string | number> = {};
    if (params.deviceSn) query.device_sn = params.deviceSn;
    if (params.eventType) query.event_type = params.eventType;
    if (params.startTime) query.start_time = params.startTime;
    if (params.endTime) query.end_time = params.endTime;
    if (params.page) query.page = params.page;
    if (params.pageSize) query.page_size = params.pageSize;

    return http.get<BackendListResponse>(BASE, { params: query }).then((res) => ({
      items: (res.data.items ?? []).map(mapBackendEventLog),
      total: res.data.total,
      page: res.data.page,
      size: res.data.size,
    }));
  },

  getById(id: string): Promise<EventLog> {
    return http
      .get<BackendEventLog>(`${BASE}/${id}`)
      .then((res) => mapBackendEventLog(res.data));
  },

  statByDevice(params: EventLogStatParams): Promise<EventLogStatResponse> {
    const query: Record<string, string | number> = {};
    if (params.deviceSn) query.device_sn = params.deviceSn;
    if (params.eventType) query.event_type = params.eventType;
    if (params.startTime) query.start_time = params.startTime;
    if (params.endTime) query.end_time = params.endTime;

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
