import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';

import {
  isTransferSystemDateBefore,
  isTransferSystemTimeAfter,
  toTransferSystemTimeRFC3339,
  toTransferSystemTimeRange,
} from '../transferTime';

describe('transfer system time helpers', () => {
  it('serializes a DatePicker wall-clock value with the configured system timezone', () => {
    const selected = dayjs('2026-07-13 10:00:00');

    const value = toTransferSystemTimeRFC3339(selected, 'Asia/Tokyo');

    expect(value).toContain('2026-07-13T10:00:00');
    expect(value).toMatch(/\+09:00$/);
  });

  it('serializes range filters without browser-local timezone conversion', () => {
    const range = toTransferSystemTimeRange([
      dayjs('2026-07-13 00:00:00'),
      dayjs('2026-07-13 23:59:59'),
    ], 'UTC');

    expect(range?.[0]).toContain('2026-07-13T00:00:00');
    expect(range?.[0]).toMatch(/(Z|\+00:00)$/);
    expect(range?.[1]).toContain('2026-07-13T23:59:59');
    expect(range?.[1]).toMatch(/(Z|\+00:00)$/);
  });

  it('compares DatePicker wall-clock values in the system timezone', () => {
    expect(isTransferSystemTimeAfter(
      dayjs('2026-07-13 10:00:00'),
      dayjs('2026-07-13 09:00:00'),
      'UTC',
    )).toBe(true);
    expect(isTransferSystemTimeAfter(
      dayjs('2026-07-13 10:00:00'),
      dayjs('2026-07-13 11:00:00'),
      'UTC',
    )).toBe(false);
  });

  it('compares DatePicker dates by system timezone day', () => {
    expect(isTransferSystemDateBefore(
      dayjs('2026-07-12 23:00:00'),
      dayjs('2026-07-13 00:30:00'),
      'UTC',
    )).toBe(true);
    expect(isTransferSystemDateBefore(
      dayjs('2026-07-13 00:00:00'),
      dayjs('2026-07-13 23:30:00'),
      'UTC',
    )).toBe(false);
  });
});
