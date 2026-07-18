import http from '../http';
import type { DeviceTask } from '../../types/deviceTask';
import { isDeviceTaskTerminal } from '../../types/deviceTask';

export interface WaitForTerminalOptions {
  timeoutMs?: number;
  intervalMs?: number;
  signal?: AbortSignal;
  onPoll?: (task: DeviceTask) => void;
}

function createAbortError(): Error {
  if (typeof DOMException !== 'undefined') {
    return new DOMException('The operation was aborted.', 'AbortError');
  }
  const error = new Error('The operation was aborted.');
  error.name = 'AbortError';
  return error;
}

export function isAbortError(error: unknown): boolean {
  return error instanceof Error && error.name === 'AbortError';
}

function throwIfAborted(signal?: AbortSignal): void {
  if (signal?.aborted) {
    throw createAbortError();
  }
}

async function delayWithSignal(delayMs: number, signal?: AbortSignal): Promise<void> {
  if (!signal) {
    await new Promise<void>((resolve) => {
      globalThis.setTimeout(resolve, delayMs);
    });
    return;
  }

  await new Promise<void>((resolve, reject) => {
    const timerId = globalThis.setTimeout(() => {
      signal.removeEventListener('abort', onAbort);
      resolve();
    }, delayMs);

    const onAbort = () => {
      globalThis.clearTimeout(timerId);
      signal.removeEventListener('abort', onAbort);
      reject(createAbortError());
    };

    signal.addEventListener('abort', onAbort, { once: true });
  });
}

/** 后端 GET /devices/tasks/:task_id 响应(信封拆出 data 后的形态)。 */
interface BackendDeviceTask {
  id: string;
  device_sn: string;
  method: string;
  status: string;
  retry_count: number;
  max_retries: number;
  created_at: string;
  sent_at?: string;
  completed_at?: string;
  expires_at?: string;
  error_code?: number;
  error_message?: string;
  result?: Record<string, unknown>;
}

export interface CreateDeviceTaskInput {
  method: string;
  params?: Record<string, unknown>;
  commandKey?: string;
  priority?: number;
  expiresIn?: number;
  maxRetries?: number;
  description?: string;
}

function mapBackendDeviceTask(bt: BackendDeviceTask): DeviceTask {
  return {
    id: bt.id,
    deviceSn: bt.device_sn,
    method: bt.method,
    status: bt.status as DeviceTask['status'],
    retryCount: bt.retry_count ?? 0,
    maxRetries: bt.max_retries ?? 0,
    createdAt: bt.created_at,
    sentAt: bt.sent_at,
    completedAt: bt.completed_at,
    expiresAt: bt.expires_at,
    errorCode: bt.error_code,
    errorMessage: bt.error_message,
    result: bt.result,
  };
}

function buildMockDeviceTask(taskId: string): DeviceTask {
  const now = new Date().toISOString();
  return {
    id: taskId,
    deviceSn: taskId.replace(/^mock-alarm-sync-/, '').split('-').slice(0, -1).join('-') || 'mock-device',
    method: 'GetParameterValues',
    status: 'completed',
    retryCount: 0,
    maxRetries: 0,
    createdAt: now,
    sentAt: now,
    completedAt: now,
  };
}

function isMockAlarmSyncTaskId(taskId: string): boolean {
  return taskId.startsWith('mock-alarm-sync-');
}

/**
 * 设备任务 API(T-0146)。
 *
 * 后端 internal/task/handler.go::GetTask;路由 GET /api/v1/devices/tasks/:task_id。
 * 前端 useDeviceTaskStatus Hook 用它轮询任务状态,直到终态后停止。
 */
export const deviceTaskApi = {
  async createTask(deviceSn: string, input: CreateDeviceTaskInput): Promise<DeviceTask> {
    const body: Record<string, unknown> = {
      device_sn: deviceSn,
      method: input.method,
      params: input.params ?? {},
    };
    if (input.commandKey) body.command_key = input.commandKey;
    if (input.priority !== undefined) body.priority = input.priority;
    if (input.expiresIn !== undefined) body.expires_in = input.expiresIn;
    if (input.maxRetries !== undefined) body.max_retries = input.maxRetries;
    if (input.description) body.description = input.description;

    const { data } = await http.post<BackendDeviceTask>(
      '/devices/tasks',
      body,
      { params: { device_sn: deviceSn } }
    );
    return mapBackendDeviceTask(data);
  },

  async getTask(taskId: string): Promise<DeviceTask> {
    const { data } = await http.get<BackendDeviceTask>(`/devices/tasks/${taskId}`);
    return mapBackendDeviceTask(data);
  },

  async waitForTerminal(
    taskId: string,
    options?: WaitForTerminalOptions,
  ): Promise<DeviceTask> {
    throwIfAborted(options?.signal);
    if (isMockAlarmSyncTaskId(taskId)) {
      const mockTask = buildMockDeviceTask(taskId);
      options?.onPoll?.(mockTask);
      return mockTask;
    }
    const timeoutMs = options?.timeoutMs ?? 10 * 60 * 1000;
    const intervalMs = options?.intervalMs ?? 2000;
    const deadlineAt = Date.now() + timeoutMs;

    while (true) {
      throwIfAborted(options?.signal);
      const task = await this.getTask(taskId);
      options?.onPoll?.(task);
      if (isDeviceTaskTerminal(task.status)) {
        return task;
      }
      if (Date.now() >= deadlineAt) {
        throw new Error(`device task ${taskId} timed out`);
      }
      await delayWithSignal(intervalMs, options?.signal);
    }
  },
};
