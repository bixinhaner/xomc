import { describe, expect, it } from 'vitest';
import {
  canApplySubmittedReadback,
  parameterReadbackValuesMatch,
  ParameterReadbackTimeoutError,
  waitForReadback,
  waitForExpectedParameterValues,
} from '../parameterReadback';

describe('waitForExpectedParameterValues', () => {
  it('keeps polling when the first read still contains the pre-save value', async () => {
    const path = 'Device.Services.FAPService.1.FAPControl.LTE.Gateway.MmeIpPlmnList';
    const reads = [
      new Map([[path, '172.24.224.88+222222']]),
      new Map([[path, '172.24.224.88+222222,172.24.224.91+333333']]),
    ];
    let calls = 0;

    const result = await waitForExpectedParameterValues({
      expected: new Map([[path, '172.24.224.88+222222,172.24.224.91+333333']]),
      read: async () => reads[Math.min(calls++, reads.length - 1)],
      intervalMs: 0,
      timeoutMs: 100,
    });

    expect(calls).toBe(2);
    expect(result.get(path)).toBe('172.24.224.88+222222,172.24.224.91+333333');
  });

  it('fails with a readback timeout instead of accepting a stale value', async () => {
    const path = 'Device.X';

    await expect(waitForExpectedParameterValues({
      expected: new Map([[path, 'new']]),
      read: async () => new Map([[path, 'old']]),
      intervalMs: 0,
      timeoutMs: 0,
    })).rejects.toBeInstanceOf(ParameterReadbackTimeoutError);
  }, 250);

  it('stops immediately when the form unmounts', async () => {
    const controller = new AbortController();
    controller.abort();
    let calls = 0;

    await expect(waitForExpectedParameterValues({
      expected: new Map([['Device.X', 'new']]),
      read: async () => {
        calls += 1;
        return new Map([['Device.X', 'new']]);
      },
      signal: controller.signal,
    })).rejects.toMatchObject({ name: 'AbortError' });
    expect(calls).toBe(0);
  });

  it('times out even when the schema read never settles', async () => {
    await expect(waitForExpectedParameterValues({
      expected: new Map([['Device.X', 'new']]),
      read: async () => new Promise<ReadonlyMap<string, string>>(() => undefined),
      timeoutMs: 20,
    })).rejects.toBeInstanceOf(ParameterReadbackTimeoutError);
  }, 250);

  it('aborts an in-flight schema read immediately', async () => {
    const controller = new AbortController();
    let readSignal: AbortSignal | undefined;
    const pending = waitForExpectedParameterValues({
      expected: new Map([['Device.X', 'new']]),
      read: async (signal) => {
        readSignal = signal;
        return new Promise<ReadonlyMap<string, string>>(() => undefined);
      },
      signal: controller.signal,
      timeoutMs: 1_000,
    });

    await Promise.resolve();
    controller.abort();

    await expect(pending).rejects.toMatchObject({ name: 'AbortError' });
    expect(readSignal?.aborted).toBe(true);
  }, 250);
});

describe('readback matching', () => {
  it('treats boolean wire aliases as the same value without weakening other comparisons', () => {
    expect(parameterReadbackValuesMatch('1', 'true')).toBe(true);
    expect(parameterReadbackValuesMatch('0', 'off')).toBe(true);
    expect(parameterReadbackValuesMatch('25', '50')).toBe(false);
  });

  it('supports polling non-parameter state such as instance and packed-list membership', async () => {
    const reads = [['old'], ['old', 'new']];
    let calls = 0;
    const result = await waitForReadback({
      read: async () => reads[Math.min(calls++, reads.length - 1)],
      matches: (actual) => actual.includes('new'),
      intervalMs: 0,
      timeoutMs: 100,
    });

    expect(calls).toBe(2);
    expect(result).toContain('new');
  });
});

describe('canApplySubmittedReadback', () => {
  it('only accepts the unsynced task whose submitted draft revision is still current', () => {
    expect(canApplySubmittedReadback({
      taskId: 'task-1',
      syncedForTaskId: undefined,
      submittedDraftRevision: 3,
      currentDraftRevision: 3,
    })).toBe(true);

    expect(canApplySubmittedReadback({
      taskId: 'task-1',
      syncedForTaskId: undefined,
      submittedDraftRevision: 3,
      currentDraftRevision: 4,
    })).toBe(false);

    expect(canApplySubmittedReadback({
      taskId: 'task-1',
      syncedForTaskId: 'task-1',
      submittedDraftRevision: 3,
      currentDraftRevision: 3,
    })).toBe(false);
  });
});
