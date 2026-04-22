import type { ConfigParam, ConfigTemplate, BaselineConfig, ConfigTask, NeighborParam } from '../../types/config';
import type { PageRequest, PageResponse } from '../../types/pagination';
import { mockConfigParams, mockConfigTemplates, mockBaselineConfigs, mockConfigTasks } from '../data/config';
import { delay, paginate, generateId } from '../utils';

const params = [...mockConfigParams];
let templates = [...mockConfigTemplates];
let baselines = [...mockBaselineConfigs];
const tasks = [...mockConfigTasks];

export const configService = {
  async getParams(
    p: { deviceSn?: string; category?: string; keyword?: string } & PageRequest
  ): Promise<PageResponse<ConfigParam>> {
    await delay(100, 200);
    let filtered = [...params];
    if (p.category) filtered = filtered.filter((x) => x.category === p.category);
    if (p.keyword) {
      filtered = filtered.filter(
        (x) => x.paramName.includes(p.keyword!) || x.paramCode.includes(p.keyword!)
      );
    }
    return paginate(filtered, p.page, p.pageSize);
  },

  async updateParam(id: string, value: string | number | boolean): Promise<ConfigParam> {
    await delay(200, 400);
    const idx = params.findIndex((x) => x.id === id);
    if (idx === -1) throw new Error(`Param ${id} not found`);
    params[idx] = { ...params[idx], paramValue: value };
    return params[idx];
  },

  async getTemplates(p: PageRequest): Promise<PageResponse<ConfigTemplate>> {
    await delay(80, 150);
    return paginate(templates, p.page, p.pageSize);
  },

  async getTemplateById(id: string): Promise<ConfigTemplate | null> {
    await delay(80, 150);
    return templates.find((t) => t.id === id) ?? null;
  },

  async createTemplate(data: Omit<ConfigTemplate, 'id' | 'createTime'>): Promise<ConfigTemplate> {
    await delay(200, 400);
    const newItem: ConfigTemplate = {
      ...data,
      id: generateId('tmpl'),
      createTime: new Date().toISOString(),
    };
    templates.push(newItem);
    return newItem;
  },

  async updateTemplate(id: string, data: Partial<ConfigTemplate>): Promise<ConfigTemplate> {
    await delay(150, 300);
    const idx = templates.findIndex((t) => t.id === id);
    if (idx === -1) throw new Error(`Template ${id} not found`);
    templates[idx] = { ...templates[idx], ...data };
    return templates[idx];
  },

  async deleteTemplates(ids: string[]): Promise<void> {
    await delay(150, 300);
    templates = templates.filter((t) => !ids.includes(t.id));
  },

  async getBaselines(p: { deviceType?: string; status?: string } & PageRequest): Promise<PageResponse<BaselineConfig>> {
    await delay(80, 150);
    let filtered = [...baselines];
    if (p.deviceType) filtered = filtered.filter((b) => b.deviceType === p.deviceType);
    if (p.status) filtered = filtered.filter((b) => b.status === p.status);
    return paginate(filtered, p.page, p.pageSize);
  },

  async getBaselineById(id: string): Promise<BaselineConfig | null> {
    await delay(80, 150);
    return baselines.find((b) => b.id === id) ?? null;
  },

  async createBaseline(data: Omit<BaselineConfig, 'id' | 'createTime' | 'updateTime'>): Promise<BaselineConfig> {
    await delay(200, 400);
    const newItem: BaselineConfig = {
      ...data,
      id: generateId('base'),
      createTime: new Date().toISOString(),
      updateTime: new Date().toISOString(),
    };
    baselines.push(newItem);
    return newItem;
  },

  async updateBaseline(id: string, data: Partial<BaselineConfig>): Promise<BaselineConfig> {
    await delay(150, 300);
    const idx = baselines.findIndex((b) => b.id === id);
    if (idx === -1) throw new Error(`Baseline ${id} not found`);
    baselines[idx] = { ...baselines[idx], ...data, updateTime: new Date().toISOString() };
    return baselines[idx];
  },

  async deleteBaselines(ids: string[]): Promise<void> {
    await delay(150, 300);
    baselines = baselines.filter((b) => !ids.includes(b.id));
  },

  async getTasks(p: PageRequest): Promise<PageResponse<ConfigTask>> {
    await delay(80, 150);
    return paginate(tasks, p.page, p.pageSize);
  },

  async createTask(data: Omit<ConfigTask, 'id' | 'createdAt' | 'updatedAt' | 'status' | 'progress' | 'successCount' | 'failCount'>): Promise<ConfigTask> {
    await delay(200, 400);
    const newItem: ConfigTask = {
      ...data,
      id: generateId('ctask'),
      status: 'pending',
      progress: 0,
      successCount: 0,
      failCount: 0,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    tasks.push(newItem);
    return newItem;
  },

  async getNeighbors(p: { sourceCellId?: string } & PageRequest): Promise<PageResponse<NeighborParam>> {
    await delay(100, 200);
    const neighbors: NeighborParam[] = [
      {
        id: 'nbr-001',
        sourceCellId: 'CELL-001-1',
        sourceCellName: '北京-eNB-0001-Cell1',
        targetCellId: 'CELL-001-2',
        targetCellName: '北京-eNB-0001-Cell2',
        neighborType: 'intra-freq',
        params: { A3Offset: 3, Hysteresis: 2, TimeToTrigger: 80 },
        createTime: '2024-01-01T00:00:00.000Z',
        updateTime: '2024-06-01T00:00:00.000Z',
      },
      {
        id: 'nbr-002',
        sourceCellId: 'CELL-001-1',
        sourceCellName: '北京-eNB-0001-Cell1',
        targetCellId: 'CELL-GNB-001-1',
        targetCellName: '北京-gNB-0001-Cell1',
        neighborType: 'inter-rat',
        params: { Threshold: -110, Hysteresis: 3 },
        createTime: '2024-02-01T00:00:00.000Z',
        updateTime: '2024-06-01T00:00:00.000Z',
      },
    ];
    let filtered = neighbors;
    if (p.sourceCellId) filtered = filtered.filter((n) => n.sourceCellId === p.sourceCellId);
    return paginate(filtered, p.page, p.pageSize);
  },
};
