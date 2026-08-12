import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import { serializeTaskTimeRange } from './taskTimeFilter';

describe('serializeTaskTimeRange', () => {
  it('interprets DatePicker values as system-timezone wall clocks', () => {
    const range: [dayjs.Dayjs, dayjs.Dayjs] = [
      dayjs('2026-08-12 08:21:00'),
      dayjs('2026-08-13 00:00:00'),
    ];

    expect(serializeTaskTimeRange(range, 'UTC')).toEqual({
      startedAfter: '2026-08-12T08:21:00Z',
      startedBefore: '2026-08-13T00:00:00Z',
    });
    expect(serializeTaskTimeRange(range, 'Asia/Shanghai')).toEqual({
      startedAfter: '2026-08-12T08:21:00+08:00',
      startedBefore: '2026-08-13T00:00:00+08:00',
    });
  });

  it('omits both boundaries when the range is empty', () => {
    expect(serializeTaskTimeRange(null, 'UTC')).toEqual({
      startedAfter: undefined,
      startedBefore: undefined,
    });
  });
});
