export function normalizePmMetricValue(value: number | null | undefined): number | null {
  return typeof value === 'number' && Number.isFinite(value) ? value : null;
}

export function isFinitePmMetricValue(value: number | null | undefined): value is number {
  return normalizePmMetricValue(value) !== null;
}

export const PM_METRIC_EMPTY_DISPLAY = '-';

export function formatPmMetricDisplayValue(value: unknown): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) return PM_METRIC_EMPTY_DISPLAY;
  return value.toFixed(2);
}

export function formatPmMetricDisplayValueWithUnit(
  value: unknown,
  unit: string | null | undefined,
  options: { appendPercentUnit?: boolean } = {},
): string {
  const displayValue = formatPmMetricDisplayValue(value);
  if (displayValue === PM_METRIC_EMPTY_DISPLAY) return displayValue;

  const normalizedUnit = unit?.trim();
  if (!normalizedUnit) return displayValue;
  if (options.appendPercentUnit === false && normalizedUnit === '%') return displayValue;
  return `${displayValue} ${normalizedUnit}`;
}
