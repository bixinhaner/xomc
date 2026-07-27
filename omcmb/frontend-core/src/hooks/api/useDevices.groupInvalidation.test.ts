import { describe, expect, it, vi } from 'vitest';
import { invalidateDeviceGroupCaches } from './useDevices';

describe('invalidateDeviceGroupCaches', () => {
  it('waits until both group caches have finished invalidating', async () => {
    let resolveFirst!: () => void;
    let resolveSecond!: () => void;
    const first = new Promise<void>((resolve) => {
      resolveFirst = resolve;
    });
    const second = new Promise<void>((resolve) => {
      resolveSecond = resolve;
    });
    const invalidateQueries = vi.fn()
      .mockReturnValueOnce(first)
      .mockReturnValueOnce(second);
    const settled = vi.fn();

    const pending = invalidateDeviceGroupCaches({ invalidateQueries }).then(settled);
    await Promise.resolve();
    expect(settled).not.toHaveBeenCalled();

    resolveFirst();
    await Promise.resolve();
    expect(settled).not.toHaveBeenCalled();

    resolveSecond();
    await pending;
    expect(settled).toHaveBeenCalledOnce();
    expect(invalidateQueries).toHaveBeenNthCalledWith(1, { queryKey: ['devices', 'groups'] });
    expect(invalidateQueries).toHaveBeenNthCalledWith(2, { queryKey: ['system', 'deviceGroups', 'all'] });
  });
});
