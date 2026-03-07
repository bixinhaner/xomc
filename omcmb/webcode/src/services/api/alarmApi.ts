import http from '../http';
import type { Alarm, AlarmRule, AlarmRuleCondition, AlarmRuleAction, AlarmFilter, AlarmCount } from '@/types/alarm';
import type { AlarmSeverity } from '@/types/common';
import type { PageRequest, PageResponse } from '@/types/pagination';

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

// Backend alarm rule model
interface BackendAlarmRule {
  id: string;
  rule_name: string;
  rule_type: string;
  severity: number; // 1=Critical, 2=Major, 3=Minor, 4=Warning
  enabled: boolean;
  conditions: Array<{
    field: string;
    operator: string;
    value: string | number | boolean;
  }>;
  actions: Array<{
    type: string;
    target?: string;
    template?: string;
    params?: Record<string, string>;
  }>;
  created_at: string;
  updated_at: string;
}

function mapBackendAlarmRule(br: BackendAlarmRule): AlarmRule {
  return {
    id: br.id,
    ruleName: br.rule_name,
    ruleType: br.rule_type,
    severity: severityNumToStr[br.severity] || 'warning',
    enabled: br.enabled,
    conditions: (br.conditions || []).map((c) => ({
      field: c.field,
      operator: c.operator as AlarmRuleCondition['operator'],
      value: c.value,
    })),
    actions: (br.actions || []).map((a) => ({
      type: a.type as AlarmRuleAction['type'],
      target: a.target,
      template: a.template,
      params: a.params,
    })),
    createTime: br.created_at,
    updateTime: br.updated_at,
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

  // Alarm rules CRUD
  async getRules(params: PageRequest): Promise<PageResponse<AlarmRule>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    const { data } = await http.get<BackendListResponse<BackendAlarmRule>>(
      '/alarms/rules',
      { params: query }
    );
    return {
      items: (data.items || []).map(mapBackendAlarmRule),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async createRule(
    data: Omit<AlarmRule, 'id' | 'createTime' | 'updateTime'>
  ): Promise<AlarmRule> {
    const payload = {
      rule_name: data.ruleName,
      rule_type: data.ruleType,
      severity: Number(
        Object.entries(severityNumToStr).find(
          ([, v]) => v === data.severity
        )?.[0] ?? 4
      ),
      enabled: data.enabled,
      conditions: data.conditions.map((c) => ({
        field: c.field,
        operator: c.operator,
        value: c.value,
      })),
      actions: data.actions.map((a) => ({
        type: a.type,
        target: a.target,
        template: a.template,
        params: a.params,
      })),
    };
    const { data: created } = await http.post<BackendAlarmRule>(
      '/alarms/rules',
      payload
    );
    return mapBackendAlarmRule(created);
  },

  async updateRule(id: string, data: Partial<AlarmRule>): Promise<AlarmRule> {
    const payload: Record<string, unknown> = {};
    if (data.ruleName !== undefined) payload.rule_name = data.ruleName;
    if (data.ruleType !== undefined) payload.rule_type = data.ruleType;
    if (data.severity !== undefined) {
      payload.severity = Number(
        Object.entries(severityNumToStr).find(
          ([, v]) => v === data.severity
        )?.[0] ?? 4
      );
    }
    if (data.enabled !== undefined) payload.enabled = data.enabled;
    if (data.conditions !== undefined) {
      payload.conditions = data.conditions.map((c) => ({
        field: c.field,
        operator: c.operator,
        value: c.value,
      }));
    }
    if (data.actions !== undefined) {
      payload.actions = data.actions.map((a) => ({
        type: a.type,
        target: a.target,
        template: a.template,
        params: a.params,
      }));
    }
    const { data: updated } = await http.put<BackendAlarmRule>(
      `/alarms/rules/${id}`,
      payload
    );
    return mapBackendAlarmRule(updated);
  },

  async deleteRules(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.delete(`/alarms/rules/${id}`);
    }
  },
};
