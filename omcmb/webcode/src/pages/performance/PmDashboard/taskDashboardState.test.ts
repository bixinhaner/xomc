import dayjs from 'dayjs';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { usePmPageStateStore, type PmPageStateSnapshot } from '@core/store/pmPageStateStore';
import {
  buildDefaultTaskDashboardFilter,
  buildDefaultTaskDashboardRange,
  buildSubmittedTaskDashboardQuery,
  defaultTaskDashboardRangeMode,
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
  it('builds default ranges with the same natural bucket alignment as KPI query presets', () => {
    const now = new Date('2026-07-23T06:23:12.000Z'); // Asia/Shanghai 2026-07-23 14:23:12

    expect(buildDefaultTaskDashboardRange('Asia/Shanghai', '15min', now).map((d) => d.format())).toEqual([
      '2026-07-23T11:15:00+08:00',
      '2026-07-23T14:15:00+08:00',
    ]);
    expect(buildDefaultTaskDashboardRange('Asia/Shanghai', 'hourly', now).map((d) => d.format())).toEqual([
      '2026-07-22T14:00:00+08:00',
      '2026-07-23T14:00:00+08:00',
    ]);
    expect(buildDefaultTaskDashboardRange('Asia/Shanghai', 'daily', now).map((d) => d.format())).toEqual([
      '2026-07-16T00:00:00+08:00',
      '2026-07-23T00:00:00+08:00',
    ]);
  });

  it('aligns submitted relative windows so empty charts cannot inherit query-time minutes or seconds', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-23T06:23:12.000Z'));
    const filter = {
      range: [
        dayjs('2026-07-22T14:23:12+08:00'),
        dayjs('2026-07-23T14:23:12+08:00'),
      ] as [dayjs.Dayjs, dayjs.Dayjs],
      weekdays: [1, 2, 3],
      hours: [8, 9],
      compare: true,
    };

    const submitted = buildSubmittedTaskDashboardQuery(filter, {
      systemTimezone: 'Asia/Shanghai',
      granularity: 'hourly',
      rangeMode: defaultTaskDashboardRangeMode('Asia/Shanghai', 'hourly'),
    });

    expect(submitted.startISO).toBe('2026-07-22T14:00:00+08:00');
    expect(submitted.endISO).toBe('2026-07-23T14:00:00+08:00');
    expect(submitted.prevStartISO).toBe('2026-07-21T14:00:00+08:00');
    expect(submitted.prevEndISO).toBe('2026-07-22T14:00:00+08:00');
    expect(submitted.rangeStartMs).toBe(dayjs('2026-07-22T14:00:00+08:00').valueOf());
    expect(submitted.rangeEndMs).toBe(dayjs('2026-07-23T14:00:00+08:00').valueOf());
  });

  it('aligns submitted absolute windows to full natural buckets only', () => {
    const filter = {
      range: [
        dayjs('2026-07-22T14:23:12+08:00'),
        dayjs('2026-07-22T16:23:12+08:00'),
      ] as [dayjs.Dayjs, dayjs.Dayjs],
      weekdays: [1, 2, 3],
      hours: [8, 9],
      compare: true,
    };

    const submitted = buildSubmittedTaskDashboardQuery(filter, {
      systemTimezone: 'Asia/Shanghai',
      granularity: 'hourly',
      rangeMode: { kind: 'absolute' },
    });

    expect(submitted.startISO).toBe('2026-07-22T15:00:00+08:00');
    expect(submitted.endISO).toBe('2026-07-22T16:00:00+08:00');
    expect(submitted.prevStartISO).toBe('2026-07-22T14:00:00+08:00');
    expect(submitted.prevEndISO).toBe('2026-07-22T15:00:00+08:00');
    expect(submitted.rangeStartMs).toBe(dayjs('2026-07-22T15:00:00+08:00').valueOf());
    expect(submitted.rangeEndMs).toBe(dayjs('2026-07-22T16:00:00+08:00').valueOf());
  });

  it('turns a manual daily range with no full day into an empty submitted window', () => {
    const filter = {
      range: [
        dayjs('2026-08-02T08:00:00Z'),
        dayjs('2026-08-03T06:06:00Z'),
      ] as [dayjs.Dayjs, dayjs.Dayjs],
      weekdays: [0, 1, 2, 3, 4, 5, 6],
      hours: Array.from({ length: 24 }, (_, i) => i),
      compare: false,
    };

    const submitted = buildSubmittedTaskDashboardQuery(filter, {
      systemTimezone: 'UTC',
      granularity: 'daily',
      rangeMode: { kind: 'absolute' },
    });

    expect(submitted.startISO).toBe('2026-08-03T00:00:00Z');
    expect(submitted.endISO).toBe('2026-08-03T00:00:00Z');
    expect(submitted.rangeStartMs).toBe(submitted.rangeEndMs);
  });

  it.each([
    {
      granularity: 'hourly' as const,
      expectedStart: '2026-07-29T12:00:00.000Z',
      expectedEnd: '2026-07-30T12:00:00.000Z',
    },
    {
      granularity: 'daily' as const,
      expectedStart: '2026-07-23T00:00:00.000Z',
      expectedEnd: '2026-07-30T00:00:00.000Z',
    },
  ])(
    'restores submitted relative range by saved effective default granularity when activeGran is empty: $granularity',
    ({ granularity, expectedStart, expectedEnd }) => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2026-07-30T12:23:12.000Z'));
      const staleFilter = {
        range: [
          dayjs('2026-07-29T12:23:12Z'),
          dayjs('2026-07-30T12:23:12Z'),
        ] as [dayjs.Dayjs, dayjs.Dayjs],
        weekdays: [1, 2],
        hours: [8, 9],
        compare: true,
      };
      const submitted = buildSubmittedTaskDashboardQuery(staleFilter, {
        systemTimezone: 'UTC',
      });

      const restored = restoreTaskDashboardState(
        snapshotOf({
          taskId: 'task-1',
          filter: staleFilter,
          dimSelected: [],
          activeGran: undefined,
          effectiveGran: granularity,
          rangeMode: { kind: 'relative', durationMs: 24 * 60 * 60 * 1000 },
          submitted,
        }),
        'UTC',
      );

      expect(restored.activeGran).toBeUndefined();
      expect(restored.submitted?.granularity).toBe(granularity);
      expect(restored.submitted?.startISO).toBe(expectedStart.replace('.000Z', 'Z'));
      expect(restored.submitted?.endISO).toBe(expectedEnd.replace('.000Z', 'Z'));
      expect(restored.submitted?.rangeStartMs).toBe(Date.parse(expectedStart));
      expect(restored.submitted?.rangeEndMs).toBe(Date.parse(expectedEnd));
      expect(restored.submitted?.prevStartISO.endsWith(':23:12Z')).toBe(false);
      expect(restored.submitted?.prevEndISO.endsWith(':23:12Z')).toBe(false);
    },
  );

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

  it('realigns a restored absolute submitted query when a saved effective granularity is available', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-30T12:00:00Z'));
    const filter = {
      range: [
        dayjs('2026-07-22T14:23:12Z'),
        dayjs('2026-07-22T16:23:12Z'),
      ] as [dayjs.Dayjs, dayjs.Dayjs],
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
        effectiveGran: 'hourly',
        rangeMode: { kind: 'absolute' },
        submitted,
      }),
      'UTC',
    );

    expect(restored.submitted?.granularity).toBe('hourly');
    expect(restored.submitted?.startISO).toBe('2026-07-22T15:00:00Z');
    expect(restored.submitted?.endISO).toBe('2026-07-22T16:00:00Z');
    expect(restored.submitted?.prevStartISO).toBe('2026-07-22T14:00:00Z');
    expect(restored.submitted?.prevEndISO).toBe('2026-07-22T15:00:00Z');
    expect(restored.submitted?.objectLdns).toEqual(['Band=42']);
  });

  it('builds a task switch reset state that clears submitted query', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-30T12:00:00Z'));

    const reset = buildTaskDashboardTaskSwitchReset('UTC');

    expect(reset.dimSelected).toEqual([]);
    expect(reset.activeGran).toBeUndefined();
    expect(reset.submitted).toBeNull();
    expect(reset.rangeMode).toEqual({ kind: 'relative', durationMs: 7 * 24 * 60 * 60 * 1000 });
    expect(reset.filter.range[0].toISOString()).toBe('2026-07-23T00:00:00.000Z');
    expect(reset.filter.range[1].toISOString()).toBe('2026-07-30T00:00:00.000Z');
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
