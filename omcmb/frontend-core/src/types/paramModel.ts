// 参数模型类型（T-0098-P4 数据字典平台化）
// 后端：internal/config/parammodel/handler.go modelView / mappingView / standardView

// T-0178: paramModel 来源(后端 source.go::ClassifySource 派生)
// 三态字符串字面量,不用 enum(项目约定 + 与后端 JSON 字段直接对齐)。
export type ParamModelSource = 'builtin' | 'custom' | 'unknown';

export interface ParamModel {
  id: string;
  name: string;
  totalEntries: number;
  totalObjects: number;
  totalParams: number;
  description: string;
  isActive: boolean;
  loadedFrom: string;
  // 2026-06-03:取消 builtin/custom 区分后,来源不再展示、全部可删;字段保留可选以兼容后端历史返回。
  source?: ParamModelSource;
  deletable?: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface ParamMapping {
  id: string;
  paramModelId: string;
  standardPath: string;
  privatePath: string;
  entryType: string;       // 'object' | 'parameter'
  access: string;          // 'readOnly' | 'readWrite' 等
  dataType: string;
  changeApplies: string;
  minValue?: string;
  maxValue?: string;
  isStorable: boolean;
  isActive: boolean;
  softwareVersion?: string;
  // T-PMSRC: 行级来源。'builtin'=XML 加载(不可删,重载重建);'custom'=管理员经 UI 新增/编辑后的覆盖项(可删,重载保留)。
  source?: ParamModelSource;
  deletable?: boolean;
}

export interface StandardParam {
  standardPath: string;
  entryType: string;
  access: string;
  dataType: string;
  changeApplies: string;
  minValue?: string;
  maxValue?: string;
}

// ISSUE-488: 标准参数树 dataType / changeApplies 枚举化 —— 三皮肤共享。
//
// dataType 枚举取值（小写，对齐 TR-069 与前端默认）。后端校验大小写不敏感
// （strings.EqualFold），且 min/max 在 dataType==='string' 时按字符串长度校验、
// 其它（数值类型）按数值取值校验（见 omcgo/internal/config/parammodel/validator.go:118-155）。
export const STANDARD_DATA_TYPES = ['string', 'int', 'unsignedInt', 'boolean', 'dateTime'] as const;
export type StandardDataType = (typeof STANDARD_DATA_TYPES)[number];

// changeApplies 枚举取值（大写，对齐字典 standard-model.xml）。
export const STANDARD_CHANGE_APPLIES = ['Immediate', 'OnReboot'] as const;
export type StandardChangeApplies = (typeof STANDARD_CHANGE_APPLIES)[number];

// min/max 是否按「字符串长度」语义（dataType==='string'）显示，否则按「数值取值」语义。
export function isStringDataType(dataType?: string): boolean {
  return dataType === 'string';
}

// min/max 取值范围语义分档（ISSUE-488 验收细化）：
//   'length' = 字符串长度范围（dataType==='string'）
//   'value'  = 数值取值范围（int / unsignedInt；未知或历史遗留值保守按数值，便于编辑旧数据）
//   'none'   = 无取值范围（boolean / dateTime）—— min/max 不适用，应禁用
export type DataTypeRangeKind = 'length' | 'value' | 'none';
export function dataTypeRangeKind(dataType?: string): DataTypeRangeKind {
  if (dataType === 'string') return 'length';
  if (dataType === 'boolean' || dataType === 'dateTime') return 'none';
  return 'value';
}

// unsignedInt 取值下界为 0（无符号）。
export function isUnsignedDataType(dataType?: string): boolean {
  return dataType === 'unsignedInt';
}

export interface DiscoveredVersion {
  softwareVersion: string;
  totalRows: number;
  lastSeenAt?: string;
}

export type TranslateDirection = 'to_private' | 'to_standard';

export interface TranslateRequest {
  productId: string;
  softwareVersion?: string;
  direction: TranslateDirection;
  paths: string[];
}

export interface TranslateItem {
  source: string;
  target?: string;
  matched: boolean;
  fromDiscovered?: boolean;
  reason?: string;
}

export interface TranslateResponse {
  productId: string;
  softwareVersion?: string;
  direction: TranslateDirection;
  results: TranslateItem[];
  source?: string;
}

export interface CreateMappingInput {
  standardPath: string;
  privatePath: string;
  entryType: string;
  access: string;
  dataType: string;
  changeApplies?: string;
  minValue?: string;
  maxValue?: string;
  isStorable?: boolean;
  isActive?: boolean;
  softwareVersion?: string;
}

export type UpdateMappingInput = Partial<CreateMappingInput>;

export interface UpsertStandardInput {
  standardPath: string;
  entryType: string;
  access: string;
  dataType: string;
  changeApplies?: string;
  minValue?: string;
  maxValue?: string;
}

export interface UpdateParamModelInput {
  description?: string;
  isActive?: boolean;
}

export interface StandardParamFilter {
  keyword?: string;
  entryType?: string;
}
