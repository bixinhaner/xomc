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
