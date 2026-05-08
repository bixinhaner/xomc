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
  indicatorDeviceType: string;
  indicatorPlatform: string;
  alarmNeType: string;
  enableFiletype11: boolean;
  deviceAttrsOverride: DeviceAttrsOverride;
  enableUnknownAlarm: boolean;
  deviceCount: number;
}

export interface ProductPattern {
  id: string;
  productId: string;
  productClass: string;
  sortOrder: number;
  isActive: boolean;
}

export interface MatchOrderRow {
  patternId: string;
  productId: string;
  productName: string;
  productClass: string;
  sortOrder: number;
  isActive: boolean;
}

export interface OrphanDevice {
  id: string;
  serialNumber: string;
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
}

export interface UpdateProductInput extends Partial<CreateProductInput> {
  clearParamModel?: boolean;
}

export interface ProductListFilter {
  vendor?: string;
  tech?: string;
  keyword?: string;
}
