import { act, renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { PropsWithChildren } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const { mmlApiMock, paramModelApiMock } = vi.hoisted(() => ({
  mmlApiMock: {
    updateTemplate: vi.fn(),
  },
  paramModelApiMock: {
    updateStandard: vi.fn(),
    deleteMapping: vi.fn(),
  },
}));

vi.mock('../../../services/api/mmlApi', () => ({ mmlApi: mmlApiMock }));
vi.mock('../../../services/api/paramModelApi', () => ({ paramModelApi: paramModelApiMock }));

import { useUpdateMMLTemplate } from '../useMML';
import { useDeleteMapping, useUpsertStandard } from '../useParamModels';

const CUSTOM_PATH_QUERY_KEY = ['mml', 'console', 'custom-command-paths'] as const;
const CUSTOM_COMMANDS_QUERY_KEY = ['mml', 'custom-commands'] as const;

function wrapper(queryClient: QueryClient) {
  return function QueryWrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('MML custom Path cache invalidation', () => {
  it('invalidates enriched custom paths after editing a custom command', async () => {
    mmlApiMock.updateTemplate.mockResolvedValue({ id: 'custom-1' });
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
    const { result } = renderHook(() => useUpdateMMLTemplate(), {
      wrapper: wrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({
        id: 'custom-1',
        data: { paramPaths: ['Device.Info.Name'] },
      });
    });

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: CUSTOM_PATH_QUERY_KEY });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: CUSTOM_COMMANDS_QUERY_KEY });
    });
  });

  it('invalidates enriched custom paths after changing standard metadata', async () => {
    paramModelApiMock.updateStandard.mockResolvedValue({});
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
    const { result } = renderHook(() => useUpsertStandard(), {
      wrapper: wrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({
        path: 'Device.Info.Name',
        input: {
          standardPath: 'Device.Info.Name',
          entryType: 'parameter',
          access: 'READ_WRITE',
          dataType: 'STRING',
        },
      });
    });

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: CUSTOM_PATH_QUERY_KEY });
    });
  });

  it('invalidates product-filtered custom commands after changing mappings', async () => {
    paramModelApiMock.deleteMapping.mockResolvedValue(undefined);
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
    const { result } = renderHook(() => useDeleteMapping(), {
      wrapper: wrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({
        name: 'BAIBLQ',
        id: 'mapping-1',
      });
    });

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: CUSTOM_COMMANDS_QUERY_KEY });
    });
  });
});
