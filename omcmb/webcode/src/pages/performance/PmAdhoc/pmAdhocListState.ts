import type { JsonObject, PmPageStateSnapshot } from '@core/store/pmPageStateStore';

export const PM_ADHOC_PAGE_KEY = '/performance/pm-adhoc';
export const PM_ADHOC_DEFAULT_PAGE_SIZE = 10;

export type PmAdhocListArea = 'builtin' | 'custom';

export interface PmAdhocTablePaginationState {
  current: number;
  pageSize: number;
}

export interface PmAdhocListSaveInput {
  activeListArea: PmAdhocListArea;
  builtinPagination: PmAdhocTablePaginationState;
  customPagination: PmAdhocTablePaginationState;
  detailTaskId: string | null;
}

export interface RestoredPmAdhocListState extends PmAdhocListSaveInput {}

export const DEFAULT_PM_ADHOC_TABLE_PAGINATION: PmAdhocTablePaginationState = {
  current: 1,
  pageSize: PM_ADHOC_DEFAULT_PAGE_SIZE,
};

export const DEFAULT_PM_ADHOC_LIST_STATE: RestoredPmAdhocListState = {
  activeListArea: 'builtin',
  builtinPagination: DEFAULT_PM_ADHOC_TABLE_PAGINATION,
  customPagination: DEFAULT_PM_ADHOC_TABLE_PAGINATION,
  detailTaskId: null,
};

function asPositiveInteger(value: unknown, fallback: number): number {
  return Number.isInteger(value) && (value as number) > 0 ? value as number : fallback;
}

function normalizeListArea(value: unknown): PmAdhocListArea {
  return value === 'custom' ? 'custom' : 'builtin';
}

function normalizePagination(value: unknown): PmAdhocTablePaginationState {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return { ...DEFAULT_PM_ADHOC_TABLE_PAGINATION };
  }
  const source = value as Record<string, unknown>;
  return {
    current: asPositiveInteger(source.current, DEFAULT_PM_ADHOC_TABLE_PAGINATION.current),
    pageSize: asPositiveInteger(source.pageSize, DEFAULT_PM_ADHOC_TABLE_PAGINATION.pageSize),
  };
}

function normalizeDetailTaskId(value: unknown): string | null {
  return typeof value === 'string' && value.trim() ? value : null;
}

export function buildPmAdhocListStateSnapshot(input: PmAdhocListSaveInput): {
  filters: JsonObject;
  view: JsonObject;
  lastAction: { submittedQuery: boolean; renderedChart: boolean; refreshed: boolean };
} {
  return {
    filters: {},
    view: {
      activeListArea: input.activeListArea,
      builtinPagination: { ...input.builtinPagination },
      customPagination: { ...input.customPagination },
      detailTaskId: input.detailTaskId,
    } as JsonObject,
    lastAction: {
      submittedQuery: false,
      renderedChart: false,
      refreshed: true,
    },
  };
}

export function restorePmAdhocListState(
  snapshot: PmPageStateSnapshot | null,
): RestoredPmAdhocListState {
  const view = (snapshot?.view ?? {}) as Record<string, unknown>;
  return {
    activeListArea: normalizeListArea(view.activeListArea),
    builtinPagination: normalizePagination(view.builtinPagination),
    customPagination: normalizePagination(view.customPagination),
    detailTaskId: normalizeDetailTaskId(view.detailTaskId),
  };
}
