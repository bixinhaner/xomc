export type RFStatusKind = 'on' | 'off' | 'error';

const ON_VALUES = new Set(['1', '3', 'true', 'on', 'enabled']);
const OFF_VALUES = new Set(['0', '2', 'false', 'off', 'disabled']);
const ERROR_VALUES = new Set(['error', 'abnormal', 'failed', 'fault']);

export function rfStatusOf(value: string | undefined | null): RFStatusKind | null {
  const normalized = String(value ?? '').trim().toLowerCase();
  if (!normalized || normalized === 'unknown' || normalized === '--') {
    return null;
  }
  if (ON_VALUES.has(normalized)) {
    return 'on';
  }
  if (OFF_VALUES.has(normalized)) {
    return 'off';
  }
  if (ERROR_VALUES.has(normalized)) {
    return 'error';
  }
  return null;
}

export function displayRFStatusOf(
  value: string | undefined | null,
  _isOnline: boolean | undefined | null,
): RFStatusKind | null {
  return rfStatusOf(value);
}

export function rfStatusLabelOf(
  value: string | undefined | null,
  labels: { on: string; off: string; error?: string },
): string | null {
  const kind = rfStatusOf(value);
  if (!kind) {
    return null;
  }
  if (kind === 'error') {
    return labels.error ?? labels.off;
  }
  return labels[kind];
}

export function displayRFStatusLabelOf(
  value: string | undefined | null,
  isOnline: boolean | undefined | null,
  labels: { on: string; off: string; error?: string },
): string | null {
  const kind = displayRFStatusOf(value, isOnline);
  if (!kind) return null;
  if (kind === 'error') return labels.error ?? labels.off;
  return labels[kind];
}
