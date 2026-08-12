import type { Dayjs } from 'dayjs';
import { toSystemTimezoneRFC3339 } from '@core/utils/systemTime';

export function serializeTaskTimeRange(
  range: [Dayjs, Dayjs] | null,
  systemTimezone?: string | null,
): { startedAfter: string | undefined; startedBefore: string | undefined } {
  return {
    startedAfter: toSystemTimezoneRFC3339(range?.[0], systemTimezone) ?? undefined,
    startedBefore: toSystemTimezoneRFC3339(range?.[1], systemTimezone) ?? undefined,
  };
}
