import { describe, expect, it } from 'vitest';
import {
  formatPmMetricDisplayValue,
  formatPmMetricDisplayValueWithUnit,
} from '../pmMetricValue';

describe('PM metric display value formatting', () => {
  it('formats finite numbers with exactly two decimals', () => {
    expect(formatPmMetricDisplayValue(12.345678901234)).toBe('12.35');
    expect(formatPmMetricDisplayValue(12)).toBe('12.00');
    expect(formatPmMetricDisplayValue(0)).toBe('0.00');
  });

  it('renders empty and non-number values as "-"', () => {
    expect(formatPmMetricDisplayValue(null)).toBe('-');
    expect(formatPmMetricDisplayValue(undefined)).toBe('-');
    expect(formatPmMetricDisplayValue(Number.NaN)).toBe('-');
    expect(formatPmMetricDisplayValue(Number.POSITIVE_INFINITY)).toBe('-');
    expect(formatPmMetricDisplayValue('12.34')).toBe('-');
  });

  it('appends normalized units only for valid numbers', () => {
    expect(formatPmMetricDisplayValueWithUnit(12.345678901234, ' Mbps ')).toBe('12.35 Mbps');
    expect(formatPmMetricDisplayValueWithUnit(12, '%')).toBe('12.00 %');
    expect(formatPmMetricDisplayValueWithUnit(null, 'Mbps')).toBe('-');
    expect(formatPmMetricDisplayValueWithUnit(12, '')).toBe('12.00');
  });

  it('can preserve chart components that do not append percent units', () => {
    expect(formatPmMetricDisplayValueWithUnit(12, '%', { appendPercentUnit: false })).toBe('12.00');
    expect(formatPmMetricDisplayValueWithUnit(12, 'Mbps', { appendPercentUnit: false })).toBe('12.00 Mbps');
  });
});
