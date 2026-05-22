// ── Indicator Group (功能集) ──────────────────────────────────────────────────

export interface IndicatorGroup {
  id: string;
  enName: string;
  cnName: string;
  operatorCode: string;
  isBuildIn: boolean;
  description: string;
  parentId: string;
  deviceType: 'ENB' | 'GSM' | 'GNB';
  indicatorCount?: number;
  children?: IndicatorGroup[];
}

// ── Tree Node (for antd Tree) ────────────────────────────────────────────────

export interface IndicatorTreeNode {
  id: string;
  text: string;
  deviceType?: string;
  children: IndicatorTreeNode[];
}

// ── Perf Indicator (指标) ────────────────────────────────────────────────────

export interface PerfIndicator {
  kpiId: string;
  kpiName: string;
  kpiNameEn?: string;
  kpiNameZh?: string;
  catagoryId: string;
  catagoryName: string;
  productClass: string;
  custName: string;
  indicatorLevel: string;
  unit: string;
  isCustomize: boolean;
  isCounter: boolean;
  isEnable: boolean;
  indicatorType: string;
  arithmetic: string;
  definition: string;
  definitionEn?: string;
  definitionZh?: string;
  statisType: string;
  updater: string;
  updateTime: string;
  calculatingStatus: string;
  keys?: string[];
  values?: string[];
  names?: string[];
  generalColor?: string;
  seriousColor?: string;
  indicatorProductRela?: Record<string, string>;
}

// ── Request / Filter types ───────────────────────────────────────────────────

export interface IndicatorListParams {
  catagoryId?: string;
  searchText?: string;
  productClass?: string;
  isEnable?: string;
  indicatorType?: string;
  indicatorLevel?: string;
  deviceType?: string;
  timeZone?: string;
  page: number;
  rows: number;
  sort?: string;
  order?: string;
}

export interface IndicatorGroupTreeParams {
  deviceType: 'ENB' | 'GSM' | 'GNB';
  operatorCode?: string;
}

export interface IndicatorCreateParams {
  kpiId?: string;
  indicatorType?: string;
  kpiName: string;
  catagoryId: string;
  unit: string;
  statisType: string;
  arithmetic: string;
  productClass?: string;
  isEnable?: string;
  definition?: string;
  custName?: string;
  indicatorLevel?: string;
  updater?: string;
}

export interface IndicatorGroupCreateParams {
  deviceType: 'ENB' | 'GSM' | 'GNB';
  enName: string;
  cnName: string;
  parentId: string;
  operatorCode?: string;
}

export interface IndicatorGroupUpdateParams {
  enName?: string;
  cnName?: string;
  description?: string;
}

export interface EnableIndicatorsParams {
  deviceType: 'ENB' | 'GSM' | 'GNB';
  operatorCode: string;
  indicatorIds: string[];
  enable: boolean;
}

export interface CustNameUpdateParams {
  deviceType: 'ENB' | 'GSM' | 'GNB';
  operatorCode: string;
  perfId: string;
  custName: string;
}

// ── Indicator Unit ───────────────────────────────────────────────────────────

export interface IndicatorUnit {
  id: string;
  enName: string;
  cnName: string;
}

// ── Indicator Type ───────────────────────────────────────────────────────────

export interface IndicatorType {
  id: string;
  name: string;
  description?: string;
}
