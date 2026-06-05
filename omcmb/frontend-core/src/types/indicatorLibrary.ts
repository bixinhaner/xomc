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
  // PM-P3:编号版公式(perf_indicators_*.arithmetic)。派生 KPI 为编号算术式(如
  // (C000060011+C000060022)/1000),原始计数为自身编号。界面公式展示用此字段
  // (对运维编号才是工作语言);标准名版 formula 表退为工程内部物,不再用于展示。
  arithmetic?: string;
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
// 一个 XML 文件即一个平台,一级列表"一个平台一条"。
// loadedFrom/source:后端 MAX(formula.loaded_from) + ClassifySource 派生,
// 一级列表渲染"加载源/来源"两展示列(删除操作仍留在 XMLFilesModal)。
export interface IndicatorPlatformSummary {
  tech: TechLower;
  platform: string;       // platform_name(rela_platform_indicator_formula_*.platform_name)
  indicators: number;     // 该平台的指标计数
  loadedFrom: string;     // 该平台对应的 XML 文件相对路径(后端 MAX(formula.loaded_from))
  source: IndicatorSource;// builtin/custom/unknown(后端由 sidecar 派生)
  deletable: boolean;     // 一级列表删除按钮可见性(后端 IsDeletable;内置置灰,2026-06-05)
  description?: string;   // 按 (tech, platform) 维度的可编辑描述
}

// IndicatorFile 是 /indicators/files?tech= 单行 — 含 source/deletable 派生 + DB 计数。
export interface IndicatorFile {
  loadedFrom: string;
  source: 'builtin' | 'custom' | 'unknown';
  deletable: boolean;
  count: number;
  onDisk: boolean;
}

export interface IndicatorUploadResult {
  uploaded: boolean;
  filename: string;
  loadedFrom: string;
  tech: TechLower;
  platform: string;     // 内容主键(<indicatorModel platform="...">)
  overwritten: boolean; // force 覆盖了既有文件(旧文件已备份 .bak.<ts>)
  reloaded: boolean;    // 同步触发 Loader.Reload 是否成功
}

export interface IndicatorDeleteFileResult {
  deleted: boolean;
  loadedFrom: string;
  tech: TechLower;
  rowsAffected: number;
  backup: string;     // ".deleted.<ts>" 文件名,空串=无备份(文件已 gone)
}
