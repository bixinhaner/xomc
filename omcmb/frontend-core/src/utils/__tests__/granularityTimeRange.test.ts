import { describe, expect, it } from 'vitest';
import { buildAlignedPresetRange, getDefaultTimeRangeForGranularity } from '../granularityTimeRange';

const NOW = new Date('2026-07-13T02:10:00.000Z'); // Asia/Shanghai 2026-07-13 10:10:00

describe('getDefaultTimeRangeForGranularity', () => {
  it('keeps weekly default at last_30d', () => {
    expect(getDefaultTimeRangeForGranularity('weekly')).toBe('last_30d');
  });
});

describe('buildAlignedPresetRange', () => {
  it('aligns 15min last_1h to four complete buckets', () => {
    expect(
      buildAlignedPresetRange({
        granularity: '15min',
        preset: 'last_1h',
        systemTimezone: 'Asia/Shanghai',
        now: NOW,
      })
    ).toEqual({
      start: '2026-07-13T09:00:00+08:00',
      end: '2026-07-13T10:00:00+08:00',
    });
  });

  it('aligns 15min last_3h to twelve complete buckets', () => {
    expect(
      buildAlignedPresetRange({
        granularity: '15min',
        preset: 'last_3h',
        systemTimezone: 'Asia/Shanghai',
        now: NOW,
      })
    ).toEqual({
      start: '2026-07-13T07:00:00+08:00',
      end: '2026-07-13T10:00:00+08:00',
    });
  });

  it('aligns hourly last_1h to the previous complete hour', () => {
    expect(
      buildAlignedPresetRange({
        granularity: 'hourly',
        preset: 'last_1h',
        systemTimezone: 'Asia/Shanghai',
        now: NOW,
      })
    ).toEqual({
      start: '2026-07-13T09:00:00+08:00',
      end: '2026-07-13T10:00:00+08:00',
    });
  });

  it('aligns daily last_7d to system-timezone natural days', () => {
    expect(
      buildAlignedPresetRange({
        granularity: 'daily',
        preset: 'last_7d',
        systemTimezone: 'Asia/Shanghai',
        now: NOW,
      })
    ).toEqual({
      start: '2026-07-06T00:00:00+08:00',
      end: '2026-07-13T00:00:00+08:00',
    });
  });

  it('aligns weekly ranges to Monday 00:00 in the system timezone', () => {
    expect(
      buildAlignedPresetRange({
        granularity: 'weekly',
        preset: 'last_30d',
        systemTimezone: 'Asia/Shanghai',
        now: NOW,
      })
    ).toEqual({
      start: '2026-06-08T00:00:00+08:00',
      end: '2026-07-13T00:00:00+08:00',
    });
  });

  it('aligns monthly last_6m to natural month boundaries', () => {
    expect(
      buildAlignedPresetRange({
        granularity: 'monthly',
        preset: 'last_6m',
        systemTimezone: 'Asia/Shanghai',
        now: NOW,
      })
    ).toEqual({
      start: '2026-01-01T00:00:00+08:00',
      end: '2026-07-01T00:00:00+08:00',
    });
  });

  it('returns null for custom ranges so callers can respect user-picked absolute times', () => {
    expect(
      buildAlignedPresetRange({
        granularity: 'hourly',
        preset: 'custom',
        systemTimezone: 'Asia/Shanghai',
        now: NOW,
      })
    ).toBeNull();
  });
});
