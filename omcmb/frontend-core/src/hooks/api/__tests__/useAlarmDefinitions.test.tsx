import { act, renderHook } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { PropsWithChildren } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { UpdateAlarmDefinitionInput } from '../../../types/alarmDefinition';

const { api, mockService } = vi.hoisted(() => ({
  api: { update: vi.fn() },
  mockService: { update: vi.fn() },
}));

vi.mock('../../../services/api/alarmDefinitionApi', () => ({
  alarmDefinitionApi: api,
}));
vi.mock('../../../mock/services/alarmDefinitionService', () => ({
  alarmDefinitionService: mockService,
}));

import { useUpdateAlarmDefinition } from '../useAlarmDefinitions';

function wrapper(queryClient: QueryClient) {
  return function QueryWrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

const input: UpdateAlarmDefinitionInput = {
  neType: 'ENB',
  cnName: 'x',
  enName: 'x',
  severityCode: 31001,
  isShow: true,
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe('useUpdateAlarmDefinition', () => {
  it('#268 成功后失效整个 alarm-definitions 族(含一级表 ne-types 统计)', async () => {
    api.update.mockResolvedValue({});
    mockService.update.mockResolvedValue({});
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    // 预置 update 此前不会触碰的三类缓存:一级聚合 / 列表 / 详情。
    queryClient.setQueryData(['alarm-definitions', 'ne-types'], { items: [] });
    queryClient.setQueryData(['alarm-definitions', 'list', { page: 1 }], { items: [] });
    queryClient.setQueryData(['alarm-definitions', 'detail', '11109'], {});
    // setQueryData 会把查询标记为 fresh,先确认基线:均未失效。
    expect(
      queryClient.getQueryState(['alarm-definitions', 'ne-types'])?.isInvalidated,
    ).toBe(false);

    const { result } = renderHook(() => useUpdateAlarmDefinition(), {
      wrapper: wrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ identifier: '11109', input });
    });

    for (const key of [
      ['alarm-definitions', 'ne-types'],
      ['alarm-definitions', 'list', { page: 1 }],
      ['alarm-definitions', 'detail', '11109'],
    ]) {
      expect(queryClient.getQueryState(key)?.isInvalidated, String(key)).toBe(true);
    }
  });
});
