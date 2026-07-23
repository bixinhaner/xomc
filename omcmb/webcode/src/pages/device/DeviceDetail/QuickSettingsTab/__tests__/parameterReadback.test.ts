import { describe, expect, it } from 'vitest';
import {
  ParameterReadbackTimeoutError,
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
});
