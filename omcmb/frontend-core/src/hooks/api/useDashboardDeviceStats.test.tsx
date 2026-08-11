import { type PropsWithChildren } from 'react';
import { act, renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { DeviceListResponse, DeviceListStats } from '../../types/device';
import * as deviceHooks from './useDevices';
import * as dashboardHooks from './useDashboard';

type DashboardDeviceStatsHook = (
  apiScope: string,
  userScope: string | undefined,
) => {
  data: DeviceListStats | undefined;
  fetchStatus: string;
  refetch: () => Promise<unknown>;
  status: string;
};

const deviceStats: DeviceListStats = {
  total: 20_002,
  online_count: 19_572,
  offline_count: 430,
  current_ue_count: 67,
  alarmed: 0,
};

function createWrapper(queryClient: QueryClient) {
  return function QueryWrapper({ children }: PropsWithChildren) {
    return (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    );
  };
}

function dashboardDeviceStatsHook(): DashboardDeviceStatsHook | undefined {
  return (dashboardHooks as unknown as {
    useDashboardDeviceStats?: DashboardDeviceStatsHook;
  }).useDashboardDeviceStats;
}

describe('useDashboardDeviceStats', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('reuses unfiltered device-list stats with the dashboard five-minute polling policy', async () => {
    const response: DeviceListResponse = {
      items: [],
      total: deviceStats.total,
      page: 1,
      pageSize: 1,
      stats: deviceStats,
    };
    const fetchDeviceList = vi.spyOn(deviceHooks, 'fetchDeviceList')
      .mockResolvedValue(response);
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const useDashboardDeviceStats = dashboardDeviceStatsHook();

    expect(useDashboardDeviceStats).toBeTypeOf('function');

    const { result } = renderHook(
      () => useDashboardDeviceStats!('http://server-a/api/v1', 'user-a'),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => {
      expect(result.current.data).toEqual(deviceStats);
    });
    expect(fetchDeviceList).toHaveBeenCalledWith({ page: 1, pageSize: 1 });

    const query = queryClient.getQueryCache().find({
      queryKey: [
        'dashboard',
        'device-stats',
        'http://server-a/api/v1',
        'user-a',
      ],
    });
    expect(query?.options.refetchInterval).toBe(300_000);
    expect(query?.options.refetchIntervalInBackground).toBe(false);
    expect(query?.options.refetchOnWindowFocus).toBe(false);
    expect(query?.options.refetchOnReconnect).toBe(false);
  });

  it('does not request device stats before the current user is available', () => {
    const fetchDeviceList = vi.spyOn(deviceHooks, 'fetchDeviceList')
      .mockRejectedValue(new Error('must not be called'));
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const useDashboardDeviceStats = dashboardDeviceStatsHook();

    expect(useDashboardDeviceStats).toBeTypeOf('function');

    const { result } = renderHook(
      () => useDashboardDeviceStats!('http://server-a/api/v1', undefined),
      { wrapper: createWrapper(queryClient) },
    );

    expect(fetchDeviceList).not.toHaveBeenCalled();
    expect(result.current.fetchStatus).toBe('idle');
    expect(result.current.status).toBe('pending');
  });

  it('keeps the last exact stats when the list API falls back to current-page counts', async () => {
    const exactResponse: DeviceListResponse = {
      items: [],
      total: deviceStats.total,
      page: 1,
      pageSize: 1,
      stats: deviceStats,
    };
    const pageFallbackResponse: DeviceListResponse = {
      items: [],
      total: deviceStats.total,
      page: 1,
      pageSize: 1,
      stats: {
        total: deviceStats.total,
        online_count: 1,
        offline_count: 0,
        alarmed: 0,
      },
    };
    vi.spyOn(deviceHooks, 'fetchDeviceList')
      .mockResolvedValueOnce(exactResponse)
      .mockResolvedValueOnce(pageFallbackResponse);
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const useDashboardDeviceStats = dashboardDeviceStatsHook();

    expect(useDashboardDeviceStats).toBeTypeOf('function');

    const { result } = renderHook(
      () => useDashboardDeviceStats!('http://server-a/api/v1', 'user-a'),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => {
      expect(result.current.data).toEqual(deviceStats);
    });

    await act(async () => {
      await result.current.refetch();
    });

    expect(deviceHooks.fetchDeviceList).toHaveBeenCalledTimes(2);
    expect(result.current.data).toEqual(deviceStats);
  });
});
