import { type PropsWithChildren } from 'react';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { dashboardApi } from '../../services/api/dashboardApi';
import type { DashboardSummary } from '../../types/dashboard';
import { useDashboardSummary } from './useDashboard';

function createSummary(total: number): DashboardSummary {
  return {
    deviceCounts: {
      total,
      online: total,
      offline: 0,
      alarm: 0,
    },
    alarmCounts: {
      critical: 0,
      major: 0,
      minor: 0,
      warning: 0,
      total: 0,
    },
    kpiSummary: {},
    kpiDeltas: {},
    taskSummary: {
      running: 0,
      pending: 0,
      success: 0,
      failed: 0,
    },
  };
}

function createWrapper(queryClient: QueryClient) {
  return function QueryWrapper({ children }: PropsWithChildren) {
    return (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    );
  };
}

describe('useDashboardSummary user scope', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('requests a separate Summary when the current user changes directly', async () => {
    const summaryA = createSummary(10);
    const summaryB = createSummary(20);
    const getSummary = vi.spyOn(dashboardApi, 'getSummary')
      .mockResolvedValueOnce(summaryA)
      .mockResolvedValueOnce(summaryB);
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
          staleTime: 30_000,
        },
      },
    });

    const { result, rerender } = renderHook(
      ({ userScope }: { userScope: string }) =>
        useDashboardSummary('http://server-a/api/v1', userScope),
      {
        initialProps: { userScope: 'user-a' },
        wrapper: createWrapper(queryClient),
      },
    );

    await waitFor(() => {
      expect(result.current.data).toEqual(summaryA);
    });

    rerender({ userScope: 'user-b' });

    await waitFor(() => {
      expect(getSummary).toHaveBeenCalledTimes(2);
      expect(result.current.data).toEqual(summaryB);
    });
    expect(
      queryClient.getQueryData([
        'dashboard',
        'summary',
        'http://server-a/api/v1',
        'user-a',
      ]),
    ).toEqual(summaryA);
    expect(
      queryClient.getQueryData([
        'dashboard',
        'summary',
        'http://server-a/api/v1',
        'user-b',
      ]),
    ).toEqual(summaryB);
  });

  it('does not request Summary before the current user is available', () => {
    const getSummary = vi.spyOn(dashboardApi, 'getSummary')
      .mockResolvedValue(createSummary(10));
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
      },
    });

    const { result } = renderHook(
      () => useDashboardSummary('http://server-a/api/v1', undefined),
      {
        wrapper: createWrapper(queryClient),
      },
    );

    expect(getSummary).not.toHaveBeenCalled();
    expect(result.current.fetchStatus).toBe('idle');
    expect(result.current.status).toBe('pending');
  });
});
