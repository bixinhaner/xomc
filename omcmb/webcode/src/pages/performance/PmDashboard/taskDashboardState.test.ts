import dayjs from 'dayjs';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { usePmPageStateStore, type PmPageStateSnapshot } from '@core/store/pmPageStateStore';
import {
  buildDefaultTaskDashboardFilter,
  buildSubmittedTaskDashboardQuery,
  buildTaskDashboardStateSnapshot,
  buildTaskDashboardTaskSwitchReset,
  restoredQueryDelayMs,
  restoreTaskDashboardState,
  taskDashboardQuerySignature,
} from './taskDashboardState';

function snapshotOf(
  input: Parameters<typeof buildTaskDashboardStateSnapshot>[0],
): PmPageStateSnapshot {
  return {
    ...buildTaskDashboardStateSnapshot(input),
    savedAt: '2026-07-29T00:00:00.000Z',
  };
}

afterEach(() => {
  vi.useRealTimers();
  usePmPageStateStore.setState({ pages: {} });
  sessionStorage.clear();
});

describe('taskDashboardState', () => {
  it('restores filters, dimension selection, active granularity and submitted query', () => {
    const filter = {
      range: [dayjs('2026-07-01T00:00:00Z'), dayjs('2026-07-02T00:00:00Z')] as [dayjs.Dayjs, dayjs.Dayjs],
      weekdays: [1, 2, 3],
      hours: [8, 9],
      compare: true,
    };
    const submitted = buildSubmittedTaskDashboardQuery(filter, {
      objectLdns: ['Band=42'],
      systemTimezone: 'UTC',
    });

    const restored = restoreTaskDashboardState(
      snapshotOf({
        taskId: 'task-1',
        filter,
        dimSelected: ['Band=42'],
        activeGran: 'hourly',
        rangeMode: { kind: 'absolute' },
        submitted,
      }),
      'UTC',
    );

    expect(restored.taskId).toBe('task-1');
    expect(restored.filter.weekdays).toEqual([1, 2, 3]);
    expect(restored.filter.hours).toEqual([8, 9]);
    expect(restored.filter.compare).toBe(true);
    expect(restored.dimSelected).toEqual(['Band=42']);
    expect(restored.activeGran).toBe('hourly');
    expect(restored.submitted?.objectLdns).toEqual(['Band=42']);
  });

  it('does not restore a submitted query when the page was only edited but not plotted', () => {
    const filter = buildDefaultTaskDashboardFilter('UTC');
    const restored = restoreTaskDashboardState(
      snapshotOf({
        taskId: 'task-1',
        filter,
        dimSelected: [],
        rangeMode: { kind: 'relative', durationMs: 7 * 24 * 60 * 60 * 1000 },
        submitted: null,
      }),
      'UTC',
    );

    expect(restored.submitted).toBeNull();
  });

  it('stores submittedQuery=false when reset clears submitted state', () => {
    const state = buildTaskDashboardStateSnapshot({
      taskId: 'task-1',
      filter: buildDefaultTaskDashboardFilter('UTC'),
      dimSelected: [],
      rangeMode: { kind: 'relative', durationMs: 7 * 24 * 60 * 60 * 1000 },
      submitted: null,
    });

    expect(state.lastAction.submittedQuery).toBe(false);
    expect(state.filters.submitted).toBeNull();
    expect(state.filters.querySignature).toBeNull();
  });

  it('omits undefined dimension fields from submitted snapshots so the shared store can save them', () => {
    const filter = {
      range: [dayjs('2026-07-01T00:00:00Z'), dayjs('2026-07-02T00:00:00Z')] as [dayjs.Dayjs, dayjs.Dayjs],
      weekdays: [1, 2],
      hours: [8, 9],
      compare: false,
    };
    const submitted = buildSubmittedTaskDashboardQuery(filter, {
      systemTimezone: 'UTC',
    });
    const state = buildTaskDashboardStateSnapshot({
      taskId: 'task-1',
      filter,
      dimSelected: [],
      rangeMode: { kind: 'absolute' },
      submitted,
    });

    expect(Object.hasOwn(submitted, 'productIds')).toBe(false);
    expect(Object.hasOwn(submitted, 'objectLdns')).toBe(false);
    const json = JSON.stringify(state);
    expect(json).not.toContain('productIds');
    expect(json).not.toContain('objectLdns');
    expect(() => {
      usePmPageStateStore.getState().savePageState('/performance', state);
    }).not.toThrow();
    expect(usePmPageStateStore.getState().getPageState('/performance')?.filters?.submitted).toEqual(submitted);
  });

  it('recalculates relative ranges from current time when restored', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-29T12:00:00Z'));
    const filter = buildDefaultTaskDashboardFilter('UTC');
    const snapshot = snapshotOf({
      taskId: 'task-1',
      filter,
      dimSelected: [],
      rangeMode: { kind: 'relative', durationMs: 2 * 60 * 60 * 1000 },
      submitted: null,
    });

    vi.setSystemTime(new Date('2026-07-30T12:00:00Z'));
    const restored = restoreTaskDashboardState(snapshot, 'UTC');

    expect(restored.filter.range[1].toISOString()).toBe('2026-07-30T12:00:00.000Z');
    expect(restored.filter.range[0].toISOString()).toBe('2026-07-30T10:00:00.000Z');
  });

  it('recalculates submitted query time when a submitted relative range is restored', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-29T12:00:00Z'));
    const filter = {
      ...buildDefaultTaskDashboardFilter('UTC'),
      weekdays: [1, 2],
      hours: [8, 9],
      compare: true,
    };
    const submitted = buildSubmittedTaskDashboardQuery(filter, {
      productIds: ['product-a'],
      systemTimezone: 'UTC',
    });
    const snapshot = snapshotOf({
      taskId: 'task-1',
      filter,
      dimSelected: ['product-a'],
      rangeMode: { kind: 'relative', durationMs: 2 * 60 * 60 * 1000 },
      submitted,
    });

    vi.setSystemTime(new Date('2026-07-30T12:00:00Z'));
    const restored = restoreTaskDashboardState(snapshot, 'UTC');

    expect(restored.filter.range[0].toISOString()).toBe('2026-07-30T10:00:00.000Z');
    expect(restored.filter.range[1].toISOString()).toBe('2026-07-30T12:00:00.000Z');
    expect(restored.submitted?.startISO).toBe('2026-07-30T10:00:00Z');
    expect(restored.submitted?.endISO).toBe('2026-07-30T12:00:00Z');
    expect(restored.submitted?.prevStartISO).toBe('2026-07-30T08:00:00Z');
    expect(restored.submitted?.prevEndISO).toBe('2026-07-30T10:00:00Z');
    expect(restored.submitted?.rangeStartMs).toBe(restored.filter.range[0].valueOf());
    expect(restored.submitted?.rangeEndMs).toBe(restored.filter.range[1].valueOf());
    expect(restored.submitted?.productIds).toEqual(['product-a']);
  });

  it('keeps absolute ranges stable when restored', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-30T12:00:00Z'));
    const snapshot = snapshotOf({
      taskId: 'task-1',
      filter: {
        range: [dayjs('2026-07-01T00:00:00Z'), dayjs('2026-07-02T00:00:00Z')] as [dayjs.Dayjs, dayjs.Dayjs],
        weekdays: [0, 1],
        hours: [0, 1],
        compare: false,
      },
      dimSelected: [],
      rangeMode: { kind: 'absolute' },
      submitted: null,
    });

    const restored = restoreTaskDashboardState(snapshot, 'UTC');

    expect(restored.filter.range[0].toISOString()).toBe('2026-07-01T00:00:00.000Z');
    expect(restored.filter.range[1].toISOString()).toBe('2026-07-02T00:00:00.000Z');
  });

  it('keeps submitted query time stable when an absolute range is restored', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-30T12:00:00Z'));
    const filter = {
      range: [dayjs('2026-07-01T00:00:00Z'), dayjs('2026-07-02T00:00:00Z')] as [dayjs.Dayjs, dayjs.Dayjs],
      weekdays: [0, 1],
      hours: [0, 1],
      compare: true,
    };
    const submitted = buildSubmittedTaskDashboardQuery(filter, {
      objectLdns: ['Band=42'],
      systemTimezone: 'UTC',
    });

    const restored = restoreTaskDashboardState(
      snapshotOf({
        taskId: 'task-1',
        filter,
        dimSelected: ['Band=42'],
        rangeMode: { kind: 'absolute' },
        submitted,
      }),
      'UTC',
    );

    expect(restored.submitted?.startISO).toBe(submitted.startISO);
    expect(restored.submitted?.endISO).toBe(submitted.endISO);
    expect(restored.submitted?.prevStartISO).toBe(submitted.prevStartISO);
    expect(restored.submitted?.prevEndISO).toBe(submitted.prevEndISO);
  });

  it('builds a task switch reset state that clears submitted query', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-30T12:00:00Z'));

    const reset = buildTaskDashboardTaskSwitchReset('UTC');

    expect(reset.dimSelected).toEqual([]);
    expect(reset.activeGran).toBeUndefined();
    expect(reset.submitted).toBeNull();
    expect(reset.rangeMode).toEqual({ kind: 'relative', durationMs: 7 * 24 * 60 * 60 * 1000 });
    expect(reset.filter.range[1].toISOString()).toBe('2026-07-30T12:00:00.000Z');
  });

  it('builds a stable signature for the same submitted query to support quick tab dedupe', () => {
    const filter = {
      range: [dayjs('2026-07-01T00:00:00Z'), dayjs('2026-07-02T00:00:00Z')] as [dayjs.Dayjs, dayjs.Dayjs],
      weekdays: [1, 2],
      hours: [8, 9],
      compare: true,
    };
    const submitted = buildSubmittedTaskDashboardQuery(filter, {
      productIds: ['product-a'],
      systemTimezone: 'UTC',
    });

    expect(taskDashboardQuerySignature('task-1', submitted)).toBe(
      taskDashboardQuerySignature('task-1', { ...submitted }),
    );
  });

  it('calculates restored query throttle delay', () => {
    expect(restoredQueryDelayMs('2026-07-30T11:59:59.000Z', Date.parse('2026-07-30T12:00:00.000Z'))).toBe(1_000);
    expect(restoredQueryDelayMs('2026-07-30T11:59:40.000Z', Date.parse('2026-07-30T12:00:00.000Z'))).toBe(0);
  });
});
