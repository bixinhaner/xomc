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
  description?: string;
  isShow: boolean;
  isUnknown?: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface AlarmSeverityLevel {
  id: string;
  code: number;
  // 真后端 alarm_severity_levels.name 是单列(ITU-T X.733 英文 Critical/Major/...);
  // mock 数据走 cnName/enName/colorHex 分列(便于看)。两套渲染都按 cnName ?? name 回退。
  name?: string;
  cnName?: string;
  enName?: string;
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
  description?: string;
  isShow?: boolean;
}

export type UpdateAlarmDefinitionInput = Partial<Omit<CreateAlarmDefinitionInput, 'identifier'>>;

export interface UnknownStatsFilter {
  productId?: string;
  days?: number;
}

// 告警 XML 来源(后端 source.go::ClassifySource 派生,前端只渲染)。
export type AlarmSource = 'builtin' | 'custom' | 'unknown';

// 自定义 XML 上传结果(对标 indicator)。
export interface AlarmUploadResult {
  uploaded: boolean;
  filename: string;
  loadedFrom: string;
  overwrite: boolean;
  backup: string;
  reloaded: boolean;
}

// 自定义 XML 删除结果(对标 indicator)。
export interface AlarmDeleteFileResult {
  deleted: boolean;
  loadedFrom: string;
  rowsAffected: number;
  backup: string;
}

// T-0179 drill-down 一级视图聚合行(后端 NeTypeStat)。
// loadedFrom 为空字符串表示历史数据未回填(Loader 未重跑过 → 显示"未知 XML 来源")。
export interface AlarmNeTypeStat {
  neType: string;
  loadedFrom: string;
  source: AlarmSource;   // 后端派生:builtin(alarm-definitions/)/custom(alarm-definitions-custom/)/unknown
  deletable: boolean;    // 后端守门:仅 custom 可删
  total: number;
  criticalCnt: number;
  majorCnt: number;
  minorCnt: number;
  warningCnt: number;
}
