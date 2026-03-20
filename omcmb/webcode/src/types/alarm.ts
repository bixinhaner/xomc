import type { AlarmSeverity } from './common';

export type AckStatus = 'acknowledged' | 'unacknowledged';

// 告警状态: 0-未确认未清除, 1-已确认未清除, 2-未确认已清除, 3-已确认已清除
export type DealState = '0' | '1' | '2' | '3';

// 事件类型
export type EventType = '30000' | '30001' | '30002' | '30003' | '30004' | '30006';

// 告警类型
export type AlarmType = 'active' | 'history';

export interface Alarm {
  id: string;
  alarmId: string;                    // 序号/告警ID
  alarmIdentifier: string;            // 告警唯一标识
  alarmCode: string;                  // 告警编码
  alarmName: string;                  // 可能原因/告警名称
  severity: AlarmSeverity;            // 告警级别
  neType: string;                     // 告警源/网元类型
  equipInfo: string;                  // 网元定位（设备信息：SN、小区名等）
  eventType: EventType;               // 事件类型
  dealState: DealState;               // 告警状态
  alarmType: AlarmType;               // 告警类型 (active/history)
  eventTime: string;                  // 故障时间
  updTime: string;                    // 更新时间
  clearTime?: string;                 // 清除时间（历史告警）
  specificProblem: string;            // 具体故障
  alarmCount: number;                 // 告警次数
  dealMemo?: string;                  // 描述/处理备注
  unread: '0' | '1';                  // 阅读状态: 0-已读, 1-未读
  // 以下为兼容旧字段
  deviceSn: string;
  deviceName: string;
  alarmContent: string;
  alarmTime: string;
  duration?: number;
  ackStatus: AckStatus;
  ackUser?: string;
  ackTime?: string;
  ackNote?: string;
  alarmSource: string;
  alarmLocation?: string;
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
  severity?: AlarmSeverity | AlarmSeverity[];
  ackStatus?: AckStatus;
  dealState?: DealState | DealState[];    // 告警状态
  eventType?: EventType | EventType[];    // 事件类型
  neType?: string;                        // 告警源
  unread?: '0' | '1';                     // 阅读状态
  deviceSn?: string;
  alarmCode?: string;
  alarmName?: string;
  alarmIdentifier?: string;               // 告警唯一标识
  keyword?: string;                       // 关键字搜索
  timeRange?: [string, string];           // 故障时间范围
  searchType?: 'Fuzzy' | 'Precise';       // 搜索方式：模糊/精确
}

export interface AlarmCount {
  critical: number;
  major: number;
  minor: number;
  warning: number;
}
