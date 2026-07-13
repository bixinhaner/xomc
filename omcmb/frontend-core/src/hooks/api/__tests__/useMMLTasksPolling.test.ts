import { describe, expect, it } from 'vitest';

import {
  getMMLTasksRefetchInterval,
  MML_TASK_LIST_ACTIVE_REFETCH_INTERVAL_MS,
} from '../useMML';

describe('useMMLTasks polling policy', () => {
  it('polls task lists while the current page contains active tasks', () => {
    expect(getMMLTasksRefetchInterval({
      items: [{ id: 'task-1', status: 'running' }],
    })).toBe(MML_TASK_LIST_ACTIVE_REFETCH_INTERVAL_MS);

    expect(getMMLTasksRefetchInterval({
      items: [{ id: 'task-1', status: 'pending' }],
    })).toBe(MML_TASK_LIST_ACTIVE_REFETCH_INTERVAL_MS);

    expect(getMMLTasksRefetchInterval({
      items: [{ id: 'task-1', status: 'paused' }],
    })).toBe(MML_TASK_LIST_ACTIVE_REFETCH_INTERVAL_MS);
  });

  it('stops polling once every task on the current page is terminal', () => {
    expect(getMMLTasksRefetchInterval({
      items: [
        { id: 'task-1', status: 'completed' },
        { id: 'task-2', status: 'failed' },
        { id: 'task-3', status: 'cancelled' },
      ],
    })).toBe(false);

    expect(getMMLTasksRefetchInterval({ items: [] })).toBe(false);
    expect(getMMLTasksRefetchInterval(undefined)).toBe(false);
  });
});
