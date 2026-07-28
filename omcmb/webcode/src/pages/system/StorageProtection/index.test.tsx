import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import StorageProtection from './index';

const mocks = vi.hoisted(() => ({
  useSystemInfo: vi.fn(),
  useStorageProtectionPolicies: vi.fn(),
  useStorageProtectionEvents: vi.fn(),
  useSaveStorageProtectionPolicy: vi.fn(),
  useUpdateStorageProtectionPolicy: vi.fn(),
}));

vi.mock('@core/hooks/api/useSystem', () => ({ useSystemInfo: mocks.useSystemInfo }));
vi.mock('@core/hooks/api/useStorageProtection', () => ({
  useStorageProtectionPolicies: mocks.useStorageProtectionPolicies,
  useStorageProtectionEvents: mocks.useStorageProtectionEvents,
  useSaveStorageProtectionPolicy: mocks.useSaveStorageProtectionPolicy,
  useUpdateStorageProtectionPolicy: mocks.useUpdateStorageProtectionPolicy,
}));
vi.mock('@/hooks/useT', () => ({ useT: () => (key: string) => key }));

describe('StorageProtection', () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it('renders capacity, policy, blocked state and audit sections from API data', () => {
    mocks.useSystemInfo.mockReturnValue({
      data: {
        storage: [{
          id: 'minio-data', kind: 'minio_cluster', label: 'MinIO', source: 'prometheus/minio',
          status: 'available', usedBytes: 90, totalBytes: 100, usedPercent: 90,
        }],
      },
    });
    mocks.useStorageProtectionPolicies.mockReturnValue({
      data: [{
        id: 'policy-1', targetType: 'minio', targetId: 'minio-data', writeScope: 'upload',
        enabled: true, warnUsedPercent: 80, blockUsedPercent: 90, recoverUsedPercent: 85,
        checkIntervalSeconds: 30, unknownBehavior: 'allow_with_alarm', currentState: 'blocked',
        stateObservations: 0, lastObservedRatio: 0.9,
      }],
      isFetching: false,
      refetch: vi.fn(),
    });
    mocks.useStorageProtectionEvents.mockReturnValue({
      data: [{
        policyId: 'policy-1', targetType: 'minio', targetId: 'minio-data', writeScope: 'upload',
        newState: 'blocked', reason: 'capacity threshold reached', policyVersion: 1,
        operatorId: 'system', createdAt: '2026-07-28T00:00:00Z',
      }],
      isFetching: false,
    });
    mocks.useSaveStorageProtectionPolicy.mockReturnValue({ mutateAsync: vi.fn(), isPending: false });
    mocks.useUpdateStorageProtectionPolicy.mockReturnValue({ mutateAsync: vi.fn(), isPending: false });

    render(<StorageProtection />);

    expect(screen.getByText('system.storageProtection.capacityOverview')).toBeInTheDocument();
    expect(screen.getByText('system.storageProtection.policyList')).toBeInTheDocument();
    expect(screen.getByText('system.storageProtection.currentBlocks')).toBeInTheDocument();
    expect(screen.getByText('system.storageProtection.audit')).toBeInTheDocument();
    expect(screen.getAllByText('system.storageProtection.state.blocked').length).toBeGreaterThan(0);
  });
});
