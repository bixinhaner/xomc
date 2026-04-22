import type { SingleTask, BatchTask } from '../../types/task';
import type { TaskStatus } from '../../store/taskStore';
import { MockWebSocket } from './index';
import { useTaskStore } from '../../store/taskStore';
import { mockDevices } from '../data/devices';

function pickRandom<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)];
}

function randomBetween(min: number, max: number): number {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

export interface TaskUpdateEvent {
  taskId: string;
  type: 'single' | 'batch';
  status: TaskStatus;
  progress: number;
  currentStep?: number;
  message?: string;
  successCount?: number;
  failCount?: number;
}

let taskIdCounter = 1000;

function createSampleTask(index: number): SingleTask {
  const device = pickRandom(mockDevices.filter((d) => d.connStatus === 'online'));
  const taskTypes = ['配置下发', '软件升级', '参数同步', '备份', 'MML执行'];
  const totalSteps = randomBetween(3, 8);
  return {
    id: `WS-TASK-${String(++taskIdCounter).padStart(6, '0')}`,
    neName: device.name,
    neSn: device.sn,
    type: taskTypes[index % taskTypes.length],
    currentStep: 0,
    totalSteps,
    status: 'pending',
    message: '任务等待执行',
    progress: 0,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  };
}

export class TaskWebSocket extends MockWebSocket {
  private managedTasks: Map<string, { task: SingleTask; startTime: number }> = new Map();
  private taskIndex = 0;

  protected onConnect(): void {
    // Occasionally create new tasks
    this.startInterval(() => this.maybeCreateNewTask(), 10000, 20000);
    // Frequently update existing tasks
    this.startInterval(() => this.updateExistingTasks(), 2000, 5000);
  }

  private maybeCreateNewTask(): void {
    if (Math.random() < 0.4) {
      const task = createSampleTask(this.taskIndex++);
      this.managedTasks.set(task.id, { task, startTime: Date.now() });

      try {
        const store = useTaskStore.getState();
        store.addTask(task);
      } catch {
        // Store may not be available
      }

      this.emit<TaskUpdateEvent>('task_update', {
        taskId: task.id,
        type: 'single',
        status: 'pending',
        progress: 0,
        currentStep: 0,
        message: '任务已创建，等待执行',
      });
    }
  }

  private updateExistingTasks(): void {
    for (const [id, entry] of this.managedTasks) {
      const { task } = entry;
      if (task.status === 'success' || task.status === 'failed' || task.status === 'cancelled') {
        // Remove completed tasks from managed set after a while
        if (Date.now() - entry.startTime > 30000) {
          this.managedTasks.delete(id);
        }
        continue;
      }

      // Transition pending -> running
      if (task.status === 'pending') {
        task.status = 'running';
        task.message = '任务执行中...';
        task.updatedAt = new Date().toISOString();
      } else if (task.status === 'running') {
        // Progress the task
        const newStep = Math.min(task.currentStep + 1, task.totalSteps);
        const newProgress = Math.round((newStep / task.totalSteps) * 100);
        const stepMessages = [
          '正在准备环境...',
          '正在建立连接...',
          '正在执行操作...',
          '正在验证结果...',
          '正在清理资源...',
          '正在生成报告...',
          '任务即将完成...',
          '完成最终检查...',
        ];

        task.currentStep = newStep;
        task.progress = newProgress;
        task.message = stepMessages[newStep % stepMessages.length];
        task.updatedAt = new Date().toISOString();

        if (newStep >= task.totalSteps) {
          // Decide success or failure
          const isFailed = Math.random() < 0.1;
          task.status = isFailed ? 'failed' : 'success';
          task.progress = 100;
          task.message = isFailed ? '任务执行失败：操作超时' : '任务执行成功';
        }
      }

      const event: TaskUpdateEvent = {
        taskId: task.id,
        type: 'single',
        status: task.status,
        progress: task.progress,
        currentStep: task.currentStep,
        message: task.message,
      };

      this.emit<TaskUpdateEvent>('task_update', event);

      try {
        const store = useTaskStore.getState();
        store.updateTask(task.id, {
          status: task.status,
          progress: task.progress,
          currentStep: task.currentStep,
          message: task.message,
          updatedAt: task.updatedAt,
        });
      } catch {
        // Store may not be available
      }
    }

    // Also simulate batch task updates
    if (Math.random() < 0.3) {
      this.simulateBatchTaskUpdate();
    }
  }

  private simulateBatchTaskUpdate(): void {
    const batchId = `WS-BATCH-${String(++taskIdCounter).padStart(6, '0')}`;
    const totalCount = randomBetween(3, 10);
    const successCount = randomBetween(0, totalCount);
    const failCount = totalCount - successCount;
    const progress = Math.round((successCount / totalCount) * 100);

    const batchEvent = {
      taskId: batchId,
      type: 'batch' as const,
      status: progress >= 100 ? ('success' as const) : ('running' as const),
      progress,
      successCount,
      failCount,
      message: `进度: ${successCount}/${totalCount}`,
    };

    this.emit<typeof batchEvent>('task_update', batchEvent);
  }

  injectTask(task: SingleTask): void {
    this.managedTasks.set(task.id, { task: { ...task }, startTime: Date.now() });
  }

  getBatchTaskSample(): BatchTask {
    const totalCount = randomBetween(3, 10);
    return {
      id: `WS-BATCH-${String(++taskIdCounter).padStart(6, '0')}`,
      taskName: pickRandom(['批量配置下发', '批量软件升级', '批量参数优化', '批量备份']),
      type: 'batch',
      totalCount,
      successCount: 0,
      failCount: 0,
      status: 'pending',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
  }
}

export const taskWebSocket = new TaskWebSocket();
