import { describe, expect, it, vi } from 'vitest';
import { QueryClient } from '@tanstack/react-query';
import { invalidateDeviceAccessQueries } from './useDeviceAccess';

describe('device-access cache invalidation', () => {
  it('does not keep a successful mutation pending while active queries refresh', () => {
    const neverSettles = new Promise<never>(() => undefined);
    const queryClient = new QueryClient();
    const invalidateQueries = vi.spyOn(queryClient, 'invalidateQueries').mockReturnValue(neverSettles);

    const result = invalidateDeviceAccessQueries(queryClient);

    expect(result).toBeUndefined();
    expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ['device-access'] });
  });
});
