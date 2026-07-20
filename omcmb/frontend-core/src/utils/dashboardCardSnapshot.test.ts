import { beforeEach, describe, expect, it } from 'vitest';
import type { DashboardSummary } from '../types/dashboard';
import {
  createDashboardCardSnapshot,
  dashboardCardSnapshotStorageKey,
  readDashboardCardSnapshot,
  resolveDashboardCardApiScope,
  resolveDashboardCardDisplay,
  writeDashboardCardSnapshot,
  type DashboardCardSnapshot,
} from './dashboardCardSnapshot';

const createSummary = (overrides?: Partial<DashboardSummary>): DashboardSummary => ({
  deviceCounts: {
    total: 20,
    online: 15,
    offline: 5,
    alarm: 2,
  },
  alarmCounts: {
    critical: 1,
    major: 1,
    minor: 1,
    warning: 1,
    total: 4,
  },
  kpiSummary: {
    UE_ACTIVE: 8.9,
  },
  kpiDeltas: {},
  taskSummary: {
    running: 0,
    pending: 0,
    success: 0,
    failed: 0,
  },
  ...overrides,
});

const previousSnapshot: DashboardCardSnapshot = {
  totalDevices: 10,
  onlineDevices: 6,
  activeAlarms: 2,
  activeUE: 3,
  updatedAt: 100,
};

describe('dashboardCardSnapshot', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it('creates the four displayed values from a successful summary', () => {
    expect(createDashboardCardSnapshot(createSummary(), 123)).toEqual({
      totalDevices: 20,
      onlineDevices: 15,
      activeAlarms: 4,
      activeUE: 8,
      updatedAt: 123,
    });
  });

  it('stores active UE as zero when a successful summary has no UE_ACTIVE', () => {
    const snapshot = createDashboardCardSnapshot(
      createSummary({ kpiSummary: {} }),
      123,
    );

    expect(snapshot.activeUE).toBe(0);
  });

  it('shows a previous snapshot without skeletons while a new request is loading', () => {
    expect(resolveDashboardCardDisplay(undefined, previousSnapshot, true)).toEqual({
      totalDevices: 10,
      onlineDevices: 6,
      activeAlarms: 2,
      activeUE: 3,
      loading: false,
    });
  });

  it('shows skeletons when the first request is loading without a snapshot', () => {
    expect(resolveDashboardCardDisplay(undefined, undefined, true)).toEqual({
      totalDevices: 0,
      onlineDevices: 0,
      activeAlarms: 0,
      activeUE: 0,
      loading: true,
    });
  });

  it('prefers a new summary over the previous snapshot', () => {
    expect(
      resolveDashboardCardDisplay(createSummary(), previousSnapshot, false),
    ).toEqual({
      totalDevices: 20,
      onlineDevices: 15,
      activeAlarms: 4,
      activeUE: 8,
      loading: false,
    });
  });

  it('round-trips a valid snapshot through storage', () => {
    writeDashboardCardSnapshot(
      window.localStorage,
      'http://server-a',
      'user-a',
      previousSnapshot,
    );

    expect(
      readDashboardCardSnapshot(window.localStorage, 'http://server-a', 'user-a'),
    ).toEqual(previousSnapshot);
  });

  it('returns undefined for corrupted storage JSON', () => {
    const scope = 'http://server-a';
    window.localStorage.setItem(
      dashboardCardSnapshotStorageKey(scope, 'user-a'),
      '{broken',
    );

    expect(
      readDashboardCardSnapshot(window.localStorage, scope, 'user-a')
    ).toBeUndefined();
  });

  it('returns undefined for an invalid snapshot shape', () => {
    const scope = 'http://server-a';
    window.localStorage.setItem(
      dashboardCardSnapshotStorageKey(scope, 'user-a'),
      JSON.stringify({ ...previousSnapshot, activeUE: '3' }),
    );

    expect(
      readDashboardCardSnapshot(window.localStorage, scope, 'user-a')
    ).toBeUndefined();
  });

  it('uses API scope in the storage key', () => {
    expect(dashboardCardSnapshotStorageKey('http://server-a', 'user-a')).not.toBe(
      dashboardCardSnapshotStorageKey('http://server-b', 'user-a'),
    );
  });

  it('uses user scope in the storage key', () => {
    expect(dashboardCardSnapshotStorageKey('http://server-a', 'user-a')).not.toBe(
      dashboardCardSnapshotStorageKey('http://server-a', 'user-b'),
    );
  });

  it('does not expose one user snapshot to another user on the same API', () => {
    writeDashboardCardSnapshot(
      window.localStorage,
      'http://server-a',
      'user-a',
      previousSnapshot,
    );

    expect(
      readDashboardCardSnapshot(window.localStorage, 'http://server-a', 'user-a'),
    ).toEqual(previousSnapshot);
    expect(
      readDashboardCardSnapshot(window.localStorage, 'http://server-a', 'user-b'),
    ).toBeUndefined();
  });

  it('resolves a relative API base against the development proxy target', () => {
    expect(
      resolveDashboardCardApiScope(
        '/api/v1',
        'http://server-a:8081/',
        'http://localhost:3000',
      ),
    ).toBe('http://server-a:8081/api/v1');
  });

  it('uses an absolute API base instead of the development proxy target', () => {
    expect(
      resolveDashboardCardApiScope(
        'https://api.example.com/api/v1/',
        'http://ignored-proxy:8081',
        'https://ui.example.com',
      ),
    ).toBe('https://api.example.com/api/v1');
  });

  it('resolves a relative production API base against the page origin', () => {
    expect(
      resolveDashboardCardApiScope(
        '/api/v1',
        undefined,
        'https://ui.example.com',
      ),
    ).toBe('https://ui.example.com/api/v1');
  });
});
