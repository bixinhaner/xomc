import type { JsonObject, PmPageStateSnapshot } from '@core/store/pmPageStateStore';
import { usePmPageStateStore } from '@core/store/pmPageStateStore';
import type { AdhocDimension, AdhocVisibility } from '@core/types/pmAdhoc';
import type { TechnologyType } from '@core/types/technology';
import type { CellSelection } from '../PmDashboard/cellDrilldownUtils';

export const PM_ADHOC_PAGE_KEY = '/performance/pm-adhoc';

export type MetricTypeFilter = 'all' | 'kpi' | 'counter';
export type CustomWizardDraftKey = { mode: 'new' } | { mode: 'edit'; taskId: string };

export interface PmAdhocCustomWizardDraft {
  current: number;
  name: string;
  technology: TechnologyType;
  expireDays: number;
  visibility: AdhocVisibility;
  dimension: AdhocDimension;
  selectedSns: string[];
  cellSel: CellSelection;
  metricPaths: string[];
  metricTypeFilter: MetricTypeFilter;
  windowStart: string;
  windowEnd: string;
  plannedEndAt: string | null;
  plannedEndTouched: boolean;
  drilldownTouched: boolean;
  originalObjectLdns: string[];
}

export interface PmAdhocBuiltinMetricDraft {
  taskId: string;
  metricPaths: string[];
  metricTypeFilter: MetricTypeFilter;
}

interface PmAdhocDraftBucket {
  customNew?: PmAdhocCustomWizardDraft;
  customEditById: Record<string, PmAdhocCustomWizardDraft>;
  builtinMetricById: Record<string, PmAdhocBuiltinMetricDraft>;
  activeBuiltinMetricTaskId?: string;
}

const EMPTY_BUCKET: PmAdhocDraftBucket = {
  customEditById: {},
  builtinMetricById: {},
};

function isPlainRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}

function asStringArray(value: unknown): string[] {
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string') : [];
}

function asMetricTypeFilter(value: unknown): MetricTypeFilter {
  return value === 'kpi' || value === 'counter' ? value : 'all';
}

function asCustomDraft(value: unknown): PmAdhocCustomWizardDraft | undefined {
  if (!isPlainRecord(value)) return undefined;
  const technology = value.technology === 'nr' || value.technology === 'gsm' ? value.technology : 'lte';
  const visibility = value.visibility === 'public' ? 'public' : 'private';
  const dimension = typeof value.dimension === 'string' ? value.dimension as AdhocDimension : 'network';
  const current = typeof value.current === 'number' && Number.isFinite(value.current)
    ? Math.min(Math.max(Math.floor(value.current), 0), 4)
    : 0;
  return {
    current,
    name: typeof value.name === 'string' ? value.name : '',
    technology,
    expireDays: typeof value.expireDays === 'number' && Number.isFinite(value.expireDays) ? value.expireDays : 60,
    visibility,
    dimension,
    selectedSns: asStringArray(value.selectedSns),
    cellSel: isPlainRecord(value.cellSel) ? value.cellSel as CellSelection : {},
    metricPaths: asStringArray(value.metricPaths),
    metricTypeFilter: asMetricTypeFilter(value.metricTypeFilter),
    windowStart: typeof value.windowStart === 'string' ? value.windowStart : '',
    windowEnd: typeof value.windowEnd === 'string' ? value.windowEnd : '',
    plannedEndAt: typeof value.plannedEndAt === 'string' ? value.plannedEndAt : null,
    plannedEndTouched: value.plannedEndTouched === true,
    drilldownTouched: value.drilldownTouched === true,
    originalObjectLdns: asStringArray(value.originalObjectLdns),
  };
}

function asBuiltinMetricDraft(value: unknown): PmAdhocBuiltinMetricDraft | undefined {
  if (!isPlainRecord(value) || typeof value.taskId !== 'string') return undefined;
  return {
    taskId: value.taskId,
    metricPaths: asStringArray(value.metricPaths),
    metricTypeFilter: asMetricTypeFilter(value.metricTypeFilter),
  };
}

export function restorePmAdhocDraftBucket(snapshot: PmPageStateSnapshot | null): PmAdhocDraftBucket {
  const rawBucket = snapshot?.view?.adhocDrafts;
  if (!isPlainRecord(rawBucket)) return { ...EMPTY_BUCKET };
  const customNew = asCustomDraft(rawBucket.customNew);
  const customEditById: Record<string, PmAdhocCustomWizardDraft> = {};
  if (isPlainRecord(rawBucket.customEditById)) {
    for (const [taskId, draft] of Object.entries(rawBucket.customEditById)) {
      const parsed = asCustomDraft(draft);
      if (parsed) customEditById[taskId] = parsed;
    }
  }
  const builtinMetricById: Record<string, PmAdhocBuiltinMetricDraft> = {};
  if (isPlainRecord(rawBucket.builtinMetricById)) {
    for (const [taskId, draft] of Object.entries(rawBucket.builtinMetricById)) {
      const parsed = asBuiltinMetricDraft(draft);
      if (parsed) builtinMetricById[taskId] = parsed;
    }
  }
  return {
    customNew,
    customEditById,
    builtinMetricById,
    activeBuiltinMetricTaskId: typeof rawBucket.activeBuiltinMetricTaskId === 'string'
      ? rawBucket.activeBuiltinMetricTaskId
      : undefined,
  };
}

function stripOldListState(view: JsonObject | undefined): JsonObject {
  const {
    activeListArea: _activeListArea,
    builtinPagination: _builtinPagination,
    customPagination: _customPagination,
    detailTaskId: _detailTaskId,
    ...rest
  } = view ?? {};
  return rest;
}

function toJsonBucket(bucket: PmAdhocDraftBucket): JsonObject {
  const jsonBucket: JsonObject = {
    customEditById: bucket.customEditById as unknown as JsonObject,
    builtinMetricById: bucket.builtinMetricById as unknown as JsonObject,
  };
  if (bucket.customNew) {
    jsonBucket.customNew = bucket.customNew as unknown as JsonObject;
  }
  if (bucket.activeBuiltinMetricTaskId) {
    jsonBucket.activeBuiltinMetricTaskId = bucket.activeBuiltinMetricTaskId;
  }
  return jsonBucket;
}

function saveBucket(nextBucket: PmAdhocDraftBucket) {
  const store = usePmPageStateStore.getState();
  const existing = store.getPageState(PM_ADHOC_PAGE_KEY);
  store.savePageState(PM_ADHOC_PAGE_KEY, {
    filters: existing?.filters,
    view: {
      ...stripOldListState(existing?.view),
      adhocDrafts: toJsonBucket(nextBucket),
    },
  });
}

function updateBucket(updater: (bucket: PmAdhocDraftBucket) => PmAdhocDraftBucket) {
  saveBucket(updater(restorePmAdhocDraftBucket(
    usePmPageStateStore.getState().getPageState(PM_ADHOC_PAGE_KEY),
  )));
}

export function getCustomWizardDraft(key: CustomWizardDraftKey): PmAdhocCustomWizardDraft | undefined {
  const bucket = restorePmAdhocDraftBucket(usePmPageStateStore.getState().getPageState(PM_ADHOC_PAGE_KEY));
  return key.mode === 'new' ? bucket.customNew : bucket.customEditById[key.taskId];
}

export function saveCustomWizardDraft(key: CustomWizardDraftKey, draft: PmAdhocCustomWizardDraft) {
  updateBucket((bucket) => {
    if (key.mode === 'new') {
      return { ...bucket, customNew: draft };
    }
    return {
      ...bucket,
      customEditById: {
        ...bucket.customEditById,
        [key.taskId]: draft,
      },
    };
  });
}

export function clearCustomWizardDraft(key: CustomWizardDraftKey) {
  updateBucket((bucket) => {
    if (key.mode === 'new') {
      const { customNew: _customNew, ...rest } = bucket;
      return rest;
    }
    const { [key.taskId]: _draft, ...customEditById } = bucket.customEditById;
    return { ...bucket, customEditById };
  });
}

export function getBuiltinMetricDraft(taskId: string): PmAdhocBuiltinMetricDraft | undefined {
  const bucket = restorePmAdhocDraftBucket(usePmPageStateStore.getState().getPageState(PM_ADHOC_PAGE_KEY));
  return bucket.builtinMetricById[taskId];
}

export function getActiveBuiltinMetricTaskId(): string | undefined {
  return restorePmAdhocDraftBucket(
    usePmPageStateStore.getState().getPageState(PM_ADHOC_PAGE_KEY),
  ).activeBuiltinMetricTaskId;
}

export function saveBuiltinMetricDraft(draft: PmAdhocBuiltinMetricDraft) {
  updateBucket((bucket) => ({
    ...bucket,
    builtinMetricById: {
      ...bucket.builtinMetricById,
      [draft.taskId]: draft,
    },
    activeBuiltinMetricTaskId: draft.taskId,
  }));
}

export function clearBuiltinMetricDraft(taskId: string) {
  updateBucket((bucket) => {
    const { [taskId]: _draft, ...builtinMetricById } = bucket.builtinMetricById;
    return {
      ...bucket,
      builtinMetricById,
      activeBuiltinMetricTaskId: bucket.activeBuiltinMetricTaskId === taskId
        ? undefined
        : bucket.activeBuiltinMetricTaskId,
    };
  });
}

export function clearActiveBuiltinMetricTaskId(taskId?: string) {
  updateBucket((bucket) => {
    if (taskId && bucket.activeBuiltinMetricTaskId !== taskId) return bucket;
    return { ...bucket, activeBuiltinMetricTaskId: undefined };
  });
}
