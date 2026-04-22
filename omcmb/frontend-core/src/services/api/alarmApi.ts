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
  severity: number; // 1=Critical, 2=Major, 3=Minor, 4=Warning
  alarm_type: string;
  alarm_code: string;
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
  by_severity: Record<string, number>;
  by_type: Record<string, number>;
}

// ---------------------------------------------------------------------------
// Backend alarm library model
// ---------------------------------------------------------------------------

interface BackendAlarmLibrary {
  id: string;
  alarm_code: string;
  alarm_source: string;
  event_type: string;
  severity: number;
  enabled: boolean;
  probable_cause: string;
  explanation?: string;
  additional_info?: Record<string, unknown>;
  carrier?: string;
  technology?: string;
  created_at: string;
  updated_at: string;
}

interface AlarmLibraryItem {
  id: string;
  alarmCode: string;
  alarmSource: string;
  eventType: string;
  severity: number;
  enabled: boolean;
  probableCause: string;
  explanation?: string;
  additionalInfo?: Record<string, unknown>;
  carrier?: string;
  technology?: string;
  createdAt: string;
  updatedAt: string;
}

function mapBackendLibrary(bl: BackendAlarmLibrary): AlarmLibraryItem {
  return {
    id: bl.id,
    alarmCode: bl.alarm_code,
    alarmSource: bl.alarm_source,
    eventType: bl.event_type,
    severity: bl.severity,
    enabled: bl.enabled,
    probableCause: bl.probable_cause,
    explanation: bl.explanation,
    additionalInfo: bl.additional_info,
    carrier: bl.carrier,
    technology: bl.technology,
    createdAt: bl.created_at,
    updatedAt: bl.updated_at,
  };
}

// ---------------------------------------------------------------------------
// Backend alarm filter rule model
// ---------------------------------------------------------------------------

interface BackendAlarmRule {
  id: string;
  name: string;
  filter_type: string; // alarm_code | alarm_source | device | device_group
  alarm_sources: string[];
  alarm_codes: string[];
  device_ids: string[];
  device_group_ids: string[];
  action: string; // default | ignore | auto_acknowledge | auto_clear
  acknowledge_desc: string;
  priority: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

// ---------------------------------------------------------------------------
// Severity helpers
// ---------------------------------------------------------------------------

const severityNumToStr: Record<number, string> = {
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

  return {
    id: ba.id,
    alarmCode: ba.alarm_code,
    alarmName: ba.probable_cause || ba.description || ba.alarm_code,
    specificProblem: ba.probable_cause || '',
    severity: (severityNumToStr[ba.severity] || 'warning') as Alarm['severity'],
    deviceSn: ba.device_sn,
    deviceName: ba.device_name || ba.device_sn,
    description: ba.description,
    neType: ba.technology || '',
    equipInfo: ba.device_name ? `${ba.device_name}(${ba.device_sn})` : ba.device_sn,
    eventType: (ba.event_type || ba.alarm_type || '') as Alarm['eventType'],
    dealState,
    eventTime: ba.raised_at,
    updTime: ba.updated_at,
    dealUser: ba.acknowledged_by,
    dealTime: ba.acknowledged_at,
    clearTime: ba.cleared_at,
    clearUser: ba.cleared_by,
    clearMemo: ba.clear_note,
    dealMemo: ba.ack_note || undefined,
    alarmType: ba.status === 'cleared' ? 'history' : 'active',
    alarmCount: ba.ack_count || 1,
    unread: ba.is_read ? '0' : '1',
    duration: ba.cleared_at && ba.raised_at
      ? Math.floor(
          (new Date(ba.cleared_at).getTime() -
            new Date(ba.raised_at).getTime()) /
            60000
        )
      : ba.acknowledged_at
        ? Math.floor(
            (new Date(ba.acknowledged_at).getTime() -
              new Date(ba.raised_at).getTime()) /
              60000
          )
        : undefined,
    alarmSource: ba.alarm_source || ba.carrier,
    technology: ba.technology || '',
    isActive: ba.status === 'active' || ba.status === 'acknowledged',
    // 后端直传字段
    probableCause: ba.probable_cause,
    additionalInfo: ba.additional_info,
    acknowledgedAt: ba.acknowledged_at,
    acknowledgedBy: ba.acknowledged_by,
    clearedAt: ba.cleared_at,
    isRead: ba.is_read,
  };
}

function mapBackendAlarmRule(br: BackendAlarmRule): AlarmRule {
  // Derive frontend conditions from backend filter fields
  const conditions: AlarmRuleCondition[] = [];

  if (br.alarm_codes.length > 0) {
    conditions.push({
      field: 'alarm_code',
      operator: 'contains',
      value: br.alarm_codes,
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

  if (data.ruleName !== undefined) payload.name = data.ruleName;
  if (data.enabled !== undefined) payload.enabled = data.enabled;
  if (data.ruleType !== undefined) payload.action = data.ruleType;

  // Extract conditions into backend fields
  const conditions = data.conditions || [];
  const alarmCodes = conditions
    .filter((c) => c.field === 'alarm_code')
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

  if (alarmCodes.length > 0) payload.alarm_codes = alarmCodes;
  if (alarmSources.length > 0) payload.alarm_sources = alarmSources;
  if (deviceIds.length > 0) payload.device_ids = deviceIds;
  if (deviceGroupIds.length > 0) payload.device_group_ids = deviceGroupIds;

  // Extract acknowledge_desc from actions
  const actions = data.actions || [];
  const suppressAction = actions.find((a) => a.type === 'suppress');
  if (suppressAction?.template) {
    payload.acknowledge_desc = suppressAction.template;
  }

  // Determine filter_type from conditions
  if (alarmCodes.length > 0) {
    payload.filter_type = 'alarm_code';
  } else if (alarmSources.length > 0) {
    payload.filter_type = 'alarm_source';
  } else if (deviceGroupIds.length > 0) {
    payload.filter_type = 'device_group';
  } else if (deviceIds.length > 0) {
    payload.filter_type = 'device';
  } else {
    payload.filter_type = 'alarm_code';
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
    const sev = Array.isArray(filter.severity) ? filter.severity[0] : filter.severity;
    if (sev) query.severity = severityStrToNum[sev];
  }
  if (filter.deviceSn) query.device_sn = filter.deviceSn;
  if (filter.timeRange) {
    query.start_time = filter.timeRange[0];
    query.end_time = filter.timeRange[1];
  }
  if (filter.keyword) query.keyword = filter.keyword;

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
      critical: bySeverity['1'] || 0,
      major: bySeverity['2'] || 0,
      minor: bySeverity['3'] || 0,
      warning: bySeverity['4'] || 0,
    };
  },

  // -- Alarm filter rules ---------------------------------------------------

  async getRules(params: PageRequest): Promise<PageResponse<AlarmRule>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
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

  // -- Alarm libraries ----------------------------------------------------

  async getAlarmLibraries(params: PageRequest & { alarmCode?: string; alarmSource?: string; severity?: string; eventType?: string; keyword?: string }): Promise<PageResponse<AlarmLibraryItem>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.alarmCode) query.alarm_code = params.alarmCode;
    if (params.alarmSource) query.alarm_source = params.alarmSource;
    if (params.severity) query.severity = params.severity;
    if (params.eventType) query.event_type = params.eventType;
    if (params.keyword) query.keyword = params.keyword;
    const { data } = await http.get<BackendListResponse<BackendAlarmLibrary>>(
      '/alarms/alarm-libraries',
      { params: query }
    );
    return {
      items: (data.items || []).map(mapBackendLibrary),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async createAlarmLibrary(payload: {
    alarm_code: string;
    alarm_source: string;
    event_type: string;
    severity: number;
    probable_cause: string;
    explanation?: string;
    carrier?: string;
    technology?: string;
    enabled?: boolean;
  }): Promise<AlarmLibraryItem> {
    const { data } = await http.post<BackendAlarmLibrary>(
      '/alarms/alarm-libraries',
      payload
    );
    return mapBackendLibrary(data);
  },

  async updateAlarmLibrary(id: string, payload: Partial<{
    severity: number;
    enabled: boolean;
    probable_cause: string;
    explanation: string;
    carrier: string;
    technology: string;
  }>): Promise<AlarmLibraryItem> {
    const { data } = await http.put<BackendAlarmLibrary>(
      `/alarms/alarm-libraries/${id}`,
      payload
    );
    return mapBackendLibrary(data);
  },

  async deleteAlarmLibrary(id: string): Promise<void> {
    await http.delete(`/alarms/alarm-libraries/${id}`);
  },
};
