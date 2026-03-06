import http from '../http';
import type { PageRequest, PageResponse } from '@/types/pagination';
import type { SystemLog, NEMessageLog } from '@/mock/data/logs';

// --- Backend response types ---

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

interface BackendSystemLog {
  id: string;
  level: string; // INFO, WARN, ERROR, DEBUG
  source: string;
  message: string;
  details?: string;
  timestamp: string;
}

interface BackendNEMessageLog {
  id: string;
  device_sn: string;
  device_id?: string;
  device_name: string;
  message_type: string; // notification, alarm, heartbeat, config_response, perf_data
  direction: string; // northbound, southbound
  protocol: string; // NETCONF, SNMP, TR-069, REST
  content: string;
  timestamp: string;
  success: boolean;
}

// --- Mapping functions ---

function mapBackendSystemLog(b: BackendSystemLog): SystemLog {
  return {
    id: b.id,
    level: b.level as SystemLog['level'],
    source: b.source,
    message: b.message,
    details: b.details,
    timestamp: b.timestamp,
  };
}

function mapBackendNEMessageLog(b: BackendNEMessageLog): NEMessageLog {
  return {
    id: b.id,
    deviceSn: b.device_sn,
    deviceName: b.device_name,
    messageType: b.message_type as NEMessageLog['messageType'],
    direction: b.direction as NEMessageLog['direction'],
    protocol: b.protocol as NEMessageLog['protocol'],
    content: b.content,
    timestamp: b.timestamp,
    success: b.success,
  };
}

// --- Exported service ---

export const logApi = {
  async getSystemLogs(
    params: { level?: SystemLog['level']; source?: string; keyword?: string } & PageRequest
  ): Promise<PageResponse<SystemLog>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.level) query.level = params.level;
    if (params.source) query.source = params.source;
    if (params.keyword) query.keyword = params.keyword;

    const { data } = await http.get<BackendListResponse<BackendSystemLog>>(
      '/logs/system',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendSystemLog),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getNEMessageLogs(
    params: { deviceSn?: string; messageType?: string } & PageRequest
  ): Promise<PageResponse<NEMessageLog>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.deviceSn) query.device_sn = params.deviceSn;
    if (params.messageType) query.message_type = params.messageType;

    const { data } = await http.get<BackendListResponse<BackendNEMessageLog>>(
      '/logs/ne-messages',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendNEMessageLog),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },
};
