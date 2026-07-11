import { act, renderHook } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { PropsWithChildren } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { QueryTemplate } from '../../../types/pmQuery';

const { api } = vi.hoisted(() => ({
  api: {
    update: vi.fn(),
  },
}));

vi.mock('../../../services/api/pmQueryApi', () => ({ pmQueryApi: api }));

import { useUpdateQueryTemplate } from '../usePmQuery';

function wrapper(queryClient: QueryClient) {
  return function QueryWrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

function template(granularity: 'daily' | 'hourly'): QueryTemplate {
  return {
    id: 'template-36',
    name: 'Issue 36',
    visibility: 'private',
    creatorId: 'user-1',
    payload: {
      deviceSns: ['SN-1'],
      metricPaths: ['KPI-1'],
      granularity,
      timeRangePreset: 'last_7d',
      deviceType: 'ENB',
    },
    createdAt: '2026-07-11T00:00:00Z',
    updatedAt: '2026-07-11T01:00:00Z',
  };
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('useUpdateQueryTemplate', () => {
  it('writes the PATCH response into list and detail caches', async () => {
    const daily = template('daily');
    const hourly = template('hourly');
    api.update.mockResolvedValue(hourly);
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const listKey = ['pm-query-templates', 'list', { pageSize: 200 }] as const;
    queryClient.setQueryData(listKey, { items: [daily], total: 1 });
    const { result } = renderHook(() => useUpdateQueryTemplate(), {
      wrapper: wrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({
        id: hourly.id,
        input: { payload: hourly.payload },
      });
    });

    expect(queryClient.getQueryData<{ items: QueryTemplate[]; total: number }>(listKey)?.items[0].payload.granularity)
      .toBe('hourly');
    expect(queryClient.getQueryData<QueryTemplate>(['pm-query-templates', 'detail', hourly.id])?.payload.granularity)
      .toBe('hourly');
  });
});
