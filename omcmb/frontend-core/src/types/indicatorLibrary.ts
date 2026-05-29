// KPI 指标库类型（T-0098-P4 数据字典平台化）
// 后端：internal/pm/indicator/rest_handler.go IndicatorInfo / GroupInfo / Unit

export type DeviceType = 'ENB' | 'GSM' | 'GNB';

export interface IndicatorInfo {
  id: string;
  name: string;
  cnName?: string;
  enName?: string;
  groupId?: string;
  groupName?: string;
  counterType?: string;
  indicatorLevel?: string;       // ENB only, GNB 无此字段
  unit?: string;
  description?: string;
  isCounter?: boolean;
  productClass?: string;
  operatorCode?: string;
  isEnabled?: boolean;
  deviceType: DeviceType;
}

export interface IndicatorListFilter {
  groupId?: string;
  keyword?: string;
  operatorCode?: string;
  productClass?: string;
  indicatorLevel?: string;
  isEnabled?: boolean;
  isCounter?: boolean;
  platformName?: string;
  page?: number;
  pageSize?: number;
}

export interface IndicatorGroup {
  id: string;
  name: string;
  parentId?: string;
  description?: string;
  operatorCode?: string;
  deviceType: DeviceType;
  children?: IndicatorGroup[];
}

export interface PlatformFormula {
  platformName: string;
  indicatorId: string;
  formula: string;
  description?: string;
}

export interface IndicatorUnit {
  id: string;
  enName: string;
  cnName: string;
}

export interface CreateIndicatorInput {
  id: string;
  name: string;
  cnName?: string;
  enName?: string;
  groupId?: string;
  counterType?: string;
  indicatorLevel?: string;
  unit?: string;
  description?: string;
  productClass?: string;
  operatorCode?: string;
}

export type UpdateIndicatorInput = Partial<Omit<CreateIndicatorInput, 'id'>>;

export interface CreateGroupInput {
  id: string;
  name: string;
  parentId?: string;
  description?: string;
  operatorCode?: string;
}

export type UpdateGroupInput = Partial<Omit<CreateGroupInput, 'id'>>;

export interface EnabledIndicatorsRequest {
  deviceType: DeviceType;
  operatorCode: string;
  indicatorIds: string[];
  enable: boolean;
}

export interface UnitInput {
  id: string;
  enName: string;
  cnName: string;
}

// ── T-0180 P4: 自定义 XML 分层目录 + drill-down 视图类型 ──────────────────

// 后端 ?tech= query 用 enb/gsm/gnb 小写;前端 DeviceType 用 ENB/GSM/GNB 大写
// (历史保留)。techFromDeviceType 在 API 层做映射,UI 用 DeviceType 即可。
export type TechLower = 'enb' | 'gsm' | 'gnb';

export function deviceTypeToTech(dt: DeviceType): TechLower {
  return dt.toLowerCase() as TechLower;
}

export function techToDeviceType(t: TechLower): DeviceType {
  return t.toUpperCase() as DeviceType;
}

// IndicatorSource 与 ParamModelSource 同枚举(builtin/custom/unknown);
// 后端 source.go::ClassifySource 派生,前端 SummaryTab 渲染"来源"列 + 守门删除。
export type IndicatorSource = 'builtin' | 'custom' | 'unknown';

// IndicatorPlatformSummary 是 /indicators/summary 端点单行 — (制式, 平台) 二元组。
// 2026-05-29 用户决策:从"3 行/制式"调为"N 行/(制式, 平台)"粒度。
// 2026-05-29 二次扩展:对齐 T-0178 param-model,补 source/deletable
// (后端唯一真值源,前端只渲染)。
export interface IndicatorPlatformSummary {
  tech: TechLower;
  platform: string;       // platform_name(rela_platform_indicator_formula_*.platform_name)
  loadedFrom: string;     // XML 文件路径(含前缀),如 "indicator-library/enb/ALL.xml"
  indicators: number;     // 该 (loaded_from, platform) 指标计数
  source: IndicatorSource;
  deletable: boolean;
}

// IndicatorFile 是 /indicators/files?tech= 单行 — 含 source/deletable 派生 + DB 计数。
export interface IndicatorFile {
  loadedFrom: string;
  source: 'builtin' | 'custom' | 'unknown';
  deletable: boolean;
  count: number;
  onDisk: boolean;
}

// IndicatorReloadMode 对应后端 ReloadMode("import" | "reload");
// import = 加法 UPSERT(默认/向后兼容);reload = destructive 全量重载 + 删孤儿
export type IndicatorReloadMode = 'import' | 'reload';

export interface IndicatorReloadResult {
  reloaded: string;  // "indicator"
  mode: IndicatorReloadMode;
  // reload 模式下三制式孤儿删除计数;import 模式不返
  orphans?: Record<TechLower, number>;
}

export interface IndicatorUploadResult {
  uploaded: boolean;
  filename: string;
  loadedFrom: string;
  tech: TechLower;
  overwrite: boolean;
  backup: string;     // ".bak.<ts>" 文件名,空串=无备份
  reloaded: boolean;  // 同步触发 Loader.Reload 是否成功
}

export interface IndicatorDeleteFileResult {
  deleted: boolean;
  loadedFrom: string;
  tech: TechLower;
  rowsAffected: number;
  backup: string;     // ".deleted.<ts>" 文件名,空串=无备份(文件已 gone)
}
