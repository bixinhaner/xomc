import { act, renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { PropsWithChildren } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const { mmlAdminApiMock } = vi.hoisted(() => ({
  mmlAdminApiMock: {
    createSubField: vi.fn(),
    updateSubField: vi.fn(),
    deleteSubField: vi.fn(),
    batchCreateSubFields: vi.fn(),
  },
}));

vi.mock('../../../services/api/mmlAdminApi', () => ({
  mmlAdminApi: mmlAdminApiMock,
}));

import {
  useBatchCreateSubFields,
  useCreateSubField,
  useDeleteSubField,
  useUpdateSubField,
} from '../useMmlAdmin';

const CONSOLE_SUB_FIELDS_QUERY_KEY = ['mml', 'console', 'sub-fields'] as const;
const GROUP_TREE_QUERY_KEY = ['mml', 'console', 'group-tree'] as const;

interface TestMutation {
  mutateAsync: (input: unknown) => Promise<unknown>;
}

function wrapper(queryClient: QueryClient) {
  return function QueryWrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  mmlAdminApiMock.createSubField.mockResolvedValue({});
  mmlAdminApiMock.updateSubField.mockResolvedValue({});
  mmlAdminApiMock.deleteSubField.mockResolvedValue(undefined);
  mmlAdminApiMock.batchCreateSubFields.mockResolvedValue([]);
});

const cases = [
  {
    name: 'create',
    useHook: useCreateSubField as unknown as () => TestMutation,
    input: {
      commandId: 'command-1',
      req: { standardPathId: 'standard-1' },
    },
  },
  {
    name: 'update',
    useHook: useUpdateSubField as unknown as () => TestMutation,
    input: {
      commandId: 'command-1',
      subFieldId: 'sub-field-1',
      req: { defaultSelected: false },
    },
  },
  {
    name: 'delete',
    useHook: useDeleteSubField as unknown as () => TestMutation,
    input: {
      commandId: 'command-1',
      subFieldId: 'sub-field-1',
    },
  },
  {
    name: 'batch create',
    useHook: useBatchCreateSubFields as unknown as () => TestMutation,
    input: {
      commandId: 'command-1',
      req: { standardPathIds: ['standard-1', 'standard-2'] },
    },
  },
] as const;

describe('MML admin Console sub-field invalidation', () => {
  it.each(cases)('invalidates Console sub-fields after $name', async ({ useHook, input }) => {
    const queryClient = new QueryClient({
      defaultOptions: { mutations: { retry: false } },
    });
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
    const { result } = renderHook(() => useHook(), {
      wrapper: wrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync(input);
    });

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: CONSOLE_SUB_FIELDS_QUERY_KEY,
      });
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: GROUP_TREE_QUERY_KEY,
      });
    });
  });
});
