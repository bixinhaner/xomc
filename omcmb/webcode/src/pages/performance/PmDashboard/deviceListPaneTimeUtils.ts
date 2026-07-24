import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import type { Granularity } from '@core/types/pmDashboard';
import { formatSystemTime, nowInSystemTimezone, toSystemTimezoneRFC3339 } from '@core/utils/systemTime';
import { previousWindow } from './dashboardFilterUtils';

function safeNowInSystemTimezone(systemTimezone?: string | null): Dayjs {
  try {
    return nowInSystemTimezone(systemTimezone);
  } catch {
    return nowInSystemTimezone('UTC');
  }
}

export function defaultRangeForGranularity(
  g: Granularity,
  systemTimezone?: string | null,
  now?: Dayjs,
): [Dayjs, Dayjs] {
  const end = (now ?? safeNowInSystemTimezone(systemTimezone)).millisecond(0);
  switch (g) {
    case '15min':
      return [end.subtract(3, 'hour'), end];
    case 'hourly':
      return [end.subtract(24, 'hour'), end];
    case 'daily':
      return [end.subtract(7, 'day'), end];
    case 'weekly':
      return [end.subtract(30, 'day'), end];
    case 'monthly':
      return [end.subtract(6, 'month'), end];
    default:
      return [end.subtract(24, 'hour'), end];
  }
}

export function toDeviceViewRequestRFC3339(
  value: Dayjs,
  systemTimezone?: string | null,
): string {
  const normalized = value.millisecond(0);
  try {
    const serialized = toSystemTimezoneRFC3339(normalized, systemTimezone);
    if (serialized) return serialized;
  } catch {
    // Fall through to UTC wall-clock fallback for invalid IANA timezone values.
  }
  return `${normalized.format('YYYY-MM-DDTHH:mm:ss')}Z`;
}

export function buildDeviceViewRequestTimeWindow(
  range: [Dayjs, Dayjs],
  systemTimezone?: string | null,
): {
  startTime: string;
  endTime: string;
  prevStartTime: string;
  prevEndTime: string;
} {
  const [start, end] = range;
  const [prevStart, prevEnd] = previousWindow(range);
  return {
    startTime: toDeviceViewRequestRFC3339(start, systemTimezone),
    endTime: toDeviceViewRequestRFC3339(end, systemTimezone),
    prevStartTime: toDeviceViewRequestRFC3339(prevStart, systemTimezone),
    prevEndTime: toDeviceViewRequestRFC3339(prevEnd, systemTimezone),
  };
}

function parseSystemWallClockTime(value: string): Dayjs | null {
  const wallClock = formatSystemTime(value, { placeholder: '' });
  if (!wallClock) return null;
  const parsed = dayjs(wallClock);
  return parsed.isValid() ? parsed : null;
}

export function actualRangeFromMeta(meta: { actualStartTime?: string | null; actualEndTime?: string | null } | undefined): [Dayjs, Dayjs] | null {
  if (!meta?.actualStartTime || !meta.actualEndTime) return null;
  const start = parseSystemWallClockTime(meta.actualStartTime);
  const end = parseSystemWallClockTime(meta.actualEndTime);
  if (!start?.isValid() || !end?.isValid()) return null;
  return [start, end];
}
