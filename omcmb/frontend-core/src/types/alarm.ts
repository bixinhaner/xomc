import type { AlarmSeverity } from './common';

// 告警状态: 0-未确认未清除, 1-已确认未清除, 2-未确认已清除, 3-已确认已清除
export type DealState = '0' | '1' | '2' | '3';

// 事件类型
export type EventType = 'communication' | 'qualityOfService' | 'processingError' | 'device' | 'environment' | 'performance';

// 告警类型
export type AlarmType = 'active' | 'history';

export interface Alarm {
  id: string;
  // 核心字段（直接映射后端）
  deviceSn: string;
  deviceName: string;
  severity: AlarmSeverity;
  alarmIdentifier: string;
  description: string;
  eventType: EventType;
  alarmSource: string;
  technology: string;
  // 展示字段（mapper 推导/复合生成）
  alarmName: string;                  // 可能原因 = probable_cause
  specificProblem: string;            // 具体故障 = description（缺失时回退 probable_cause）
  neType: string;                     // 网元类型 = technology
  equipInfo: string;                  // 网元定位 = device_name(device_sn)
  eventTime: string;                  // 故障时间 = raised_at
  updTime: string;                    // 更新时间 = last_updated_at，缺失时回退 updated_at
  dealState: DealState;               // 告警状态（由 status + ack + clear 推导）
  dealUser?: string;                  // 确认人 = acknowledged_by
  dealTime?: string;                  // 确认时间 = acknowledged_at
  dealMemo?: string;                  // 确认备注 = ack_note
  clearTime?: string;                 // 清除时间 = cleared_at
  clearUser?: string;                 // 清除人 = cleared_by
  clearMemo?: string;                 // 清除备注 = clear_note
  duration?: number;                  // 持续时长（分钟，计算字段）
  // 列表字段
  alarmType: AlarmType;               // 告警类型: active | history（由 status 推导）
  alarmCount: number;                 // 告警次数 = ack_count
  unread: '0' | '1';                  // 阅读状态: 0-已读, 1-未读（由 is_read 取反）
  isActive: boolean;                  // 是否活动告警（由 status 推导）
  // 后端直传字段
  probableCause?: string;
  additionalInfo?: Record<string, string>;
  additionalText?: string;
  acknowledgedAt?: string;
  acknowledgedBy?: string;
  clearedAt?: string;
  isRead?: boolean;
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
  emailRecipients?: string[];  // 邮件通知动作的收件人（最多 50 个）
  effectiveStart?: string;     // 规则生效时间（ISO 8601）
  effectiveEnd?: string;       // 规则失效时间（ISO 8601，左闭右开）
  clearEffectiveWindow?: boolean; // 更新时显式清除生效区间
  createTime: string;
  updateTime: string;
}

export interface AlarmRuleCondition {
  field: string;
  operator: 'eq' | 'ne' | 'gt' | 'lt' | 'gte' | 'lte' | 'contains' | 'startsWith' | 'endsWith';
  value: string | number | boolean | string[];
}

export interface AlarmRuleAction {
  type: 'notify' | 'email' | 'sms' | 'webhook' | 'suppress';
  target?: string;
  template?: string;
  params?: Record<string, string>;
}

export interface AlarmFilter {
  severity?: AlarmSeverity | AlarmSeverity[];
  dealState?: DealState | DealState[];    // 告警状态
  eventType?: EventType | EventType[];    // 事件类型
  neType?: string;                        // 告警源
  unread?: '0' | '1';                     // 阅读状态
  deviceSn?: string;
  alarmName?: string;                     // 可能原因（模糊查询）
  alarmIdentifier?: string;               // 告警唯一标识（精确查询）
  equipInfo?: string;                     // 网元定位（模糊查询）
  keyword?: string;                       // 关键字搜索
  timeRange?: [string, string];           // 故障时间范围
  searchType?: 'Fuzzy' | 'Precise';       // 搜索方式：模糊/精确
  isUnknown?: 'true' | 'false';           // 未识别告警过滤：true 仅未识别 / false 仅已识别 / 不传 全部
}

export interface AlarmCount {
  total_active: number;
  unacknowledged: number;
  unread: number;
  stateVersion?: number;
  critical: number;
  major: number;
  minor: number;
  warning: number;
}
