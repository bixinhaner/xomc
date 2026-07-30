import { type PropsWithChildren } from 'react';
import { act, renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { adminApi } from '../../services/api/adminApi';
import { useAppStore } from '../../store/appStore';
import { useUserStore } from '../../store/userStore';
import { usePublicOmcName } from './useOmcName';
import { usePublicSecuritySettings } from './useSecuritySettings';
import { useBatchUpdateSysConfigs } from './useSystem';
import { SYSTEM_TIMEZONE_QUERY_KEY, useSystemTimezone } from './useSystemTimezone';
import { UI_CUSTOM_DEFAULTS, usePublicUICustom } from './useUICustom';

function createQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
}

function createWrapper(queryClient: QueryClient) {
  return function QueryWrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

function consoleOutput(spy: ReturnType<typeof vi.spyOn>) {
  return spy.mock.calls.flat().join(' ');
}

describe('config query error handling', () => {
  afterEach(() => {
    vi.restoreAllMocks();
    act(() => {
      useUserStore.setState({ isAuthenticated: false });
      useAppStore.setState({ omcName: undefined, systemTimezone: undefined });
    });
  });

  it('OMC 名称查询失败时保留回退行为并向 React Query 传播原始错误', async () => {
    const failure = new Error('network unavailable');
    vi.spyOn(adminApi, 'getPublicSysConfigsByCategory').mockRejectedValue(failure);
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const queryClient = createQueryClient();

    const { result } = renderHook(() => usePublicOmcName(), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => {
      expect(queryClient.getQueryState(['public', 'sysConfig', 'basic'])?.status).toBe('error');
    });
    expect(queryClient.getQueryState(['public', 'sysConfig', 'basic'])?.error).toBe(failure);
    expect(result.current.omcName).toBeUndefined();
    expect(consoleOutput(consoleError)).not.toContain('Query data cannot be undefined');
  });

  it('公开安全配置查询失败时保留未配置回退并传播原始错误', async () => {
    const failure = new Error('network unavailable');
    vi.spyOn(adminApi, 'getPublicSysConfigsByCategory').mockRejectedValue(failure);
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const queryClient = createQueryClient();

    const { result } = renderHook(() => usePublicSecuritySettings(), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => {
      expect(queryClient.getQueryState(['public', 'sysConfig', 'security'])?.status).toBe('error');
    });
    expect(queryClient.getQueryState(['public', 'sysConfig', 'security'])?.error).toBe(failure);
    expect(result.current.settings).toBeUndefined();
    expect(consoleOutput(consoleError)).not.toContain('Query data cannot be undefined');
  });

  it('UI 定制查询失败时保留静态资源回退并传播原始错误', async () => {
    const failure = new Error('network unavailable');
    vi.spyOn(adminApi, 'getPublicSysConfigsByCategory').mockRejectedValue(failure);
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const queryClient = createQueryClient();

    const { result } = renderHook(() => usePublicUICustom(), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => {
      expect(queryClient.getQueryState(['public', 'sysConfig', 'ui_custom'])?.status).toBe('error');
    });
    expect(queryClient.getQueryState(['public', 'sysConfig', 'ui_custom'])?.error).toBe(failure);
    expect(result.current.settings).toEqual(UI_CUSTOM_DEFAULTS);
    expect(consoleOutput(consoleError)).not.toContain('Query data cannot be undefined');
  });

  it('系统时区查询失败时保留持久化值并传播原始错误', async () => {
    const failure = new Error('network unavailable');
    vi.spyOn(adminApi, 'getSysConfigsByCategory').mockRejectedValue(failure);
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const queryClient = createQueryClient();
    act(() => {
      useUserStore.setState({ isAuthenticated: true });
      useAppStore.setState({ systemTimezone: 'Asia/Shanghai' });
    });

    const { result } = renderHook(() => useSystemTimezone(), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => {
      expect(queryClient.getQueryState(['sysConfig', 'basic', 'timezone'])?.status).toBe('error');
    });
    expect(queryClient.getQueryState(['sysConfig', 'basic', 'timezone'])?.error).toBe(failure);
    expect(result.current.systemTimezone).toBe('Asia/Shanghai');
    expect(consoleOutput(consoleError)).not.toContain('Query data cannot be undefined');
  });

  it('保存 basic.timezoneCode 成功后立即同步全局系统时区', async () => {
    vi.spyOn(adminApi, 'batchUpdateSysConfigs').mockResolvedValue({
      updated: 1,
      batch: {
        id: 'batch-timezone',
        category: 'basic',
        configVersion: 2,
        status: 'applied',
        createdAt: '',
        updatedAt: '',
        targets: [],
      },
    });
    const queryClient = createQueryClient();
    queryClient.setQueryData(SYSTEM_TIMEZONE_QUERY_KEY, [
      {
        id: 'tz',
        category: 'basic',
        key: 'timezoneCode',
        value: 'UTC',
        valueType: 'string',
      },
    ]);
    act(() => {
      useAppStore.setState({ systemTimezone: 'UTC' });
    });

    const { result } = renderHook(() => useBatchUpdateSysConfigs(), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({
        category: 'basic',
        items: [{ key: 'timezoneCode', value: 'Asia/Shanghai', value_type: 'string' }],
      });
    });

    expect(useAppStore.getState().systemTimezone).toBe('Asia/Shanghai');
    expect(queryClient.getQueryData(SYSTEM_TIMEZONE_QUERY_KEY)).toMatchObject([
      { key: 'timezoneCode', value: 'Asia/Shanghai' },
    ]);
  });
});
