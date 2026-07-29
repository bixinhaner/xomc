import dayjs from 'dayjs';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { usePmPageStateStore, type PmPageStateSnapshot } from '@core/store/pmPageStateStore';
import {
  buildDeviceViewStateSnapshot,
  buildDeviceViewSubmittedQuery,
  deviceViewQuerySignature,
  PM_DEVICE_VIEW_PAGE_KEY,
  restoredDeviceViewQueryDelayMs,
  restoreDeviceViewState,
} from './deviceViewState';

function baseFilter() {
  return {
    range: [dayjs('2026-07-01T00:00:00Z'), dayjs('2026-07-02T00:00:00Z')] as [dayjs.Dayjs, dayjs.Dayjs],
    weekdays: [1, 2, 3],
    hours: [8, 9],
    compare: true,
  };
}

function snapshotOf(
  input: Parameters<typeof buildDeviceViewStateSnapshot>[0],
): PmPageStateSnapshot {
  return {
    ...buildDeviceViewStateSnapshot(input),
    savedAt: '2026-07-29T00:00:00.000Z',
  };
}

afterEach(() => {
  vi.useRealTimers();
  usePmPageStateStore.setState({ pages: {} });
  sessionStorage.clear();
});

describe('deviceViewState', () => {
  it('restores edit fields and a submitted query snapshot', () => {
    const filter = baseFilter();
    const submitted = buildDeviceViewSubmittedQuery({
      tech: 'nr',
      deviceSns: ['GNB00001'],
      metricPaths: ['K001'],
      granularity: 'hourly',
      filter,
      allowedLdns: ['Cell=1'],
      systemTimezone: 'UTC',
    });

    const restored = restoreDeviceViewState(
      snapshotOf({
        tech: 'nr',
        deviceSns: ['GNB00001'],
        cellSel: { GNB00001: ['Cell=1'] },
        metricPaths: ['K001'],
        metricsTouched: true,
        granularity: 'hourly',
        rangeTouched: true,
        filter,
        submitted,
      }),
      'UTC',
    );

    expect(restored.tech).toBe('nr');
    expect(restored.deviceSns).toEqual(['GNB00001']);
    expect(restored.cellSel).toEqual({ GNB00001: ['Cell=1'] });
    expect(restored.metricPaths).toEqual(['K001']);
    expect(restored.filter.weekdays).toEqual([1, 2, 3]);
    expect(restored.filter.hours).toEqual([8, 9]);
    expect(restored.filter.compare).toBe(true);
    expect(restored.submitted?.tech).toBe('nr');
    expect(restored.submitted?.allowedLdns).toEqual(['Cell=1']);
    expect(restored.shouldRestoreQuery).toBe(true);
  });

  it('does not restore a submitted query when conditions were edited but not plotted', () => {
    const restored = restoreDeviceViewState(
      snapshotOf({
        tech: 'lte',
        deviceSns: ['ENB00001'],
        cellSel: {},
        metricPaths: ['K001'],
        metricsTouched: true,
        granularity: '15min',
        rangeTouched: true,
        filter: baseFilter(),
        submitted: null,
      }),
      'UTC',
    );

    expect(restored.submitted).toBeNull();
    expect(restored.shouldRestoreQuery).toBe(false);
  });

  it('keeps submitted query stable when edit fields changed after plotting', () => {
    const submittedFilter = baseFilter();
    const editedFilter = {
      range: [dayjs('2026-07-03T00:00:00Z'), dayjs('2026-07-04T00:00:00Z')] as [dayjs.Dayjs, dayjs.Dayjs],
      weekdays: [4],
      hours: [10],
      compare: false,
    };
    const submitted = buildDeviceViewSubmittedQuery({
      tech: 'lte',
      deviceSns: ['ENB00001'],
      metricPaths: ['K001'],
      granularity: 'hourly',
      filter: submittedFilter,
      allowedLdns: ['Cell=1'],
      systemTimezone: 'UTC',
    });

    const restored = restoreDeviceViewState(
      snapshotOf({
        tech: 'lte',
        deviceSns: ['ENB00002'],
        cellSel: { ENB00002: ['Cell=9'] },
        metricPaths: ['K002'],
        metricsTouched: true,
        granularity: 'daily',
        rangeTouched: true,
        filter: editedFilter,
        submitted,
      }),
      'UTC',
    );

    expect(restored.filter.weekdays).toEqual([4]);
    expect(restored.metricPaths).toEqual(['K002']);
    expect(restored.submitted?.metricPaths).toEqual(['K001']);
    expect(restored.submitted?.startTime).toBe(submitted.startTime);
    expect(restored.submitted?.weekdays).toEqual([1, 2, 3]);
    expect(restored.submitted?.allowedLdns).toEqual(['Cell=1']);
  });

  it('recalculates restored relative submitted time from current time', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-29T12:00:00Z'));
    const initialFilter = {
      range: [dayjs('2026-07-29T10:00:00Z'), dayjs('2026-07-29T12:00:00Z')] as [dayjs.Dayjs, dayjs.Dayjs],
      weekdays: [1],
      hours: [8],
      compare: false,
    };
    const snapshot = snapshotOf({
      tech: 'lte',
      deviceSns: ['ENB00001'],
      cellSel: {},
      metricPaths: ['K001'],
      metricsTouched: true,
      granularity: 'hourly',
      rangeTouched: false,
      filter: initialFilter,
      submitted: buildDeviceViewSubmittedQuery({
        tech: 'lte',
        deviceSns: ['ENB00001'],
        metricPaths: ['K001'],
        granularity: 'hourly',
        filter: initialFilter,
        allowedLdns: [],
        systemTimezone: 'UTC',
      }),
    });

    vi.setSystemTime(new Date('2026-07-30T12:00:00Z'));
    const restored = restoreDeviceViewState(snapshot, 'UTC');

    expect(restored.filter.range[1].toISOString()).toBe('2026-07-30T12:00:00.000Z');
    expect(restored.submitted?.endTime).toBe('2026-07-30T12:00:00Z');
  });

  it('keeps the submitted technology in query signatures and persisted store data', () => {
    const submitted = buildDeviceViewSubmittedQuery({
      tech: 'nr',
      deviceSns: ['GNB00001'],
      metricPaths: ['K001'],
      granularity: 'daily',
      filter: baseFilter(),
      allowedLdns: [],
      systemTimezone: 'UTC',
    });
    const state = buildDeviceViewStateSnapshot({
      tech: 'lte',
      deviceSns: ['ENB00001'],
      cellSel: {},
      metricPaths: ['K002'],
      metricsTouched: true,
      granularity: '15min',
      rangeTouched: true,
      filter: baseFilter(),
      submitted,
    });

    expect(deviceViewQuerySignature(submitted)).toContain('"tech":"nr"');
    usePmPageStateStore.getState().savePageState(PM_DEVICE_VIEW_PAGE_KEY, state);
    expect(
      (usePmPageStateStore.getState().getPageState(PM_DEVICE_VIEW_PAGE_KEY)?.filters?.submitted as { tech?: string }).tech,
    ).toBe('nr');
  });

  it('calculates restored query throttle delay', () => {
    expect(restoredDeviceViewQueryDelayMs('2026-07-30T11:59:58.000Z', Date.parse('2026-07-30T12:00:00.000Z'))).toBe(8_000);
    expect(restoredDeviceViewQueryDelayMs('2026-07-30T11:59:40.000Z', Date.parse('2026-07-30T12:00:00.000Z'))).toBe(0);
  });
});
