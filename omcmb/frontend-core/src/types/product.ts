// 产品装配件类型（T-0098-P4 数据字典平台化）
// 后端：internal/product/handler.go productView 等 DTO

export interface DeviceAttrsOverride {
  data_type?: boolean;
  access?: boolean;
  min_value?: boolean;
  max_value?: boolean;
  change_applies?: boolean;
  [key: string]: boolean | undefined;
}

export interface Product {
  id: string;
  name: string;
  vendor: string;
  tech: string;
  radioModes: string;
  description: string;
  paramModelId?: string;
  paramModelName?: string; // 后端反查 param_models.name；列表展示「参数模型库名称」
  indicatorDeviceType: string;
  indicatorPlatform: string;
  alarmNeType: string;
  enableFiletype11: boolean;
  deviceAttrsOverride: DeviceAttrsOverride;
  enableUnknownAlarm: boolean;
  deviceCount: number;
  isBuiltin: boolean; // true=products.xml 装配的内置产品，前端禁止删除
  patterns: string[]; // active 正则（按 sort_order 升序）；列表页直接展示用
}

/** 正则来源：builtin=products.xml 内置(UI 只读，重灌覆盖)；custom=UI 新增(重灌保留)。 */
export type ProductPatternSource = 'builtin' | 'custom';

export interface ProductPattern {
  id: string;
  productId: string;
  productClass: string;
  sortOrder: number;
  isActive: boolean;
  source: ProductPatternSource;
  /** deletable=true 仅 custom；前端据此决定编辑/删除/移动按钮可用性。 */
  deletable: boolean;
}

export interface MatchOrderRow {
  patternId: string;
  productId: string;
  productName: string;
  productClass: string;
  sortOrder: number;
  isActive: boolean;
  source: ProductPatternSource;
}

export interface OrphanDevice {
  id: string;
  serialNumber: string;
  deviceName?: string;
  oui: string;
  productClass: string;
  carrier: string;
  manufacturer: string;
  lastInformAt?: string;
}

export interface ProductMatchResult {
  matched: boolean;
  product?: Product;
  matchedPattern?: string;
  globalOrder?: number;
  productClass?: string;
}

export interface CreateProductInput {
  name: string;
  vendor?: string;
  tech?: string;
  radioModes?: string;
  description?: string;
  paramModelName?: string;
  paramModelId?: string;
  indicatorDeviceType: string;
  indicatorPlatform: string;
  alarmNeType: string;
  enableFiletype11?: boolean;
  deviceAttrsOverride?: DeviceAttrsOverride;
  enableUnknownAlarm?: boolean;
  patterns?: string[]; // 可选；后端事务内随 product 一并创建
}

export interface UpdateProductInput extends Partial<CreateProductInput> {
  clearParamModel?: boolean;
}

export interface ProductListFilter {
  vendor?: string;
  tech?: string;
  keyword?: string;
}
