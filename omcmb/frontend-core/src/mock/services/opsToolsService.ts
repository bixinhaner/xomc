import type { PageRequest, PageResponse } from '../../types/pagination';
import type { OpsTemplate, OpsCommandRecord, OpsTask } from '../data/opsTools';
import { mockOpsTemplates, mockOpsCommandRecords, mockOpsTasks } from '../data/opsTools';
import { delay, paginate, generateId } from '../utils';

let templates = [...mockOpsTemplates];
let commandRecords = [...mockOpsCommandRecords];
let tasks = [...mockOpsTasks];

export const opsToolsService = {
  async getTemplates(
    params: { category?: string; keyword?: string; targetDeviceType?: string } & PageRequest
  ): Promise<PageResponse<OpsTemplate>> {
    await delay(100, 200);
    let filtered = [...templates];
    if (params.category) filtered = filtered.filter((t) => t.category === params.category);
    if (params.keyword) {
      const kw = params.keyword.toLowerCase();
      filtered = filtered.filter(
        (t) =>
          t.templateName.toLowerCase().includes(kw) ||
          t.description.toLowerCase().includes(kw)
      );
    }
    if (params.targetDeviceType) {
      filtered = filtered.filter((t) =>
        t.targetDeviceTypes.includes(params.targetDeviceType!)
      );
    }
    return paginate(filtered, params.page, params.pageSize);
  },

  async getTemplateById(id: string): Promise<OpsTemplate | null> {
    await delay(80, 150);
    return templates.find((t) => t.id === id) ?? null;
  },

  async createTemplate(data: Omit<OpsTemplate, 'id' | 'createTime' | 'updateTime' | 'useCount'>): Promise<OpsTemplate> {
    await delay(200, 400);
    const newItem: OpsTemplate = {
      ...data,
      id: generateId('opst'),
      createTime: new Date().toISOString(),
      updateTime: new Date().toISOString(),
      useCount: 0,
    };
    templates.push(newItem);
    return newItem;
  },

  async updateTemplate(id: string, data: Partial<OpsTemplate>): Promise<OpsTemplate> {
    await delay(150, 300);
    const idx = templates.findIndex((t) => t.id === id);
    if (idx === -1) throw new Error(`Template ${id} not found`);
    templates[idx] = { ...templates[idx], ...data, updateTime: new Date().toISOString() };
    return templates[idx];
  },

  async deleteTemplates(ids: string[]): Promise<void> {
    await delay(150, 300);
    templates = templates.filter((t) => !ids.includes(t.id));
  },

  async getCommandRecords(
    params: { deviceSn?: string; operator?: string; success?: boolean } & PageRequest
  ): Promise<PageResponse<OpsCommandRecord>> {
    await delay(100, 200);
    let filtered = [...commandRecords];
    if (params.deviceSn) filtered = filtered.filter((r) => r.deviceSn === params.deviceSn);
    if (params.operator) filtered = filtered.filter((r) => r.operator === params.operator);
    if (params.success !== undefined) filtered = filtered.filter((r) => r.success === params.success);
    filtered.sort((a, b) => b.executeTime.localeCompare(a.executeTime));
    return paginate(filtered, params.page, params.pageSize);
  },

  async addCommandRecord(data: Omit<OpsCommandRecord, 'id'>): Promise<OpsCommandRecord> {
    await delay(50, 100);
    const newItem: OpsCommandRecord = {
      ...data,
      id: generateId('ocmd'),
    };
    commandRecords.unshift(newItem);
    return newItem;
  },

  async getTasks(
    params: { status?: string; templateId?: string } & PageRequest
  ): Promise<PageResponse<OpsTask>> {
    await delay(100, 200);
    let filtered = [...tasks];
    if (params.status) filtered = filtered.filter((t) => t.status === params.status);
    if (params.templateId) filtered = filtered.filter((t) => t.templateId === params.templateId);
    filtered.sort((a, b) => b.createdAt.localeCompare(a.createdAt));
    return paginate(filtered, params.page, params.pageSize);
  },

  async getTaskById(id: string): Promise<OpsTask | null> {
    await delay(80, 150);
    return tasks.find((t) => t.id === id) ?? null;
  },

  async createTask(data: Omit<OpsTask, 'id' | 'status' | 'currentStep' | 'progress' | 'successCount' | 'failCount' | 'createdAt'>): Promise<OpsTask> {
    await delay(200, 400);
    const template = data.templateId ? templates.find((t) => t.id === data.templateId) : null;
    const newItem: OpsTask = {
      ...data,
      id: generateId('opstask'),
      status: 'pending',
      currentStep: 0,
      totalSteps: template?.steps.length ?? data.totalSteps,
      progress: 0,
      successCount: 0,
      failCount: 0,
      createdAt: new Date().toISOString(),
    };
    tasks.push(newItem);
    if (data.templateId) {
      const tmplIdx = templates.findIndex((t) => t.id === data.templateId);
      if (tmplIdx !== -1) {
        templates[tmplIdx] = { ...templates[tmplIdx], useCount: templates[tmplIdx].useCount + 1 };
      }
    }
    return newItem;
  },

  async cancelTask(id: string): Promise<void> {
    await delay(150, 300);
    const idx = tasks.findIndex((t) => t.id === id);
    if (idx !== -1) {
      tasks[idx] = { ...tasks[idx], status: 'cancelled', completedAt: new Date().toISOString() };
    }
  },

  async pauseTask(id: string): Promise<void> {
    await delay(100, 200);
    const idx = tasks.findIndex((t) => t.id === id);
    if (idx !== -1) {
      tasks[idx] = { ...tasks[idx], status: 'paused' };
    }
  },

  async resumeTask(id: string): Promise<void> {
    await delay(100, 200);
    const idx = tasks.findIndex((t) => t.id === id);
    if (idx !== -1) {
      tasks[idx] = { ...tasks[idx], status: 'running' };
    }
  },
};
