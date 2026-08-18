import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { PropsWithChildren } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const { getCommandSubFieldsMock } = vi.hoisted(() => ({
  getCommandSubFieldsMock: vi.fn(),
}));

vi.mock('../../../services/api/mmlApi', () => ({
  mmlApi: {
    getCommandSubFields: getCommandSubFieldsMock,
  },
}));

import { useCommandSubFields } from '../useMmlConsole';

function wrapper(queryClient: QueryClient) {
  return function QueryWrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  getCommandSubFieldsMock.mockResolvedValue([]);
});

describe('useCommandSubFields', () => {
  it('重新打开命令弹框时重新校验曾缓存为空的参数路径', async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const { rerender } = renderHook(
      ({ active }: { active: boolean }) => useCommandSubFields(
        'command-halob',
        'zh-CN',
        '1202000534228JB0007',
        'FAP/BSC7041C243',
        active,
      ),
      {
        initialProps: { active: true },
        wrapper: wrapper(queryClient),
      },
    );

    await waitFor(() => expect(getCommandSubFieldsMock).toHaveBeenCalledTimes(1));

    rerender({ active: false });
    rerender({ active: true });

    await waitFor(() => expect(getCommandSubFieldsMock).toHaveBeenCalledTimes(2));
    expect(getCommandSubFieldsMock).toHaveBeenLastCalledWith(
      'command-halob',
      'zh-CN',
      '1202000534228JB0007',
      'FAP/BSC7041C243',
    );
  });
});
