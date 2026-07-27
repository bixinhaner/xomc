import type { AlarmSyncTriggerResult } from '@core/services/api/alarmApi';

interface AlarmRefreshTask {
  status: string;
  errorMessage?: string;
}

interface DeviceAlarmRefreshInput {
  deviceSn: string;
  trigger: (deviceSn: string) => Promise<AlarmSyncTriggerResult>;
  waitForTerminal: (taskId: string, signal?: AbortSignal) => Promise<AlarmRefreshTask>;
  refreshCurrentAlarms: () => Promise<unknown>;
  signal?: AbortSignal;
  projectionRefreshDelaysMs?: number[];
}

const DEFAULT_PROJECTION_REFRESH_DELAYS_MS = [0, 500, 1500];

function createAbortError(): Error {
  if (typeof DOMException !== 'undefined') {
    return new DOMException('The operation was aborted.', 'AbortError');
  }
  const error = new Error('The operation was aborted.');
  error.name = 'AbortError';
  return error;
}

function throwIfAborted(signal?: AbortSignal): void {
  if (signal?.aborted) throw createAbortError();
}

async function delayWithSignal(delayMs: number, signal?: AbortSignal): Promise<void> {
  throwIfAborted(signal);
  if (delayMs <= 0) return;

  await new Promise<void>((resolve, reject) => {
    const timerId = globalThis.setTimeout(() => {
      signal?.removeEventListener('abort', onAbort);
      resolve();
    }, delayMs);
    const onAbort = () => {
      globalThis.clearTimeout(timerId);
      signal?.removeEventListener('abort', onAbort);
      reject(createAbortError());
    };
    signal?.addEventListener('abort', onAbort, { once: true });
  });
}

export async function runDeviceAlarmRefresh({
  deviceSn,
  trigger,
  waitForTerminal,
  refreshCurrentAlarms,
  signal,
  projectionRefreshDelaysMs = DEFAULT_PROJECTION_REFRESH_DELAYS_MS,
}: DeviceAlarmRefreshInput): Promise<void> {
  const triggerResult = await trigger(deviceSn);
  if (!triggerResult.taskId) {
    throw new Error('alarm sync task id is missing');
  }

  const task = await waitForTerminal(triggerResult.taskId, signal);
  if (task.status !== 'completed') {
    throw new Error(task.errorMessage || `alarm sync task ended with status ${task.status}`);
  }

  // device_tasks 会先进入 completed，告警处理器随后消费 GPV 事件并写库。
  // 在这个很短的投影窗口内做有上限的重查，避免只命中 completed 与落库之间的旧数据。
  for (const delayMs of projectionRefreshDelaysMs) {
    await delayWithSignal(delayMs, signal);
    throwIfAborted(signal);
    await refreshCurrentAlarms();
    throwIfAborted(signal);
  }
}
