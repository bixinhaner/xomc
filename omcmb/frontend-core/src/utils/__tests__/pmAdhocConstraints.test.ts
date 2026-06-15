import { describe, it, expect } from 'vitest';
import {
  isGranularityDimensionSupported,
  UNSUPPORTED_GRANULARITY_DIMENSIONS,
} from '../pmAdhocConstraints';

describe('isGranularityDimensionSupported (#363)', () => {
  it('rejects 15min × device_group (no 15min group aggregation source)', () => {
    expect(isGranularityDimensionSupported('15min', 'device_group')).toBe(false);
  });

  it('allows hourly/daily/weekly/monthly × device_group', () => {
    expect(isGranularityDimensionSupported('hourly', 'device_group')).toBe(true);
    expect(isGranularityDimensionSupported('daily', 'device_group')).toBe(true);
    expect(isGranularityDimensionSupported('weekly', 'device_group')).toBe(true);
    expect(isGranularityDimensionSupported('monthly', 'device_group')).toBe(true);
  });

  it('allows 15min under device / aggregate_group / product / band / network', () => {
    expect(isGranularityDimensionSupported('15min', 'device')).toBe(true);
    expect(isGranularityDimensionSupported('15min', 'aggregate_group')).toBe(true);
    expect(isGranularityDimensionSupported('15min', 'product')).toBe(true);
    expect(isGranularityDimensionSupported('15min', 'band')).toBe(true);
    expect(isGranularityDimensionSupported('15min', 'network')).toBe(true);
  });

  it('treats missing granularity/dimension as supported (no premature block)', () => {
    expect(isGranularityDimensionSupported(undefined, 'device_group')).toBe(true);
    expect(isGranularityDimensionSupported('15min', undefined)).toBe(true);
  });

  it('keeps the unsupported table to exactly the documented single combination', () => {
    expect(UNSUPPORTED_GRANULARITY_DIMENSIONS).toHaveLength(1);
    expect(UNSUPPORTED_GRANULARITY_DIMENSIONS[0]).toEqual({
      granularity: '15min',
      dimension: 'device_group',
    });
  });
});
