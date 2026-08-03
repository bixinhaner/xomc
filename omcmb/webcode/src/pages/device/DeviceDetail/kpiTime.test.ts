import { describe, expect, it } from 'vitest';
import {
  buildKpiQueryWindow,
  buildKpiTooltipRangeLabels,
  formatKpiBucketRangeLabel,
  previousKpiQueryWindow,
} from './kpiTime';

const NOW = new Date('2026-07-13T02:10:37.123Z'); // Asia/Shanghai 2026-07-13 10:10:37

describe('device detail KPI time window', () => {
  it('hourly day mode ends at the current system-timezone hour boundary', () => {
    expect(buildKpiQueryWindow('day', 'Asia/Shanghai', NOW)).toEqual({
      granularity: 'hourly',
      startTime: '2026-07-12T10:00:00+08:00',
      endTime: '2026-07-13T10:00:00+08:00',
    });
  });

  it('daily week mode ends at the current system-timezone natural day boundary', () => {
    expect(buildKpiQueryWindow('week', 'Asia/Shanghai', NOW)).toEqual({
      granularity: 'daily',
      startTime: '2026-07-06T00:00:00+08:00',
      endTime: '2026-07-13T00:00:00+08:00',
    });
  });

  it('previous window keeps the same system-timezone RFC3339 offset', () => {
    expect(
      previousKpiQueryWindow(
        '2026-07-12T10:00:00+08:00',
        '2026-07-13T10:00:00+08:00',
        'Asia/Shanghai',
        'hourly',
      ),
    ).toEqual({
      startTime: '2026-07-11T10:00:00+08:00',
      endTime: '2026-07-12T10:00:00+08:00',
    });
  });

  it('daily previous window keeps natural day boundaries', () => {
    expect(
      previousKpiQueryWindow(
        '2026-07-06T00:00:00+08:00',
        '2026-07-13T00:00:00+08:00',
        'Asia/Shanghai',
        'daily',
      ),
    ).toEqual({
      startTime: '2026-06-29T00:00:00+08:00',
      endTime: '2026-07-06T00:00:00+08:00',
    });
  });

  it('daily previous window keeps midnight across DST changes', () => {
    expect(
      previousKpiQueryWindow(
        '2026-03-02T00:00:00-05:00',
        '2026-03-09T00:00:00-04:00',
        'America/New_York',
        'daily',
      ),
    ).toEqual({
      startTime: '2026-02-23T00:00:00-05:00',
      endTime: '2026-03-02T00:00:00-05:00',
    });
  });

  it('formats tooltip xDataFull as the current bucket start and end', () => {
    expect(
      buildKpiTooltipRangeLabels(
        ['2026-07-13T09:00:00+08:00'],
        ['2026-07-13T10:00:00+08:00'],
        'hourly',
        'Asia/Shanghai',
      ),
    ).toEqual(['07-13 09:00 ~ 07-13 10:00']);
  });

  it('infers a full bucket end when backend rows do not carry endTime', () => {
    expect(
      formatKpiBucketRangeLabel(
        '2026-07-13T09:00:00+08:00',
        '',
        'hourly',
        'Asia/Shanghai',
      ),
    ).toBe('07-13 09:00 ~ 07-13 10:00');
  });
});
