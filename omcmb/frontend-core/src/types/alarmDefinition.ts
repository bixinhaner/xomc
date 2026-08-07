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
  eventType?: number | string;
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
  loadedFrom?: string;
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
  eventType?: number;
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

// 自定义 XML 上传结果(2026-06-05:重复允许 force 覆盖,新增 overwritten)。
export interface AlarmUploadResult {
  uploaded: boolean;
  filename: string;
  loadedFrom: string;
  neType: string;
  overwritten: boolean; // force 覆盖了既有文件(旧文件已备份 .bak.<ts>)
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
  // 2026-06-03:取消 builtin/custom 区分后,来源不再展示、全部可删;字段保留可选以兼容后端历史返回。
  source?: AlarmSource;
  deletable?: boolean;
  total: number;
  criticalCnt: number;
  majorCnt: number;
  minorCnt: number;
  warningCnt: number;
}
