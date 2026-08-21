import { act, renderHook } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { PropsWithChildren } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const { provisionApiMock } = vi.hoisted(() => ({
  provisionApiMock: {
    createPolicy: vi.fn(),
    updatePolicy: vi.fn(),
  },
}));

vi.mock('../../../services/api/provisionApi', () => ({ provisionApi: provisionApiMock }));

import { useSavePlugAndPlayPolicy } from '../useProvisioning';

function wrapper(queryClient: QueryClient) {
  return function QueryWrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

function policyWithMode(paramConfigMode: 'common' | 'specified') {
  return {
    id: 'policy-1',
    name: 'policy',
    enabled: true,
    productName: 'BNQ',
    productNames: ['BNQ'],
    productClass: 'BNQ',
    productClasses: ['BNQ'],
    executeType: 'auto' as const,
    priority: 100,
    upgradeEnabled: false,
    targetVersion: '',
    licenseEnabled: false,
    selfConfigEnabled: true,
    config: { paramConfigMode },
    createdAt: '',
    updatedAt: '',
  };
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('useSavePlugAndPlayPolicy cache update', () => {
  it('updates the single policy cache after saving so reopening edit uses the selected mode', async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    queryClient.setQueryData(['provisioning', 'policy', 'policy-1'], policyWithMode('specified'));
    provisionApiMock.updatePolicy.mockResolvedValue(policyWithMode('common'));

    const { result } = renderHook(() => useSavePlugAndPlayPolicy('policy-1'), {
      wrapper: wrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync(policyWithMode('common'));
    });

    expect(queryClient.getQueryData(['provisioning', 'policy', 'policy-1']))
      .toMatchObject({ config: { paramConfigMode: 'common' } });
  });
});
