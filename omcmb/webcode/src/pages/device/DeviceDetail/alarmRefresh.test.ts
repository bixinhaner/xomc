import { describe, expect, it, vi } from 'vitest';
import * as alarmRefresh from './alarmRefresh';

describe('runDeviceAlarmRefresh', () => {
  it('waits for the device task before retrying the projected alarm list refresh', async () => {
    const order: string[] = [];
    const trigger = vi.fn(async (deviceSn: string) => {
      order.push(`trigger:${deviceSn}`);
      return { deviceSn, taskId: 'task-149' };
    });
    const waitForTerminal = vi.fn(async (taskId: string) => {
      order.push(`wait:${taskId}`);
      return { status: 'completed' };
    });
    const refreshCurrentAlarms = vi.fn(async () => {
      order.push('refresh');
    });

    await alarmRefresh.runDeviceAlarmRefresh({
      deviceSn: 'SN-149',
      trigger,
      waitForTerminal,
      refreshCurrentAlarms,
      projectionRefreshDelaysMs: [0, 0, 0],
    });

    expect(order).toEqual([
      'trigger:SN-149',
      'wait:task-149',
      'refresh',
      'refresh',
      'refresh',
    ]);
  });

  it('does not refresh the local list when the device task fails', async () => {
    const refreshCurrentAlarms = vi.fn();

    await expect(alarmRefresh.runDeviceAlarmRefresh({
      deviceSn: 'SN-149',
      trigger: vi.fn(async () => ({ deviceSn: 'SN-149', taskId: 'task-149' })),
      waitForTerminal: vi.fn(async () => ({ status: 'failed', errorMessage: 'device offline' })),
      refreshCurrentAlarms,
    })).rejects.toThrow('device offline');

    expect(refreshCurrentAlarms).not.toHaveBeenCalled();
  });

  it('rejects a response without a task id instead of reporting completion', async () => {
    const waitForTerminal = vi.fn();
    const refreshCurrentAlarms = vi.fn();

    await expect(alarmRefresh.runDeviceAlarmRefresh({
      deviceSn: 'SN-149',
      trigger: vi.fn(async () => ({ deviceSn: 'SN-149' })),
      waitForTerminal,
      refreshCurrentAlarms,
    })).rejects.toThrow('task id is missing');

    expect(waitForTerminal).not.toHaveBeenCalled();
    expect(refreshCurrentAlarms).not.toHaveBeenCalled();
  });

  it('stops projected-list retries after navigation aborts the refresh', async () => {
    const abortController = new AbortController();
    const refreshCurrentAlarms = vi.fn(async () => {
      abortController.abort();
    });

    await expect(alarmRefresh.runDeviceAlarmRefresh({
      deviceSn: 'SN-149',
      trigger: vi.fn(async () => ({ deviceSn: 'SN-149', taskId: 'task-149' })),
      waitForTerminal: vi.fn(async () => ({ status: 'completed' })),
      refreshCurrentAlarms,
      signal: abortController.signal,
      projectionRefreshDelaysMs: [0],
    })).rejects.toMatchObject({ name: 'AbortError' });

    expect(refreshCurrentAlarms).toHaveBeenCalledTimes(1);
  });
});
