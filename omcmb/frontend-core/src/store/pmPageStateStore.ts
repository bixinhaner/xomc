import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';

export type JsonPrimitive = string | number | boolean | null;
export type JsonValue = JsonPrimitive | JsonValue[] | { [key: string]: JsonValue };
export type JsonObject = { [key: string]: JsonValue };

export interface PmPageLastActionState {
  submittedQuery: boolean;
  renderedChart: boolean;
  refreshed: boolean;
}

export interface PmPageStateSnapshot {
  filters?: JsonObject;
  view?: JsonObject;
  lastAction: PmPageLastActionState;
  savedAt: string;
}

interface PmPageStateStore {
  pages: Record<string, PmPageStateSnapshot>;
  savePageState: (
    pageKey: string,
    snapshot: {
      filters?: JsonObject;
      view?: JsonObject;
      lastAction?: Partial<PmPageLastActionState>;
    },
  ) => void;
  getPageState: (pageKey: string) => PmPageStateSnapshot | null;
  clearPageState: (pageKey: string) => void;
  clearPageStates: (pageKeys: string[]) => void;
  clearAllPageStates: () => void;
}

const PM_STATE_STORAGE_KEY = 'omc-pm-page-state-store';

const DEFAULT_LAST_ACTION: PmPageLastActionState = {
  submittedQuery: false,
  renderedChart: false,
  refreshed: false,
};

const PERFORMANCE_PAGE_KEYS = new Set([
  '/performance',
  '/performance/pm-adhoc',
  '/performance/device-view',
  '/performance/query',
]);

const FORBIDDEN_FIELD_NAMES = new Set([
  'tableRows',
  'rows',
  'rowData',
  'dataSource',
  'chartSeries',
  'series',
  'exportContent',
  'exportData',
  'exportTask',
  'result',
  'results',
  'response',
  'request',
  'requestConfig',
  'abortController',
  'loading',
  'isLoading',
  'error',
  'errorObject',
]);

function ensurePlainSerializable(value: JsonValue, path = 'snapshot'): JsonValue {
  if (value === null) return value;
  const valueType = typeof value;
  if (valueType === 'string' || valueType === 'number' || valueType === 'boolean') {
    if (valueType === 'number' && !Number.isFinite(value as number)) {
      throw new Error(`${path} must be JSON serializable`);
    }
    return value;
  }
  if (Array.isArray(value)) {
    return value.map((item, index) => ensurePlainSerializable(item, `${path}[${index}]`));
  }
  if (valueType !== 'object' || Object.getPrototypeOf(value) !== Object.prototype) {
    throw new Error(`${path} must be JSON serializable`);
  }
  const result: JsonObject = {};
  for (const [key, child] of Object.entries(value as JsonObject)) {
    if (FORBIDDEN_FIELD_NAMES.has(key)) continue;
    result[key] = ensurePlainSerializable(child, `${path}.${key}`);
  }
  return result;
}

function normalizeJsonObject(value: JsonObject | undefined, path: string): JsonObject | undefined {
  if (value === undefined) return undefined;
  return ensurePlainSerializable(value, path) as JsonObject;
}

function normalizePageKey(pageKey: string): string {
  return pageKey.split('?')[0].replace(/\/+$/, '') || pageKey;
}

export function isPerformanceTabPath(path: string): boolean {
  const basePath = normalizePageKey(path);
  return PERFORMANCE_PAGE_KEYS.has(basePath);
}

export function performancePageKeyFromPath(path: string): string | null {
  if (!isPerformanceTabPath(path)) return null;
  return normalizePageKey(path);
}

function requirePerformancePageKey(pageKey: string): string {
  const normalizedPageKey = normalizePageKey(pageKey);
  if (!PERFORMANCE_PAGE_KEYS.has(normalizedPageKey)) {
    throw new Error(`unsupported performance page state key: ${pageKey}`);
  }
  return normalizedPageKey;
}

export const usePmPageStateStore = create<PmPageStateStore>()(
  persist(
    (set, get) => ({
      pages: {},

      savePageState: (pageKey, snapshot) => {
        const normalizedPageKey = requirePerformancePageKey(pageKey);
        const existing = get().pages[normalizedPageKey];
        set({
          pages: {
            ...get().pages,
            [normalizedPageKey]: {
              filters: normalizeJsonObject(snapshot.filters, 'snapshot.filters'),
              view: normalizeJsonObject(snapshot.view, 'snapshot.view'),
              lastAction: {
                ...(existing?.lastAction ?? DEFAULT_LAST_ACTION),
                ...(snapshot.lastAction ?? {}),
              },
              savedAt: new Date().toISOString(),
            },
          },
        });
      },

      getPageState: (pageKey) => {
        return get().pages[normalizePageKey(pageKey)] ?? null;
      },

      clearPageState: (pageKey) => {
        const normalizedPageKey = normalizePageKey(pageKey);
        const nextPages = { ...get().pages };
        delete nextPages[normalizedPageKey];
        set({ pages: nextPages });
      },

      clearPageStates: (pageKeys) => {
        const normalizedPageKeys = new Set(pageKeys.map(normalizePageKey));
        const nextPages = Object.fromEntries(
          Object.entries(get().pages).filter(([pageKey]) => !normalizedPageKeys.has(pageKey)),
        );
        set({ pages: nextPages });
      },

      clearAllPageStates: () => {
        set({ pages: {} });
      },
    }),
    {
      name: PM_STATE_STORAGE_KEY,
      storage: createJSONStorage(() => sessionStorage),
      partialize: (state) => ({ pages: state.pages }),
    },
  ),
);
