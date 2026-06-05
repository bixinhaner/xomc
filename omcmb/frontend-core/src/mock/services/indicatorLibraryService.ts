import type {
  DeviceType,
  IndicatorInfo,
  IndicatorListFilter,
  IndicatorGroup,
  PlatformFormula,
  CreateIndicatorInput,
  UpdateIndicatorInput,
  CreateGroupInput,
  UpdateGroupInput,
  EnabledIndicatorsRequest,
  IndicatorPlatformSummary,
  IndicatorFile,
  IndicatorUploadResult,
  IndicatorDeleteFileResult,
  TechLower,
} from '../../types/indicatorLibrary';
import {
  mockIndicators,
  mockGroups,
  mockFormulas,
  mockEnabledIndicators,
} from '../data/indicatorLibrary';
import { extractXmlRootAttr } from '../../utils/xmlRootAttr';
import { saveBlob } from '../../utils/saveBlob';

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
      productClass: input.productClass,
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
      ...(input.productClass !== undefined && { productClass: input.productClass }),
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

  // 与真实 api 对齐(按 tech, platform upsert 描述);mock 下无持久化,空实现即可。
  async updateFileDescription(_tech: TechLower, _platform: string, _description: string): Promise<void> {
    return;
  },

  async summary(): Promise<{ items: IndicatorPlatformSummary[] }> {
    // mock (制式, 平台) 多行聚合 — 按 indicator.productClass 分组,每组生成一行
    const items: IndicatorPlatformSummary[] = [];
    const techRows: TechLower[] = ['enb', 'gsm', 'gnb'];
    for (const tech of techRows) {
      const dt = tech.toUpperCase() as DeviceType;
      const arr = indicators[dt] || [];
      const byPlatform = new Map<string, number>();
      for (const i of arr) {
        const p = i.productClass || '';
        if (!p) continue;
        byPlatform.set(p, (byPlatform.get(p) || 0) + 1);
      }
      for (const [platform, count] of Array.from(byPlatform.entries()).sort(([a], [b]) => a.localeCompare(b))) {
        items.push({
          tech,
          platform,
          indicators: count,
          loadedFrom: `indicator-library/${tech}/${platform}.xml`,
          source: 'builtin',
          deletable: false, // mock 数据均为内置 → 删除按钮置灰
          description: '',
        });
      }
    }
    return { items };
  },

  async listFiles(tech: TechLower): Promise<{ items: IndicatorFile[]; tech: TechLower }> {
    // mock 单个内置文件 + 0 行自定义
    const items: IndicatorFile[] = [
      {
        loadedFrom: `indicator-library/${tech === 'enb' ? 'enb/ALL.xml' : tech.toUpperCase() + '.xml'}`,
        source: 'builtin',
        deletable: false,
        count: (indicators[tech.toUpperCase() as DeviceType] || []).length,
        onDisk: true,
      },
    ];
    return { items, tech };
  },

  // mock 上传:名称取自 XML platform 属性(与后端同口径);
  // 落地目录:ENB → enb/ 子目录,GSM/GNB → indicator-library/ 根级。
  // force = 二次确认后的覆盖(mock 不真存盘,直接回成功)。
  async uploadXml(tech: TechLower, file: File, force = false): Promise<IndicatorUploadResult> {
    const platform = (await extractXmlRootAttr(file, 'platform')) ?? 'CUSTOM';
    const loadedFrom =
      tech === 'enb'
        ? `indicator-library/enb/${platform}.xml`
        : `indicator-library/${platform}.xml`;
    return {
      uploaded: true,
      filename: `${platform}.xml`,
      loadedFrom,
      tech,
      platform,
      overwritten: force,
      reloaded: true,
    };
  },

  // mock 下载:生成占位 XML 触发浏览器另存(无真实文件)。
  async downloadXml(loadedFrom: string): Promise<void> {
    const name = loadedFrom.split('/').pop() || 'indicator.xml';
    saveBlob(`<?xml version="1.0" encoding="UTF-8"?>\n<!-- mock ${name} -->\n<indicatorModel platform="${name.replace(/\.xml$/i, '')}"></indicatorModel>\n`, name);
  },

  async deleteFile(loadedFrom: string): Promise<IndicatorDeleteFileResult> {
    const seg = loadedFrom.split('/');
    const tech = (seg.length >= 2 ? seg[1] : 'enb') as TechLower;
    return {
      deleted: true,
      loadedFrom,
      tech,
      rowsAffected: 0,
      backup: `${seg[seg.length - 1]}.deleted.20260101120000`,
    };
  },
};
