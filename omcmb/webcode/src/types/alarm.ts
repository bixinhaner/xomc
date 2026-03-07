import type { AlarmSeverity } from './common';

export type AckStatus = 'acknowledged' | 'unacknowledged';

export interface Alarm {
  id: string;
  alarmCode: string;
  alarmName: string;
  severity: AlarmSeverity;
  deviceSn: string;
  deviceName: string;
  neType: string;
  alarmContent: string;
  alarmTime: string;
  clearTime?: string;
  duration?: number;
  ackStatus: AckStatus;
  ackUser?: string;
  ackTime?: string;
  ackNote?: string;
  alarmSource: string;
  alarmLocation?: string;
  alarmType?: string;
  isActive: boolean;
}

export interface AlarmRule {
  id: string;
  ruleName: string;
  ruleType: string;
  severity: AlarmSeverity;
  enabled: boolean;
  conditions: AlarmRuleCondition[];
  actions: AlarmRuleAction[];
  createTime: string;
  updateTime: string;
}

export interface AlarmRuleCondition {
  field: string;
  operator: 'eq' | 'ne' | 'gt' | 'lt' | 'gte' | 'lte' | 'contains' | 'startsWith' | 'endsWith';
  value: string | number | boolean;
}

export interface AlarmRuleAction {
  type: 'notify' | 'email' | 'sms' | 'webhook' | 'suppress';
  target?: string;
  template?: string;
  params?: Record<string, string>;
}

export interface AlarmFilter {
  severity?: AlarmSeverity;
  ackStatus?: AckStatus;
  deviceSn?: string;
  alarmCode?: string;
  alarmName?: string;
  timeRange?: [string, string];
  neType?: string;
  keyword?: string;
}

export interface AlarmCount {
  critical: number;
  major: number;
  minor: number;
  warning: number;
}
