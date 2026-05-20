import type {
  DeviceType,
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
} from '../../types/indicatorLibrary';
import {
  mockIndicators,
  mockGroups,
  mockFormulas,
  mockEnabledIndicators,
  mockUnits,
} from '../data/indicatorLibrary';

const indicators: Record<DeviceType, IndicatorInfo[]> = {
  ENB: [...mockIndicators.ENB],
  GSM: [...mockIndicators.GSM],
  GNB: [...mockIndicators.GNB],
};

const groups: Record<DeviceType, IndicatorGroup[]> = {
  ENB: [...mockGroups.ENB],
  GSM: [...mockGroups.GSM],
  GNB: [...mockGroups.GNB],
};

const formulas: Record<string, PlatformFormula[]> = JSON.parse(JSON.stringify(mockFormulas));

const enabled: Record<DeviceType, Record<string, string[]>> = {
  ENB: { ...mockEnabledIndicators.ENB },
  GSM: { ...mockEnabledIndicators.GSM },
  GNB: { ...mockEnabledIndicators.GNB },
};

let units: IndicatorUnit[] = [...mockUnits];

function clone<T>(v: T): T {
  return JSON.parse(JSON.stringify(v)) as T;
}

export const indicatorLibraryService = {
  async list(deviceType: DeviceType, filter?: IndicatorListFilter) {
    let items = [...indicators[deviceType]];
    if (filter?.groupId) items = items.filter((i) => i.groupId === filter.groupId);
    if (filter?.keyword) {
      const k = filter.keyword.toLowerCase();
      items = items.filter((i) => (i.id + i.name + (i.enName || '')).toLowerCase().includes(k));
    }
    if (filter?.operatorCode) items = items.filter((i) => i.operatorCode === filter.operatorCode);
    if (filter?.isEnabled !== undefined) items = items.filter((i) => Boolean(i.isEnabled) === filter.isEnabled);
    if (filter?.platformName) {
      const platform = filter.platformName;
      items = items.filter((i) => (formulas[i.id] || []).some((f) => f.platformName === platform));
    }
    return { items: clone(items), total: items.length };
  },

  async listPlatforms(deviceType: DeviceType) {
    const ids = new Set(indicators[deviceType].map((i) => i.id));
    const names = new Set<string>();
    for (const id of ids) {
      for (const f of formulas[id] || []) names.add(f.platformName);
    }
    const items = Array.from(names).sort();
    return { items, total: items.length };
  },

  async get(deviceType: DeviceType, id: string): Promise<IndicatorInfo> {
    const i = indicators[deviceType].find((x) => x.id === id);
    if (!i) throw new Error(`indicator ${id} not found`);
    return clone(i);
  },

  async create(deviceType: DeviceType, input: CreateIndicatorInput): Promise<IndicatorInfo> {
    const created: IndicatorInfo = {
      id: input.id,
      name: input.name,
      cnName: input.cnName,
      enName: input.enName,
      groupId: input.groupId,
      counterType: input.counterType,
      indicatorLevel: input.indicatorLevel,
      unit: input.unit,
      description: input.description,
      productType: input.productType,
      operatorCode: input.operatorCode || 'default',
      isEnabled: false,
      deviceType,
    };
    indicators[deviceType].push(created);
    return clone(created);
  },

  async update(deviceType: DeviceType, id: string, input: UpdateIndicatorInput): Promise<IndicatorInfo> {
    const arr = indicators[deviceType];
    const idx = arr.findIndex((i) => i.id === id);
    if (idx < 0) throw new Error(`indicator ${id} not found`);
    arr[idx] = {
      ...arr[idx],
      ...(input.name !== undefined && { name: input.name }),
      ...(input.cnName !== undefined && { cnName: input.cnName }),
      ...(input.enName !== undefined && { enName: input.enName }),
      ...(input.groupId !== undefined && { groupId: input.groupId }),
      ...(input.counterType !== undefined && { counterType: input.counterType }),
      ...(input.indicatorLevel !== undefined && { indicatorLevel: input.indicatorLevel }),
      ...(input.unit !== undefined && { unit: input.unit }),
      ...(input.description !== undefined && { description: input.description }),
      ...(input.productType !== undefined && { productType: input.productType }),
      ...(input.operatorCode !== undefined && { operatorCode: input.operatorCode }),
    };
    return clone(arr[idx]);
  },

  async delete(deviceType: DeviceType, id: string): Promise<void> {
    indicators[deviceType] = indicators[deviceType].filter((i) => i.id !== id);
    delete formulas[id];
  },

  async listFormulas(_deviceType: DeviceType, indicatorId: string) {
    const items = formulas[indicatorId] || [];
    return { items: clone(items), total: items.length };
  },

  async getFormula(_deviceType: DeviceType, indicatorId: string, platform: string): Promise<PlatformFormula> {
    const arr = formulas[indicatorId] || [];
    const f = arr.find((x) => x.platformName === platform);
    if (!f) throw new Error(`formula ${indicatorId}/${platform} not found`);
    return clone(f);
  },

  async upsertFormula(_deviceType: DeviceType, indicatorId: string, platform: string, formula: string) {
    const arr = formulas[indicatorId] || [];
    const idx = arr.findIndex((x) => x.platformName === platform);
    if (idx >= 0) arr[idx].formula = formula;
    else arr.push({ platformName: platform, indicatorId, formula });
    formulas[indicatorId] = arr;
    return { indicatorId, platform, formula };
  },

  async deleteFormula(_deviceType: DeviceType, indicatorId: string, platform: string): Promise<void> {
    formulas[indicatorId] = (formulas[indicatorId] || []).filter((x) => x.platformName !== platform);
  },

  async listGroups(deviceType: DeviceType, operatorCode?: string) {
    let items = [...groups[deviceType]];
    if (operatorCode) items = items.filter((g) => g.operatorCode === operatorCode);
    return { items: clone(items), total: items.length };
  },

  async createGroup(deviceType: DeviceType, input: CreateGroupInput): Promise<IndicatorGroup> {
    const created: IndicatorGroup = {
      id: input.id,
      name: input.name,
      parentId: input.parentId,
      description: input.description,
      operatorCode: input.operatorCode || 'default',
      deviceType,
    };
    groups[deviceType].push(created);
    return clone(created);
  },

  async updateGroup(deviceType: DeviceType, id: string, input: UpdateGroupInput): Promise<IndicatorGroup> {
    const arr = groups[deviceType];
    const idx = arr.findIndex((g) => g.id === id);
    if (idx < 0) throw new Error(`group ${id} not found`);
    arr[idx] = {
      ...arr[idx],
      ...(input.name !== undefined && { name: input.name }),
      ...(input.parentId !== undefined && { parentId: input.parentId }),
      ...(input.description !== undefined && { description: input.description }),
      ...(input.operatorCode !== undefined && { operatorCode: input.operatorCode }),
    };
    return clone(arr[idx]);
  },

  async deleteGroup(deviceType: DeviceType, id: string): Promise<void> {
    groups[deviceType] = groups[deviceType].filter((g) => g.id !== id);
  },

  async listEnabled(deviceType: DeviceType, operatorCode: string) {
    const ids = enabled[deviceType][operatorCode] || [];
    return { items: [...ids], total: ids.length, operatorCode };
  },

  async setEnabled(request: EnabledIndicatorsRequest) {
    const cur = new Set(enabled[request.deviceType][request.operatorCode] || []);
    if (request.enable) request.indicatorIds.forEach((id) => cur.add(id));
    else request.indicatorIds.forEach((id) => cur.delete(id));
    enabled[request.deviceType][request.operatorCode] = Array.from(cur);
    return {
      updated: request.indicatorIds.length,
      operatorCode: request.operatorCode,
      enable: request.enable,
    };
  },

  async listUnits() {
    return { items: clone(units), total: units.length };
  },

  async upsertUnit(input: UnitInput): Promise<IndicatorUnit> {
    const idx = units.findIndex((u) => u.id === input.id);
    const item: IndicatorUnit = { id: input.id, enName: input.enName, cnName: input.cnName };
    if (idx >= 0) units[idx] = item;
    else units.push(item);
    return clone(item);
  },

  async updateUnit(id: string, input: UnitInput): Promise<IndicatorUnit> {
    const idx = units.findIndex((u) => u.id === id);
    if (idx < 0) throw new Error(`unit ${id} not found`);
    units[idx] = { id: input.id, enName: input.enName, cnName: input.cnName };
    return clone(units[idx]);
  },

  async deleteUnit(id: string): Promise<void> {
    units = units.filter((u) => u.id !== id);
  },

  async cacheRefresh() {
    return { refreshed: true, note: 'mock refresh' };
  },

  async importDirectory() {
    return { reloaded: 'indicator' };
  },
};
