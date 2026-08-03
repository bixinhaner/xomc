import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import StorageProtection from './index';

const mocks = vi.hoisted(() => ({
  useSystemInfo: vi.fn(),
  useSysConfigsByCategory: vi.fn(),
  useBatchUpdateSysConfigs: vi.fn(),
  useStorageProtectionPolicies: vi.fn(),
  useSaveStorageProtectionPolicy: vi.fn(),
  useUpdateStorageProtectionPolicy: vi.fn(),
}));

vi.mock('@core/hooks/api/useSystem', () => ({
  useSystemInfo: mocks.useSystemInfo,
  useSysConfigsByCategory: mocks.useSysConfigsByCategory,
  useBatchUpdateSysConfigs: mocks.useBatchUpdateSysConfigs,
}));
vi.mock('@core/hooks/api/useStorageProtection', () => ({
  useStorageProtectionPolicies: mocks.useStorageProtectionPolicies,
  useSaveStorageProtectionPolicy: mocks.useSaveStorageProtectionPolicy,
  useUpdateStorageProtectionPolicy: mocks.useUpdateStorageProtectionPolicy,
}));
vi.mock('@/hooks/useT', () => ({ useT: () => (key: string) => key }));

describe('StorageProtection', () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  beforeEach(() => {
    mocks.useSysConfigsByCategory.mockReturnValue({
      data: [],
      isLoading: false,
      isFetching: false,
      isError: false,
      isSuccess: true,
      refetch: vi.fn(),
    });
    mocks.useBatchUpdateSysConfigs.mockReturnValue({ mutateAsync: vi.fn() });
  });

  it('renders the singleton policy summary and hides the persistent create form', () => {
    mocks.useSystemInfo.mockReturnValue({
      data: {
        storage: [{
          id: 'host-root', kind: 'host_filesystem', label: '/', source: 'prometheus/node_exporter', mountpoint: '/',
          status: 'available', usedBytes: 90, totalBytes: 100, usedPercent: 90,
        }],
      },
    });
    mocks.useStorageProtectionPolicies.mockReturnValue({
      data: [{
        id: 'policy-1', targetType: 'filesystem', targetId: 'root', writeScope: 'all',
        enabled: true, warnUsedPercent: 80, blockUsedPercent: 90, recoverUsedPercent: 85,
        checkIntervalSeconds: 30, unknownBehavior: 'allow_with_alarm', currentState: 'blocked',
        stateObservations: 0, lastObservedRatio: 0.9,
      }],
      isFetching: false,
      refetch: vi.fn(),
    });
    mocks.useSaveStorageProtectionPolicy.mockReturnValue({ mutateAsync: vi.fn(), isPending: false });
    mocks.useUpdateStorageProtectionPolicy.mockReturnValue({ mutateAsync: vi.fn(), isPending: false });

    render(<StorageProtection />);

    expect(screen.getByText('system.storageProtection.capacityOverview')).toBeInTheDocument();
    expect(screen.getByText('system.storageProtection.policySummary')).toBeInTheDocument();
    expect(screen.getByText('system.storageProtection.currentBlocks')).toBeInTheDocument();
    expect(screen.getByText('system.storageProtection.retentionTitle')).toBeInTheDocument();
    expect(screen.getByText('logCfg.retention.title')).toBeInTheDocument();
    expect(screen.queryByText('system.storageProtection.audit')).not.toBeInTheDocument();
    expect(screen.queryByText('system.storageProtection.newPolicy')).not.toBeInTheDocument();
    expect(screen.getByText('common.edit')).toBeInTheDocument();
    expect(screen.getAllByText('system.storageProtection.state.blocked').length).toBeGreaterThan(0);
  });

  it('shows a one-time configuration entry when no policy exists', () => {
    mocks.useSystemInfo.mockReturnValue({ data: { storage: [] } });
    mocks.useStorageProtectionPolicies.mockReturnValue({ data: [], isFetching: false, refetch: vi.fn() });
    mocks.useSaveStorageProtectionPolicy.mockReturnValue({ mutateAsync: vi.fn(), isPending: false });
    mocks.useUpdateStorageProtectionPolicy.mockReturnValue({ mutateAsync: vi.fn(), isPending: false });

    render(<StorageProtection />);

    expect(screen.getByText('system.storageProtection.noPolicyDescription')).toBeInTheDocument();
    expect(screen.getByText('system.storageProtection.configurePolicy')).toBeInTheDocument();
    expect(screen.getByText('system.storageProtection.retentionTitle')).toBeInTheDocument();
    expect(screen.getByText('logCfg.rotation.title')).toBeInTheDocument();
    expect(screen.queryByText('system.storageProtection.newPolicy')).not.toBeInTheDocument();
  });
});
