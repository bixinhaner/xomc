import http from '../http';
import type { Alarm, AlarmRule, AlarmFilter, AlarmCount } from '@/types/alarm';
import type { AlarmSeverity } from '@/types/common';
import type { PageRequest, PageResponse } from '@/types/pagination';
import { alarmService } from '@/mock/services/alarmService';

// Backend alarm model
interface BackendAlarm {
  id: string;
  device_id: string;
  device_sn: string;
  carrier: string;
  severity: number; // 1=Critical, 2=Major, 3=Minor, 4=Warning
  alarm_type: string;
  alarm_code: string;
  description: string;
  status: string; // active, acknowledged, cleared
  raised_at: string;
  acknowledged_at?: string;
  cleared_at?: string;
  acknowledged_by?: string;
  additional_info?: Record<string, string>;
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

interface BackendAlarmStatistics {
  total_active: number;
  by_severity: Record<string, number>;
  by_type: Record<string, number>;
}

// Severity mapping: backend number ↔ frontend string
const severityNumToStr: Record<number, AlarmSeverity> = {
  1: 'critical',
  2: 'major',
  3: 'minor',
  4: 'warning',
};

const severityStrToNum: Record<string, string> = {
  critical: '1',
  major: '2',
  minor: '3',
  warning: '4',
};

function mapBackendAlarm(ba: BackendAlarm): Alarm {
  return {
    id: ba.id,
    alarmCode: ba.alarm_code,
    alarmName: ba.alarm_type || ba.alarm_code,
    severity: severityNumToStr[ba.severity] || 'warning',
    deviceSn: ba.device_sn,
    deviceName: ba.device_sn, // Backend doesn't include device name; use SN
    neType: '',
    alarmContent: ba.description,
    alarmTime: ba.raised_at,
    clearTime: ba.cleared_at,
    duration: ba.cleared_at
      ? Math.floor(
          (new Date(ba.cleared_at).getTime() - new Date(ba.raised_at).getTime()) / 60000
        )
      : undefined,
    ackStatus: ba.acknowledged_at ? 'acknowledged' : 'unacknowledged',
    ackUser: ba.acknowledged_by,
    ackTime: ba.acknowledged_at,
    alarmSource: ba.carrier,
    alarmLocation: ba.additional_info?.location,
    alarmType: ba.alarm_type,
    isActive: ba.status === 'active' || ba.status === 'acknowledged',
  };
}

function mapListResponse(resp: BackendListResponse<BackendAlarm>): PageResponse<Alarm> {
  return {
    items: (resp.items || []).map(mapBackendAlarm),
    total: resp.total,
    page: resp.page,
    pageSize: resp.page_size,
  };
}

function buildAlarmQuery(
  filter: AlarmFilter,
  pagination: PageRequest
): Record<string, unknown> {
  const query: Record<string, unknown> = {
    page: pagination.page,
    pageSize: pagination.pageSize,
    sortField: pagination.sortField,
    sortOrder: pagination.sortOrder,
  };

  if (filter.severity) query.severity = severityStrToNum[filter.severity];
  if (filter.deviceSn) query.device_sn = filter.deviceSn;
  if (filter.timeRange) {
    query.start_time = filter.timeRange[0];
    query.end_time = filter.timeRange[1];
  }

  return query;
}

export const alarmApi = {
  async getCurrentAlarms(
    params: AlarmFilter & PageRequest
  ): Promise<PageResponse<Alarm>> {
    const query = buildAlarmQuery(params, params);
    const { data } = await http.get<BackendListResponse<BackendAlarm>>(
      '/alarms/active',
      { params: query }
    );
    return mapListResponse(data);
  },

  async getHistoricalAlarms(
    params: AlarmFilter & PageRequest
  ): Promise<PageResponse<Alarm>> {
    const query = buildAlarmQuery(params, params);
    const { data } = await http.get<BackendListResponse<BackendAlarm>>(
      '/alarms/history',
      { params: query }
    );
    return mapListResponse(data);
  },

  async getList(
    params: AlarmFilter & PageRequest & { isActive?: boolean }
  ): Promise<PageResponse<Alarm>> {
    if (params.isActive === false) {
      return alarmApi.getHistoricalAlarms(params);
    }
    return alarmApi.getCurrentAlarms(params);
  },

  async getById(id: string): Promise<Alarm | null> {
    try {
      const { data } = await http.get<BackendAlarm>(`/alarms/${id}`);
      return mapBackendAlarm(data);
    } catch {
      return null;
    }
  },

  async acknowledgeAlarms(ids: string[], note?: string): Promise<void> {
    // Backend only supports single alarm acknowledge; loop through
    for (const id of ids) {
      await http.post(`/alarms/${id}/acknowledge`, {
        acknowledged_by: note || 'operator',
      });
    }
  },

  async clearAlarms(ids: string[]): Promise<void> {
    // Backend only supports single alarm clear; loop through
    for (const id of ids) {
      await http.post(`/alarms/${id}/clear`);
    }
  },

  async getAlarmCount(): Promise<AlarmCount> {
    const { data } = await http.get<BackendAlarmStatistics>(
      '/alarms/statistics'
    );
    const bySeverity = data.by_severity || {};
    return {
      critical: bySeverity['1'] || 0,
      major: bySeverity['2'] || 0,
      minor: bySeverity['3'] || 0,
      warning: bySeverity['4'] || 0,
    };
  },

  // Alarm rules — not yet implemented in backend (Sprint 4), delegate to mock
  getRules: alarmService.getRules.bind(alarmService),
  createRule: alarmService.createRule.bind(alarmService),
  updateRule: alarmService.updateRule.bind(alarmService),
  deleteRules: alarmService.deleteRules.bind(alarmService),
};
