export type DeviceSyncStatusKind = 'synchronized' | 'gps' | 'beidou' | 'ntp' | 'rem' | 'error';

type DeviceSyncStatusLabels = Record<DeviceSyncStatusKind, string>;

export const DEVICE_SYNC_STATUS_LABELS_ZH: DeviceSyncStatusLabels = {
  synchronized: '已同步',
  gps: 'GPS 已同步',
  beidou: '北斗已同步',
  ntp: 'NTP/1588 已同步',
  rem: 'REM 已同步',
  error: '未同步',
};

export const DEVICE_SYNC_STATUS_LABELS_EN: DeviceSyncStatusLabels = {
  synchronized: 'Synchronized',
  gps: 'GPS synchronized',
  beidou: 'Beidou synchronized',
  ntp: 'NTP/1588 synchronized',
  rem: 'REM synchronized',
  error: 'Not synchronized',
};

const DEVICE_SYNC_STATUS_ALIASES: Record<string, DeviceSyncStatusKind> = {
  synchronized: 'synchronized',
  synced: 'synchronized',
  'gps synchronized': 'gps',
  gps: 'gps',
  'beidou synchronized': 'beidou',
  beidou: 'beidou',
  '1588 synchronized': 'ntp',
  'ntp synchronized': 'ntp',
  '1588': 'ntp',
  ntp: 'ntp',
  'rem synchronized': 'rem',
  rem: 'rem',
  error: 'error',
  'not synchronized': 'error',
};

function normalizeLookupKey(value: string): string {
  return value.trim().toLowerCase().replace(/[_\s]+/g, ' ');
}

export function getDeviceSyncStatusKind(value: string | null | undefined): DeviceSyncStatusKind | null {
  if (!value) return null;
  return DEVICE_SYNC_STATUS_ALIASES[normalizeLookupKey(value)] ?? null;
}

export function normalizeDeviceSyncStatus(value: string | null | undefined): string {
  if (!value) return '';
  const trimmed = value.trim();
  if (!trimmed) return '';
  return getDeviceSyncStatusKind(trimmed) ?? trimmed;
}

export function formatDeviceSyncStatus(
  value: string | null | undefined,
  labels: DeviceSyncStatusLabels,
): string {
  const normalized = normalizeDeviceSyncStatus(value);
  if (!normalized) return '';
  const kind = getDeviceSyncStatusKind(normalized);
  return kind ? labels[kind] : normalized;
}