import { act, renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { PropsWithChildren } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const { api } = vi.hoisted(() => ({
  api: {
    validateScriptImport: vi.fn(),
    createImportedScript: vi.fn(),
    replaceImportedScript: vi.fn(),
    createScriptExecution: vi.fn(),
  },
}));

vi.mock('../../../services/api/mmlApi', () => ({ mmlApi: api }));

import {
  useCreateImportedMMLScript,
  useCreateMMLScriptExecution,
  useReplaceImportedMMLScript,
  useValidateMMLScriptImport,
} from '../useMML';

function wrapper(queryClient: QueryClient) {
  return function QueryWrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('MML TXT import hooks', () => {
  it('validates the selected TXT file through the shared API', async () => {
    api.validateScriptImport.mockResolvedValue({ validationToken: 'token', planItems: [], summary: {}, issues: [] });
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    const { result } = renderHook(() => useValidateMMLScriptImport(), { wrapper: wrapper(queryClient) });
    const file = new File(['LST DEVICE_INFO;SN1'], 'script.txt');

    await act(async () => result.current.mutateAsync(file));

    expect(api.validateScriptImport).toHaveBeenCalledWith(file);
  });

  it('invalidates script lists after saving an imported script', async () => {
    api.createImportedScript.mockResolvedValue({ id: 'script-1' });
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
    const { result } = renderHook(() => useCreateImportedMMLScript(), { wrapper: wrapper(queryClient) });

    await act(async () => result.current.mutateAsync({ validationToken: 'token', scriptName: '巡检', description: '', tags: [] }));

    await waitFor(() => expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['mml', 'scripts'] }));
  });

  it('invalidates list and detail caches after replacing an imported script', async () => {
    api.replaceImportedScript.mockResolvedValue({ id: 'script-1' });
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
    const { result } = renderHook(() => useReplaceImportedMMLScript(), { wrapper: wrapper(queryClient) });

    await act(async () => result.current.mutateAsync({ id: 'script-1', input: { validationToken: 'token', scriptName: '替换', description: '', tags: [], expectedUpdatedAt: '2026-07-10T00:00:00Z' } }));

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['mml', 'scripts'] });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['mml', 'scripts', 'script-1'] });
    });
  });

  it('invalidates task lists after creating script execution', async () => {
    api.createScriptExecution.mockResolvedValue({ task: { id: 'task-1' }, validation: { issues: [] } });
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
    const { result } = renderHook(() => useCreateMMLScriptExecution(), { wrapper: wrapper(queryClient) });

    await act(async () => result.current.mutateAsync({ id: 'script-1', input: { taskName: '执行', executeType: 'immediate', offlineRetry: false, offlineRetryWait: 60, failedRetry: false, failedRetryCount: 3, failedRetryInterval: 5, confirmWarnings: false } }));

    await waitFor(() => expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['mml', 'tasks'] }));
  });
});
