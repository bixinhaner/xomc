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
import { mockIndicatorGroups, mockIndicators, mockIndicatorUnits, mockIndicatorTypes } from '../data/indicator';
import { delay, paginate, generateId } from '../utils';

// ── Mutable state for mutations ──────────────────────────────────────────────

let groups = structuredClone(mockIndicatorGroups);
let indicators = [...mockIndicators];

function resetState(): void {
  groups = structuredClone(mockIndicatorGroups);
  indicators = [...mockIndicators];
}

// ── Service ──────────────────────────────────────────────────────────────────

export const indicatorService = {
  // ── Group Tree ────────────────────────────────────────────────────────────

  async getIndicatorGroupTree(params: IndicatorGroupTreeParams): Promise<IndicatorGroup[]> {
    await delay(80, 150);
    const root = groups[params.deviceType];
    if (!root) return [];
    return [root];
  },

  // ── Group CRUD ────────────────────────────────────────────────────────────

  async addIndicatorGroup(params: IndicatorGroupCreateParams): Promise<IndicatorGroup> {
    await delay(150, 300);
    const root = groups[params.deviceType];
    const newGroup: IndicatorGroup = {
      id: generateId('grp'),
      enName: params.enName,
      cnName: params.cnName,
      operatorCode: params.operatorCode || 'cmcc',
      isBuildIn: false,
      description: '',
      parentId: params.parentId,
      deviceType: params.deviceType,
      indicatorCount: 0,
      children: [],
    };
    if (root) {
      if (params.parentId === root.id) {
        root.children = root.children || [];
        root.children.push(newGroup);
      } else {
        const parent = findGroupById(root.children || [], params.parentId);
        if (parent) {
          parent.children = parent.children || [];
          parent.children.push(newGroup);
        }
      }
    }
    return newGroup;
  },

  async getIndicatorGroupInfo(groupId: string, _deviceType?: string): Promise<IndicatorGroup> {
    await delay(80, 150);
    for (const root of Object.values(groups)) {
      const found = findGroupById([root], groupId);
      if (found) return found;
    }
    throw new Error(`Group ${groupId} not found`);
  },

  async modifyIndicatorGroup(
    groupId: string,
    params: IndicatorGroupUpdateParams,
    _deviceType?: string,
  ): Promise<IndicatorGroup> {
    await delay(150, 300);
    for (const root of Object.values(groups)) {
      const found = findGroupById([root], groupId);
      if (found) {
        if (params.enName !== undefined) found.enName = params.enName;
        if (params.cnName !== undefined) found.cnName = params.cnName;
        if (params.description !== undefined) found.description = params.description;
        return found;
      }
    }
    throw new Error(`Group ${groupId} not found`);
  },

  async deleteIndicatorGroup(groupId: string, _deviceType?: string): Promise<void> {
    await delay(150, 300);
    for (const root of Object.values(groups)) {
      removeGroupById(root.children || [], groupId);
    }
  },

  // ── Indicator List ────────────────────────────────────────────────────────

  async getIndicatorListByPage(params: IndicatorListParams): Promise<PageResponse<PerfIndicator>> {
    await delay(100, 200);
    let filtered = [...indicators];

    // Filter by device type (derived from catagoryId prefix)
    if (params.deviceType) {
      const prefix = params.deviceType.toLowerCase();
      filtered = filtered.filter((ind) => ind.catagoryId.startsWith(prefix));
    }

    // Filter by category (group)
    if (params.catagoryId) {
      filtered = filtered.filter((ind) => ind.catagoryId === params.catagoryId);
    }

    // Filter by search text
    if (params.searchText) {
      const lower = params.searchText.toLowerCase();
      filtered = filtered.filter(
        (ind) =>
          ind.kpiId.toLowerCase().includes(lower) ||
          ind.kpiName.toLowerCase().includes(lower),
      );
    }

    // Filter by product type
    if (params.productClass) {
      filtered = filtered.filter((ind) => ind.productClass === params.productClass);
    }

    // Filter by enable status
    if (params.isEnable !== undefined && params.isEnable !== '') {
      const isEnabled = params.isEnable === '1' || params.isEnable === 'true';
      filtered = filtered.filter((ind) => ind.isEnable === isEnabled);
    }

    // Filter by indicator level
    if (params.indicatorLevel) {
      filtered = filtered.filter((ind) => ind.indicatorLevel === params.indicatorLevel);
    }

    return paginate(filtered, params.page, params.rows);
  },

  // ── Effective Indicators ──────────────────────────────────────────────────

  async getEffectiveIndicators(
    deviceType: string,
    _operatorCode?: string,
  ): Promise<PerfIndicator[]> {
    await delay(80, 150);
    const prefix = deviceType.toLowerCase();
    return indicators.filter((ind) => ind.catagoryId.startsWith(prefix) && ind.isEnable);
  },

  // ── Indicator Detail ──────────────────────────────────────────────────────

  async getIndicatorInfo(indicatorId: string, _deviceType?: string): Promise<PerfIndicator> {
    await delay(80, 150);
    const found = indicators.find((ind) => ind.kpiId === indicatorId);
    if (!found) throw new Error(`Indicator ${indicatorId} not found`);
    return found;
  },

  // ── Indicator Create / Update ─────────────────────────────────────────────

  async addOrModifyIndicator(
    params: IndicatorCreateParams & { deviceType?: string },
  ): Promise<PerfIndicator> {
    await delay(200, 400);
    const isCounter = params.indicatorType === 'counter';

    if (params.kpiId) {
      // Update existing
      const idx = indicators.findIndex((ind) => ind.kpiId === params.kpiId);
      if (idx !== -1) {
        indicators[idx] = {
          ...indicators[idx],
          kpiName: params.kpiName,
          catagoryId: params.catagoryId,
          unit: params.unit,
          statisType: params.statisType,
          arithmetic: params.arithmetic,
          definition: params.definition || indicators[idx].definition,
          custName: params.custName || indicators[idx].custName,
          isEnable: params.isEnable === '1',
          isCounter,
          indicatorType: isCounter ? 'counter' : 'kpi',
          updateTime: new Date().toISOString().replace('T', ' ').substring(0, 19),
        };
        return indicators[idx];
      }
    }

    // Create new
    const newIndicator: PerfIndicator = {
      kpiId: generateId('kpi'),
      kpiName: params.kpiName,
      catagoryId: params.catagoryId,
      catagoryName: '',
      productClass: params.productClass || 'BBU',
      custName: params.custName || '',
      indicatorLevel: params.indicatorLevel || 'device',
      unit: params.unit,
      isCustomize: true,
      isCounter,
      isEnable: params.isEnable === '1',
      indicatorType: isCounter ? 'counter' : 'kpi',
      arithmetic: params.arithmetic,
      definition: params.definition || '',
      statisType: params.statisType,
      updater: 'admin',
      updateTime: new Date().toISOString().replace('T', ' ').substring(0, 19),
      calculatingStatus: isCounter ? '' : 'active',
    };
    indicators.push(newIndicator);
    return newIndicator;
  },

  // ── Indicator Delete ──────────────────────────────────────────────────────

  async deleteIndicator(indicatorId: string, _deviceType?: string): Promise<void> {
    await delay(150, 300);
    indicators = indicators.filter((ind) => ind.kpiId !== indicatorId);
  },

  // ── Export ────────────────────────────────────────────────────────────────

  async exportAllIndicator(
    _deviceType: string,
    _groupId?: string,
    _operatorCode?: string,
  ): Promise<Blob> {
    await delay(300, 600);
    const csvContent = 'kpiId,kpiName,unit,arithmetic\n' +
      indicators.map((ind) => `${ind.kpiId},${ind.kpiName},${ind.unit},${ind.arithmetic}`).join('\n');
    return new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
  },

  // ── Custom Name Updates ───────────────────────────────────────────────────

  async updateBaseKpiCustName(params: CustNameUpdateParams): Promise<void> {
    await delay(100, 200);
    const idx = indicators.findIndex((ind) => ind.kpiId === params.perfId);
    if (idx !== -1) {
      indicators[idx].custName = params.custName;
    }
  },

  async updateEnbIndicatorsName(indicatorId: string, name: string): Promise<void> {
    await delay(100, 200);
    const idx = indicators.findIndex((ind) => ind.kpiId === indicatorId);
    if (idx !== -1) {
      indicators[idx].kpiName = name;
    }
  },

  async updateGsmIndicatorsName(indicatorId: string, name: string): Promise<void> {
    await delay(100, 200);
    const idx = indicators.findIndex((ind) => ind.kpiId === indicatorId);
    if (idx !== -1) {
      indicators[idx].kpiName = name;
    }
  },

  async updateGnbIndicatorsName(indicatorId: string, name: string): Promise<void> {
    await delay(100, 200);
    const idx = indicators.findIndex((ind) => ind.kpiId === indicatorId);
    if (idx !== -1) {
      indicators[idx].kpiName = name;
    }
  },

  // ── Unit & Type lookups ───────────────────────────────────────────────────

  async getIndicatorUnitList(): Promise<IndicatorUnit[]> {
    await delay(50, 100);
    return mockIndicatorUnits;
  },

  async getIndicatorGroupList(
    deviceType: string,
    _operatorCode?: string,
  ): Promise<IndicatorGroup[]> {
    await delay(80, 150);
    const root = groups[deviceType];
    if (!root) return [];
    return root.children || [];
  },

  async getIndicatorTypes(_deviceType: string): Promise<IndicatorType[]> {
    await delay(50, 100);
    return mockIndicatorTypes;
  },

  // ── Enable / Disable ─────────────────────────────────────────────────────

  async enableIndicator(params: EnableIndicatorsParams): Promise<void> {
    await delay(150, 300);
    for (const id of params.indicatorIds) {
      const idx = indicators.findIndex((ind) => ind.kpiId === id);
      if (idx !== -1) {
        indicators[idx].isEnable = true;
      }
    }
  },

  async disableIndicator(params: EnableIndicatorsParams): Promise<void> {
    await delay(150, 300);
    for (const id of params.indicatorIds) {
      const idx = indicators.findIndex((ind) => ind.kpiId === id);
      if (idx !== -1) {
        indicators[idx].isEnable = false;
      }
    }
  },

  async isIndicatorInTemplate(indicatorId: string): Promise<boolean> {
    await delay(50, 100);
    // Mock: some indicators are in templates
    return ['RRC_CONN_REQ', 'RRC_CONN_SUCC', 'NR_RRC_CONN_REQ'].includes(indicatorId);
  },

  // ── Reset (for testing) ──────────────────────────────────────────────────

  _resetState: resetState,
};

// ── Helpers ──────────────────────────────────────────────────────────────────

function findGroupById(groups: IndicatorGroup[], id: string): IndicatorGroup | null {
  for (const g of groups) {
    if (g.id === id) return g;
    if (g.children) {
      const found = findGroupById(g.children, id);
      if (found) return found;
    }
  }
  return null;
}

function removeGroupById(groups: IndicatorGroup[], id: string): boolean {
  const idx = groups.findIndex((g) => g.id === id);
  if (idx !== -1) {
    groups.splice(idx, 1);
    return true;
  }
  for (const g of groups) {
    if (g.children && removeGroupById(g.children, id)) {
      return true;
    }
  }
  return false;
}
