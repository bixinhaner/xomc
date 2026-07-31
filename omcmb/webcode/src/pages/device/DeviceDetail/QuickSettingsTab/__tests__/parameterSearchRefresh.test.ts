import { QueryClient, QueryObserver } from '@tanstack/react-query';
import { describe, expect, it } from 'vitest';
import { applyDeviceParameterSearchReadback } from '../parameterSearchRefresh';

describe('quick-settings parameter search refresh', () => {
  it('applies schema readback even while the search endpoint still returns stale data', async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });
    let calls = 0;
    const observer = new QueryObserver(queryClient, {
      queryKey: ['devices', 'parameters', 'search', 'device-1', 'ExistPlmnidList', 50],
      queryFn: async () => {
        calls += 1;
        return [{
          parameterPath: 'Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList',
          parameterValue: '44190,44191,44192',
        }];
      },
    });
    const unsubscribe = observer.subscribe(() => undefined);

    await observer.refetch();
    applyDeviceParameterSearchReadback(queryClient, {
      deviceId: 'device-1',
      searchQuery: 'ExistPlmnidList',
      replacePathPrefix: 'Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList',
      parameters: [{
        parameterPath: 'Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList',
        parameterValue: '44190,44192,44193',
      }],
    });

    expect(observer.getCurrentResult().data).toEqual([{
      parameterPath: 'Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList',
      parameterValue: '44190,44192,44193',
    }]);
    expect(calls).toBe(1);
    unsubscribe();
    queryClient.clear();
  });
});
