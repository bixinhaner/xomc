import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import type { JsonObject, PmPageStateSnapshot } from '@core/store/pmPageStateStore';
import type { DeviceType } from '@core/types/indicatorLibrary';
import type { Granularity } from '@core/types/pmDashboard';
import type { QueryTemplatePayload, TimeRangePreset } from '@core/types/pmQuery';
import type { CellSelection } from '../PmDashboard/cellDrilldownUtils';

export const PM_KPI_QUERY_PAGE_KEY = '/performance/query';
export const KPI_QUERY_RESTORED_QUERY_THROTTLE_MS = 10_000;
export const KPI_QUERY_DEFAULT_PIVOT_PAGE_SIZE = 50;

export interface KpiQuerySubmittedQuery {
  payload: QueryTemplatePayload;
  range: { start: string; end: string };
  cellSel: CellSelection;
}

export type KpiQuerySubmittedSnapshot = KpiQuerySubmittedQuery;

export interface KpiQuerySaveInput {
  payload: QueryTemplatePayload;
  customRange: [Dayjs, Dayjs] | null;
  timeRangeDirty: boolean;
  cellSel: CellSelection;
  submitted: KpiQuerySubmittedQuery | null;
  pivotPage: number;
  pivotPageSize: number;
  templateTab: 'public' | 'private';
  activeTemplateId?: string;
  sidebarCollapsed: boolean;
}

export interface RestoredKpiQueryState {
  payload: QueryTemplatePayload;
  customRange: [Dayjs, Dayjs] | null;
  timeRangeDirty: boolean;
  cellSel: CellSelection;
  submitted: KpiQuerySubmittedQuery | null;
  pivotPage: number;
  pivotPageSize: number;
  templateTab: 'public' | 'private';
  activeTemplateId?: string;
  sidebarCollapsed: boolean;
  shouldRestoreQuery: boolean;
  savedAt?: string;
}

const DEFAULT_PAYLOAD: QueryTemplatePayload = {
  deviceSns: [],
  metricPaths: [],
  granularity: '15min',
  timeRangePreset: 'last_3h',
  deviceType: 'ENB',
};

function asString(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value : undefined;
}

function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value.filter((item): item is string => typeof item === 'string');
}

function asPositiveInteger(value: unknown, fallback: number): number {
  return Number.isInteger(value) && (value as number) > 0 ? value as number : fallback;
}

function asBoolean(value: unknown, fallback = false): boolean {
  return typeof value === 'boolean' ? value : fallback;
}

function normalizeDeviceType(value: unknown): DeviceType {
  return value === 'GNB' || value === 'GSM' || value === 'ENB' ? value : 'ENB';
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
  return '15min';
}

function normalizeTimeRangePreset(value: unknown): TimeRangePreset {
  if (
    value === 'last_1h' ||
    value === 'last_3h' ||
    value === 'last_24h' ||
    value === 'last_7d' ||
    value === 'last_30d' ||
    value === 'last_6m' ||
    value === 'custom'
  ) {
    return value;
  }
  return 'last_3h';
}

function normalizePayload(value: unknown): QueryTemplatePayload {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return { ...DEFAULT_PAYLOAD, deviceSns: [], metricPaths: [] };
  }
  const source = value as Record<string, unknown>;
  const payload: QueryTemplatePayload = {
    deviceSns: asStringArray(source.deviceSns),
    metricPaths: asStringArray(source.metricPaths),
    granularity: normalizeGranularity(source.granularity),
    timeRangePreset: normalizeTimeRangePreset(source.timeRangePreset),
    deviceType: normalizeDeviceType(source.deviceType),
  };
  const absoluteStart = asString(source.absoluteStart);
  const absoluteEnd = asString(source.absoluteEnd);
  if (absoluteStart) payload.absoluteStart = absoluteStart;
  if (absoluteEnd) payload.absoluteEnd = absoluteEnd;
  return payload;
}

function serializablePayload(payload: QueryTemplatePayload): JsonObject {
  const result: JsonObject = {
    deviceSns: [...payload.deviceSns],
    metricPaths: [...payload.metricPaths],
    granularity: payload.granularity,
    timeRangePreset: payload.timeRangePreset,
    deviceType: payload.deviceType ?? 'ENB',
  };
  if (payload.absoluteStart) result.absoluteStart = payload.absoluteStart;
  if (payload.absoluteEnd) result.absoluteEnd = payload.absoluteEnd;
  return result;
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

function serializableCellSelection(value: CellSelection): JsonObject {
  return Object.fromEntries(
    Object.entries(value).map(([deviceSn, selected]) => [deviceSn, [...selected]]),
  ) as JsonObject;
}

function parseCustomRange(filters: Record<string, unknown>): [Dayjs, Dayjs] | null {
  const start = asString(filters.customRangeStartISO);
  const end = asString(filters.customRangeEndISO);
  const parsedStart = start ? dayjs(start) : null;
  const parsedEnd = end ? dayjs(end) : null;
  if (parsedStart?.isValid() && parsedEnd?.isValid()) return [parsedStart, parsedEnd];
  return null;
}

function normalizeSubmitted(value: unknown): KpiQuerySubmittedQuery | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null;
  const source = value as Record<string, unknown>;
  const range = source.range;
  if (!range || typeof range !== 'object' || Array.isArray(range)) return null;
  const start = asString((range as Record<string, unknown>).start);
  const end = asString((range as Record<string, unknown>).end);
  if (!start || !end) return null;
  if (!dayjs(start).isValid() || !dayjs(end).isValid()) return null;
  const payload = normalizePayload(source.payload);
  if (payload.deviceSns.length === 0 || payload.metricPaths.length === 0) return null;
  return {
    payload,
    range: { start, end },
    cellSel: normalizeCellSelection(source.cellSel),
  };
}

export function kpiQuerySignature(submitted: KpiQuerySubmittedQuery | null): string | null {
  if (!submitted) return null;
  return JSON.stringify({
    payload: submitted.payload,
    range: submitted.range,
    cellSel: submitted.cellSel,
  });
}

export function buildKpiQuerySubmittedSnapshot(
  payload: QueryTemplatePayload,
  range: { start: string; end: string },
  cellSel: CellSelection,
): KpiQuerySubmittedQuery {
  return {
    payload: normalizePayload(payload),
    range: { ...range },
    cellSel: normalizeCellSelection(cellSel),
  };
}

export function restoredKpiQueryDelayMs(
  savedAt: string | undefined,
  nowMs: number,
  throttleMs = KPI_QUERY_RESTORED_QUERY_THROTTLE_MS,
): number {
  if (!savedAt) return 0;
  const savedMs = Date.parse(savedAt);
  if (!Number.isFinite(savedMs)) return 0;
  return Math.max(0, throttleMs - (nowMs - savedMs));
}

export function buildKpiQueryStateSnapshot(input: KpiQuerySaveInput): {
  filters: JsonObject;
  view: JsonObject;
  lastAction: { submittedQuery: boolean; renderedChart: boolean; refreshed: boolean };
} {
  return {
    filters: {
      payload: serializablePayload(input.payload),
      customRangeStartISO: input.customRange?.[0].toISOString() ?? null,
      customRangeEndISO: input.customRange?.[1].toISOString() ?? null,
      timeRangeDirty: input.timeRangeDirty,
      cellSel: serializableCellSelection(input.cellSel),
      submitted: input.submitted ? {
        payload: serializablePayload(input.submitted.payload),
        range: input.submitted.range as unknown as JsonObject,
        cellSel: serializableCellSelection(input.submitted.cellSel),
      } : null,
      querySignature: kpiQuerySignature(input.submitted),
    } as JsonObject,
    view: {
      pivotPage: input.pivotPage,
      pivotPageSize: input.pivotPageSize,
      templateTab: input.templateTab,
      activeTemplateId: input.activeTemplateId ?? null,
      sidebarCollapsed: input.sidebarCollapsed,
    },
    lastAction: {
      submittedQuery: Boolean(input.submitted),
      renderedChart: Boolean(input.submitted),
      refreshed: false,
    },
  };
}

export function restoreKpiQueryState(snapshot: PmPageStateSnapshot | null): RestoredKpiQueryState {
  const filters = (snapshot?.filters ?? {}) as Record<string, unknown>;
  const view = (snapshot?.view ?? {}) as Record<string, unknown>;
  const submitted = snapshot?.lastAction.submittedQuery ? normalizeSubmitted(filters.submitted) : null;
  const templateTab = view.templateTab === 'private' ? 'private' : 'public';
  return {
    payload: normalizePayload(filters.payload),
    customRange: parseCustomRange(filters),
    timeRangeDirty: asBoolean(filters.timeRangeDirty),
    cellSel: normalizeCellSelection(filters.cellSel),
    submitted,
    pivotPage: asPositiveInteger(view.pivotPage, 1),
    pivotPageSize: asPositiveInteger(view.pivotPageSize, KPI_QUERY_DEFAULT_PIVOT_PAGE_SIZE),
    templateTab,
    activeTemplateId: asString(view.activeTemplateId),
    sidebarCollapsed: asBoolean(view.sidebarCollapsed),
    shouldRestoreQuery: Boolean(
      submitted &&
      (snapshot?.lastAction.submittedQuery || snapshot?.lastAction.renderedChart || snapshot?.lastAction.refreshed),
    ),
    savedAt: snapshot?.savedAt,
  };
}
