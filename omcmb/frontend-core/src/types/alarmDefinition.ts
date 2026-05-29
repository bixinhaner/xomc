// 告警定义类型（T-0098-P4 数据字典平台化）
// 后端：internal/alarm/definition/handler.go defView

export interface AlarmDefinition {
  id: string;
  identifier: string;
  neType: string;
  cnName: string;
  enName: string;
  severityId?: string;
  severityCode: number;
  severityName: string;
  eventType?: string;
  cnProbableCause?: string;
  enProbableCause?: string;
  cnSuggestion?: string;
  enSuggestion?: string;
  isShow: boolean;
  isUnknown?: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface AlarmSeverityLevel {
  id: string;
  code: number;
  cnName: string;
  enName: string;
  colorHex?: string;
}

export interface UnknownAlarmStat {
  productId?: string;
  productName?: string;
  identifier: string;
  count: number;
  lastSeenAt?: string;
}

export interface AlarmDefinitionFilter {
  neType?: string;
  severityCode?: number;
  keyword?: string;
  isUnknown?: boolean;
  page?: number;
  pageSize?: number;
}

export interface CreateAlarmDefinitionInput {
  identifier: string;
  neType: string;
  cnName: string;
  enName: string;
  severityCode: number;
  eventType?: string;
  cnProbableCause?: string;
  enProbableCause?: string;
  cnSuggestion?: string;
  enSuggestion?: string;
  isShow?: boolean;
}

export type UpdateAlarmDefinitionInput = Partial<Omit<CreateAlarmDefinitionInput, 'identifier'>>;

export interface UnknownStatsFilter {
  productId?: string;
  days?: number;
}

// T-0179 drill-down 一级视图聚合行(后端 NeTypeStat)。
// loadedFrom 为空字符串表示历史数据未回填(Loader 未重跑过 → 显示"未知 XML 来源")。
export interface AlarmNeTypeStat {
  neType: string;
  loadedFrom: string;
  total: number;
  criticalCnt: number;
  majorCnt: number;
  minorCnt: number;
  warningCnt: number;
}
