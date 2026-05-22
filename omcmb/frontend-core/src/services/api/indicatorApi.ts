import http from '../http';
import type {
  IndicatorGroup,
  PerfIndicator,
  IndicatorListParams,
  IndicatorGroupTreeParams,
  IndicatorCreateParams,
  IndicatorGroupCreateParams,
  IndicatorGroupUpdateParams,
  EnableIndicatorsParams,
  CustNameUpdateParams,
  IndicatorUnit,
  IndicatorType,
} from '../../types/indicator';
import type { PageResponse } from '../../types/pagination';

// ── Backend response types ───────────────────────────────────────────────────

interface BackendIndicatorGroup {
  id: string;
  en_name: string;
  cn_name: string;
  operator_code: string;
  is_build_in: string;
  description: string;
  parent_id: string;
  indicator_count: number;
  children?: BackendIndicatorGroup[];
}

interface BackendPerfIndicator {
  id: string;
  en_name: string;
  cn_name: string;
  en_description: string;
  cn_description: string;
  group_id: string;
  operator_code: string;
  data_type: string;
  unit_id: string;
  updator: string;
  is_build_in: string;
  is_counter: string;
  arithmetic: string;
  statis_type: string;
  calculating_status: string;
  product_types: string;
  indicator_level: string;
  is_enabled: boolean;
  cust_name: string;
  created_at: string;
  updated_at: string;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

interface BackendUnit {
  id: string;
  en_name: string;
  cn_name: string;
}

interface BackendIndicatorType {
  id: string;
  name: string;
  description?: string;
}

// ── Mapping functions ────────────────────────────────────────────────────────

function mapBackendGroup(g: BackendIndicatorGroup): IndicatorGroup {
  return {
    id: g.id,
    enName: g.en_name,
    cnName: g.cn_name,
    operatorCode: g.operator_code,
    isBuildIn: g.is_build_in === '1' || g.is_build_in === 'true',
    description: g.description,
    parentId: g.parent_id,
    deviceType: 'ENB', // derived from request context
    indicatorCount: g.indicator_count,
    children: g.children ? g.children.map(mapBackendGroup) : undefined,
  };
}

function mapBackendIndicator(ind: BackendPerfIndicator): PerfIndicator {
  return {
    kpiId: ind.id,
    kpiName: ind.cn_name || ind.en_name,
    kpiNameEn: ind.en_name,
    kpiNameZh: ind.cn_name,
    catagoryId: ind.group_id,
    catagoryName: '',
    productClass: ind.product_types,
    custName: ind.cust_name,
    indicatorLevel: ind.indicator_level,
    unit: ind.unit_id,
    isCustomize: ind.is_build_in !== '1' && ind.is_build_in !== 'true',
    isCounter: ind.is_counter === '1' || ind.is_counter === 'true',
    isEnable: ind.is_enabled,
    indicatorType: ind.is_counter === '1' || ind.is_counter === 'true' ? 'counter' : 'kpi',
    arithmetic: ind.arithmetic,
    definition: ind.cn_description || ind.en_description,
    definitionEn: ind.en_description,
    definitionZh: ind.cn_description,
    statisType: ind.statis_type,
    updater: ind.updator,
    updateTime: ind.updated_at,
    calculatingStatus: ind.calculating_status,
  };
}

function mapBackendUnit(u: BackendUnit): IndicatorUnit {
  return {
    id: u.id,
    enName: u.en_name,
    cnName: u.cn_name,
  };
}

// ── Helper: device-type-prefixed path ────────────────────────────────────────

function basePath(deviceType?: string): string {
  if (deviceType === 'GNB') return '/gnb/pm/indicatormg';
  return '/pm/indicatormg';
}

// ── Exported service ─────────────────────────────────────────────────────────

export const indicatorApi = {
  // ── Group Tree ────────────────────────────────────────────────────────────

  async getIndicatorGroupTree(params: IndicatorGroupTreeParams): Promise<IndicatorGroup[]> {
    const { data } = await http.post<BackendIndicatorGroup[]>(
      `${basePath(params.deviceType)}/getIndicatorGroupTree`,
      { device_type: params.deviceType, operator_code: params.operatorCode },
    );
    return (data || []).map(mapBackendGroup);
  },

  // ── Group CRUD ────────────────────────────────────────────────────────────

  async addIndicatorGroup(params: IndicatorGroupCreateParams): Promise<IndicatorGroup> {
    const payload = {
      device_type: params.deviceType,
      en_name: params.enName,
      cn_name: params.cnName,
      parent_id: params.parentId,
      operator_code: params.operatorCode,
    };
    const { data } = await http.post<BackendIndicatorGroup>(
      `${basePath(params.deviceType)}/addIndicatorGroup`,
      payload,
    );
    return mapBackendGroup(data);
  },

  async getIndicatorGroupInfo(groupId: string, deviceType?: string): Promise<IndicatorGroup> {
    const { data } = await http.post<BackendIndicatorGroup>(
      `${basePath(deviceType)}/getIndicatorGroupInfo`,
      { id: groupId },
    );
    return mapBackendGroup(data);
  },

  async modifyIndicatorGroup(
    groupId: string,
    params: IndicatorGroupUpdateParams,
    deviceType?: string,
  ): Promise<IndicatorGroup> {
    const payload: Record<string, unknown> = { id: groupId, device_type: (deviceType || 'ENB').toUpperCase() };
    if (params.enName !== undefined) payload.en_name = params.enName;
    if (params.cnName !== undefined) payload.cn_name = params.cnName;
    if (params.description !== undefined) payload.description = params.description;

    const { data } = await http.post<BackendIndicatorGroup>(
      `${basePath(deviceType)}/modifyIndicatorGroup`,
      payload,
    );
    return mapBackendGroup(data);
  },

  async deleteIndicatorGroup(groupId: string, deviceType?: string): Promise<void> {
    await http.post(`${basePath(deviceType)}/delIndicatorGroup`, {
      id: groupId,
      device_type: (deviceType || 'ENB').toUpperCase(),
    });
  },

  // ── Indicator List & Detail ───────────────────────────────────────────────

  async getIndicatorListByPage(
    params: IndicatorListParams,
  ): Promise<PageResponse<PerfIndicator>> {
    const payload: Record<string, unknown> = {
      device_type: params.deviceType,
      operator_code: 'default',
      page: params.page,
      page_size: params.rows,
    };
    if (params.catagoryId) payload.group_id = params.catagoryId;
    if (params.searchText) payload.keyword = params.searchText;
    if (params.productClass) payload.product_type = params.productClass;
    if (params.isEnable !== undefined && params.isEnable !== '') payload.is_enabled = params.isEnable;
    if (params.indicatorType !== undefined && params.indicatorType !== '') payload.is_counter = params.indicatorType;
    if (params.indicatorLevel) payload.indicator_level = params.indicatorLevel;

    const { data } = await http.post<BackendListResponse<BackendPerfIndicator>>(
      `${basePath(params.deviceType)}/getIndicatorListByPage`,
      payload,
    );

    return {
      items: (data.items || []).map(mapBackendIndicator),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getEffectiveIndicators(
    deviceType: string,
    operatorCode?: string,
  ): Promise<PerfIndicator[]> {
    const payload: Record<string, unknown> = { device_type: deviceType };
    if (operatorCode) payload.operator_code = operatorCode;

    const { data } = await http.post<BackendPerfIndicator[]>(
      `${basePath(deviceType)}/getEffectiveIndicators`,
      payload,
    );
    return (data || []).map(mapBackendIndicator);
  },

  async getIndicatorInfo(
    indicatorId: string,
    deviceType?: string,
  ): Promise<PerfIndicator> {
    const { data } = await http.post<BackendPerfIndicator>(
      `${basePath(deviceType)}/getIndicatorInfo`,
      { id: indicatorId, device_type: (deviceType || 'ENB').toUpperCase() },
    );
    return mapBackendIndicator(data);
  },

  // ── Indicator Create / Update ─────────────────────────────────────────────

  async addOrModifyIndicator(
    params: IndicatorCreateParams & { deviceType?: string },
  ): Promise<PerfIndicator> {
    const payload: Record<string, unknown> = {
      device_type: (params.deviceType || 'ENB').toUpperCase(),
      cn_name: params.kpiName,
      en_name: params.kpiName,
      group_id: params.catagoryId,
      unit_id: params.unit,
      statis_type: params.statisType,
      arithmetic: params.arithmetic,
    };
    if (params.kpiId) payload.id = params.kpiId;
    if (params.indicatorType) {
      payload.is_counter = params.indicatorType === 'counter' ? '1' : '0';
    }
    if (params.productClass) payload.product_types = params.productClass;
    if (params.indicatorLevel) payload.indicator_level = params.indicatorLevel;
    if (params.custName) payload.cust_name = params.custName;
    if (params.definition) payload.cn_description = params.definition;
    if (params.updater) payload.updator = params.updater;

    const { data } = await http.post<BackendPerfIndicator>(
      `${basePath(params.deviceType)}/addOrModifyIndicator`,
      payload,
    );
    return mapBackendIndicator(data);
  },

  // ── Indicator Delete ──────────────────────────────────────────────────────

  async deleteIndicator(
    indicatorId: string,
    deviceType?: string,
  ): Promise<void> {
    await http.post(`${basePath(deviceType)}/delIndicator`, { id: indicatorId, device_type: deviceType || 'ENB' });
  },

  // ── Export ────────────────────────────────────────────────────────────────

  async exportAllIndicator(
    deviceType: string,
    groupId?: string,
    operatorCode?: string,
  ): Promise<Blob> {
    const params: Record<string, unknown> = { device_type: deviceType };
    if (groupId) params.group_id = groupId;
    if (operatorCode) params.operator_code = operatorCode;

    const { data } = await http.post<Blob>(
      `${basePath(deviceType)}/exportAllIndicator`,
      params,
      { responseType: 'blob' },
    );
    return data;
  },

  // ── Custom Name Updates ───────────────────────────────────────────────────

  async updateBaseKpiCustName(params: CustNameUpdateParams): Promise<void> {
    await http.post(`${basePath(params.deviceType)}/updateBaseKpiCustName`, {
      device_type: params.deviceType,
      operator_code: params.operatorCode,
      perf_id: params.perfId,
      cust_name: params.custName,
    });
  },

  async updateEnbIndicatorsName(
    indicatorId: string,
    name: string,
  ): Promise<void> {
    await http.post('/pm/indicatormg/updateEnbIndicatorsName', {
      id: indicatorId,
      cn_name: name,
    });
  },

  async updateGsmIndicatorsName(
    indicatorId: string,
    name: string,
  ): Promise<void> {
    await http.post('/pm/indicatormg/updateGsmIndicatorsName', {
      id: indicatorId,
      cn_name: name,
    });
  },

  // ── Unit & Type lookups ───────────────────────────────────────────────────

  async getIndicatorUnitList(): Promise<IndicatorUnit[]> {
    const { data } = await http.get<BackendUnit[]>(
      '/pm/indicatormg/getIndicatorUnitList',
    );
    return (data || []).map(mapBackendUnit);
  },

  async getIndicatorGroupList(
    deviceType: string,
    operatorCode?: string,
  ): Promise<IndicatorGroup[]> {
    const payload: Record<string, unknown> = { device_type: deviceType };
    if (operatorCode) payload.operator_code = operatorCode;

    const { data } = await http.post<BackendIndicatorGroup[]>(
      `${basePath(deviceType)}/getIndicatorGroupList`,
      payload,
    );
    return (data || []).map(mapBackendGroup);
  },

  async getIndicatorTypes(
    deviceType: string,
  ): Promise<IndicatorType[]> {
    const { data } = await http.post<BackendIndicatorType[]>(
      `${basePath(deviceType)}/getIndicatorTypes`,
      { device_type: deviceType },
    );
    return (data || []).map((t) => ({
      id: t.id,
      name: t.name,
      description: t.description,
    }));
  },

  // ── Enable / Disable ─────────────────────────────────────────────────────

  async enableIndicator(params: EnableIndicatorsParams): Promise<void> {
    await http.post('/cell/perfmgmt/kpimanage/enableIndicator', {
      device_type: params.deviceType,
      operator_code: params.operatorCode,
      indicator_ids: params.indicatorIds,
      enable: true,
    });
  },

  async disableIndicator(params: EnableIndicatorsParams): Promise<void> {
    await http.post('/cell/perfmgmt/kpimanage/disableIndicator', {
      device_type: params.deviceType,
      operator_code: params.operatorCode,
      indicator_ids: params.indicatorIds,
      enable: false,
    });
  },

  async isIndicatorInTemplate(indicatorId: string): Promise<boolean> {
    const { data } = await http.get<{ in_template: boolean }>(
      '/cell/perfmgmt/kpimanage/isIndicatorInTemplate',
      { params: { indicator_id: indicatorId } },
    );
    return data.in_template;
  },

  // ── GNB-specific endpoints ────────────────────────────────────────────────

  async updateGnbIndicatorsName(
    indicatorId: string,
    name: string,
  ): Promise<void> {
    await http.post('/gnb/pm/indicatormg/updateGnbIndicatorsName', {
      id: indicatorId,
      cn_name: name,
    });
  },
};
