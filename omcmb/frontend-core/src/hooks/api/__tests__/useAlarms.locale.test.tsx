import { type PropsWithChildren } from 'react';
import { act, renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { alarmApi } from '../../../services/api/alarmApi';
import { useAppStore } from '../../../store/appStore';
import { useCurrentAlarms, useHistoricalAlarms } from '../useAlarms';

vi.mock('../../../services/apiSwitch', () => ({
  createApiSwitchWithMock: (_mock: unknown, real: unknown) => real,
}));

const emptyPage = { items: [], total: 0, page: 1, pageSize: 20 };
const params = { page: 1, pageSize: 20 };

function createWrapper(queryClient: QueryClient) {
  return function QueryWrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
}

describe('alarm list locale query keys', () => {
  beforeEach(() => {
    useAppStore.setState({ locale: 'zh-CN' });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    useAppStore.setState({ locale: 'zh-CN' });
  });

  it('refetches current alarms when locale changes', async () => {
    const getCurrentAlarms = vi.spyOn(alarmApi, 'getCurrentAlarms').mockResolvedValue(emptyPage);
    const queryClient = createQueryClient();
    const { unmount } = renderHook(
      () => useCurrentAlarms(params, { refetchIntervalMs: false }),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(getCurrentAlarms).toHaveBeenCalledTimes(1));

    act(() => useAppStore.getState().setLocale('en-US'));

    await waitFor(() => expect(getCurrentAlarms).toHaveBeenCalledTimes(2));
    expect(queryClient.getQueryCache().findAll({ queryKey: ['alarms', 'current'] }))
      .toHaveLength(2);

    unmount();
    queryClient.clear();
  });

  it('refetches historical alarms when locale changes', async () => {
    const getHistoricalAlarms = vi.spyOn(alarmApi, 'getHistoricalAlarms').mockResolvedValue(emptyPage);
    const queryClient = createQueryClient();
    const { unmount } = renderHook(
      () => useHistoricalAlarms(params, { refetchIntervalMs: false }),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(getHistoricalAlarms).toHaveBeenCalledTimes(1));

    act(() => useAppStore.getState().setLocale('en-US'));

    await waitFor(() => expect(getHistoricalAlarms).toHaveBeenCalledTimes(2));
    expect(queryClient.getQueryCache().findAll({ queryKey: ['alarms', 'historical'] }))
      .toHaveLength(2);

    unmount();
    queryClient.clear();
  });
});
