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
  // T-0178: 由后端 source.go 派生,前端不重新推导(唯一真值源在后端)。
  // source: builtin → 内置出厂 XML(数据来自镜像);
  //         custom  → 自定义上传 XML(数据来自 host 持久化目录);
  //         unknown → 历史无前缀数据(迁移前的旧行),保守对待。
  // deletable: 删除按钮可见性。仅 custom 行 true,内置/未知一律置灰 + Tooltip。
  source: ParamModelSource;
  deletable: boolean;
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
