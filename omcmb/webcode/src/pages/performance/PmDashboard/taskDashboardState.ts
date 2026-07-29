import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import type { JsonObject, PmPageStateSnapshot } from '@core/store/pmPageStateStore';
import { nowInSystemTimezone, toSystemTimezoneRFC3339 } from '@core/utils/systemTime';
import type { DashboardFilterValue } from './DashboardFilterBar';
import { ALL_HOURS, ALL_WEEKDAYS, previousWindow } from './dashboardFilterUtils';

export const PM_DASHBOARD_PAGE_KEY = '/performance';

export type DashboardRangeMode =
  | { kind: 'relative'; durationMs: number }
  | { kind: 'absolute' };

export interface TaskDashboardSubmittedQuery {
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
  rangeMode: DashboardRangeMode;
  submitted: TaskDashboardSubmittedQuery | null;
}

const DEFAULT_RELATIVE_DURATION_MS = 7 * 24 * 60 * 60 * 1000;
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

function buildDefaultRange(systemTimezone?: string | null): [Dayjs, Dayjs] {
  const now = nowInSystemTimezone(systemTimezone);
  return [now.subtract(DEFAULT_RELATIVE_DURATION_MS, 'millisecond'), now];
}

export function buildDefaultTaskDashboardFilter(systemTimezone?: string | null): DashboardFilterValue {
  return {
    range: buildDefaultRange(systemTimezone),
    weekdays: [...ALL_WEEKDAYS],
    hours: [...ALL_HOURS],
    compare: false,
  };
}

export function buildTaskDashboardTaskSwitchReset(systemTimezone?: string | null): {
  filter: DashboardFilterValue;
  rangeMode: DashboardRangeMode;
  dimSelected: string[];
  activeGran?: string;
  submitted: null;
} {
  return {
    filter: buildDefaultTaskDashboardFilter(systemTimezone),
    rangeMode: { ...DEFAULT_RANGE_MODE },
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
): [Dayjs, Dayjs] {
  if (rangeMode.kind === 'relative') {
    const now = nowInSystemTimezone(systemTimezone);
    return [now.subtract(rangeMode.durationMs, 'millisecond'), now];
  }

  const start = asString(filters.rangeStartISO);
  const end = asString(filters.rangeEndISO);
  const parsedStart = start ? dayjs(start) : null;
  const parsedEnd = end ? dayjs(end) : null;
  if (parsedStart?.isValid() && parsedEnd?.isValid()) return [parsedStart, parsedEnd];
  return buildDefaultRange(systemTimezone);
}

export function buildSubmittedTaskDashboardQuery(
  filter: DashboardFilterValue,
  params: {
    productIds?: string[];
    objectLdns?: string[];
    systemTimezone?: string | null;
  },
): TaskDashboardSubmittedQuery {
  const [s, e] = filter.range;
  const sISO = toSystemTimezoneRFC3339(s, params.systemTimezone) ?? s.toISOString();
  const eISO = toSystemTimezoneRFC3339(e, params.systemTimezone) ?? e.toISOString();
  const [ps, pe] = previousWindow(filter.range);
  const submitted: TaskDashboardSubmittedQuery = {
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
  const filter: DashboardFilterValue = {
    range: restoreRange(filters, rangeMode, systemTimezone),
    weekdays: asNumberArray(filters.weekdays, ALL_WEEKDAYS),
    hours: asNumberArray(filters.hours, ALL_HOURS),
    compare: asBoolean(filters.compare),
  };

  const savedSubmitted = snapshot?.lastAction.submittedQuery && filters.submitted && typeof filters.submitted === 'object'
    ? (filters.submitted as unknown as TaskDashboardSubmittedQuery)
    : null;
  const submitted = savedSubmitted && rangeMode.kind === 'relative'
    ? buildSubmittedTaskDashboardQuery(filter, {
        productIds: savedSubmitted.productIds,
        objectLdns: savedSubmitted.objectLdns,
        systemTimezone,
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
