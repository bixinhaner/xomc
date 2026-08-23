import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import type { JsonObject, PmPageStateSnapshot } from '@core/store/pmPageStateStore';
import type { Granularity } from '@core/types/pmDashboard';
import type { TechnologyType } from '@core/types/technology';
import type { CellSelection } from './cellDrilldownUtils';
import type { DashboardFilterValue } from './DashboardFilterBar';
import { ALL_HOURS, ALL_WEEKDAYS } from './dashboardFilterUtils';
import {
  buildDeviceViewRequestTimeWindow,
  defaultRangeForGranularity,
} from './deviceListPaneTimeUtils';

export const PM_DEVICE_VIEW_PAGE_KEY = '/performance/device-view';
export const DEVICE_VIEW_RESTORED_QUERY_THROTTLE_MS = 2_000;

export interface DeviceViewSubmittedQuery {
  tech: TechnologyType;
  deviceSns: string[];
  metricPaths: string[];
  granularity: Granularity;
  startTime: string;
  endTime: string;
  weekdays: number[];
  hours: number[];
  compare: boolean;
  offsetMs: number;
  prevStartTime: string;
  prevEndTime: string;
  allowedLdns: string[];
}

export interface DeviceViewSaveInput {
  tech: TechnologyType;
  deviceSns: string[];
  cellSel: CellSelection;
  metricPaths: string[];
  metricsTouched: boolean;
  granularity: Granularity;
  rangeTouched: boolean;
  filter: DashboardFilterValue;
  submitted: DeviceViewSubmittedQuery | null;
  refreshed?: boolean;
}

export interface RestoredDeviceViewState {
  tech: TechnologyType;
  deviceSns: string[];
  cellSel: CellSelection;
  metricPaths: string[];
  metricsTouched: boolean;
  granularity: Granularity;
  rangeTouched: boolean;
  filter: DashboardFilterValue;
  submitted: DeviceViewSubmittedQuery | null;
  shouldRestoreQuery: boolean;
  savedAt?: string;
}

const DEFAULT_TECH: TechnologyType = 'lte';
const DEFAULT_GRANULARITY: Granularity = '15min';

function asString(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value : undefined;
}

function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value.filter((item): item is string => typeof item === 'string');
}

function asNumberArray(value: unknown, fallback: number[]): number[] {
  if (!Array.isArray(value)) return [...fallback];
  const numbers = value.filter((item): item is number => Number.isInteger(item));
  return numbers.length > 0 ? numbers : [...fallback];
}

function asBoolean(value: unknown, fallback = false): boolean {
  return typeof value === 'boolean' ? value : fallback;
}

function normalizeGranularity(value: unknown): Granularity {
  if (
    value === '15min' ||
    value === 'hourly' ||
    value === 'daily' ||
    value === 'weekly' ||
    value === 'monthly'
  ) {
    return value;
  }
  return DEFAULT_GRANULARITY;
}

function normalizeTech(value: unknown): TechnologyType {
  return value === 'nr' || value === 'gsm' || value === 'lte' ? value : DEFAULT_TECH;
}

function parseRange(filters: Record<string, unknown>): [Dayjs, Dayjs] | null {
  const start = asString(filters.rangeStartISO);
  const end = asString(filters.rangeEndISO);
  const parsedStart = start ? dayjs(start) : null;
  const parsedEnd = end ? dayjs(end) : null;
  if (parsedStart?.isValid() && parsedEnd?.isValid()) return [parsedStart, parsedEnd];
  return null;
}

function normalizeCellSelection(value: unknown): CellSelection {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {};
  const result: CellSelection = {};
  for (const [deviceSn, selected] of Object.entries(value as Record<string, unknown>)) {
    const selectedLdns = asStringArray(selected);
    if (selectedLdns.length > 0) result[deviceSn] = selectedLdns;
  }
  return result;
}

export function buildDeviceViewSubmittedQuery(
  input: {
    deviceSns: string[];
    tech: TechnologyType;
    metricPaths: string[];
    granularity: Granularity;
    filter: DashboardFilterValue;
    allowedLdns: string[];
    systemTimezone?: string | null;
  },
): DeviceViewSubmittedQuery {
  const [start, end] = input.filter.range;
  const requestWindow = buildDeviceViewRequestTimeWindow(input.filter.range, input.systemTimezone);
  return {
    tech: input.tech,
    deviceSns: [...input.deviceSns],
    metricPaths: [...input.metricPaths],
    granularity: input.granularity,
    startTime: requestWindow.startTime,
    endTime: requestWindow.endTime,
    weekdays: [...input.filter.weekdays],
    hours: [...input.filter.hours],
    compare: input.filter.compare,
    offsetMs: end.valueOf() - start.valueOf(),
    prevStartTime: requestWindow.prevStartTime,
    prevEndTime: requestWindow.prevEndTime,
    allowedLdns: [...input.allowedLdns],
  };
}

export function deviceViewQuerySignature(submitted: DeviceViewSubmittedQuery | null): string | null {
  if (!submitted) return null;
  return JSON.stringify({
    deviceSns: submitted.deviceSns,
    tech: submitted.tech,
    metricPaths: submitted.metricPaths,
    granularity: submitted.granularity,
    startTime: submitted.startTime,
    endTime: submitted.endTime,
    weekdays: submitted.weekdays,
    hours: submitted.hours,
    compare: submitted.compare,
    prevStartTime: submitted.prevStartTime,
    prevEndTime: submitted.prevEndTime,
    allowedLdns: submitted.allowedLdns,
  });
}

export function restoredDeviceViewQueryDelayMs(
  savedAt: string | undefined,
  nowMs: number,
  throttleMs = DEVICE_VIEW_RESTORED_QUERY_THROTTLE_MS,
): number {
  if (!savedAt) return 0;
  const savedMs = Date.parse(savedAt);
  if (!Number.isFinite(savedMs)) return 0;
  return Math.max(0, throttleMs - (nowMs - savedMs));
}

export function buildDeviceViewStateSnapshot(input: DeviceViewSaveInput): {
  filters: JsonObject;
  view: JsonObject;
  lastAction: { submittedQuery: boolean; renderedChart: boolean; refreshed: boolean };
} {
  const [rangeStart, rangeEnd] = input.filter.range;
  return {
    filters: {
      tech: input.tech,
      deviceSns: input.deviceSns,
      cellSel: input.cellSel as JsonObject,
      metricPaths: input.metricPaths,
      metricsTouched: input.metricsTouched,
      granularity: input.granularity,
      rangeTouched: input.rangeTouched,
      rangeStartISO: rangeStart.toISOString(),
      rangeEndISO: rangeEnd.toISOString(),
      weekdays: input.filter.weekdays,
      hours: input.filter.hours,
      compare: input.filter.compare,
      submitted: input.submitted ? { ...input.submitted } : null,
      querySignature: deviceViewQuerySignature(input.submitted),
    } as JsonObject,
    view: {},
    lastAction: {
      submittedQuery: Boolean(input.submitted),
      renderedChart: Boolean(input.submitted),
      refreshed: input.refreshed ?? false,
    },
  };
}

export function restoreDeviceViewState(
  snapshot: PmPageStateSnapshot | null,
  systemTimezone?: string | null,
): RestoredDeviceViewState {
  const filters = (snapshot?.filters ?? {}) as Record<string, unknown>;
  const granularity = normalizeGranularity(filters.granularity);
  const rangeTouched = asBoolean(filters.rangeTouched);
  const range = rangeTouched
    ? parseRange(filters) ?? defaultRangeForGranularity(granularity, systemTimezone)
    : defaultRangeForGranularity(granularity, systemTimezone);
  const filter: DashboardFilterValue = {
    range,
    weekdays: asNumberArray(filters.weekdays, ALL_WEEKDAYS),
    hours: asNumberArray(filters.hours, ALL_HOURS),
    compare: asBoolean(filters.compare),
  };
  const savedSubmitted = snapshot?.lastAction.submittedQuery && filters.submitted && typeof filters.submitted === 'object'
    ? (filters.submitted as unknown as DeviceViewSubmittedQuery)
    : null;
  const canRestoreSubmitted = savedSubmitted
    ? savedSubmitted.deviceSns.length === 0 || savedSubmitted.allowedLdns.length > 0
    : false;
  const submitted = savedSubmitted && canRestoreSubmitted
    ? rangeTouched
      ? { ...savedSubmitted, tech: savedSubmitted.tech ?? normalizeTech(filters.tech) }
      : buildDeviceViewSubmittedQuery({
          tech: savedSubmitted.tech ?? normalizeTech(filters.tech),
          deviceSns: savedSubmitted.deviceSns,
          metricPaths: savedSubmitted.metricPaths,
          granularity: savedSubmitted.granularity,
          filter,
          allowedLdns: savedSubmitted.allowedLdns,
          systemTimezone,
        })
    : null;

  return {
    tech: normalizeTech(filters.tech),
    deviceSns: asStringArray(filters.deviceSns),
    cellSel: normalizeCellSelection(filters.cellSel),
    metricPaths: asStringArray(filters.metricPaths),
    metricsTouched: asBoolean(filters.metricsTouched),
    granularity,
    rangeTouched,
    filter,
    submitted,
    shouldRestoreQuery: Boolean(
      submitted &&
      (
        snapshot?.lastAction.submittedQuery ||
        snapshot?.lastAction.renderedChart ||
        snapshot?.lastAction.refreshed
      ),
    ),
    savedAt: snapshot?.savedAt,
  };
}
