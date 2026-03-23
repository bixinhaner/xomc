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
  // 详情页字段（对应 alarm_detail.jsp）
  alarmId: string;                    // 序号/告警ID (ALARM_ID)
  alarmIdentifier: string;            // 告警唯一标识 (ALARM_IDENTIFIER)
  alarmCode: string;                  // 告警编码
  alarmName: string;                  // 可能原因/告警名称 (ALARM_NAME)
  specificProblem: string;            // 具体故障 (SPECIFIC_PROBLEM)
  additionalInformation?: string;     // 附件信息 (ADDITIONAL_INFORMATION)
  additionalText?: string;            // 附件文本 (ADDITIONAL_TEXT)
  severity: AlarmSeverity;            // 严重程度 (ALARM_SERVERITY)
  eventType: EventType;               // 事件类型 (EVENT_TYPE)
  neType: string;                     // 告警源/网元类型 (NE_TYPE)
  equipInfo: string;                  // 网元定位 (EQUIP_INFO)
  dealState: DealState;               // 告警状态 (DEAL_STATE)
  eventTime: string;                  // 故障时间 (EVENT_TIME)
  updTime: string;                    // 更新时间 (UPD_TIME)
  dealUser?: string;                  // 确认人 (DEAL_USER)
  dealTime?: string;                  // 确认时间 (DEAL_TIME)
  clearUser?: string;                 // 告警清除人 (CLEAR_USER)
  clearTime?: string;                 // 清除时间 (ClEAR_TIME)
  suggestion?: string;                // 处理建议 (SUGGESTION)
  dealMemo?: string;                  // 描述/处理备注 (DEAL_MEMO)
  // 列表页字段
  alarmType: AlarmType;               // 告警类型 (active/history)
  alarmCount: number;                 // 告警次数
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
  ruleType: string;           // 执行动作: 0-禁止上报, 1-不入库不显示, 2-入库不显示, 3-自动确认
  deviceType?: string;         // 告警源: ENB/UPS/CPE/GNB/WCG/GSM
  severity: AlarmSeverity;
  enabled: boolean;
  isDefault?: boolean;         // 是否默认规则
  userCode?: string;           // 操作人
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
  alarmName?: string;                     // 可能原因（模糊查询）
  alarmIdentifier?: string;               // 告警唯一标识（精确查询）
  equipInfo?: string;                     // 网元定位（模糊查询）
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
