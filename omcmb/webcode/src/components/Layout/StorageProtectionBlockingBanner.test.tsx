import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type {
  StorageProtectionPolicy,
  StorageProtectionTarget,
} from '@core/services/api/storageProtectionApi';
import StorageProtectionBlockingBanner from './StorageProtectionBlockingBanner';

const mocks = vi.hoisted(() => ({
  navigate: vi.fn(),
  openTab: vi.fn(),
  policies: [] as StorageProtectionPolicy[],
  targets: [] as StorageProtectionTarget[],
  tKeys: [] as string[],
}));

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mocks.navigate,
  };
});

vi.mock('@core/hooks/api/useStorageProtection', () => ({
  useStorageProtectionPolicies: () => ({ data: mocks.policies }),
  useStorageProtectionTargets: () => ({ data: mocks.targets }),
}));

vi.mock('@core/store/tabStore', () => ({
  useTabStore: (selector: (state: { openTab: typeof mocks.openTab }) => unknown) =>
    selector({ openTab: mocks.openTab }),
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (key: string, values?: Record<string, string | number>) => {
    mocks.tKeys.push(key);
    if (key === 'system.storageProtection.globalBlock.description' && values) {
      return `${key}:${values.target}:${values.usedPercent}:${values.blockPercent}:${values.recoverPercent}`;
    }
    return key;
  },
}));

function blockedPolicy(overrides: Partial<StorageProtectionPolicy> = {}): StorageProtectionPolicy {
  return {
    id: 'policy-1',
    targetType: 'filesystem',
    targetId: 'root',
    writeScope: 'all',
    enabled: true,
    warnUsedPercent: 80,
    recoverUsedPercent: 85,
    blockUsedPercent: 90,
    checkIntervalSeconds: 30,
    unknownBehavior: 'allow_with_alarm',
    currentState: 'blocked',
    stateObservations: 2,
    lastObservedRatio: 0.91,
    ...overrides,
  };
}

function target(overrides: Partial<StorageProtectionTarget> = {}): StorageProtectionTarget {
  return {
    targetType: 'filesystem',
    targetId: 'root',
    mountpoint: '/',
    components: [],
    sourcePaths: ['/data/omc'],
    protectedPaths: ['/'],
    capacityBytes: 100,
    usedBytes: 92,
    usedRatio: 0.923,
    available: true,
    currentState: 'blocked',
    writeScopes: ['all'],
    ...overrides,
  };
}

function renderBanner() {
  return render(
    <MemoryRouter>
      <StorageProtectionBlockingBanner />
    </MemoryRouter>,
  );
}

describe('StorageProtectionBlockingBanner', () => {
  beforeEach(() => {
    mocks.navigate.mockReset();
    mocks.openTab.mockReset();
    mocks.tKeys = [];
    mocks.policies = [];
    mocks.targets = [];
  });

  it('storage protection blocked 时显示常驻告警并使用 i18n key', () => {
    mocks.policies = [blockedPolicy()];
    mocks.targets = [target()];

    renderBanner();

    expect(screen.getByText('system.storageProtection.globalBlock.title')).toBeInTheDocument();
    expect(screen.getByText('system.storageProtection.globalBlock.description:/:92.3%:90.0%:85.0%')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'system.storageProtection.globalBlock.viewDetail' })).toBeInTheDocument();
    expect(mocks.tKeys).toEqual(expect.arrayContaining([
      'system.storageProtection.globalBlock.title',
      'system.storageProtection.globalBlock.description',
      'system.storageProtection.globalBlock.viewDetail',
    ]));
    expect(screen.getAllByRole('button')).toHaveLength(1);
  });

  it('点击查看详情时打开资源保留与背压页面', () => {
    mocks.policies = [blockedPolicy()];
    mocks.targets = [target()];

    renderBanner();
    fireEvent.click(screen.getByRole('button', { name: 'system.storageProtection.globalBlock.viewDetail' }));

    expect(mocks.openTab).toHaveBeenCalledWith({
      key: 'system-config-retention-bp',
      label: 'system.config.retentionBp',
      labelRaw: false,
      path: '/system/config?tab=retention_bp',
      closable: true,
    });
    expect(mocks.navigate).toHaveBeenCalledWith('/system/config?tab=retention_bp');
  });

  it('状态恢复为非 blocked 后隐藏', () => {
    mocks.policies = [blockedPolicy({ currentState: 'normal' })];
    mocks.targets = [target({ currentState: 'normal', usedRatio: 0.4 })];

    renderBanner();

    expect(screen.queryByText('system.storageProtection.globalBlock.title')).not.toBeInTheDocument();
  });

  it('找不到匹配 target 时使用策略目标兜底而不是误报其他目录', () => {
    mocks.policies = [blockedPolicy({ targetId: 'root' })];
    mocks.targets = [
      target({
        targetId: 'other',
        mountpoint: '/other',
        currentState: 'normal',
        usedRatio: 0.3,
      }),
    ];

    renderBanner();

    expect(screen.getByText('system.storageProtection.globalBlock.description:root:91.0%:90.0%:85.0%')).toBeInTheDocument();
  });

  it('精确匹配 target 正常但其他展开挂载点 blocked 时优先展示 blocked 目录', () => {
    mocks.policies = [blockedPolicy({ targetId: 'root' })];
    mocks.targets = [
      target({
        targetId: 'root',
        mountpoint: '/',
        currentState: 'normal',
        usedRatio: 0.3,
      }),
      target({
        targetId: 'data',
        mountpoint: '/data',
        currentState: 'blocked',
        usedRatio: 0.94,
      }),
    ];

    renderBanner();

    expect(screen.getByText('system.storageProtection.globalBlock.description:/data:94.0%:90.0%:85.0%')).toBeInTheDocument();
  });

  it('容量总览 target 已 blocked 但策略状态尚未刷新时仍显示提醒', () => {
    mocks.policies = [blockedPolicy({ currentState: 'normal' })];
    mocks.targets = [target({ mountpoint: '/data', currentState: 'blocked', usedRatio: 0.94 })];

    renderBanner();

    expect(screen.getByText('system.storageProtection.globalBlock.title')).toBeInTheDocument();
    expect(screen.getByText('system.storageProtection.globalBlock.description:/data:94.0%:90.0%:85.0%')).toBeInTheDocument();
  });

  it('容量总览 target blocked 时优先使用匹配 target 的策略阈值', () => {
    mocks.policies = [
      blockedPolicy({
        id: 'policy-root',
        targetId: 'root',
        currentState: 'normal',
        recoverUsedPercent: 85,
        blockUsedPercent: 90,
      }),
      blockedPolicy({
        id: 'policy-data',
        targetId: 'data',
        currentState: 'normal',
        recoverUsedPercent: 60,
        blockUsedPercent: 70,
      }),
    ];
    mocks.targets = [target({
      targetId: 'data',
      mountpoint: '/data',
      currentState: 'blocked',
      usedRatio: 0.94,
    })];

    renderBanner();

    expect(screen.getByText('system.storageProtection.globalBlock.description:/data:94.0%:70.0%:60.0%')).toBeInTheDocument();
  });

  it('多个目录 blocked 时展示占用最高的目录', () => {
    mocks.policies = [blockedPolicy({ targetId: 'root' })];
    mocks.targets = [
      target({
        targetId: 'logs',
        mountpoint: '/logs',
        currentState: 'blocked',
        usedRatio: 0.91,
      }),
      target({
        targetId: 'data',
        mountpoint: '/data',
        currentState: 'blocked',
        usedRatio: 0.96,
      }),
    ];

    renderBanner();

    expect(screen.getByText('system.storageProtection.globalBlock.description:/data:96.0%:90.0%:85.0%')).toBeInTheDocument();
  });
});
