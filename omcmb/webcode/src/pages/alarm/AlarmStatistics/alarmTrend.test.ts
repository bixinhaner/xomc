import { describe, expect, it } from 'vitest';
import { selectAlarmTrendBuckets } from './alarmTrend';

describe('selectAlarmTrendBuckets', () => {
  it('uses backend dates as authoritative buckets across timezone day boundaries', () => {
    const buckets = selectAlarmTrendBuckets([
      { date: '2026-07-27', critical: 1, major: 0, minor: 0, warning: 0 },
      { date: '2026-07-28', critical: 5, major: 1, minor: 2, warning: 0 },
    ], 2);

    expect(buckets.map((bucket) => bucket.date)).toEqual(['2026-07-27', '2026-07-28']);
    expect(buckets.map((bucket) => bucket.label)).toEqual(['07/27', '07/28']);
    expect(buckets[1].critical).toBe(5);
  });

  it('sorts and limits buckets without synthesizing browser-local dates', () => {
    const buckets = selectAlarmTrendBuckets([
      { date: '2026-07-28', critical: 2, major: 0, minor: 0, warning: 0 },
      { date: '2026-07-26', critical: 1, major: 0, minor: 0, warning: 0 },
      { date: '2026-07-27', critical: 0, major: 1, minor: 0, warning: 0 },
    ], 2);

    expect(buckets.map((bucket) => bucket.date)).toEqual(['2026-07-27', '2026-07-28']);
  });
});
