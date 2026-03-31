import type { Alarm, AlarmRule, AlarmFilter } from '@/types/alarm';
import type { AlarmSeverity } from '@/types/common';
import type { PageRequest, PageResponse } from '@/types/pagination';
import { mockActiveAlarms, mockHistoricalAlarms } from '../data/alarms';
import { delay, paginate, sortBy, generateId } from '../utils';

let activeAlarms = [...mockActiveAlarms];
let historicalAlarms = [...mockHistoricalAlarms];

// 持久化的告警规则数据
let alarmRules: import('@/types/alarm').AlarmRule[] = [
  {
    id: 'rule-001',
    ruleName: '默认告警过滤规则',
    ruleType: '1',
    deviceType: '',
    severity: 'warning',
    enabled: true,
    isDefault: true,
    userCode: 'system',
    conditions: [],
    actions: [],
    createTime: '2024-01-01T00:00:00.000Z',
    updateTime: '2024-06-01T00:00:00.000Z',
  },
  {
    id: 'rule-002',
    ruleName: 'S1链路中断告警过滤',
    ruleType: '3',
    deviceType: 'ENB',
    severity: 'major',
    enabled: true,
    isDefault: false,
    userCode: 'admin',
    conditions: [{ field: 'alarmCode', operator: 'eq', value: 'A0002' }],
    actions: [{ type: 'notify', target: '传输组' }],
    createTime: '2024-01-01T00:00:00.000Z',
    updateTime: '2024-04-01T00:00:00.000Z',
  },
  {
    id: 'rule-003',
    ruleName: '温度过高抑制规则',
    ruleType: '0',
    deviceType: 'ENB',
    severity: 'warning',
    enabled: false,
    isDefault: false,
    userCode: 'admin',
    conditions: [{ field: 'alarmCode', operator: 'eq', value: 'A0004' }, { field: 'alarmContent', operator: 'contains', value: '温度' }],
    actions: [{ type: 'suppress' }],
    createTime: '2024-03-01T00:00:00.000Z',
    updateTime: '2024-03-01T00:00:00.000Z',
  },
];

function applyAlarmFilter(items: Alarm[], filter: AlarmFilter): Alarm[] {
  let result = [...items];

  // severity 支持单值和数组
  if (filter.severity) {
    const severities = Array.isArray(filter.severity) ? filter.severity : [filter.severity];
    result = result.filter((a) => severities.includes(a.severity));
  }

  // dealState 支持单值和数组
  if (filter.dealState) {
    const states = Array.isArray(filter.dealState) ? filter.dealState : [filter.dealState];
    result = result.filter((a) => states.includes(a.dealState));
  }

  // eventType 支持单值和数组
  if (filter.eventType) {
    const types = Array.isArray(filter.eventType) ? filter.eventType : [filter.eventType];
    result = result.filter((a) => types.includes(a.eventType));
  }

  // unread 阅读状态
  if (filter.unread) {
    result = result.filter((a) => a.unread === filter.unread);
  }

  if (filter.ackStatus) result = result.filter((a) => a.ackStatus === filter.ackStatus);
  if (filter.deviceSn) result = result.filter((a) => a.deviceSn.includes(filter.deviceSn!));
  if (filter.alarmCode) result = result.filter((a) => a.alarmCode.includes(filter.alarmCode!));
  if (filter.neType) result = result.filter((a) => a.neType === filter.neType);

  // 告警标识 - 精确查询
  if (filter.alarmIdentifier) {
    const identifier = filter.alarmIdentifier.toLowerCase();
    result = result.filter((a) => a.alarmIdentifier.toLowerCase() === identifier);
  }

  // 可能原因 - 模糊查询
  if (filter.alarmName) {
    const name = filter.alarmName.toLowerCase();
    result = result.filter((a) => a.alarmName.toLowerCase().includes(name));
  }

  // 网元定位 - 模糊查询
  if (filter.equipInfo) {
    const equip = filter.equipInfo.toLowerCase();
    result = result.filter((a) => a.equipInfo.toLowerCase().includes(equip));
  }

  if (filter.keyword) {
    const kw = filter.keyword.toLowerCase();
    result = result.filter(
      (a) =>
        a.alarmName.toLowerCase().includes(kw) ||
        a.deviceName.toLowerCase().includes(kw) ||
        a.alarmContent.toLowerCase().includes(kw)
    );
  }
  if (filter.timeRange) {
    const [start, end] = filter.timeRange;
    result = result.filter(
      (a) => a.alarmTime >= start && a.alarmTime <= end
    );
  }
  return result;
}

export const alarmService = {
  async getCurrentAlarms(
    params: AlarmFilter & PageRequest
  ): Promise<PageResponse<Alarm>> {
    await delay(100, 250);
    let filtered = applyAlarmFilter(activeAlarms, params);
    if (params.sortField) {
      filtered = sortBy(filtered, params.sortField as keyof Alarm, params.sortOrder ?? 'descend');
    } else {
      filtered = filtered.sort((a, b) => {
        const severityOrder: Record<AlarmSeverity, number> = { critical: 0, major: 1, minor: 2, warning: 3 };
        return severityOrder[a.severity] - severityOrder[b.severity];
      });
    }
    return paginate(filtered, params.page, params.pageSize);
  },

  async getHistoricalAlarms(
    params: AlarmFilter & PageRequest
  ): Promise<PageResponse<Alarm>> {
    await delay(100, 300);
    let filtered = applyAlarmFilter(historicalAlarms, params);
    if (params.sortField) {
      filtered = sortBy(filtered, params.sortField as keyof Alarm, params.sortOrder ?? 'descend');
    }
    return paginate(filtered, params.page, params.pageSize);
  },

  async getList(params: AlarmFilter & PageRequest & { isActive?: boolean }): Promise<PageResponse<Alarm>> {
    const source = params.isActive === false ? historicalAlarms : activeAlarms;
    await delay(100, 250);
    let filtered = applyAlarmFilter(source, params);
    if (params.sortField) {
      filtered = sortBy(filtered, params.sortField as keyof Alarm, params.sortOrder ?? 'descend');
    }
    return paginate(filtered, params.page, params.pageSize);
  },

  async getById(id: string): Promise<Alarm | null> {
    await delay(80, 150);
    return (
      activeAlarms.find((a) => a.id === id) ??
      historicalAlarms.find((a) => a.id === id) ??
      null
    );
  },

  async acknowledgeAlarms(ids: string[], note?: string): Promise<void> {
    await delay(200, 400);
    const now = new Date().toISOString();
    activeAlarms = activeAlarms.map((a) =>
      ids.includes(a.id)
        ? {
            ...a,
            ackStatus: 'acknowledged' as const,
            ackUser: 'admin',
            ackTime: now,
            ackNote: note ?? '已确认',
          }
        : a
    );
  },

  async clearAlarms(ids: string[]): Promise<void> {
    await delay(200, 400);
    const now = new Date().toISOString();
    const cleared = activeAlarms.filter((a) => ids.includes(a.id)).map((a) => ({
      ...a,
      clearTime: now,
      isActive: false,
      duration: Math.floor(
        (new Date(now).getTime() - new Date(a.alarmTime).getTime()) / 60000
      ),
    }));
    activeAlarms = activeAlarms.filter((a) => !ids.includes(a.id));
    historicalAlarms = [...cleared, ...historicalAlarms];
  },

  async getAlarmCount(): Promise<{ critical: number; major: number; minor: number; warning: number }> {
    await delay(50, 100);
    return {
      critical: activeAlarms.filter((a) => a.severity === 'critical').length,
      major: activeAlarms.filter((a) => a.severity === 'major').length,
      minor: activeAlarms.filter((a) => a.severity === 'minor').length,
      warning: activeAlarms.filter((a) => a.severity === 'warning').length,
    };
  },

  // Alarm rules
  async getRules(params: PageRequest): Promise<PageResponse<AlarmRule>> {
    await delay(100, 200);
    return paginate(alarmRules, params.page, params.pageSize);
  },

  async createRule(data: Omit<AlarmRule, 'id' | 'createTime' | 'updateTime'>): Promise<AlarmRule> {
    await delay(200, 400);
    const newRule: AlarmRule = {
      ...data,
      id: generateId('rule'),
      isDefault: false,
      createTime: new Date().toISOString(),
      updateTime: new Date().toISOString(),
    };
    alarmRules.push(newRule);
    return newRule;
  },

  async updateRule(id: string, data: Partial<AlarmRule>): Promise<AlarmRule> {
    await delay(150, 300);
    const index = alarmRules.findIndex((r) => r.id === id);
    if (index === -1) {
      throw new Error('Rule not found');
    }
    const existing = alarmRules[index];
    const updated: AlarmRule = {
      ...existing,
      ...data,
      id,
      updateTime: new Date().toISOString(),
    };
    alarmRules[index] = updated;
    return updated;
  },

  async deleteRules(ids: string[]): Promise<void> {
    await delay(150, 300);
    alarmRules = alarmRules.filter((r) => !ids.includes(r.id));
  },
};
