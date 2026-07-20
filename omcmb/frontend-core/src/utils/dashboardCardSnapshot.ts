import type { DashboardSummary } from '../types/dashboard';

const STORAGE_KEY_PREFIX = 'xomc:dashboard-card-snapshot:v1';

type SnapshotStorage = Pick<Storage, 'getItem' | 'setItem'>;

export interface DashboardCardSnapshot {
  totalDevices: number;
  onlineDevices: number;
  activeAlarms: number;
  activeUE: number;
  updatedAt: number;
}

export interface DashboardCardDisplay {
  totalDevices: number;
  onlineDevices: number;
  activeAlarms: number;
  activeUE: number;
  loading: boolean;
}

function toFiniteNumber(value: number | undefined): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : 0;
}

function summaryValues(summary: DashboardSummary): Omit<DashboardCardSnapshot, 'updatedAt'> {
  return {
    totalDevices: toFiniteNumber(summary.deviceCounts.total),
    onlineDevices: toFiniteNumber(summary.deviceCounts.online),
    activeAlarms: toFiniteNumber(summary.alarmCounts.total),
    activeUE: Math.floor(toFiniteNumber(summary.kpiSummary['UE_ACTIVE'])),
  };
}

function isDashboardCardSnapshot(value: unknown): value is DashboardCardSnapshot {
  if (!value || typeof value !== 'object') return false;

  const candidate = value as Record<string, unknown>;
  return [
    'totalDevices',
    'onlineDevices',
    'activeAlarms',
    'activeUE',
    'updatedAt',
  ].every((key) => (
    typeof candidate[key] === 'number' && Number.isFinite(candidate[key])
  ));
}

export function createDashboardCardSnapshot(
  summary: DashboardSummary,
  updatedAt: number,
): DashboardCardSnapshot {
  return {
    ...summaryValues(summary),
    updatedAt,
  };
}

export function resolveDashboardCardApiScope(
  apiBaseURL: string | undefined,
  proxyTarget: string | undefined,
  locationOrigin: string,
): string {
  const apiBase = apiBaseURL?.trim() || '/api/v1';
  const requestOrigin = proxyTarget?.trim() || locationOrigin;
  const resolved = new URL(apiBase, requestOrigin);
  resolved.hash = '';
  resolved.search = '';
  return resolved.toString().replace(/\/$/, '');
}

export function resolveDashboardCardDisplay(
  summary: DashboardSummary | undefined,
  snapshot: DashboardCardSnapshot | undefined,
  isLoading: boolean,
): DashboardCardDisplay {
  const values = summary
    ? summaryValues(summary)
    : snapshot ?? {
      totalDevices: 0,
      onlineDevices: 0,
      activeAlarms: 0,
      activeUE: 0,
    };

  return {
    totalDevices: values.totalDevices,
    onlineDevices: values.onlineDevices,
    activeAlarms: values.activeAlarms,
    activeUE: values.activeUE,
    loading: isLoading && !summary && !snapshot,
  };
}

export function dashboardCardSnapshotStorageKey(
  apiScope: string,
  userScope: string,
): string {
  return [
    STORAGE_KEY_PREFIX,
    encodeURIComponent(apiScope),
    encodeURIComponent(userScope),
  ].join(':');
}

export function readDashboardCardSnapshot(
  storage: SnapshotStorage,
  apiScope: string,
  userScope: string,
): DashboardCardSnapshot | undefined {
  try {
    const raw = storage.getItem(
      dashboardCardSnapshotStorageKey(apiScope, userScope)
    );
    if (!raw) return undefined;

    const parsed: unknown = JSON.parse(raw);
    return isDashboardCardSnapshot(parsed) ? parsed : undefined;
  } catch {
    return undefined;
  }
}

export function writeDashboardCardSnapshot(
  storage: SnapshotStorage,
  apiScope: string,
  userScope: string,
  snapshot: DashboardCardSnapshot,
): void {
  try {
    storage.setItem(
      dashboardCardSnapshotStorageKey(apiScope, userScope),
      JSON.stringify(snapshot),
    );
  } catch {
    // localStorage 可能因隐私模式或容量限制不可写；不影响 Dashboard 正常请求。
  }
}
