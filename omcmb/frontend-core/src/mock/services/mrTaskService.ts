/**
 * Mock 实现，对应 services/api/mrTaskApi.ts。
 * useMock=true 时由 hooks/api/useMrTasks 切换调用，便于前端独立开发。
 *
 * 状态机：waitting → on → off （fake）。Mock 不模拟 SPV 时序与 cell 进度，
 * 仅维持任务级状态便于 UI 流程演示。
 */

import type { PageResponse } from '../../types/pagination';
import type {
  CreateMRTaskRequest,
  MRTask,
  MRTaskListFilter,
  MRTaskProgress,
  MRTaskProgressFilter,
} from '../../types/mrTask';
import { delay, paginate } from '../utils';

let _idCounter = 1;
const _now = () => new Date().toISOString();

function makeMockTask(req: CreateMRTaskRequest): MRTask {
  const id = `mr-task-mock-${String(_idCounter++).padStart(4, '0')}`;
  return {
    taskId: id,
    taskName: req.taskName,
    mrType: req.mrType ?? 'MRS,MRE,MRO',
    statisPeriod: req.statisPeriod ?? '5120',
    reportPeriod: req.reportPeriod ?? '15',
    startTime: req.startTime,
    endTime: req.endTime,
    taskStatus: 'waitting',
    creator: 'mock-admin', // 后端从 auth ctx 取
    targetDeviceSns: req.targetDeviceSns ?? [],
    createdAt: _now(),
    updatedAt: _now(),
  };
}

// 初始 3 条样例数据
const tasks: MRTask[] = [
  {
    taskId: 'mr-task-mock-demo-1',
    taskName: 'MR0226_demo',
    mrType: 'MRS,MRE,MRO',
    statisPeriod: '5120',
    reportPeriod: '15',
    startTime: '2026-02-26T00:00:00Z',
    endTime: '2026-02-26T10:00:00Z',
    taskStatus: 'off',
    creator: 'admin',
    targetDeviceSns: ['SN001', 'SN002'],
    createdAt: '2026-02-25T18:00:00Z',
    updatedAt: '2026-02-26T10:00:00Z',
  },
  {
    taskId: 'mr-task-mock-demo-2',
    taskName: 'MR30SITES',
    mrType: 'MRS,MRE,MRO',
    statisPeriod: '5120',
    reportPeriod: '30',
    startTime: '2026-02-23T00:00:00Z',
    endTime: '2026-02-23T04:00:00Z',
    taskStatus: 'on',
    creator: 'planner',
    targetDeviceSns: ['SN010', 'SN011', 'SN012'],
    createdAt: '2026-02-23T08:00:00Z',
    updatedAt: '2026-02-23T08:30:00Z',
  },
  {
    taskId: 'mr-task-mock-demo-3',
    taskName: 'MR_optimization_test',
    mrType: 'MRS,MRE,MRO',
    statisPeriod: '10240',
    reportPeriod: '60',
    startTime: '2026-05-26T20:00:00Z',
    taskStatus: 'waitting',
    creator: 'admin',
    targetDeviceSns: ['SN100'],
    createdAt: '2026-05-25T12:00:00Z',
    updatedAt: '2026-05-25T12:00:00Z',
  },
];

const progressByTask = new Map<string, MRTaskProgress[]>([
  ['mr-task-mock-demo-1', [
    { id: 'p1', taskId: 'mr-task-mock-demo-1', smallCellCode: 'CELL001', serialNumber: 'SN001', hostName: 'cell-001', progressStatus: 'closeSuccess', healthStatus: 'unknown', missedHeartbeat: 0, createdAt: '2026-02-26T00:00:00Z', updatedAt: '2026-02-26T10:00:00Z' },
    { id: 'p2', taskId: 'mr-task-mock-demo-1', smallCellCode: 'CELL002', serialNumber: 'SN002', hostName: 'cell-002', progressStatus: 'closeSuccess', healthStatus: 'unknown', missedHeartbeat: 0, createdAt: '2026-02-26T00:00:00Z', updatedAt: '2026-02-26T10:00:00Z' },
  ]],
  ['mr-task-mock-demo-2', [
    { id: 'p3', taskId: 'mr-task-mock-demo-2', smallCellCode: 'CELL010', serialNumber: 'SN010', progressStatus: 'openSuccess', healthStatus: 'normal', lastHeartbeat: _now(), missedHeartbeat: 0, createdAt: '2026-02-23T08:30:00Z', updatedAt: _now() },
    { id: 'p4', taskId: 'mr-task-mock-demo-2', smallCellCode: 'CELL011', serialNumber: 'SN011', progressStatus: 'openSuccess', healthStatus: 'abnormal', missedHeartbeat: 3, createdAt: '2026-02-23T08:30:00Z', updatedAt: _now() },
    { id: 'p5', taskId: 'mr-task-mock-demo-2', smallCellCode: 'CELL012', serialNumber: 'SN012', progressStatus: 'unsupport', healthStatus: 'unknown', faultCode: 'platform_unsupport', missedHeartbeat: 0, createdAt: '2026-02-23T08:30:00Z', updatedAt: '2026-02-23T08:30:00Z' },
  ]],
]);

export const mrTaskService = {
  async create(req: CreateMRTaskRequest): Promise<MRTask> {
    await delay(150, 300);
    const t = makeMockTask(req);
    tasks.unshift(t);
    // 简化模型：targets 由后端 scheduler 动态枚举，mock 留空数组
    progressByTask.set(t.taskId, []);
    return t;
  },

  async list(filter: MRTaskListFilter): Promise<PageResponse<MRTask>> {
    await delay(80, 150);
    let filtered = [...tasks];
    if (filter.status) filtered = filtered.filter((t) => t.taskStatus === filter.status);
    if (filter.keyword) {
      const kw = filter.keyword.toLowerCase();
      filtered = filtered.filter((t) => t.taskName.toLowerCase().includes(kw));
    }
    return paginate(filtered, filter.page, filter.pageSize);
  },

  async get(taskId: string): Promise<MRTask> {
    await delay(50, 100);
    const t = tasks.find((x) => x.taskId === taskId);
    if (!t) throw new Error(`mock: mr task ${taskId} not found`);
    return t;
  },

  async stop(taskId: string): Promise<void> {
    await delay(100, 200);
    const t = tasks.find((x) => x.taskId === taskId);
    if (!t) throw new Error(`mock: mr task ${taskId} not found`);
    if (t.taskStatus !== 'waitting' && t.taskStatus !== 'on') {
      throw new Error(`mock: cannot stop task in status ${t.taskStatus}`);
    }
    t.taskStatus = 'termination';
    t.updatedAt = _now();
  },

  async delete(taskId: string): Promise<void> {
    await delay(100, 200);
    const idx = tasks.findIndex((x) => x.taskId === taskId);
    if (idx === -1) throw new Error(`mock: mr task ${taskId} not found`);
    const t = tasks[idx];
    if (t.taskStatus !== 'off' && t.taskStatus !== 'termination') {
      throw new Error(`mock: cannot delete task in status ${t.taskStatus}`);
    }
    tasks.splice(idx, 1);
    progressByTask.delete(taskId);
  },

  async listProgress(
    taskId: string,
    filter: MRTaskProgressFilter,
  ): Promise<PageResponse<MRTaskProgress>> {
    await delay(80, 150);
    const all = progressByTask.get(taskId) ?? [];
    let filtered = [...all];
    if (filter.status) filtered = filtered.filter((p) => p.progressStatus === filter.status);
    if (filter.health) filtered = filtered.filter((p) => p.healthStatus === filter.health);
    return paginate(filtered, filter.page, filter.pageSize);
  },
};
