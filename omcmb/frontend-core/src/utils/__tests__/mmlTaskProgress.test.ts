import { describe, expect, it } from 'vitest';
import { getMmlTaskProgress } from '../mmlTaskProgress';
import type { MMLTask } from '../../types/mml';

const baseTask: MMLTask = {
  id: 'task-1',
  taskName: 'task',
  taskOrigin: 'console',
  deviceSns: [],
  commands: [],
  status: 'running',
  results: [],
  createdAt: '2026-07-11T00:00:00Z',
  updatedAt: '2026-07-11T00:00:00Z',
  creator: 'admin',
  executeType: 'immediate',
  offlineRetry: false,
  offlineRetryWait: 60,
  failedRetry: false,
  failedRetryCount: 3,
  failedRetryInterval: 5,
  totalDevices: 3,
  successCount: 2,
  failedCount: 1,
};

describe('getMmlTaskProgress', () => {
  it('uses lightweight command counts when list rows omit command arrays', () => {
    expect(getMmlTaskProgress({ ...baseTask, commandCount: 4 })).toEqual({
      kind: 'execution',
      done: 3,
      total: 12,
    });
  });

  it('uses lightweight plan item counts for device-bound list rows', () => {
    expect(getMmlTaskProgress({
      ...baseTask,
      executeMode: 'device_bound',
      commandCount: 0,
      planItemCount: 9,
    })).toEqual({
      kind: 'execution',
      done: 3,
      total: 9,
    });
  });

  it('uses expanded executable command count for device-bound progress', () => {
    expect(getMmlTaskProgress({
      ...baseTask,
      executeMode: 'device_bound',
      commandCount: 5,
      planItemCount: 4,
      successCount: 4,
      failedCount: 1,
    })).toEqual({
      kind: 'execution',
      done: 5,
      total: 5,
    });
  });

  it('uses latest periodic run progress for periodic parent rows', () => {
    expect(getMmlTaskProgress({
      ...baseTask,
      taskOrigin: 'script',
      executeType: 'periodic',
      commandCount: 6,
      successCount: 0,
      failedCount: 0,
      latestRun: {
        id: 'run-1',
        executeType: 'immediate',
        status: 'completed',
        result: 'partial',
        executeMode: 'device_bound',
        commandCount: 6,
        planItemCount: 6,
        totalDevices: 2,
        successCount: 5,
        failedCount: 1,
        createdAt: '2026-07-20T00:00:00Z',
        updatedAt: '2026-07-20T00:00:00Z',
      },
    })).toEqual({
      kind: 'execution',
      done: 6,
      total: 6,
    });
  });

  it('marks periodic parent rows without runs as awaiting first execution', () => {
    expect(getMmlTaskProgress({
      ...baseTask,
      taskOrigin: 'script',
      executeType: 'periodic',
      commandCount: 6,
      successCount: 0,
      failedCount: 0,
    })).toEqual({
      done: 0,
      total: 0,
      kind: 'awaiting_first_run',
    });
  });
});
