import http from '../http';
import type {
  Alarm,
  AlarmRule,
  AlarmRuleCondition,
  AlarmRuleAction,
  AlarmFilter,
  AlarmCount,
} from '../../types/alarm';
import type { PageRequest, PageResponse } from '../../types/pagination';

// ---------------------------------------------------------------------------
// Backend alarm model
// ---------------------------------------------------------------------------

interface BackendAlarm {
  id: string;
  device_id: string;
  device_sn: string;
  carrier: string;
  severity: number; // 1..4 or legacy 31001..31004
  alarm_type: string;
  alarm_identifier: string;
  description: string;
  status: string; // active, acknowledged, cleared
  raised_at: string;
  acknowledged_at?: string;
  acknowledged_by?: string;
  ack_note?: string;
  cleared_at?: string;
  cleared_by?: string;
  clear_note?: string;
  device_name?: string;
  technology?: string;
  alarm_source?: string;
  event_type?: string;
  is_read?: boolean;
  ack_count?: number;
  first_raised_at?: string;
  last_updated_at?: string;
  probable_cause?: string;
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
  unacknowledged?: number;
  unread?: number;
  state_version?: number;
  by_severity: Record<string, number>;
  by_type: Record<string, number>;
}

interface BackendAlarmSyncTriggerResponse {
  device_sn?: string;
  deviceSn?: string;
  task_id?: string;
  taskId?: string;
}

export interface AlarmSyncTriggerResult {
  deviceSn: string;
  taskId?: string;
}

// ---------------------------------------------------------------------------
// T-0098-P5-06：旧 alarm_libraries 接口已下线，改由 alarmDefinitionApi（P4-01）
// 提供 /alarms/alarm-definitions CRUD（super_admin 治理）。
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Backend alarm filter rule model
// ---------------------------------------------------------------------------

interface BackendAlarmRule {
  id: string;
  name: string;
  filter_type: string; // alarm_identifier | alarm_source | device | device_group
  alarm_sources: string[];
  alarm_identifiers: string[];
  device_ids: string[];
  device_group_ids: string[];
  action: string; // default | ignore | auto_acknowledge | auto_clear
  acknowledge_desc: string;
  priority: number;
  enabled: boolean;
  created_by?: string;
  created_at: string;
  updated_by?: string;
  updated_at: string;
}

export interface AlarmEmailGlobalSetting {
  enabled: boolean;
  default_recipients: string[];
  updated_at?: string;
}

export interface AlarmEmailSubscription {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  interval_minutes: 0 | 10 | 30 | 60;
  tolerance_minutes: 0 | 10 | 30 | 60;
  recipients: string[];
  include_default_recipients: boolean;
  alarm_identifiers: string[];
  severities: number[];
  alarm_sources: string[];
  event_types: string[];
  device_ids: string[];
  device_group_ids: string[];
  created_at?: string;
  updated_at?: string;
}

export type AlarmEmailSubscriptionInput = Omit<
  AlarmEmailSubscription,
  'id' | 'created_at' | 'updated_at'
>;

// ---------------------------------------------------------------------------
// Severity helpers
// ---------------------------------------------------------------------------

const severityNumToStr: Record<number, string> = {
  1: 'critical',
  31001: 'critical',
  2: 'major',
  31002: 'major',
  3: 'minor',
  31003: 'minor',
  4: 'warning',
  31004: 'warning',
};

const severityStrToNum: Record<string, string> = {
  critical: '1',
  major: '2',
  minor: '3',
  warning: '4',
};

// ---------------------------------------------------------------------------
// Event type normalization: TR-069 raw values → frontend enum
// ---------------------------------------------------------------------------

const EVENT_TYPE_MAP: Record<string, Alarm['eventType']> = {
  'communicationsalarm': 'communication',
  'communications alarm': 'communication',
  'qualityofservicealarm': 'qualityOfService',
  'quality of service alarm': 'qualityOfService',
  'processingerroralarm': 'processingError',
  'processing error alarm': 'processingError',
  'equipmentalarm': 'device',
  'equipment alarm': 'device',
  'environmentalarm': 'environment',
  'environment alarm': 'environment',
  'servicealarm': 'performance',
  'service alarm': 'performance',
};

function normalizeEventType(raw: string | undefined): Alarm['eventType'] {
  if (!raw) return 'communication';
  const key = raw.toLowerCase().replace(/[\s-]/g, '');
  return EVENT_TYPE_MAP[key] || 'communication';
}

function hasBusinessTimestamp(value: string | undefined): value is string {
  if (!value) return false;
  const normalized = value.trim();
  if (!normalized || normalized.startsWith('0001-01-01')) {
    return false;
  }
  const parsed = Date.parse(normalized);
  if (Number.isNaN(parsed)) {
    return true;
  }
  return new Date(parsed).getUTCFullYear() > 1;
}

function resolveUpdatedAt(ba: BackendAlarm): string {
  if (hasBusinessTimestamp(ba.last_updated_at)) {
    return ba.last_updated_at;
  }
  if (hasBusinessTimestamp(ba.updated_at)) {
    return ba.updated_at;
  }
  return ba.raised_at;
}

function resolveEventTime(ba: BackendAlarm): string {
  if (hasBusinessTimestamp(ba.first_raised_at)) {
    return ba.first_raised_at;
  }
  return ba.raised_at;
}

// ---------------------------------------------------------------------------
// Mappers
// ---------------------------------------------------------------------------

function mapBackendAlarm(ba: BackendAlarm): Alarm {
  // Derive dealState from status + ack/clear state
  // 0=未确认未清除, 1=已确认未清除, 2=未确认已清除, 3=已确认已清除
  const isAcked = !!ba.acknowledged_at;
  const isCleared = !!ba.cleared_at;
  let dealState: Alarm['dealState'];
  if (isCleared) {
    dealState = isAcked ? '3' : '2';
  } else if (isAcked) {
    dealState = '1';
  } else {
    dealState = '0';
  }

  const eventTime = resolveEventTime(ba);

  return {
    id: ba.id,
    alarmIdentifier: ba.alarm_identifier,
    alarmName: ba.probable_cause || '',
    specificProblem: ba.description || ba.probable_cause || '',
    severity: (severityNumToStr[ba.severity] || 'warning') as Alarm['severity'],
    deviceSn: ba.device_sn,
    deviceName: ba.device_name || ba.device_sn,
    description: ba.description,
    neType: ba.technology || '',
    equipInfo: ba.device_name ? `${ba.device_name}(${ba.device_sn})` : ba.device_sn,
    eventType: normalizeEventType(ba.event_type || ba.alarm_type),
    dealState,
    eventTime,
    updTime: resolveUpdatedAt(ba),
    dealUser: ba.acknowledged_by,
    dealTime: ba.acknowledged_at,
    clearTime: ba.cleared_at,
    clearUser: ba.cleared_by,
    clearMemo: ba.clear_note,
    dealMemo: ba.ack_note || undefined,
    alarmType: ba.status === 'cleared' ? 'history' : 'active',
    alarmCount: ba.ack_count || 1,
    unread: ba.is_read ? '0' : '1',
    duration: ba.cleared_at && eventTime
      ? Math.floor(
          (new Date(ba.cleared_at).getTime() -
            new Date(eventTime).getTime()) /
            60000
        )
      : ba.acknowledged_at
        ? Math.floor(
            (new Date(ba.acknowledged_at).getTime() -
              new Date(eventTime).getTime()) /
              60000
          )
        : undefined,
    alarmSource: ba.alarm_source || ba.carrier,
    technology: ba.technology || '',
    isActive: ba.status === 'active' || ba.status === 'acknowledged',
    // 后端直传字段
    probableCause: ba.probable_cause,
    additionalInfo: ba.additional_info,
    additionalText: ba.additional_info?.additional_text,
    acknowledgedAt: ba.acknowledged_at,
    acknowledgedBy: ba.acknowledged_by,
    clearedAt: ba.cleared_at,
    isRead: ba.is_read,
  };
}

function mapBackendAlarmRule(br: BackendAlarmRule): AlarmRule {
  // Derive frontend conditions from backend filter fields
  const conditions: AlarmRuleCondition[] = [];

  if (br.alarm_identifiers.length > 0) {
    conditions.push({
      field: 'alarm_identifier',
      operator: 'contains',
      value: br.alarm_identifiers,
    });
  }

  if (br.alarm_sources.length > 0) {
    conditions.push({
      field: 'alarm_source',
      operator: 'contains',
      value: br.alarm_sources,
    });
  }

  if (br.device_ids.length > 0) {
    conditions.push({
      field: 'device_id',
      operator: 'contains',
      value: br.device_ids,
    });
  }

  if (br.device_group_ids.length > 0) {
    conditions.push({
      field: 'device_group_id',
      operator: 'contains',
      value: br.device_group_ids,
    });
  }

  // Derive frontend actions from backend action field
  const actions: AlarmRuleAction[] = [];
  if (br.action === 'auto_acknowledge' || br.action === 'auto_clear') {
    actions.push({
      type: br.action === 'auto_acknowledge' ? 'suppress' : 'suppress',
      target: br.action,
      template: br.acknowledge_desc || undefined,
    });
  } else if (br.action === 'ignore') {
    actions.push({
      type: 'suppress',
      target: 'ignore',
    });
  }

  return {
    id: br.id,
    ruleName: br.name,
    ruleType: br.action,
    severity: 'warning' as Alarm['severity'],
    enabled: br.enabled,
    userCode: br.updated_by || br.created_by || undefined,
    conditions,
    actions,
    createTime: br.created_at,
    updateTime: br.updated_at,
  };
}

function mapListResponse(
  resp: BackendListResponse<BackendAlarm>
): PageResponse<Alarm> {
  return {
    items: (resp.items || []).map(mapBackendAlarm),
    total: resp.total,
    page: resp.page,
    pageSize: resp.page_size,
  };
}

function mapRuleListResponse(
  resp: BackendListResponse<BackendAlarmRule>
): PageResponse<AlarmRule> {
  return {
    items: (resp.items || []).map(mapBackendAlarmRule),
    total: resp.total,
    page: resp.page,
    pageSize: resp.page_size,
  };
}

// ---------------------------------------------------------------------------
// Helpers to convert frontend AlarmRule → backend payload
// ---------------------------------------------------------------------------

function ruleToBackendPayload(
  data: Partial<AlarmRule>
): Record<string, unknown> {
  const payload: Record<string, unknown> = {};
  const hasConditions = data.conditions !== undefined;

  if (data.ruleName !== undefined) payload.name = data.ruleName;
  if (data.enabled !== undefined) payload.enabled = data.enabled;
  if (data.ruleType !== undefined) payload.action = data.ruleType;

  // Extract conditions into backend fields
  const conditions = data.conditions || [];
  const alarmIdentifiers = conditions
    .filter((c) => c.field === 'alarm_identifier')
    .flatMap((c) => (Array.isArray(c.value) ? c.value : [c.value]));
  const alarmSources = conditions
    .filter((c) => c.field === 'alarm_source')
    .flatMap((c) => (Array.isArray(c.value) ? c.value : [c.value]));
  const deviceIds = conditions
    .filter((c) => c.field === 'device_id')
    .flatMap((c) => (Array.isArray(c.value) ? c.value : [c.value]));
  const deviceGroupIds = conditions
    .filter((c) => c.field === 'device_group_id')
    .flatMap((c) => (Array.isArray(c.value) ? c.value : [c.value]));

  if (hasConditions) {
    if (alarmIdentifiers.length > 0) payload.alarm_identifiers = alarmIdentifiers;
    if (alarmSources.length > 0) payload.alarm_sources = alarmSources;
    if (deviceIds.length > 0) payload.device_ids = deviceIds;
    if (deviceGroupIds.length > 0) payload.device_group_ids = deviceGroupIds;
  }

  // Extract acknowledge_desc from actions
  const actions = data.actions || [];
  const suppressAction = actions.find((a) => a.type === 'suppress');
  if (suppressAction?.template) {
    payload.acknowledge_desc = suppressAction.template;
  }

  // Determine filter_type from conditions
  if (hasConditions) {
    if (alarmIdentifiers.length > 0) {
      payload.filter_type = 'alarm_identifier';
    } else if (alarmSources.length > 0) {
      payload.filter_type = 'alarm_source';
    } else if (deviceGroupIds.length > 0) {
      payload.filter_type = 'device_group';
    } else if (deviceIds.length > 0) {
      payload.filter_type = 'device';
    } else {
      payload.filter_type = 'alarm_identifier';
    }
  }

  return payload;
}

// ---------------------------------------------------------------------------
// Query builder
// ---------------------------------------------------------------------------

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

  if (filter.severity) {
    const severities = (Array.isArray(filter.severity) ? filter.severity : [filter.severity])
      .map((severity) => severityStrToNum[severity])
      .filter((severity): severity is string => Boolean(severity));
    if (severities.length > 0) {
      query.severity = severities.join(',');
    }
  }
  if (filter.eventType) {
    const eventType = Array.isArray(filter.eventType) ? filter.eventType[0] : filter.eventType;
    if (eventType) query.event_type = eventType;
  }
  if (filter.dealState) {
    const dealState = Array.isArray(filter.dealState) ? filter.dealState[0] : filter.dealState;
    if (dealState === '0') {
      query.status = 'active';
    } else if (dealState === '1') {
      query.status = 'acknowledged';
    }
  }
  if (filter.unread !== undefined) {
    query.is_read = filter.unread === '0' ? 'true' : 'false';
  }
  if (filter.neType) query.ne_type = filter.neType;
  if (filter.deviceSn) query.device_sn = filter.deviceSn;
  if (filter.timeRange) {
    query.start_time = filter.timeRange[0];
    query.end_time = filter.timeRange[1];
  }
  if (filter.alarmIdentifier) {
    query.keyword = filter.alarmIdentifier;
  } else if (filter.keyword) {
    query.keyword = filter.keyword;
  }
  // 未识别告警过滤（设计 §3.3 治理闭环）：'true'/'false' 字符串透传给后端 form:"is_unknown"
  if (filter.isUnknown !== undefined) {
    query.is_unknown = filter.isUnknown;
  }

  return query;
}

// ---------------------------------------------------------------------------
// API service
// ---------------------------------------------------------------------------

export const alarmApi = {
  // -- Active alarms --------------------------------------------------------

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

  // -- Batch operations -----------------------------------------------------

  async acknowledgeAlarms(ids: string[], note?: string): Promise<void> {
    await http.post('/alarms/active/batch/acknowledge', {
      ids,
      acknowledged_by: note || 'operator',
    });
  },

  async unacknowledgeAlarms(ids: string[]): Promise<void> {
    await http.post('/alarms/active/batch/unacknowledge', { ids });
  },

  async acknowledgeHistoryAlarms(ids: string[], note?: string): Promise<void> {
    await http.post('/alarms/history/batch/acknowledge', {
      ids,
      acknowledged_by: note || 'operator',
    });
  },

  async unacknowledgeHistoryAlarms(ids: string[]): Promise<void> {
    await http.post('/alarms/history/batch/unacknowledge', { ids });
  },

  async deleteHistoryAlarms(ids: string[]): Promise<void> {
    await http.post('/alarms/history/batch/delete', { ids });
  },

  async clearAlarms(ids: string[], note?: string): Promise<void> {
    await http.post('/alarms/active/batch/clear', { ids, clear_note: note });
  },

  async markAlarmRead(id: string): Promise<void> {
    await http.post(`/alarms/active/${id}/read`);
  },

  // -- Statistics -----------------------------------------------------------

  async getAlarmCount(): Promise<AlarmCount> {
    const { data } = await http.get<BackendAlarmStatistics>(
      '/alarms/statistics'
    );
    const bySeverity = data.by_severity || {};
    return {
      total_active: data.total_active || 0,
      unacknowledged: data.unacknowledged || 0,
      unread: data.unread || 0,
      stateVersion: data.state_version || 0,
      critical: bySeverity['1'] || 0,
      major: bySeverity['2'] || 0,
      minor: bySeverity['3'] || 0,
      warning: bySeverity['4'] || 0,
    };
  },

  async getHistoryAlarmCount(): Promise<AlarmCount> {
    const { data } = await http.get<BackendAlarmStatistics>(
      '/alarms/history/statistics'
    );
    const bySeverity = data.by_severity || {};
    return {
      total_active: data.total_active || 0,
      unacknowledged: 0,
      unread: 0,
      stateVersion: data.state_version || 0,
      critical: bySeverity['1'] || 0,
      major: bySeverity['2'] || 0,
      minor: bySeverity['3'] || 0,
      warning: bySeverity['4'] || 0,
    };
  },

  // -- Alarm filter rules ---------------------------------------------------

  async getRules(
    params: PageRequest & { keyword?: string; filterType?: string[]; action?: string; enabled?: string }
  ): Promise<PageResponse<AlarmRule>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.keyword) {
      query.keyword = params.keyword;
    }
      if (params.filterType && params.filterType.length > 0) {
        query.filter_type = params.filterType.join(',');
    }
    if (params.action) {
      query.action = params.action;
    }
    if (params.enabled) {
      query.enabled = params.enabled;
    }
    const { data } = await http.get<BackendListResponse<BackendAlarmRule>>(
      '/alarms/alarm-filters',
      { params: query }
    );
    return mapRuleListResponse(data);
  },

  async createRule(
    data: Omit<AlarmRule, 'id' | 'createTime' | 'updateTime'>
  ): Promise<AlarmRule> {
    const payload = ruleToBackendPayload(data);
    const { data: created } = await http.post<BackendAlarmRule>(
      '/alarms/alarm-filters',
      payload
    );
    return mapBackendAlarmRule(created);
  },

  async updateRule(id: string, data: Partial<AlarmRule>): Promise<AlarmRule> {
    const payload = ruleToBackendPayload(data);
    const { data: updated } = await http.put<BackendAlarmRule>(
      `/alarms/alarm-filters/${id}`,
      payload
    );
    return mapBackendAlarmRule(updated);
  },

  async deleteRules(ids: string[]): Promise<void> {
    await Promise.all(
      ids.map((id) => http.delete(`/alarms/alarm-filters/${id}`))
    );
  },

  async toggleRule(id: string): Promise<AlarmRule> {
    const { data: updated } = await http.post<BackendAlarmRule>(
      `/alarms/alarm-filters/${id}/toggle`
    );
    return mapBackendAlarmRule(updated);
  },

  async getAlarmEmailSetting(): Promise<AlarmEmailGlobalSetting> {
    const { data } = await http.get<AlarmEmailGlobalSetting>('/alarms/email-settings');
    return data;
  },

  async updateAlarmEmailSetting(setting: AlarmEmailGlobalSetting): Promise<AlarmEmailGlobalSetting> {
    const { data } = await http.put<AlarmEmailGlobalSetting>('/alarms/email-settings', setting);
    return data;
  },

  async getAlarmEmailSubscriptions(): Promise<AlarmEmailSubscription[]> {
    const { data } = await http.get<{ items: AlarmEmailSubscription[] }>('/alarms/email-subscriptions');
    return data.items ?? [];
  },

  async createAlarmEmailSubscription(input: AlarmEmailSubscriptionInput): Promise<AlarmEmailSubscription> {
    const { data } = await http.post<AlarmEmailSubscription>('/alarms/email-subscriptions', input);
    return data;
  },

  async updateAlarmEmailSubscription(id: string, input: AlarmEmailSubscriptionInput): Promise<AlarmEmailSubscription> {
    const { data } = await http.put<AlarmEmailSubscription>(`/alarms/email-subscriptions/${id}`, input);
    return data;
  },

  async deleteAlarmEmailSubscription(id: string): Promise<void> {
    await http.delete(`/alarms/email-subscriptions/${id}`);
  },

  // T-0098-P5-06：旧 /alarms/alarm-libraries 接口已下线，治理走 alarmDefinitionApi（/alarms/alarm-definitions）。

  // -- Alarm sync ----------------------------------------------------------

  async triggerAlarmSync(deviceSN: string): Promise<AlarmSyncTriggerResult> {
    const { data } = await http.post<BackendAlarmSyncTriggerResponse>(`/alarms/sync/${deviceSN}`);
    return {
      deviceSn: data.deviceSn ?? data.device_sn ?? deviceSN,
      taskId: data.taskId ?? data.task_id,
    };
  },
};
