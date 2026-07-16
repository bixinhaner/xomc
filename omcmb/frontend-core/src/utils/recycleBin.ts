export function formatRecycleOperator(deletedBy?: string | null): string {
  const value = deletedBy?.trim();
  if (!value) return '—';
  if (value === 'system' || value.startsWith('system:')) return 'system';
  return value;
}

export type RecycleType = 'manual' | 'auto';

export function resolveRecycleType(
  recycleType?: string | null,
  deletedBy?: string | null,
): RecycleType {
  if (recycleType === 'auto') return 'auto';
  if (recycleType === 'manual') return 'manual';

  const operator = deletedBy?.trim().toLowerCase() ?? '';
  return operator === 'system' || operator.startsWith('system:') ? 'auto' : 'manual';
}

export function normalizeRecycleOfflineDays(offlineDays?: number | null): number {
  if (offlineDays === null || offlineDays === undefined || !Number.isFinite(offlineDays)) {
    return 0;
  }
  return Math.max(0, Math.floor(offlineDays));
}
