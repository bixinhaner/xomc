import http from '../http';
import type {
  IndicatorInfo,
  IndicatorListFilter,
  IndicatorGroup,
  PlatformFormula,
  IndicatorUnit,
  CreateIndicatorInput,
  UpdateIndicatorInput,
  CreateGroupInput,
  UpdateGroupInput,
  EnabledIndicatorsRequest,
  UnitInput,
  DeviceType,
} from '../../types/indicatorLibrary';

interface BackendIndicator {
  id: string;
  name: string;
  cn_name?: string;
  en_name?: string;
  group_id?: string;
  group_name?: string;
  counter_type?: string;
  indicator_level?: string;
  unit?: string;
  description?: string;
  is_counter?: boolean;
  product_type?: string;
  operator_code?: string;
  is_enabled?: boolean;
  device_type?: DeviceType;
}

interface BackendGroup {
  id: string;
  name: string;
  parent_id?: string;
  description?: string;
  operator_code?: string;
  device_type?: DeviceType;
  children?: BackendGroup[];
}

interface BackendFormula {
  platform_name: string;
  indicator_id: string;
  formula: string;
  description?: string;
}

interface BackendUnit {
  id: string;
  en_name: string;
  cn_name: string;
}

function mapIndicator(b: BackendIndicator, deviceType: DeviceType): IndicatorInfo {
  return {
    id: b.id,
    name: b.name,
    cnName: b.cn_name,
    enName: b.en_name,
    groupId: b.group_id,
    groupName: b.group_name,
    counterType: b.counter_type,
    indicatorLevel: b.indicator_level,
    unit: b.unit,
    description: b.description,
    isCounter: b.is_counter,
    productClass: b.product_type,
    operatorCode: b.operator_code,
    isEnabled: b.is_enabled,
    deviceType: b.device_type ?? deviceType,
  };
}

function mapGroup(b: BackendGroup, deviceType: DeviceType): IndicatorGroup {
  return {
    id: b.id,
    name: b.name,
    parentId: b.parent_id,
    description: b.description,
    operatorCode: b.operator_code,
    deviceType: b.device_type ?? deviceType,
    children: b.children?.map((c) => mapGroup(c, deviceType)),
  };
}

function mapFormula(b: BackendFormula): PlatformFormula {
  return {
    platformName: b.platform_name,
    indicatorId: b.indicator_id,
    formula: b.formula,
    description: b.description,
  };
}

function mapUnit(b: BackendUnit): IndicatorUnit {
  return { id: b.id, enName: b.en_name, cnName: b.cn_name };
}

function indicatorPayload(
  input: CreateIndicatorInput | UpdateIndicatorInput
): Record<string, unknown> {
  const p: Record<string, unknown> = {};
  if ('id' in input && input.id !== undefined) p.id = input.id;
  if (input.name !== undefined) p.name = input.name;
  if (input.cnName !== undefined) p.cn_name = input.cnName;
  if (input.enName !== undefined) p.en_name = input.enName;
  if (input.groupId !== undefined) p.group_id = input.groupId;
  if (input.counterType !== undefined) p.counter_type = input.counterType;
  if (input.indicatorLevel !== undefined) p.indicator_level = input.indicatorLevel;
  if (input.unit !== undefined) p.unit = input.unit;
  if (input.description !== undefined) p.description = input.description;
  if (input.productClass !== undefined) p.product_type = input.productClass;
  if (input.operatorCode !== undefined) p.operator_code = input.operatorCode;
  return p;
}

function groupPayload(input: CreateGroupInput | UpdateGroupInput): Record<string, unknown> {
  const p: Record<string, unknown> = {};
  if ('id' in input && input.id !== undefined) p.id = input.id;
  if (input.name !== undefined) p.name = input.name;
  if (input.parentId !== undefined) p.parent_id = input.parentId;
  if (input.description !== undefined) p.description = input.description;
  if (input.operatorCode !== undefined) p.operator_code = input.operatorCode;
  return p;
}

export const indicatorLibraryApi = {
  async list(deviceType: DeviceType, filter?: IndicatorListFilter): Promise<{
    items: IndicatorInfo[];
    total: number;
  }> {
    const params: Record<string, unknown> = { deviceType };
    if (filter?.groupId) params.groupId = filter.groupId;
    if (filter?.keyword) params.keyword = filter.keyword;
    if (filter?.operatorCode) params.operatorCode = filter.operatorCode;
    if (filter?.productClass) params.productClass = filter.productClass;
    if (filter?.indicatorLevel) params.indicatorLevel = filter.indicatorLevel;
    if (filter?.isEnabled !== undefined) params.isEnabled = filter.isEnabled;
    if (filter?.isCounter !== undefined) params.isCounter = filter.isCounter;
    if (filter?.platformName) params.platformName = filter.platformName;
    if (filter?.page) params.page = filter.page;
    if (filter?.pageSize) params.pageSize = filter.pageSize;

    const { data } = await http.get<{ items: BackendIndicator[]; total: number }>(
      '/indicators',
      { params }
    );
    return {
      items: (data.items || []).map((b) => mapIndicator(b, deviceType)),
      total: data.total || 0,
    };
  },

  async get(deviceType: DeviceType, id: string): Promise<IndicatorInfo> {
    const { data } = await http.get<BackendIndicator>(`/indicators/${id}`, { params: { deviceType } });
    return mapIndicator(data, deviceType);
  },

  async listPlatforms(deviceType: DeviceType): Promise<{ items: string[]; total: number }> {
    const { data } = await http.get<{ items: string[]; total: number }>('/indicators/platforms', {
      params: { deviceType },
    });
    return { items: data.items || [], total: data.total || 0 };
  },

  async create(deviceType: DeviceType, input: CreateIndicatorInput): Promise<IndicatorInfo> {
    const { data } = await http.post<BackendIndicator>('/indicators', indicatorPayload(input), {
      params: { deviceType },
    });
    return mapIndicator(data, deviceType);
  },

  async update(deviceType: DeviceType, id: string, input: UpdateIndicatorInput): Promise<IndicatorInfo> {
    const { data } = await http.put<BackendIndicator>(`/indicators/${id}`, indicatorPayload(input), {
      params: { deviceType },
    });
    return mapIndicator(data, deviceType);
  },

  async delete(deviceType: DeviceType, id: string): Promise<void> {
    await http.delete(`/indicators/${id}`, { params: { deviceType } });
  },

  async listFormulas(deviceType: DeviceType, indicatorId: string): Promise<{ items: PlatformFormula[]; total: number }> {
    const { data } = await http.get<{ items: BackendFormula[]; total: number }>(
      `/indicators/${indicatorId}/formulas`,
      { params: { deviceType } }
    );
    return {
      items: (data.items || []).map(mapFormula),
      total: data.total || 0,
    };
  },

  async getFormula(deviceType: DeviceType, indicatorId: string, platform: string): Promise<PlatformFormula> {
    const { data } = await http.get<BackendFormula>(
      `/indicators/${indicatorId}/formulas/${encodeURIComponent(platform)}`,
      { params: { deviceType } }
    );
    return mapFormula(data);
  },

  async upsertFormula(
    deviceType: DeviceType,
    indicatorId: string,
    platform: string,
    formula: string
  ): Promise<{ indicatorId: string; platform: string; formula: string }> {
    const { data } = await http.post<{ indicator_id: string; platform: string; formula: string }>(
      `/indicators/${indicatorId}/formulas`,
      { platform, formula },
      { params: { deviceType } }
    );
    return { indicatorId: data.indicator_id, platform: data.platform, formula: data.formula };
  },

  async deleteFormula(deviceType: DeviceType, indicatorId: string, platform: string): Promise<void> {
    await http.delete(`/indicators/${indicatorId}/formulas/${encodeURIComponent(platform)}`, {
      params: { deviceType },
    });
  },

  async listGroups(deviceType: DeviceType, operatorCode?: string): Promise<{ items: IndicatorGroup[]; total: number }> {
    const params: Record<string, unknown> = { deviceType };
    if (operatorCode) params.operatorCode = operatorCode;
    const { data } = await http.get<{ items: BackendGroup[]; total: number }>('/indicator-groups', { params });
    return {
      items: (data.items || []).map((b) => mapGroup(b, deviceType)),
      total: data.total || 0,
    };
  },

  async createGroup(deviceType: DeviceType, input: CreateGroupInput): Promise<IndicatorGroup> {
    const { data } = await http.post<BackendGroup>('/indicator-groups', groupPayload(input), {
      params: { deviceType },
    });
    return mapGroup(data, deviceType);
  },

  async updateGroup(deviceType: DeviceType, id: string, input: UpdateGroupInput): Promise<IndicatorGroup> {
    const { data } = await http.put<BackendGroup>(`/indicator-groups/${id}`, groupPayload(input), {
      params: { deviceType },
    });
    return mapGroup(data, deviceType);
  },

  async deleteGroup(deviceType: DeviceType, id: string): Promise<void> {
    await http.delete(`/indicator-groups/${id}`, { params: { deviceType } });
  },

  async listEnabled(deviceType: DeviceType, operatorCode: string): Promise<{
    items: string[];
    total: number;
    operatorCode: string;
  }> {
    const { data } = await http.get<{ items: string[]; total: number; operator_code: string }>(
      '/enabled-indicators',
      { params: { deviceType, operatorCode } }
    );
    return {
      items: data.items || [],
      total: data.total || 0,
      operatorCode: data.operator_code,
    };
  },

  async setEnabled(request: EnabledIndicatorsRequest): Promise<{ updated: number; operatorCode: string; enable: boolean }> {
    const { data } = await http.put<{ updated: number; operator_code: string; enable: boolean }>(
      '/enabled-indicators',
      { indicator_ids: request.indicatorIds, enable: request.enable },
      { params: { deviceType: request.deviceType, operatorCode: request.operatorCode } }
    );
    return {
      updated: data.updated,
      operatorCode: data.operator_code,
      enable: data.enable,
    };
  },

  async listUnits(): Promise<{ items: IndicatorUnit[]; total: number }> {
    const { data } = await http.get<{ items: BackendUnit[]; total: number }>('/indicator-units');
    return {
      items: (data.items || []).map(mapUnit),
      total: data.total || 0,
    };
  },

  async upsertUnit(input: UnitInput): Promise<IndicatorUnit> {
    const { data } = await http.post<BackendUnit>('/indicator-units', {
      id: input.id,
      en_name: input.enName,
      cn_name: input.cnName,
    });
    return mapUnit(data);
  },

  async updateUnit(id: string, input: UnitInput): Promise<IndicatorUnit> {
    const { data } = await http.put<BackendUnit>(`/indicator-units/${id}`, {
      id: input.id,
      en_name: input.enName,
      cn_name: input.cnName,
    });
    return mapUnit(data);
  },

  async deleteUnit(id: string): Promise<void> {
    await http.delete(`/indicator-units/${id}`);
  },

  async cacheRefresh(): Promise<{ refreshed: boolean; note?: string }> {
    const { data } = await http.post<{ refreshed: boolean; note?: string }>('/indicators/cache/refresh');
    return data;
  },

  async importDirectory(): Promise<{ reloaded: string }> {
    const { data } = await http.post<{ reloaded: string }>('/indicators/import-directory');
    return data;
  },
};
