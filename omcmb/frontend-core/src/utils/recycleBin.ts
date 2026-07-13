export function formatRecycleOperator(deletedBy?: string | null): string {
  const value = deletedBy?.trim();
  if (!value) return '—';
  if (value === 'system' || value.startsWith('system:')) return 'system';
  return value;
}
