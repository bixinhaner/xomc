import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';

import { toSystemTimezoneRFC3339 } from '@core/utils/systemTime';

type TransferWallClock = string | number | Date | Dayjs | null | undefined;

function asTransferWallClock(value: unknown): TransferWallClock {
  return value as TransferWallClock;
}

function fallbackToISOString(value: unknown): string | undefined {
  if (value === null || value === undefined || value === '') return undefined;
  if (dayjs.isDayjs(value)) return value.toISOString();
  const parsed = dayjs(asTransferWallClock(value));
  return parsed.isValid() ? parsed.toISOString() : undefined;
}

export function toTransferSystemTimeRFC3339(
  value: unknown,
  systemTimezone?: string | null,
): string | undefined {
  return toSystemTimezoneRFC3339(asTransferWallClock(value), systemTimezone) ?? fallbackToISOString(value);
}

export function toTransferSystemTimeRange(
  range: [Dayjs, Dayjs] | null | undefined,
  systemTimezone?: string | null,
): [string, string] | undefined {
  if (!range) return undefined;
  const start = toTransferSystemTimeRFC3339(range[0], systemTimezone);
  const end = toTransferSystemTimeRFC3339(range[1], systemTimezone);
  return start && end ? [start, end] : undefined;
}

export function isTransferSystemTimeAfter(
  value: unknown,
  baseline: unknown,
  systemTimezone?: string | null,
): boolean {
  const valueIso = toTransferSystemTimeRFC3339(value, systemTimezone);
  const baselineIso = toTransferSystemTimeRFC3339(baseline, systemTimezone);
  return Boolean(valueIso && baselineIso && dayjs(valueIso).isAfter(dayjs(baselineIso)));
}

export function isTransferSystemDateBefore(
  value: unknown,
  baseline: unknown,
  systemTimezone?: string | null,
): boolean {
  const valueIso = toTransferSystemTimeRFC3339(value, systemTimezone);
  const baselineIso = toTransferSystemTimeRFC3339(baseline, systemTimezone);
  return Boolean(valueIso && baselineIso && valueIso.slice(0, 10) < baselineIso.slice(0, 10));
}
