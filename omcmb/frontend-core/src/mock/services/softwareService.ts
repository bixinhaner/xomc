import type { PageRequest, PageResponse } from '../../types/pagination';
import type { SoftwareVersion, UpgradePlan, UpgradeTaskInfo, UpgradeSubTaskInfo } from '../data/software';
import { mockSoftwareVersions, mockUpgradePlans } from '../data/software';
import { delay, paginate, generateId } from '../utils';

let versions = [...mockSoftwareVersions];
const upgradePlans = [...mockUpgradePlans];

// Mock upgrade tasks (simulates upgrade_tasks table)
let taskIdCounter = 100;
const mockUpgradeTasks: UpgradeTaskInfo[] = [];

// Helper: create a mock UpgradeTaskInfo
function createMockTask(overrides: Partial<UpgradeTaskInfo> & { taskName: string; productClass: string }): UpgradeTaskInfo {
  const total = Math.floor(Math.random() * 10) + 1;
  const success = Math.floor(Math.random() * total);
  const fail = total - success;
  return {
    id: generateId('task'),
    taskType: 1,
    status: 'ended',
    result: fail === 0 ? 'success' : success === 0 ? 'failed' : 'partial',
    operatorCode: 'cmcc',
    isKeepConfig: true,
    createStatus: 'active',
    createUser: 'admin',
    totalCount: total,
    successCount: success,
    failCount: fail,
    maxConcurrent: 5,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    ...overrides,
  };
}

export const softwareService = {
  async getVersions(
    params: { deviceType?: string; status?: string; vendor?: string; fileType?: number } & PageRequest
  ): Promise<PageResponse<SoftwareVersion>> {
    await delay(100, 200);
    let filtered = [...versions];
    if (params.deviceType) filtered = filtered.filter((v) => v.deviceType === params.deviceType);
    if (params.status) filtered = filtered.filter((v) => v.status === params.status);
    if (params.vendor) filtered = filtered.filter((v) => v.vendor === params.vendor);
    if (params.fileType !== undefined) filtered = filtered.filter((v) => v.fileType === params.fileType);
    return paginate(filtered, params.page, params.pageSize);
  },

  async getVersionById(id: string): Promise<SoftwareVersion | null> {
    await delay(80, 150);
    return versions.find((v) => v.id === id) ?? null;
  },

  async uploadVersion(data: Omit<SoftwareVersion, 'id' | 'releaseDate'>): Promise<SoftwareVersion> {
    await delay(500, 1500);
    const newItem: SoftwareVersion = {
      ...data,
      id: generateId('ver'),
      releaseDate: new Date().toISOString(),
    };
    versions.push(newItem);
    return newItem;
  },

  async deleteVersions(ids: string[]): Promise<void> {
    await delay(150, 300);
    versions = versions.filter((v) => !ids.includes(v.id));
  },

  async toggleRecommend(id: string): Promise<SoftwareVersion> {
    await delay(100, 200);
    const ver = versions.find((v) => v.id === id);
    if (ver) ver.recommend = !ver.recommend;
    return ver!;
  },

  async getUpgradeTasks(
    params: { taskType?: number; status?: string; productClass?: string; createUser?: string } & PageRequest
  ): Promise<PageResponse<UpgradeTaskInfo>> {
    await delay(100, 200);
    let filtered = [...mockUpgradeTasks];
    if (params.taskType !== undefined) filtered = filtered.filter((t) => t.taskType === params.taskType);
    if (params.status) filtered = filtered.filter((t) => t.status === params.status);
    if (params.productClass) filtered = filtered.filter((t) => t.productClass === params.productClass);
    return paginate(filtered, params.page, params.pageSize);
  },

  async getUpgradeTaskById(id: string): Promise<UpgradeTaskInfo | null> {
    await delay(80, 150);
    return mockUpgradeTasks.find((t) => t.id === id) ?? null;
  },

  async createUpgradeTask(req: {
    deviceIds: string[];
    firmwareId: string;
    taskName: string;
    taskType?: number;
    isKeepConfig?: boolean;
    concurrency?: number;
  }): Promise<UpgradeTaskInfo> {
    await delay(200, 400);
    const task = createMockTask({
      taskName: req.taskName,
      productClass: 'PM-B4860',
      taskType: (req.taskType ?? 1) as UpgradeTaskInfo['taskType'],
      isKeepConfig: req.isKeepConfig ?? true,
      status: 'in_progress',
      totalCount: req.deviceIds.length,
      successCount: 0,
      failCount: 0,
      maxConcurrent: req.concurrency ?? 5,
    });
    mockUpgradeTasks.unshift(task);
    return task;
  },

  async suspendTask(id: string): Promise<void> {
    await delay(100, 200);
    const task = mockUpgradeTasks.find((t) => t.id === id);
    if (task) task.status = 'suspended';
  },

  async resumeTask(id: string): Promise<void> {
    await delay(100, 200);
    const task = mockUpgradeTasks.find((t) => t.id === id);
    if (task) task.status = 'in_progress';
  },

  async terminateTask(id: string): Promise<void> {
    await delay(100, 200);
    const task = mockUpgradeTasks.find((t) => t.id === id);
    if (task) {
      task.status = 'ended';
      task.result = 'terminated';
    }
  },

  async retryTask(id: string): Promise<void> {
    await delay(100, 200);
    const task = mockUpgradeTasks.find((t) => t.id === id);
    if (task) {
      task.status = 'in_progress';
      task.failCount = 0;
    }
  },

  // ---- Canary stage transitions (T-0019, mirrors backend T-0018) ----
  // Mock implementations only mutate the in-memory task; deeper canary
  // simulation lives in the backend. Stage_status / current_stage updates
  // are minimal no-ops sufficient for the dev/test workflow.

  async advanceCanary(id: string): Promise<void> {
    await delay(50, 150);
    const task = mockUpgradeTasks.find((t) => t.id === id);
    if (task && task.strategy === 'canary' && (task.currentStage ?? 0) > 0) {
      const next = (task.currentStage ?? 0) + 1;
      const max = task.canaryStages?.length ?? 0;
      if (next > max) {
        task.stageStatus = 'completed';
      } else {
        task.currentStage = next;
        task.stageStatus = 'running';
      }
    }
  },

  async pauseCanary(id: string): Promise<void> {
    await delay(50, 150);
    const task = mockUpgradeTasks.find((t) => t.id === id);
    if (task && task.strategy === 'canary') {
      task.stageStatus = 'paused';
    }
  },

  async resumeCanary(id: string): Promise<void> {
    await delay(50, 150);
    const task = mockUpgradeTasks.find((t) => t.id === id);
    if (task && task.strategy === 'canary') {
      task.stageStatus = 'running';
    }
  },

  async abortCanary(id: string): Promise<void> {
    await delay(50, 150);
    const task = mockUpgradeTasks.find((t) => t.id === id);
    if (task && task.strategy === 'canary') {
      task.stageStatus = 'aborted';
    }
  },

  async createRollback(req: {
    deviceIds: string[];
    taskName: string;
    operatorCode: string;
    createUser: string;
  }): Promise<UpgradeTaskInfo> {
    await delay(200, 400);
    const task = createMockTask({
      taskName: req.taskName,
      productClass: 'PM-B4860',
      taskType: 2,
      status: 'in_progress',
      totalCount: req.deviceIds.length,
      successCount: 0,
      failCount: 0,
      operatorCode: req.operatorCode,
      createUser: req.createUser,
    });
    mockUpgradeTasks.unshift(task);
    return task;
  },

  async getSubTasks(
    taskId: string,
    params: { status?: string } & PageRequest
  ): Promise<PageResponse<UpgradeSubTaskInfo>> {
    await delay(100, 200);
    void taskId;
    const subTasks: UpgradeSubTaskInfo[] = [];
    return paginate(subTasks, params.page, params.pageSize);
  },

  async getSubTaskById(id: string): Promise<UpgradeSubTaskInfo | null> {
    await delay(80, 150);
    void id;
    return null;
  },

  // Legacy functions (kept for backwards compatibility)

  async getUpgradePlans(
    params: { status?: string } & PageRequest
  ): Promise<PageResponse<UpgradePlan>> {
    await delay(100, 200);
    let filtered = [...upgradePlans];
    if (params.status) filtered = filtered.filter((p) => p.status === params.status);
    return paginate(filtered, params.page, params.pageSize);
  },

  async getUpgradePlanById(id: string): Promise<UpgradePlan | null> {
    await delay(80, 150);
    return upgradePlans.find((p) => p.id === id) ?? null;
  },

  async createUpgradePlan(data: Omit<UpgradePlan, 'id' | 'status' | 'progress' | 'successCount' | 'failCount' | 'createdAt' | 'updatedAt'>): Promise<UpgradePlan> {
    await delay(200, 400);
    const newItem: UpgradePlan = {
      ...data,
      id: generateId('upg'),
      status: data.scheduledTime ? 'scheduled' : 'pending',
      progress: 0,
      successCount: 0,
      failCount: 0,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    upgradePlans.push(newItem);
    return newItem;
  },

  async cancelUpgradePlan(id: string): Promise<void> {
    await delay(200, 400);
    const idx = upgradePlans.findIndex((p) => p.id === id);
    if (idx !== -1) {
      upgradePlans[idx] = {
        ...upgradePlans[idx],
        status: 'cancelled',
        updatedAt: new Date().toISOString(),
      };
    }
  },

  async precheck(deviceSns: string[], versionId: string): Promise<Array<{ deviceSn: string; passed: boolean; issues: string[] }>> {
    await delay(1000, 3000);
    void versionId;
    return deviceSns.map((sn) => ({
      deviceSn: sn,
      passed: Math.random() > 0.15,
      issues: Math.random() > 0.8 ? ['磁盘空间不足，需要2GB', '当前版本不支持直升'] : [],
    }));
  },
};
