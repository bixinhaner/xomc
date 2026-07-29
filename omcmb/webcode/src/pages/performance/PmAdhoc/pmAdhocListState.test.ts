import { describe, expect, it } from 'vitest';
import type { PmPageStateSnapshot } from '@core/store/pmPageStateStore';
import {
  buildPmAdhocListStateSnapshot,
  DEFAULT_PM_ADHOC_LIST_STATE,
  PM_ADHOC_DEFAULT_PAGE_SIZE,
  restorePmAdhocListState,
} from './pmAdhocListState';

describe('pmAdhocListState', () => {
  it('restores built-in/custom list area, pagination, and opened detail task id', () => {
    const snapshot: PmPageStateSnapshot = {
      ...buildPmAdhocListStateSnapshot({
        activeListArea: 'custom',
        builtinPagination: { current: 2, pageSize: 20 },
        customPagination: { current: 3, pageSize: 50 },
        detailTaskId: 'task-42',
      }),
      savedAt: '2026-07-29T00:00:00.000Z',
    };

    expect(restorePmAdhocListState(snapshot)).toEqual({
      activeListArea: 'custom',
      builtinPagination: { current: 2, pageSize: 20 },
      customPagination: { current: 3, pageSize: 50 },
      detailTaskId: 'task-42',
    });
  });

  it('falls back to the default list state when saved values are invalid', () => {
    const snapshot: PmPageStateSnapshot = {
      filters: {},
      view: {
        activeListArea: 'other',
        builtinPagination: { current: -1, pageSize: 0 },
        customPagination: { current: '2', pageSize: Number.NaN },
        detailTaskId: '',
      },
      lastAction: { submittedQuery: false, renderedChart: false, refreshed: true },
      savedAt: '2026-07-29T00:00:00.000Z',
    };

    expect(restorePmAdhocListState(snapshot)).toEqual(DEFAULT_PM_ADHOC_LIST_STATE);
  });

  it('keeps snapshots scoped to view state without task rows or result payloads', () => {
    const snapshot = buildPmAdhocListStateSnapshot({
      activeListArea: 'builtin',
      builtinPagination: { current: 4, pageSize: PM_ADHOC_DEFAULT_PAGE_SIZE },
      customPagination: { current: 1, pageSize: PM_ADHOC_DEFAULT_PAGE_SIZE },
      detailTaskId: null,
    });

    expect(snapshot.filters).toEqual({});
    expect(snapshot.view).toEqual({
      activeListArea: 'builtin',
      builtinPagination: { current: 4, pageSize: PM_ADHOC_DEFAULT_PAGE_SIZE },
      customPagination: { current: 1, pageSize: PM_ADHOC_DEFAULT_PAGE_SIZE },
      detailTaskId: null,
    });
    expect(JSON.stringify(snapshot)).not.toContain('tasks');
    expect(JSON.stringify(snapshot)).not.toContain('results');
  });
});
