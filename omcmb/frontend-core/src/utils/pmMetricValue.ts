export function normalizePmMetricValue(value: number | null | undefined): number | null {
  return typeof value === 'number' && Number.isFinite(value) ? value : null;
}

export function isFinitePmMetricValue(value: number | null | undefined): value is number {
  return normalizePmMetricValue(value) !== null;
}
