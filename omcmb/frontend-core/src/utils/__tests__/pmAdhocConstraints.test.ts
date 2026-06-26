import { describe, it, expect } from 'vitest';
import {
  isGranularityDimensionSupported,
  UNSUPPORTED_GRANULARITIES,
} from '../pmAdhocConstraints';

describe('isGranularityDimensionSupported (#669)', () => {
  it('rejects 15min under any dimension (15min granularity is no longer supported for ad-hoc tasks)', () => {
    expect(isGranularityDimensionSupported('15min', 'device')).toBe(false);
    expect(isGranularityDimensionSupported('15min', 'device_group')).toBe(false);
    expect(isGranularityDimensionSupported('15min', 'aggregate_group')).toBe(false);
    expect(isGranularityDimensionSupported('15min', 'product')).toBe(false);
    expect(isGranularityDimensionSupported('15min', 'band')).toBe(false);
    expect(isGranularityDimensionSupported('15min', 'network')).toBe(false);
  });

  it('allows hourly/daily/weekly/monthly under any dimension', () => {
    const grans = ['hourly', 'daily', 'weekly', 'monthly'] as const;
    const dims = ['device', 'device_group', 'aggregate_group', 'product', 'band', 'network'] as const;
    for (const g of grans) {
      for (const d of dims) {
        expect(isGranularityDimensionSupported(g, d)).toBe(true);
      }
    }
  });

  it('treats missing granularity as supported (no premature block before user picks one)', () => {
    expect(isGranularityDimensionSupported(undefined, 'device_group')).toBe(true);
  });

  it('does not consult dimension (dim arg is reserved for future per-dim constraints)', () => {
    // 15min 命中粒度黑名单，dim 是 undefined / device / device_group 都拒。
    expect(isGranularityDimensionSupported('15min', undefined)).toBe(false);
    expect(isGranularityDimensionSupported('15min', 'device')).toBe(false);
  });

  it('keeps the unsupported-granularity list to exactly 15min', () => {
    expect(UNSUPPORTED_GRANULARITIES).toEqual(['15min']);
  });
});
