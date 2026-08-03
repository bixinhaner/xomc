import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import type { JsonObject, PmPageStateSnapshot } from '@core/store/pmPageStateStore';
import type { Granularity } from '@core/types/pmDashboard';
import {
  buildAlignedPresetRange,
  getDefaultTimeRangeForGranularity,
} from '@core/utils/granularityTimeRange';
import { nowInSystemTimezone, toSystemTimezoneRFC3339 } from '@core/utils/systemTime';
import type { DashboardFilterValue } from './DashboardFilterBar';
import { ALL_HOURS, ALL_WEEKDAYS, previousWindow } from './dashboardFilterUtils';

export const PM_DASHBOARD_PAGE_KEY = '/performance';

export type DashboardRelativeRangeMode = { kind: 'relative'; durationMs: number };

export type DashboardRangeMode =
  | DashboardRelativeRangeMode
  | { kind: 'absolute' };

export interface TaskDashboardSubmittedQuery {
  granularity?: Granularity;
  startISO: string;
  endISO: string;
  productIds?: string[];
  objectLdns?: string[];
  weekdays: number[];
  hours: number[];
  compare: boolean;
  offsetMs: number;
  prevStartISO: string;
  prevEndISO: string;
  rangeStartMs: number;
  rangeEndMs: number;
}

export interface RestoredTaskDashboardState {
  taskId?: string;
  filter: DashboardFilterValue;
  dimSelected: string[];
  activeGran?: string;
  rangeMode: DashboardRangeMode;
  submitted: TaskDashboardSubmittedQuery | null;
  savedAt?: string;
}

export interface TaskDashboardSaveInput {
  taskId: string;
  filter: DashboardFilterValue;
  dimSelected: string[];
  activeGran?: string;
  effectiveGran?: string;
  rangeMode: DashboardRangeMode;
  submitted: TaskDashboardSubmittedQuery | null;
}

const DEFAULT_RELATIVE_DURATION_MS = 7 * 24 * 60 * 60 * 1000;
const DEFAULT_TASK_GRANULARITY: Granularity = 'daily';
const DEFAULT_RANGE_MODE: DashboardRangeMode = {
  kind: 'relative',
  durationMs: DEFAULT_RELATIVE_DURATION_MS,
};
export const RESTORED_QUERY_THROTTLE_MS = 2_000;

function asString(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value : undefined;
}

function asNumberArray(value: unknown, fallback: number[]): number[] {
  if (!Array.isArray(value)) return [...fallback];
  const numbers = value.filter((item): item is number => Number.isInteger(item));
  return numbers.length > 0 ? numbers : [...fallback];
}

function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value.filter((item): item is string => typeof item === 'string');
}

function asBoolean(value: unknown, fallback = false): boolean {
  return typeof value === 'boolean' ? value : fallback;
}

function asGranularity(value: unknown): Granularity | undefined {
  if (
    value === '15min' ||
    value === 'hourly' ||
    value === 'daily' ||
    value === 'weekly' ||
    value === 'monthly'
  ) {
    return value;
  }
  return undefined;
}

function parseRFC3339KeepingOffset(value: string): Dayjs {
  const offsetMatch = value.match(/(Z|([+-])(\d{2}):?(\d{2}))$/i);
  if (!offsetMatch) return dayjs(value);
  const wall = dayjs(value.replace(/(Z|[+-]\d{2}:?\d{2})$/i, ''));
  if (!wall.isValid()) return dayjs(value);
  if (offsetMatch[1].toUpperCase() === 'Z') return wall.utcOffset(0, true);
  const sign = offsetMatch[2] === '-' ? -1 : 1;
  const hours = Number(offsetMatch[3]);
  const minutes = Number(offsetMatch[4]);
  if (!Number.isFinite(hours) || !Number.isFinite(minutes)) return dayjs(value);
  return wall.utcOffset(sign * (hours * 60 + minutes), true);
}

function alignWeeklyStart(d: Dayjs): Dayjs {
  const weekday = d.day();
  const daysSinceMonday = (weekday + 6) % 7;
  return d.subtract(daysSinceMonday, 'day').startOf('day');
}

function alignBucketStart(d: Dayjs, granularity: Granularity): Dayjs {
  switch (granularity) {
    case '15min': {
      const minute = Math.floor(d.minute() / 15) * 15;
      return d.minute(minute).second(0).millisecond(0);
    }
    case 'hourly':
      return d.startOf('hour');
    case 'daily':
      return d.startOf('day');
    case 'weekly':
      return alignWeeklyStart(d);
    case 'monthly':
      return d.startOf('month');
    default:
      return d;
  }
}

function addBucket(d: Dayjs, granularity: Granularity): Dayjs {
  switch (granularity) {
    case '15min':
      return d.add(15, 'minute');
    case 'hourly':
      return d.add(1, 'hour');
    case 'daily':
      return d.add(1, 'day');
    case 'weekly':
      return d.add(1, 'week');
    case 'monthly':
      return d.add(1, 'month');
    default:
      return d;
  }
}

function alignRangeToFullBuckets(range: [Dayjs, Dayjs], granularity: Granularity): [Dayjs, Dayjs] {
  const [rawStart, rawEnd] = range;
  let start = alignBucketStart(rawStart, granularity);
  if (start.valueOf() < rawStart.valueOf()) {
    start = addBucket(start, granularity);
  }
  const end = alignBucketStart(rawEnd, granularity);
  if (end.valueOf() <= start.valueOf()) {
    return [start, start];
  }
  return [start, end];
}

export function buildDefaultTaskDashboardRange(
  systemTimezone?: string | null,
  granularity: Granularity = DEFAULT_TASK_GRANULARITY,
  now?: Date,
): [Dayjs, Dayjs] {
  const range = buildAlignedPresetRange({
    granularity,
    preset: getDefaultTimeRangeForGranularity(granularity),
    systemTimezone,
    now,
  });
  if (range) return [parseRFC3339KeepingOffset(range.start), parseRFC3339KeepingOffset(range.end)];

  const fallbackNow = now ? dayjs(now) : nowInSystemTimezone(systemTimezone);
  const end = fallbackNow.startOf('hour');
  return [end.subtract(DEFAULT_RELATIVE_DURATION_MS, 'millisecond'), end];
}

export function defaultTaskDashboardRangeMode(
  systemTimezone?: string | null,
  granularity: Granularity = DEFAULT_TASK_GRANULARITY,
  now?: Date,
): DashboardRelativeRangeMode {
  const [start, end] = buildDefaultTaskDashboardRange(systemTimezone, granularity, now);
  return {
    kind: 'relative',
    durationMs: Math.max(1, end.valueOf() - start.valueOf()),
  };
}

export function buildDefaultTaskDashboardFilter(
  systemTimezone?: string | null,
  granularity: Granularity = DEFAULT_TASK_GRANULARITY,
  now?: Date,
): DashboardFilterValue {
  return {
    range: buildDefaultTaskDashboardRange(systemTimezone, granularity, now),
    weekdays: [...ALL_WEEKDAYS],
    hours: [...ALL_HOURS],
    compare: false,
  };
}

export function buildTaskDashboardTaskSwitchReset(
  systemTimezone?: string | null,
  granularity: Granularity = DEFAULT_TASK_GRANULARITY,
): {
  filter: DashboardFilterValue;
  rangeMode: DashboardRangeMode;
  dimSelected: string[];
  activeGran?: string;
  submitted: null;
} {
  const rangeMode = defaultTaskDashboardRangeMode(systemTimezone, granularity);
  return {
    filter: buildDefaultTaskDashboardFilter(systemTimezone, granularity),
    rangeMode,
    dimSelected: [],
    activeGran: undefined,
    submitted: null,
  };
}

function normalizeRangeMode(value: unknown): DashboardRangeMode {
  if (!value || typeof value !== 'object') return { ...DEFAULT_RANGE_MODE };
  const obj = value as Record<string, unknown>;
  if (obj.kind === 'absolute') return { kind: 'absolute' };
  if (obj.kind === 'relative' && typeof obj.durationMs === 'number' && Number.isFinite(obj.durationMs) && obj.durationMs > 0) {
    return { kind: 'relative', durationMs: obj.durationMs };
  }
  return { ...DEFAULT_RANGE_MODE };
}

function restoreRange(
  filters: Record<string, unknown>,
  rangeMode: DashboardRangeMode,
  systemTimezone?: string | null,
  activeGranularity?: Granularity,
): [Dayjs, Dayjs] {
  if (rangeMode.kind === 'relative') {
    if (activeGranularity) return buildDefaultTaskDashboardRange(systemTimezone, activeGranularity);
    const now = nowInSystemTimezone(systemTimezone);
    const end = now.startOf('hour');
    return [end.subtract(rangeMode.durationMs, 'millisecond'), end];
  }

  const start = asString(filters.rangeStartISO);
  const end = asString(filters.rangeEndISO);
  const parsedStart = start ? parseRFC3339KeepingOffset(start) : null;
  const parsedEnd = end ? parseRFC3339KeepingOffset(end) : null;
  if (parsedStart?.isValid() && parsedEnd?.isValid()) return [parsedStart, parsedEnd];
  return buildDefaultTaskDashboardRange(systemTimezone, activeGranularity);
}

export function buildSubmittedTaskDashboardQuery(
  filter: DashboardFilterValue,
  params: {
    productIds?: string[];
    objectLdns?: string[];
    systemTimezone?: string | null;
    granularity?: string;
    rangeMode?: DashboardRangeMode;
  },
): TaskDashboardSubmittedQuery {
  const submittedGranularity = asGranularity(params.granularity);
  const baseRange =
    params.rangeMode?.kind === 'relative' && submittedGranularity
      ? buildDefaultTaskDashboardRange(params.systemTimezone, submittedGranularity)
      : filter.range;
  const submittedRange = submittedGranularity && params.rangeMode?.kind === 'absolute'
    ? alignRangeToFullBuckets(baseRange, submittedGranularity)
    : baseRange;
  const [s, e] = submittedRange;
  const sISO = toSystemTimezoneRFC3339(s, params.systemTimezone) ?? s.toISOString();
  const eISO = toSystemTimezoneRFC3339(e, params.systemTimezone) ?? e.toISOString();
  const [ps, pe] = previousWindow(submittedRange);
  const submitted: TaskDashboardSubmittedQuery = {
    ...(submittedGranularity ? { granularity: submittedGranularity } : {}),
    startISO: sISO,
    endISO: eISO,
    weekdays: [...filter.weekdays],
    hours: [...filter.hours],
    compare: filter.compare,
    offsetMs: e.valueOf() - s.valueOf(),
    prevStartISO: toSystemTimezoneRFC3339(ps, params.systemTimezone) ?? ps.toISOString(),
    prevEndISO: toSystemTimezoneRFC3339(pe, params.systemTimezone) ?? pe.toISOString(),
    rangeStartMs: s.valueOf(),
    rangeEndMs: e.valueOf(),
  };
  if (params.productIds && params.productIds.length > 0) {
    submitted.productIds = [...params.productIds];
  }
  if (params.objectLdns && params.objectLdns.length > 0) {
    submitted.objectLdns = [...params.objectLdns];
  }
  return submitted;
}

export function taskDashboardQuerySignature(
  taskId: string,
  submitted: TaskDashboardSubmittedQuery | null,
): string | null {
  if (!submitted) return null;
  const signature: Record<string, unknown> = {
    taskId,
    startISO: submitted.startISO,
    endISO: submitted.endISO,
    weekdays: submitted.weekdays,
    hours: submitted.hours,
    compare: submitted.compare,
    granularity: submitted.granularity,
    prevStartISO: submitted.prevStartISO,
    prevEndISO: submitted.prevEndISO,
  };
  if (submitted.productIds && submitted.productIds.length > 0) {
    signature.productIds = submitted.productIds;
  }
  if (submitted.objectLdns && submitted.objectLdns.length > 0) {
    signature.objectLdns = submitted.objectLdns;
  }
  return JSON.stringify(signature);
}

export function restoredQueryDelayMs(
  savedAt: string | undefined,
  nowMs: number,
  throttleMs = RESTORED_QUERY_THROTTLE_MS,
): number {
  if (!savedAt) return 0;
  const savedMs = Date.parse(savedAt);
  if (!Number.isFinite(savedMs)) return 0;
  return Math.max(0, throttleMs - (nowMs - savedMs));
}

export function buildTaskDashboardStateSnapshot(input: TaskDashboardSaveInput): {
  filters: JsonObject;
  view: JsonObject;
  lastAction: { submittedQuery: boolean; renderedChart: boolean; refreshed: boolean };
} {
  const [rangeStart, rangeEnd] = input.filter.range;
  return {
    filters: {
      taskId: input.taskId,
      rangeMode: input.rangeMode,
      rangeStartISO: rangeStart.toISOString(),
      rangeEndISO: rangeEnd.toISOString(),
      weekdays: input.filter.weekdays,
      hours: input.filter.hours,
      compare: input.filter.compare,
      dimSelected: input.dimSelected,
      submitted: input.submitted ? { ...input.submitted } : null,
      querySignature: taskDashboardQuerySignature(input.taskId, input.submitted),
    } as JsonObject,
    view: {
      activeGran: input.activeGran ?? null,
      effectiveGran: input.effectiveGran ?? input.activeGran ?? input.submitted?.granularity ?? null,
    },
    lastAction: {
      submittedQuery: Boolean(input.submitted),
      renderedChart: Boolean(input.submitted),
      refreshed: false,
    },
  };
}

export function restoreTaskDashboardState(
  snapshot: PmPageStateSnapshot | null,
  systemTimezone?: string | null,
): RestoredTaskDashboardState {
  const filters = (snapshot?.filters ?? {}) as Record<string, unknown>;
  const view = (snapshot?.view ?? {}) as Record<string, unknown>;
  const rangeMode = normalizeRangeMode(filters.rangeMode);
  const savedSubmitted = snapshot?.lastAction.submittedQuery && filters.submitted && typeof filters.submitted === 'object'
    ? (filters.submitted as unknown as TaskDashboardSubmittedQuery)
    : null;
  const activeGranularity =
    asGranularity(view.activeGran) ??
    asGranularity(view.effectiveGran) ??
    asGranularity(savedSubmitted?.granularity);
  const filter: DashboardFilterValue = {
    range: restoreRange(filters, rangeMode, systemTimezone, activeGranularity),
    weekdays: asNumberArray(filters.weekdays, ALL_WEEKDAYS),
    hours: asNumberArray(filters.hours, ALL_HOURS),
    compare: asBoolean(filters.compare),
  };

  const submitted = savedSubmitted && (activeGranularity || rangeMode.kind === 'relative')
    ? buildSubmittedTaskDashboardQuery(filter, {
        productIds: savedSubmitted.productIds,
        objectLdns: savedSubmitted.objectLdns,
        systemTimezone,
        granularity: activeGranularity,
        rangeMode,
      })
    : savedSubmitted;

  return {
    taskId: asString(filters.taskId),
    filter,
    dimSelected: asStringArray(filters.dimSelected),
    activeGran: asString(view.activeGran),
    rangeMode,
    submitted,
    savedAt: snapshot?.savedAt,
  };
}
