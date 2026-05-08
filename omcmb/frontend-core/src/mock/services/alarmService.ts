import type { Alarm, AlarmRule, AlarmFilter } from '../../types/alarm';
import type { AlarmSeverity } from '../../types/common';
import type { PageRequest, PageResponse } from '../../types/pagination';
import { mockActiveAlarms, mockHistoricalAlarms } from '../data/alarms';
import { delay, paginate, sortBy, generateId } from '../utils';

let activeAlarms = [...mockActiveAlarms];
let historicalAlarms = [...mockHistoricalAlarms];

// 持久化的告警规则数据
let alarmRules: import('../../types/alarm').AlarmRule[] = [
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
    conditions: [{ field: 'alarmIdentifier', operator: 'eq', value: 'A0002' }],
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
    conditions: [{ field: 'alarmIdentifier', operator: 'eq', value: 'A0004' }, { field: 'description', operator: 'contains', value: '温度' }],
    actions: [{ type: 'suppress' }],
    createTime: '2024-03-01T00:00:00.000Z',
    updateTime: '2024-03-01T00:00:00.000Z',
  },
  {
    id: 'rule-004',
    ruleName: 'gNB光模块故障过滤',
    ruleType: '1',
    deviceType: 'GNB',
    severity: 'major',
    enabled: true,
    isDefault: false,
    userCode: 'admin',
    conditions: [{ field: 'alarmIdentifier', operator: 'eq', value: 'A0010' }],
    actions: [{ type: 'suppress' }],
    createTime: '2024-04-10T08:00:00.000Z',
    updateTime: '2024-05-12T10:30:00.000Z',
  },
  {
    id: 'rule-005',
    ruleName: 'CPE信号异常自动确认',
    ruleType: '3',
    deviceType: 'CPE',
    severity: 'minor',
    enabled: true,
    isDefault: false,
    userCode: 'user1',
    conditions: [{ field: 'alarmName', operator: 'contains', value: '信号' }],
    actions: [{ type: 'notify', target: '无线组' }],
    createTime: '2024-04-15T09:00:00.000Z',
    updateTime: '2024-06-20T14:00:00.000Z',
  },
  {
    id: 'rule-006',
    ruleName: 'UPS电源告警抑制',
    ruleType: '0',
    deviceType: 'UPS',
    severity: 'warning',
    enabled: false,
    isDefault: false,
    userCode: 'admin',
    conditions: [{ field: 'alarmIdentifier', operator: 'eq', value: 'A0020' }],
    actions: [{ type: 'suppress' }],
    createTime: '2024-04-20T10:00:00.000Z',
    updateTime: '2024-04-20T10:00:00.000Z',
  },
  {
    id: 'rule-007',
    ruleName: 'eNB风扇故障通知',
    ruleType: '3',
    deviceType: 'ENB',
    severity: 'minor',
    enabled: true,
    isDefault: false,
    userCode: 'admin',
    conditions: [{ field: 'alarmName', operator: 'contains', value: '风扇' }],
    actions: [{ type: 'email', target: 'maintenance@baicell.com' }],
    createTime: '2024-05-01T08:00:00.000Z',
    updateTime: '2024-07-10T09:00:00.000Z',
  },
  {
    id: 'rule-008',
    ruleName: 'WCG链路中断自动确认',
    ruleType: '3',
    deviceType: 'WCG',
    severity: 'critical',
    enabled: true,
    isDefault: false,
    userCode: 'user2',
    conditions: [{ field: 'alarmName', operator: 'contains', value: '链路中断' }],
    actions: [{ type: 'sms', target: '13800138000' }],
    createTime: '2024-05-05T11:00:00.000Z',
    updateTime: '2024-08-15T16:00:00.000Z',
  },
  {
    id: 'rule-009',
    ruleName: 'GSM时钟同步异常过滤',
    ruleType: '1',
    deviceType: 'GSM',
    severity: 'major',
    enabled: false,
    isDefault: false,
    userCode: 'admin',
    conditions: [{ field: 'alarmName', operator: 'contains', value: '时钟' }],
    actions: [{ type: 'suppress' }],
    createTime: '2024-05-10T14:00:00.000Z',
    updateTime: '2024-05-10T14:00:00.000Z',
  },
  {
    id: 'rule-010',
    ruleName: 'gNB射频单元功率异常',
    ruleType: '2',
    deviceType: 'GNB',
    severity: 'critical',
    enabled: true,
    isDefault: false,
    userCode: 'admin',
    conditions: [{ field: 'alarmName', operator: 'contains', value: '射频' }],
    actions: [{ type: 'notify', target: '射频组' }],
    createTime: '2024-05-15T09:00:00.000Z',
    updateTime: '2024-09-01T11:00:00.000Z',
  },
  {
    id: 'rule-011',
    ruleName: 'eNB内存使用率告警过滤',
    ruleType: '1',
    deviceType: 'ENB',
    severity: 'warning',
    enabled: true,
    isDefault: false,
    userCode: 'user1',
    conditions: [{ field: 'alarmName', operator: 'contains', value: '内存' }],
    actions: [{ type: 'suppress' }],
    createTime: '2024-05-20T08:00:00.000Z',
    updateTime: '2024-06-01T10:00:00.000Z',
  },
  {
    id: 'rule-012',
    ruleName: 'CPE SIM卡异常自动确认',
    ruleType: '3',
    deviceType: 'CPE',
    severity: 'warning',
    enabled: false,
    isDefault: false,
    userCode: 'admin',
    conditions: [{ field: 'alarmName', operator: 'contains', value: 'SIM' }],
    actions: [{ type: 'notify', target: '运维组' }],
    createTime: '2024-06-01T10:00:00.000Z',
    updateTime: '2024-06-01T10:00:00.000Z',
  },
  {
    id: 'rule-013',
    ruleName: 'gNB X2接口告警抑制',
    ruleType: '0',
    deviceType: 'GNB',
    severity: 'minor',
    enabled: true,
    isDefault: false,
    userCode: 'user2',
    conditions: [{ field: 'alarmName', operator: 'contains', value: 'X2' }],
    actions: [{ type: 'suppress' }],
    createTime: '2024-06-05T14:00:00.000Z',
    updateTime: '2024-07-20T09:00:00.000Z',
  },
  {
    id: 'rule-014',
    ruleName: 'eNB PCI冲突告警通知',
    ruleType: '3',
    deviceType: 'ENB',
    severity: 'warning',
    enabled: true,
    isDefault: false,
    userCode: 'admin',
    conditions: [{ field: 'alarmName', operator: 'contains', value: 'PCI' }],
    actions: [{ type: 'webhook', target: 'https://hooks.example.com/pci-alert' }],
    createTime: '2024-06-10T09:00:00.000Z',
    updateTime: '2024-08-01T15:00:00.000Z',
  },
  {
    id: 'rule-015',
    ruleName: 'UPS电池电量低告警',
    ruleType: '2',
    deviceType: 'UPS',
    severity: 'major',
    enabled: true,
    isDefault: false,
    userCode: 'admin',
    conditions: [{ field: 'alarmName', operator: 'contains', value: '电池' }],
    actions: [{ type: 'email', target: 'power@baicell.com' }],
    createTime: '2024-06-15T11:00:00.000Z',
    updateTime: '2024-09-10T08:00:00.000Z',
  },
  {
    id: 'rule-016',
    ruleName: 'gNB NR切换失败告警',
    ruleType: '1',
    deviceType: 'GNB',
    severity: 'major',
    enabled: false,
    isDefault: false,
    userCode: 'user1',
    conditions: [{ field: 'alarmName', operator: 'contains', value: '切换失败' }],
    actions: [{ type: 'suppress' }],
    createTime: '2024-06-20T08:00:00.000Z',
    updateTime: '2024-06-20T08:00:00.000Z',
  },
  {
    id: 'rule-017',
    ruleName: 'WCG ARP表溢出过滤',
    ruleType: '0',
    deviceType: 'WCG',
    severity: 'warning',
    enabled: true,
    isDefault: false,
    userCode: 'admin',
    conditions: [{ field: 'alarmName', operator: 'contains', value: 'ARP' }],
    actions: [{ type: 'suppress' }],
    createTime: '2024-07-01T10:00:00.000Z',
    updateTime: '2024-08-05T14:00:00.000Z',
  },
  {
    id: 'rule-018',
    ruleName: 'eNB CPRI链路误码告警',
    ruleType: '2',
    deviceType: 'ENB',
    severity: 'major',
    enabled: true,
    isDefault: false,
    userCode: 'user2',
    conditions: [{ field: 'alarmName', operator: 'contains', value: 'CPRI' }],
    actions: [{ type: 'notify', target: '传输组' }],
    createTime: '2024-07-05T09:00:00.000Z',
    updateTime: '2024-09-15T10:00:00.000Z',
  },
  {
    id: 'rule-019',
    ruleName: 'GSM驻波比异常自动确认',
    ruleType: '3',
    deviceType: 'GSM',
    severity: 'critical',
    enabled: true,
    isDefault: false,
    userCode: 'admin',
    conditions: [{ field: 'alarmName', operator: 'contains', value: '驻波比' }],
    actions: [{ type: 'sms', target: '13900139000' }],
    createTime: '2024-07-10T14:00:00.000Z',
    updateTime: '2024-10-01T09:00:00.000Z',
  },
  {
    id: 'rule-020',
    ruleName: 'gNB GPS定位失效告警',
    ruleType: '1',
    deviceType: 'GNB',
    severity: 'major',
    enabled: false,
    isDefault: false,
    userCode: 'admin',
    conditions: [{ field: 'alarmName', operator: 'contains', value: 'GPS' }],
    actions: [{ type: 'suppress' }],
    createTime: '2024-07-15T08:00:00.000Z',
    updateTime: '2024-07-15T08:00:00.000Z',
  },
  {
    id: 'rule-021',
    ruleName: 'eNB电源模块故障通知',
    ruleType: '3',
    deviceType: 'ENB',
    severity: 'critical',
    enabled: true,
    isDefault: false,
    userCode: 'user1',
    conditions: [{ field: 'alarmName', operator: 'contains', value: '电源' }],
    actions: [{ type: 'email', target: 'power@baicell.com' }],
    createTime: '2024-07-20T10:00:00.000Z',
    updateTime: '2024-10-10T11:00:00.000Z',
  },
  {
    id: 'rule-022',
    ruleName: 'CPE在线率低告警抑制',
    ruleType: '0',
    deviceType: 'CPE',
    severity: 'minor',
    enabled: true,
    isDefault: false,
    userCode: 'admin',
    conditions: [{ field: 'alarmName', operator: 'contains', value: '在线率' }],
    actions: [{ type: 'suppress' }],
    createTime: '2024-07-25T09:00:00.000Z',
    updateTime: '2024-08-30T10:00:00.000Z',
  },
  {
    id: 'rule-023',
    ruleName: 'WCG磁盘空间不足告警',
    ruleType: '2',
    deviceType: 'WCG',
    severity: 'warning',
    enabled: true,
    isDefault: false,
    userCode: 'user2',
    conditions: [{ field: 'alarmName', operator: 'contains', value: '磁盘' }],
    actions: [{ type: 'webhook', target: 'https://hooks.example.com/disk-alert' }],
    createTime: '2024-08-01T08:00:00.000Z',
    updateTime: '2024-10-20T14:00:00.000Z',
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

  if (filter.deviceSn) result = result.filter((a) => a.deviceSn.includes(filter.deviceSn!));
  if (filter.alarmIdentifier) result = result.filter((a) => a.alarmIdentifier.includes(filter.alarmIdentifier!));
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
        a.description.toLowerCase().includes(kw)
    );
  }
  if (filter.timeRange) {
    const [start, end] = filter.timeRange;
    result = result.filter(
      (a) => a.eventTime >= start && a.eventTime <= end
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
            dealState: (a.dealState === '2' ? '3' : '1') as Alarm['dealState'],
            dealUser: 'admin',
            dealTime: now,
            dealMemo: note ?? '已确认',
          }
        : a
    );
  },

  async unacknowledgeAlarms(ids: string[]): Promise<void> {
    await delay(200, 400);
    activeAlarms = activeAlarms.map((a) =>
      ids.includes(a.id)
        ? {
            ...a,
            dealState: (a.dealState === '3' ? '2' : '0') as Alarm['dealState'],
            dealUser: undefined,
            dealTime: undefined,
            dealMemo: undefined,
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
      dealState: (a.dealState === '1' ? '3' : '2') as Alarm['dealState'],
      isActive: false,
      duration: Math.floor(
        (new Date(now).getTime() - new Date(a.eventTime).getTime()) / 60000
      ),
    }));
    activeAlarms = activeAlarms.filter((a) => !ids.includes(a.id));
    historicalAlarms = [...cleared, ...historicalAlarms];
  },

  async acknowledgeHistoryAlarms(ids: string[], note?: string): Promise<void> {
    await delay(200, 400);
    const now = new Date().toISOString();
    historicalAlarms = historicalAlarms.map((a) =>
      ids.includes(a.id)
        ? { ...a, dealState: (a.dealState === '2' ? '3' : '1') as Alarm['dealState'], dealUser: 'admin', dealTime: now, dealMemo: note ?? '已确认' }
        : a
    );
  },

  async unacknowledgeHistoryAlarms(ids: string[]): Promise<void> {
    await delay(200, 400);
    historicalAlarms = historicalAlarms.map((a) =>
      ids.includes(a.id)
        ? { ...a, dealState: (a.dealState === '3' ? '2' : '0') as Alarm['dealState'], dealUser: undefined, dealTime: undefined, dealMemo: undefined }
        : a
    );
  },

  async deleteHistoryAlarms(ids: string[]): Promise<void> {
    await delay(200, 400);
    historicalAlarms = historicalAlarms.filter((a) => !ids.includes(a.id));
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

  async toggleRule(id: string): Promise<AlarmRule> {
    await delay(150, 300);
    const index = alarmRules.findIndex((r) => r.id === id);
    if (index === -1) throw new Error('Rule not found');
    alarmRules[index] = { ...alarmRules[index], enabled: !alarmRules[index].enabled, updateTime: new Date().toISOString() };
    return alarmRules[index];
  },

  async markAlarmRead(_id: string): Promise<void> {
    await delay(50, 100);
  },

  // T-0098-P5-06：旧 alarm-library mock 已删，治理走 alarmDefinitionService（mock/services/alarmDefinitionService.ts）。
};
