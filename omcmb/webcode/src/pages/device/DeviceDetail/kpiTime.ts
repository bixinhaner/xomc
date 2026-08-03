import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import utc from 'dayjs/plugin/utc';
import timezone from 'dayjs/plugin/timezone';
import { formatSystemTime, nowInSystemTimezone, toSystemTimezoneRFC3339 } from '@core/utils/systemTime';

dayjs.extend(utc);
dayjs.extend(timezone);

export type KpiTimeMode = 'day' | 'week';
export type KpiGranularity = 'hourly' | 'daily';

export interface KpiQueryWindow {
  granularity: KpiGranularity;
  startTime: string;
  endTime: string;
}

function safeSystemNow(systemTimezone?: string | null, now?: Date): Dayjs {
  const tz = systemTimezone?.trim() || 'UTC';
  if (now) {
    try {
      const zoned = dayjs(now).tz(tz);
      if (zoned.isValid()) return zoned;
    } catch {
      // Fall back below.
    }
    return dayjs(now).utc();
  }

  try {
    return nowInSystemTimezone(systemTimezone);
  } catch {
    return nowInSystemTimezone('UTC');
  }
}

function systemTimeFromMs(ms: number, systemTimezone?: string | null): Dayjs {
  const tz = systemTimezone?.trim() || 'UTC';
  try {
    const zoned = dayjs(ms).tz(tz);
    if (zoned.isValid()) return zoned;
  } catch {
    // Fall back below.
  }
  return dayjs(ms).utc();
}

function parseSystemTime(value: string, systemTimezone?: string | null): Dayjs | null {
  const ms = Date.parse(value);
  if (!Number.isFinite(ms)) return null;
  return systemTimeFromMs(ms, systemTimezone);
}

function serializeSystemTime(value: Dayjs, systemTimezone?: string | null): string {
  return toSystemTimezoneRFC3339(value.millisecond(0), systemTimezone) ?? value.toISOString();
}

export function buildKpiQueryWindow(
  mode: KpiTimeMode,
  systemTimezone?: string | null,
  now?: Date,
): KpiQueryWindow {
  const granularity: KpiGranularity = mode === 'day' ? 'hourly' : 'daily';
  const end = mode === 'day'
    ? safeSystemNow(systemTimezone, now).startOf('hour')
    : safeSystemNow(systemTimezone, now).startOf('day');
  const start = mode === 'day' ? end.subtract(24, 'hour') : end.subtract(7, 'day');

  return {
    granularity,
    startTime: serializeSystemTime(start, systemTimezone),
    endTime: serializeSystemTime(end, systemTimezone),
  };
}

export function previousKpiQueryWindow(
  startTime: string,
  endTime: string,
  systemTimezone?: string | null,
  granularity: KpiGranularity = 'hourly',
): { startTime: string; endTime: string } {
  const start = parseSystemTime(startTime, systemTimezone);
  const end = parseSystemTime(endTime, systemTimezone);
  if (!start || !end || !end.isAfter(start)) {
    return { startTime, endTime: startTime };
  }

  const unit = granularity === 'daily' ? 'day' : 'hour';
  const rawLength = end.diff(start, unit, true);
  const length = Math.max(1, Math.round(rawLength));
  const previousStart = start.subtract(length, unit);

  return {
    startTime: serializeSystemTime(previousStart, systemTimezone),
    endTime: serializeSystemTime(start, systemTimezone),
  };
}

export function formatKpiAxisLabel(
  iso: string,
  granularity: KpiGranularity,
  systemTimezone?: string | null,
): string {
  return formatSystemTime(iso, {
    format: granularity === 'hourly' ? 'MM-DD HH:mm' : 'MM-DD',
    placeholder: iso,
    systemTimezone,
  });
}

function inferredBucketEnd(
  startIso: string,
  granularity: KpiGranularity,
  systemTimezone?: string | null,
): string {
  const startMs = Date.parse(startIso);
  if (!Number.isFinite(startMs)) return '';
  const start = systemTimeFromMs(startMs, systemTimezone);
  const end = granularity === 'hourly' ? start.add(1, 'hour') : start.add(1, 'day');
  return serializeSystemTime(end, systemTimezone);
}

export function formatKpiBucketRangeLabel(
  startIso: string,
  endIso: string | null | undefined,
  granularity: KpiGranularity,
  systemTimezone?: string | null,
): string | undefined {
  if (!startIso) return undefined;
  const start = formatKpiAxisLabel(startIso, granularity, systemTimezone);
  const effectiveEnd = endIso && endIso !== startIso
    ? endIso
    : inferredBucketEnd(startIso, granularity, systemTimezone);
  const end = effectiveEnd
    ? formatKpiAxisLabel(effectiveEnd, granularity, systemTimezone)
    : '';
  return end && end !== start ? `${start} ~ ${end}` : start;
}

export function buildKpiTooltipRangeLabels(
  xData: string[],
  xEnds: string[],
  granularity: KpiGranularity,
  systemTimezone?: string | null,
): string[] {
  return xData.map((iso, i) =>
    formatKpiBucketRangeLabel(iso, xEnds[i], granularity, systemTimezone)
    ?? formatKpiAxisLabel(iso, granularity, systemTimezone),
  );
}
